package smsc

import (
	"context"
	"errors"
	log "log/slog"
	"strings"
	"sync"
	"time"

	"github.com/linxGnu/gosmpp"
	"github.com/linxGnu/gosmpp/data"
	"github.com/linxGnu/gosmpp/pdu"
)

const (
	// TypeOfNumber (TON)
	tonUNKNOWN           = 0
	tonINTERNATIONAL     = 1
	tonNATIONAL          = 2
	tonNETWORK_SPECIFIC  = 3
	tonSUBSCRIBER_NUMBER = 4
	tonALPHANUMERIC      = 5
	tonABBREVIATED       = 7

	// NumberingPlanIndicator (NPI)
	npiUNKNOWN       = 0
	npiISDN          = 1
	npiDATA          = 3
	npiTELEX         = 4
	npiLAND_MOBILE   = 6
	npiNATIONAL      = 8
	npiPRIVATE       = 9
	npiERMES         = 10
	npiINTERNATIONAL = 14
	npiWAP           = 18
)

type SMSCenter struct {
	trans  *gosmpp.Session
	auth   gosmpp.Auth
	dryRun bool
	//
	config        *ConfigSMSC
	resultTimeout time.Duration
	sendMu        sync.Mutex
	restartMu     sync.Mutex
	waitMu        sync.Mutex
	waitChs       map[*pdu.SubmitSM]chan string
}

func NewSMSCenter(config *ConfigSMSC) *SMSCenter {
	resultT, err := time.ParseDuration(config.ResultTimeout)
	if err != nil || resultT <= 0 {
		resultT = 5 * time.Second
	}
	return &SMSCenter{
		dryRun:        config.DryRun,
		config:        config,
		resultTimeout: resultT,
		waitChs:       make(map[*pdu.SubmitSM]chan string),
		auth:          gosmpp.Auth{SMSC: config.SMSC, SystemID: config.SystemID, Password: config.Password, SystemType: config.SystemType},
	}
}

func (s *SMSCenter) GetName() string {
	return "SMSCenter"
}

func (s *SMSCenter) Start(ctx context.Context) error {
	s.restartMu.Lock()
	defer s.restartMu.Unlock()
	return s.startLocked(ctx)
}

func (s *SMSCenter) startLocked(ctx context.Context) error {
	var err error
	if s.dryRun {
		log.Warn("DryRun")
	} else {
		log.Debug("Start", log.String("SMSC", s.auth.SMSC))
		s.trans, err = gosmpp.NewSession(
			gosmpp.TRXConnector(gosmpp.NonTLSDialer, s.auth),
			gosmpp.Settings{
				EnquireLink: 60 * time.Second,
				ReadTimeout: 2 * time.Minute,
				OnSubmitError: func(p pdu.PDU, err error) {
					log.Error("SubmitPDU error", log.String("message", err.Error()))
				},
				OnReceivingError: func(err error) {
					log.Error("Receiving PDU/Network", log.String("message", err.Error()))
				},
				OnRebindingError: func(err error) {
					log.Error("Rebinding but error", log.String("message", err.Error()))
				},
				OnClosed: func(state gosmpp.State) {
					log.Error("OnClosed", log.String("state", state.String()))
				},
				// OnPDU: func(pdu pdu.PDU, responded bool) {log.Debug("OnPDU", log.Any("pdu", pdu))},
				// OnPDU is not invoked while WindowedRequestTracking is set (legacy hook).
				OnPDU: s.handlePDU(),
				// OnAllPDU is not invoked while WindowedRequestTracking is set (legacy hook).
				OnAllPDU: func(pdu pdu.PDU) (responsePdu pdu.PDU, closeBind bool) {
					log.Debug("OnAllPDU", log.Any("pdu", pdu))
					return pdu, false
				},

				WindowedRequestTracking: &gosmpp.WindowedRequestTracking{
					OnExpectedPduResponse:   s.handleExpectedPduResponse(),
					OnReceivedPduRequest:    s.handleReceivedPduRequest(),
					OnUnexpectedPduResponse: s.handleUnexpectedPduResponse(),
					OnClosePduRequest:       s.handleOnClosePduRequest(),
					OnExpiredPduRequest:     s.handleExpirePduRequest(),
					PduExpireTimeOut:        30 * time.Second,
					ExpireCheckTimer:        10 * time.Second,
					// the library multiplies this value by time.Millisecond internally
					StoreAccessTimeOut: 5000 * time.Millisecond,
					EnableAutoRespond:  false,
					MaxWindowSize:      30,
				},
			},
			5*time.Second,
			// gosmpp.WithRequestStore(NewCustomStore()),
		)
	}
	return err
}

// handlePDU returns the OnPDU handler. It is not invoked while
// WindowedRequestTracking is set (legacy hook).
func (s *SMSCenter) handlePDU() func(pdu.PDU, bool) {

	return func(p pdu.PDU, _ bool) {
		switch pd := p.(type) {
		case *pdu.SubmitSMResp:
			log.Debug("SubmitSMResp", log.Any("resp", pd))

		case *pdu.GenericNack:
			log.Debug("GenericNack Received")

		case *pdu.EnquireLinkResp:
			log.Debug("EnquireLinkResp Received")

		case *pdu.DataSM:
			log.Debug("DataSM", log.Any("pdu", pd))

		case *pdu.DeliverSM:
			s, e := pd.Message.GetMessage()
			log.Debug("DeliverSM", log.Int("messageSize", len(s)), log.Any("err", e))
			// region concatenated sms (sample code)

		// endregion

		default:
			log.Debug("Pdu", log.Any("pdu", pd))
		}
	}
}

func (s *SMSCenter) Stop() error {
	s.restartMu.Lock()
	defer s.restartMu.Unlock()
	return s.stopLocked()
}

func (s *SMSCenter) stopLocked() error {
	if !s.dryRun && s.trans != nil {
		return s.trans.Close()
	}
	return nil
}

func (s *SMSCenter) Restart() error {
	log.Debug("Restart")
	s.restartMu.Lock()
	defer s.restartMu.Unlock()
	if err := s.stopLocked(); err != nil {
		log.Warn("Stop", log.Any("error", err))
	}
	return s.startLocked(context.Background())
}

func (s *SMSCenter) handleExpirePduRequest() func(pdu.PDU) bool {
	return func(p pdu.PDU) bool {
		switch p.(type) {

		case *pdu.SubmitSM:
			log.Debug("Expired SubmitSM", log.Any("pdu", p))

		case *pdu.EnquireLink:
			log.Debug("Expired EnquireLink", log.Any("pdu", p))
			return true // if the enquire_link expired, usually means the bind is stale

		case *pdu.DataSM:
			log.Debug("Expired DataSM", log.Any("pdu", p))

		default:
			log.Debug("Expired", log.Any("pdu", p))
		}

		return false
	}
}

func (s *SMSCenter) handleReceivedPduRequest() func(pdu.PDU) (pdu.PDU, bool) {
	return func(p pdu.PDU) (pdu.PDU, bool) {
		switch pd := p.(type) {
		case *pdu.Unbind:
			log.Debug("Unbind Received")
			return pd.GetResponse(), true

		case *pdu.GenericNack:
			log.Debug("GenericNack Received")

		case *pdu.EnquireLinkResp:
			log.Debug("EnquireLinkResp Received")

		case *pdu.EnquireLink:
			log.Debug("EnquireLink Received")
			return pd.GetResponse(), false

		case *pdu.DataSM:
			log.Debug("DataSM Received")
			return pd.GetResponse(), false

		case *pdu.DeliverSM:
			deliver := p.(*pdu.DeliverSM)
			// log.Debug("DeliverSM Received", log.Any("Deliver", deliver))
			if m, err := deliver.Message.GetMessage(); err == nil {
				log.Debug("Message", log.Int("size", len(m)))
			}
			return pd.GetResponse(), false

		default:
			log.Debug("handleReceivedPduRequest", log.Any("default", p))
		}
		return nil, false
	}
}

func (s *SMSCenter) handleOnClosePduRequest() func(pdu.PDU) {
	return func(p pdu.PDU) {
		switch p.(type) {
		case *pdu.Unbind:
			resp := p.(*pdu.Unbind)
			log.Debug("OnClose Unbind", log.Any("Unbind", resp))

		case *pdu.SubmitSM:
			resp := p.(*pdu.SubmitSM)
			log.Debug("OnClose SubmitSM", log.Any("SubmitSM", resp))

		case *pdu.EnquireLink:
			resp := p.(*pdu.EnquireLink)
			log.Debug("OnClose EnquireLink", log.Any("EnquireLink", resp))

		case *pdu.DataSM:
			resp := p.(*pdu.DataSM)
			log.Debug("OnClose DataSM", log.Any("DataSM", resp))
		default:
			log.Debug("handleOnClosePduRequest", log.Any("default", p))
		}
	}
}

func (s *SMSCenter) handleUnexpectedPduResponse() func(pdu.PDU) {
	return func(p pdu.PDU) {
		log.Debug("handleUnexpectedPduResponse", log.Any("response", p))
	}
}

func (s *SMSCenter) handleExpectedPduResponse() func(response gosmpp.Response) {
	return func(response gosmpp.Response) {
		switch response.PDU.(type) {
		case *pdu.UnbindResp:
			resp := response.PDU.(*pdu.UnbindResp)
			log.Debug("UnbindResp Received", log.Any("response", resp))

		case *pdu.SubmitSMResp:
			s.handleSubmitSMResponse(response)

		case *pdu.EnquireLinkResp:
		default:
			log.Debug("default", log.Any("response", response))
		}
	}
}

func (s *SMSCenter) handleSubmitSMResponse(response gosmpp.Response) {
	resp := response.PDU.(*pdu.SubmitSMResp)
	log.Debug("SubmitSMResp Received", log.Any("resp", resp))
	orig, ok := response.OriginalRequest.PDU.(*pdu.SubmitSM)
	if !ok {
		log.Debug("SubmitSMResp without original request", log.String("messageID", resp.MessageID))
		return
	}
	s.waitMu.Lock()
	ch, okWait := s.waitChs[orig]
	if okWait {
		delete(s.waitChs, orig)
	}
	s.waitMu.Unlock()
	if okWait {
		select {
		case ch <- resp.MessageID:
		default:
		}
	} else {
		log.Debug("SubmitSMResp without waiter", log.String("messageID", resp.MessageID))
	}
}

func (s *SMSCenter) SendMessage(from, to, text string) (string, error) {
	var lastErr error
	for i := 1; i < 6; i++ {
		if result, err := s.sendMessage(from, to, text); err == nil {
			return result, err
		} else {
			lastErr = err
			log.Debug("SendMessage", log.Int("retry", i))
			// gosmpp's exact error text: matching by prefix is intentional
			if strings.HasPrefix(err.Error(), "connection is closing, can not send PDU to SMSC") {
				log.Debug("Restarting server")
				if err = s.Restart(); err != nil {
					return "", err
				}
			} else if i < 5 {
				time.Sleep(time.Duration(i) * 100 * time.Millisecond)
			}
		}
	}
	if lastErr != nil {
		return "", lastErr
	}
	return "", errors.New("can't send message")
}

func (s *SMSCenter) sendMessage(from, to, text string) (string, error) {
	if s.dryRun {
		return "DryRun", nil
	}
	s.sendMu.Lock()
	defer s.sendMu.Unlock()
	partChs, err := s.sendSMS(from, to, text)
	if err != nil {
		return "", err
	}
	resultCh := make(chan string, 1)
	done := make(chan struct{})
	defer close(done)
	for _, pc := range partChs {
		go func(pc chan string) {
			select {
			case res := <-pc:
				select {
				case resultCh <- res:
				case <-done:
				}
			case <-done:
			}
		}(pc)
	}
	select {
	case res := <-resultCh:
		log.Debug("MsgID", log.String("value", res))
		s.dropWaitingParts()
		return res, nil
	case <-time.After(s.resultTimeout):
		s.dropWaitingParts()
		return "", errors.New("send message timeout while result message is waiting")
	}
}

func (s *SMSCenter) dropWaitingParts() {
	s.waitMu.Lock()
	defer s.waitMu.Unlock()
	for k := range s.waitChs {
		delete(s.waitChs, k)
	}
}

func (s *SMSCenter) dropWaitingPartsFrom(messages []*pdu.SubmitSM, from int) {
	s.waitMu.Lock()
	defer s.waitMu.Unlock()
	for _, msg := range messages[from:] {
		delete(s.waitChs, msg)
	}
}

func (s *SMSCenter) sendSMS(from, to, text string) ([]chan string, error) {
	log.Debug("SendSMS", log.String("from", from), log.String("to", to), log.Int("textSize", len(text)))
	s.restartMu.Lock()
	trans := s.trans
	s.restartMu.Unlock()
	if trans == nil {
		return nil, errors.New("smsc session is not started")
	}
	if from == "" {
		from = s.config.SourceAddr
	}
	srcAddr := pdu.NewAddress()
	srcAddr.SetTon(tonALPHANUMERIC)
	srcAddr.SetNpi(npiUNKNOWN)
	if err := srcAddr.SetAddress(from); err != nil {
		log.Error("SetAddress.from", log.String("message", err.Error()))
		return nil, err
	}
	destAddr := pdu.NewAddress()
	destAddr.SetTon(tonINTERNATIONAL)
	destAddr.SetNpi(npiISDN)
	if err := destAddr.SetAddress(to); err != nil {
		log.Error("SetAddress.to", log.String("message", err.Error()))
		return nil, err
	}
	submitSM := pdu.NewSubmitSM().(*pdu.SubmitSM)
	submitSM.SourceAddr = srcAddr
	submitSM.DestAddr = destAddr
	//if err = submitSM.Message.SetMessageWithEncoding(text, data.UCS2); err == nil {
	enc := data.UCS2
	if len(data.ValidateGSM7String(text)) == 0 {
		enc = data.GSM7BIT
	}
	if err := submitSM.Message.SetLongMessageWithEnc(text, enc); err != nil {
		log.Error("SetLongMessageWithEnc", log.String("message", err.Error()))
		return nil, err
	}
	submitSM.ReplaceIfPresentFlag = 0
	submitSM.RegisteredDelivery = 1
	submitSM.PriorityFlag = 1
	submitSM.ProtocolID = 0
	submitSM.EsmClass = 0
	messages, err := submitSM.Split()
	if err != nil {
		log.Error("Split", log.String("message", err.Error()))
		return nil, err
	}
	partChs := make([]chan string, 0, len(messages))
	for i, msg := range messages {
		if message, merr := msg.Message.GetMessage(); merr == nil {
			log.Debug("message", log.Int("part", i), log.Int("size", len(message)))
		}
		partCh := make(chan string, 1)
		s.waitMu.Lock()
		s.waitChs[msg] = partCh
		s.waitMu.Unlock()
		if err := trans.Transceiver().Submit(msg); err != nil {
			log.Error("Submit", log.String("message", err.Error()))
			s.dropWaitingPartsFrom(messages, i)
			return nil, err
		}
		partChs = append(partChs, partCh)
	}
	return partChs, nil
}

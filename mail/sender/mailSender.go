package sender

import (
	"context"
	"errors"
	"fmt"
	log "log/slog"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/wneessen/go-mail"
)

const defaultSendTimeout = 30 * time.Second

var errSendTimeout = errors.New("smtp send timed out")

// mailSenderState holds the persistent SMTP connection shared by a MailSender.
type mailSenderState struct {
	mu      sync.Mutex
	client  *mail.Client
	host    string
	port    int
	timeout time.Duration
}

type MailSender struct {
	config *ConfigMailSender
	st     *mailSenderState
}

func NewMailSender(config *ConfigMailSender) *MailSender {
	return &MailSender{config: config, st: &mailSenderState{}}
}

func (m *MailSender) GetName() string {
	return "MailSender"
}

func (m *MailSender) Start(ctx context.Context) error {
	if m.config.Server == "" {
		return errors.New("mail server is not configured")
	}
	st := m.st
	if st == nil {
		st = &mailSenderState{}
		m.st = st
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	host, port, err := m.hostAndPort()
	if err != nil {
		return err
	}
	st.host, st.port = host, port
	return nil
}

func (m *MailSender) Stop() error {
	if m.st == nil {
		return nil
	}
	m.st.mu.Lock()
	defer m.st.mu.Unlock()
	if m.st.client != nil {
		err := m.st.client.Close()
		m.st.client = nil
		return err
	}
	return nil
}

//////////////////////////////////////////////////////

func (m *MailSender) SendMessage(sender, recipient, message, subj, msgType string) error {
	if m.config.DryRun {
		log.Info("DryRun, mail is not sent", log.String("to", recipient), log.String("subject", subj))
		return nil
	}
	mm := mail.NewMsg()
	if err := mm.From(sender); err != nil {
		return err
	}
	if err := mm.To(recipient); err != nil {
		return err
	}
	mm.Subject(subj)
	var contentType mail.ContentType
	if strings.ToLower(msgType) == "html" {
		contentType = mail.TypeTextHTML
	} else {
		contentType = mail.TypeTextPlain
	}
	mm.SetBodyString(contentType, message)

	st := m.st
	if st == nil {
		st = &mailSenderState{}
	}
	st.mu.Lock()
	defer st.mu.Unlock()

	host, port, timeout, err := m.endpointLocked(st)
	if err != nil {
		return err
	}
	client, err := m.dialLocked(st, host, port, timeout)
	if err != nil {
		return err
	}
	if err := sendWithTimeout(client, mm, timeout); err != nil {
		if errors.Is(err, errSendTimeout) {
			m.closeLocked(st)
			return err
		}
		if !isConnLevelErr(err) {
			return err
		}
		// the connection may have been lost: drop it, redial once and retry
		m.closeLocked(st)
		if client, err = m.dialLocked(st, host, port, timeout); err != nil {
			return err
		}
		if err = sendWithTimeout(client, mm, timeout); err != nil {
			m.closeLocked(st)
		}
		return err
	}
	return nil
}

func (m *MailSender) endpointLocked(st *mailSenderState) (string, int, time.Duration, error) {
	if st.host == "" {
		host, port, err := m.hostAndPort()
		if err != nil {
			return "", 0, 0, err
		}
		st.host, st.port = host, port
	}
	if st.timeout <= 0 {
		timeout, err := time.ParseDuration(m.config.Timeout)
		if err != nil || timeout <= 0 {
			if err != nil {
				log.Error("invalid mail Timeout", log.String("Timeout", m.config.Timeout), log.String("Message", err.Error()))
			}
			timeout = defaultSendTimeout
		}
		st.timeout = timeout
	}
	return st.host, st.port, st.timeout, nil
}

func sendWithTimeout(client *mail.Client, mm *mail.Msg, timeout time.Duration) error {
	done := make(chan error, 1)
	go func() {
		done <- client.Send(mm)
	}()
	select {
	case err := <-done:
		return err
	case <-time.After(timeout):
		return errSendTimeout
	}
}

func isConnLevelErr(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, mail.ErrNoActiveConnection) {
		return true
	}
	var se *mail.SendError
	if errors.As(err, &se) {
		return se.Reason == mail.ErrConnCheck
	}
	var nerr net.Error
	return errors.As(err, &nerr)
}

// dialLocked requires st.mu; returns a dialed client, reusing the persistent
// connection when it is still alive.
func (m *MailSender) dialLocked(st *mailSenderState, host string, port int, timeout time.Duration) (*mail.Client, error) {
	if st.client != nil {
		return st.client, nil
	}
	policy := mail.TLSOpportunistic
	if m.config.WithSSL {
		policy = mail.TLSMandatory
	}
	client, err := mail.NewClient(
		host,
		mail.WithPort(port),
		mail.WithSMTPAuth(mail.SMTPAuthLogin),
		mail.WithUsername(m.config.User),
		mail.WithPassword(m.config.Password),
		mail.WithTLSPolicy(policy),
		mail.WithTimeout(timeout),
	)
	if err != nil {
		return nil, err
	}
	if err := client.DialWithContext(context.Background()); err != nil {
		return nil, err
	}
	st.client = client
	return client, nil
}

func (m *MailSender) closeLocked(st *mailSenderState) {
	if st.client != nil {
		_ = st.client.Close()
		st.client = nil
	}
}

func (m *MailSender) hostAndPort() (string, int, error) {
	host, portStr, err := net.SplitHostPort(m.config.Server)
	if err != nil {
		return "", 0, fmt.Errorf("invalid mail server address %q: %w", m.config.Server, err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return "", 0, fmt.Errorf("invalid mail server port %q: %w", portStr, err)
	}
	return host, port, nil
}

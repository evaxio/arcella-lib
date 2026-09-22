package trueConfClient

import (
	pkgTCCommon "axgit.vixiv.ru/snake/arcella-lib/tc"
	"axgit.vixiv.ru/snake/arcella-lib/utils"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	log "log/slog"
	"math/rand"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

const (
	wsMsgQueueFullTimeout = 5 * time.Second
	wsReadDeadline        = 60 * time.Second
	maxRestartBackoff     = 5 * time.Minute
)

var cPing = []byte("ping")

type WSMsg struct {
	Type int
	Text []byte
}

func (tc *TcClient) wsRestart() {
	if !tc.restartFlag.CompareAndSwap(false, true) {
		log.Warn("Already restarting !!!")
		return
	}
	defer tc.restartFlag.Store(false)
	log.Debug("****************** Restart ******************")
	count := 1
	for {
		if tc.stopped.Load() {
			log.Debug("Client is stopped, aborting restart")
			return
		}
		if err := tc.wsDisconnect(); err != nil {
			log.Error("wsDisconnect", log.String("message", err.Error()))
		}
		if count > 1 {
			backoff := time.Duration(count) * time.Minute
			if backoff > maxRestartBackoff {
				backoff = maxRestartBackoff
			}
			backoff = backoff + time.Duration(rand.Int63n(int64(backoff/2)))
			log.Debug("Sleep", log.Duration("backoff", backoff))
			select {
			case <-time.After(backoff):
			case <-tc.stopCtx.Done():
				return
			}
		}
		count = count + 1
		if err := tc.wsConnect(""); err != nil {
			log.Error("wsConnect", log.String("message", err.Error()))
			if tc.stopped.Load() {
				return
			}
		} else {
			log.Debug("Success", log.Int("count", count))
			break
		}
	}
	log.Debug("****************** Restart.end ******************")
}

func (tc *TcClient) wsDisconnect() error {
	log.Debug("wsDisconnect")
	tc.mu.Lock()
	if tc.cancel != nil {
		tc.cancel()
		tc.cancel = nil
	} else {
		log.Warn("cancel is null")
	}
	conn := tc.wsconn
	tc.wsconn = nil
	tc.connecting = false
	tc.mu.Unlock()
	if conn != nil {
		err := conn.Close()
		tc.wg.Wait()
		return err
	}
	return nil
}

// Deprecated: debug method; forces a reconnect after 15 seconds.
func (tc *TcClient) Test() {
	log.Debug("******************************************")
	log.Debug("************** INIT TEST *****************")
	log.Debug("******************************************")
	go func() {
		ticker := time.NewTimer(15 * time.Second)
		for {
			select {
			case <-ticker.C:
				log.Debug("TIME TO RESTART")
				if !tc.restartFlag.Load() {
					go tc.wsRestart()
				}
				ticker.Stop()
				ticker = nil
				return
			}
		}
	}()
}
func (tc *TcClient) wsConnect(token string) error {
	log.Debug("wsConnect")
	tc.mu.Lock()
	if tc.wsconn != nil {
		tc.mu.Unlock()
		return errors.New("already connected")
	}
	if tc.connecting {
		tc.mu.Unlock()
		return errors.New("connect already in progress")
	}
	tc.connecting = true
	baseCtx := tc.parentCtx
	if baseCtx == nil {
		baseCtx = context.Background()
	}
	tc.mu.Unlock()

	connCtx, cancel := context.WithCancel(baseCtx)
	if token == "" {
		tc.mu.Lock()
		if tc.token != "" && time.Now().Before(tc.tokenExpires) {
			token = tc.token
		}
		tc.mu.Unlock()
	}
	if token == "" {
		log.Debug("No token. Requesting new token")
		if err, accessToken := tc.GetNewToken(); err == nil {
			token = accessToken
		} else {
			tc.mu.Lock()
			tc.connecting = false
			tc.mu.Unlock()
			cancel()
			log.Error("GetNewToken", log.String("message", err.Error()))
			return err
		}
	}

	log.Debug("connecting", log.String("to", tc.wsurl))
	var err error
	var resp *http.Response

	headers := http.Header{}
	headers.Add("Sec-WebSocket-Protocol", "json.v1")
	dialer := websocket.Dialer{
		ReadBufferSize:    1024,
		WriteBufferSize:   1024,
		EnableCompression: false,
		HandshakeTimeout:  time.Second * 45,
		TLSClientConfig:   &tls.Config{InsecureSkipVerify: tc.config.InsecureSkipVerify},
	}
	var conn *websocket.Conn
	if conn, resp, err = dialer.DialContext(connCtx, tc.wsurl, headers); err != nil {
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
		log.Debug(err.Error(), log.Any("🚩🚩🚩 resp", resp))
		tc.mu.Lock()
		tc.connecting = false
		tc.mu.Unlock()
		cancel()
		return err
	}
	tc.mu.Lock()
	if tc.stopped.Load() {
		tc.connecting = false
		tc.mu.Unlock()
		cancel()
		_ = conn.Close()
		return errors.New("client is stopped")
	}
	tc.ctx, tc.cancel = connCtx, cancel
	tc.wsconn = conn
	tc.connecting = false
	tc.mu.Unlock()
	conn.SetReadLimit(1 << 20)
	_ = conn.SetReadDeadline(time.Now().Add(wsReadDeadline))
	conn.SetPingHandler(func(appData string) error {
		return conn.WriteControl(websocket.PongMessage, []byte(appData), time.Now().Add(5*time.Second))
	})
	tc.wg.Add(1)
	go func() {
		defer tc.wg.Done()
		tc.writeMsg()
	}()
	tc.wg.Add(1)
	go func() {
		defer tc.wg.Done()
		tc.readMsg()
	}()

	// https://trueconf.ru/docs/chatbot-connector/ru/connect-and-auth/#%D0%B0%D0%B2%D1%82%D0%BE%D1%80%D0%B8%D0%B7%D0%B0%D1%86%D0%B8%D1%8F-%D0%BF%D0%BE%D0%B4%D0%BA%D0%BB%D1%8E%D1%87%D0%B5%D0%BD%D0%B8%D0%B5
	jsonB := utils.ToJsonB(&pkgTCCommon.WSHeader{
		Type:   pkgTCCommon.MT_REQUEST,
		Id:     tc.msgId.Add(1),
		Method: "auth",
		Payload: utils.ToJsonB(&pkgTCCommon.WSAuthPayload{
			Token:         token,
			TokenType:     "JWT",
			ReceiveUnread: false,
		}),
	})
	// log.Debug("🔑 Sending auth message 🔑", log.String("json", string(jsonB)))
	if err := tc.pushWSMsg(WSMsg{websocket.TextMessage, jsonB}); err != nil {
		_ = tc.wsDisconnect()
		return err
	}
	// log.Debug("👀 Done 👀")
	return nil
}

type ResultData struct { // ResultData = {"id":575,"type":2,"payload":{"errorCode":300}}
	Id      int64       `json:"id"`
	Type    int         `json:"type"`
	Payload payloadData `json:"payload"`
}

type payloadData struct {
	ErrorCode int `json:"errorCode"`
}

func (tc *TcClient) readMsg() {
	tc.mu.Lock()
	conn := tc.wsconn
	tc.mu.Unlock()
	log.Debug("🌞Waiting for message🌞", log.String("Subprotocol", conn.Subprotocol()))
	for {
		if t, p, err := conn.ReadMessage(); err != nil {
			log.Error("🚩🚩🚩 read", log.String("message", err.Error()))
			if !tc.stopped.Load() && !tc.restartFlag.Load() {
				go tc.wsRestart()
			}
			return
		} else {
			_ = conn.SetReadDeadline(time.Now().Add(wsReadDeadline))
			if !tc.config.ShowAnswers {
				continue
			}
			log.Debug("🍥 msg", log.Int("type", t), log.Any("data", string(p)))
			var res ResultData
			if err = json.Unmarshal(p, &res); err == nil {
				if res.Payload.ErrorCode >= 300 {
					log.Error("Error found", log.Int("code", res.Payload.ErrorCode))
					if !tc.stopped.Load() && !tc.restartFlag.Load() {
						go tc.wsRestart()
					}
				}
			}
		}
	}
}

func (tc *TcClient) writeMsg() {
	tc.mu.Lock()
	conn := tc.wsconn
	tc.mu.Unlock()
	log.Debug("🍥 Waiting for message 🍥")
	ticker := time.NewTicker(25 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-tc.ctx.Done():
			log.Debug("👀 writeMsg done")
			return
		case <-ticker.C:
			if err := conn.WriteMessage(websocket.PingMessage, cPing); err != nil {
				log.Error("🚩ticker.WriteMessage", log.String("message", err.Error()))
				if !tc.stopped.Load() && !tc.restartFlag.Load() {
					go tc.wsRestart()
				}
				return
			}
		case msg, ok := <-tc.wsMsg:
			if ok {
				if tc.restartFlag.Load() || tc.stopped.Load() {
					log.Warn("Drop message: restart in progress or client is stopped", log.Int("type", msg.Type))
				} else {
					if err := conn.WriteMessage(msg.Type, msg.Text); err != nil {
						log.Error("🚩msg.WriteMessage", log.String("message", err.Error()))
						if !tc.stopped.Load() && !tc.restartFlag.Load() {
							go tc.wsRestart()
						}
						return
					}
				}
			} else {
				log.Debug("👀 write done")
				return
			}
		}
	}
}

func (tc *TcClient) pushWSMsg(m WSMsg) error {
	select {
	case tc.wsMsg <- m:
		return nil
	case <-time.After(wsMsgQueueFullTimeout):
		log.Error("ws message queue is full", log.Int("type", m.Type))
		return errors.New("ws message queue is full")
	}
}

func (tc *TcClient) SendMsg(msg, chatId, mode string) error {
	if tc.stopped.Load() {
		return errors.New("tc client is stopped")
	}
	if mode != "html" && mode != "text" {
		mode = "markdown"
	}
	jsonB := utils.ToJsonB(&pkgTCCommon.WSHeader{
		Id:     tc.msgId.Add(1),
		Type:   pkgTCCommon.MT_REQUEST,
		Method: "sendMessage",
		Payload: utils.ToJsonB(&pkgTCCommon.WSMessagePayload{
			ChatId: chatId,
			Content: pkgTCCommon.WSMessageContentPayload{
				Text:      msg,
				ParseMode: mode, // text, markdown, html
			},
		})})
	log.Debug("SendMsg to queue", log.String("json", string(jsonB)))
	if err := tc.pushWSMsg(WSMsg{Type: websocket.TextMessage, Text: jsonB}); err != nil {
		return err
	}
	log.Debug("Sent")
	return nil
}

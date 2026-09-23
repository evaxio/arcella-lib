package trueConfClient

import (
	pkgTCCommon "github.com/evaxio/arcella-lib/tc"
	"context"
	"errors"
	log "log/slog"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

type TcClient struct {
	config *ConfigTcClient
	//
	msgId  atomic.Uint32 // https://trueconf.ru/docs/chatbot-connector/ru/base-format-message/#%D0%BF%D0%B0%D1%80%D0%B0%D0%BC%D0%B5%D1%82%D1%80-id
	ctx    context.Context
	cancel context.CancelFunc
	mu     sync.Mutex
	wg     sync.WaitGroup
	wsconn *websocket.Conn
	wsurl  string
	//
	started      bool
	connecting   bool
	restartFlag  atomic.Bool
	token        string
	tokenIssued  time.Time
	tokenExpires time.Time
	parentCtx    context.Context
	stopped      atomic.Bool
	stopCtx      context.Context
	stopCancel   context.CancelFunc
	httpClient   *http.Client
	wsMsg        chan WSMsg
}

func NewTcClient(config *ConfigTcClient) *TcClient {
	wurl := url.URL{Scheme: "wss", Host: serverHost(config.Server), Path: "/websocket/chat_bot"}
	stopCtx, stopCancel := context.WithCancel(context.Background())
	return &TcClient{
		config:     config,
		wsurl:      wurl.String(),
		wsMsg:      make(chan WSMsg, 256),
		stopCtx:    stopCtx,
		stopCancel: stopCancel,
		httpClient: &http.Client{
			Timeout: time.Second * 10,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     time.Second * 90,
			},
		},
	}
}

func (tc *TcClient) GetName() string {
	return "TcClient"
}

func (tc *TcClient) Start(ctx context.Context) error {
	tc.mu.Lock()
	if tc.started {
		tc.mu.Unlock()
		return errors.New("already started")
	}
	tc.started = true
	tc.parentCtx = ctx
	tc.stopped.Store(false)
	tc.mu.Unlock()
	var atResp *pkgTCCommon.AuthTokenResponse
	var err error
	if err, atResp = tc.getAuthorizationToken(); err == nil {
		tc.mu.Lock()
		tc.token = atResp.AccessToken
		tc.tokenIssued = time.Now()
		tc.tokenExpires = time.Unix(atResp.ExpiresAt, 0)
		tc.mu.Unlock()
		go tc.tokenRefreshLoop()
		if err = tc.wsConnect(atResp.AccessToken); err == nil {
			return nil
		}
	}
	tc.mu.Lock()
	tc.started = false
	tc.mu.Unlock()
	return err
}

func (tc *TcClient) Stop() error {
	tc.mu.Lock()
	tc.started = false
	tc.mu.Unlock()
	tc.stopped.Store(true)
	tc.stopCancel()
	return tc.wsDisconnect()
}

func (tc *TcClient) tokenRefreshLoop() {
	retryIn := time.Duration(0)
	for {
		var wait time.Duration
		tc.mu.Lock()
		if !tc.started {
			tc.mu.Unlock()
			return
		}
		if retryIn > 0 {
			wait = retryIn
		} else if tc.tokenExpires.IsZero() {
			wait = 5 * time.Second
		} else {
			refreshAt := tc.tokenIssued.Add(time.Duration(float64(tc.tokenExpires.Sub(tc.tokenIssued)) * 0.75))
			wait = time.Until(refreshAt)
		}
		tc.mu.Unlock()
		if wait < 0 {
			wait = 0
		}
		timer := time.NewTimer(wait)
		select {
		case <-tc.stopCtx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
		if err, atResp := tc.getAuthorizationToken(); err == nil {
			tc.mu.Lock()
			tc.token = atResp.AccessToken
			tc.tokenIssued = time.Now()
			tc.tokenExpires = time.Unix(atResp.ExpiresAt, 0)
			tc.mu.Unlock()
			log.Debug("Token refreshed", log.Time("expires", tc.tokenExpires))
			retryIn = 0
		} else {
			log.Error("Token refresh failed", log.String("message", err.Error()))
			retryIn = 30 * time.Second
		}
	}
}

func (tc *TcClient) GetNewToken() (error, string) {
	log.Debug("Get new token")
	if err, a := tc.getAuthorizationToken(); err != nil {
		return err, ""
	} else {
		return err, a.AccessToken
	}
}

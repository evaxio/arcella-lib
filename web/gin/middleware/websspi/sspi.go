package websspi

import (
	"context"
	"github.com/quasoft/websspi"
	"net/http"
)

type SSPI struct {
	auth *websspi.Authenticator
}

func NewSSPI() *SSPI {
	return &SSPI{}
}

func (ss *SSPI) GetName() string {
	return "SSPI"
}

func (ss *SSPI) Start(ctx context.Context) error {
	var err error
	config := websspi.NewConfig()
	config.EnumerateGroups = true // If groups should be resolved
	// config.ServerName = "..."  // If static instead of dynamic group membership should be resolved
	ss.auth, err = websspi.New(config)
	return err
}

func (ss *SSPI) Stop() error {
	if ss.auth == nil {
		return nil
	}
	return ss.auth.Free()
}

func (ss *SSPI) WithAuth(next http.Handler) http.Handler {
	return ss.auth.WithAuth(next)
}

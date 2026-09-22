package ldap

import (
	"crypto/tls"
	"errors"
	"fmt"
	log "log/slog"
	"net"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	goLdap "github.com/go-ldap/ldap/v3"
)

const (
	ufAccountDisable        = 0x2
	accountExpiresNever     = 9223372036854775807
	fileTimeEpochOffsetUnix = 11644473600000

	connectTimeout = 5 * time.Second
)

type Client struct {
	config        *ConfigLDAPClient
	searchTimeout time.Duration
	tlsConf       *tls.Config
	mu            sync.Mutex
	conn          goLdap.Client
}

func NewClient(config *ConfigLDAPClient) (*Client, error) {
	timeout, err := time.ParseDuration(config.SearchTimeout)
	if err != nil || timeout <= 0 {
		if err != nil {
			log.Error("invalid SearchTimeout", log.String("SearchTimeout", config.SearchTimeout), log.String("message", err.Error()))
		}
		timeout = 10 * time.Second
	}
	c := &Client{
		config:        config,
		searchTimeout: timeout,
		tlsConf: &tls.Config{
			MinVersion:         tls.VersionTLS12,
			InsecureSkipVerify: config.Insecure,
		},
	}
	if err := c.connect(); err != nil {
		return nil, err
	}
	return c, nil
}

// VerifyUser checks that the user exists in LDAP and the account is enabled and not expired.
// The domain from the NTLM message is not used in the filter: the search is limited by ConfigLDAPClient.Base.
func (c *Client) VerifyUser(username, domain string) error {
	if username == "" {
		return errors.New("empty username")
	}
	c.mu.Lock()
	if err := c.connectLocked(); err != nil {
		c.mu.Unlock()
		return err
	}
	c.mu.Unlock()

	req := &goLdap.SearchRequest{
		BaseDN:     c.config.Base,
		Scope:      goLdap.ScopeWholeSubtree,
		Attributes: []string{"userAccountControl", "accountExpires"},
		Filter:     "(sAMAccountName=" + goLdap.EscapeFilter(username) + ")",
		SizeLimit:  5,
	}
	res, err := c.searchWithTimeout(req)
	if err != nil {
		return err
	}
	if len(res.Entries) == 0 {
		return fmt.Errorf("user %q not found", username)
	}
	entry := res.Entries[0]

	if uac, err := strconv.ParseInt(entry.GetAttributeValue("userAccountControl"), 10, 64); err == nil {
		if uac&ufAccountDisable != 0 {
			return fmt.Errorf("user %q is disabled", username)
		}
	}
	if exp := entry.GetAttributeValue("accountExpires"); exp != "" {
		if v, err := strconv.ParseInt(exp, 10, 64); err == nil && v != 0 && v != accountExpiresNever {
			if t := time.UnixMilli(v/10000 - fileTimeEpochOffsetUnix); t.Before(time.Now()) {
				return fmt.Errorf("user %q account is expired", username)
			}
		}
	}
	log.Debug("LDAP user verified", log.String("user", username), log.String("domain", domain))
	return nil
}

func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn != nil {
		err := c.conn.Close()
		c.conn = nil
		return err
	}
	return nil
}

// connect requires that the mutex is NOT held
func (c *Client) connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.connectLocked()
}

// connectLocked requires that the mutex IS held
func (c *Client) connectLocked() error {
	if c.conn != nil {
		return nil
	}
	c.mu.Unlock()
	lconn, err := c.dial()
	c.mu.Lock()
	if err != nil {
		return err
	}
	if c.conn != nil {
		_ = lconn.Close()
		return nil
	}
	c.conn = lconn
	return nil
}

func (c *Client) dial() (goLdap.Client, error) {
	u, err := url.Parse(c.config.Url)
	if err != nil {
		return nil, err
	}
	if u.Host == "" {
		return nil, errors.New("empty ldap url host")
	}
	dialer := &net.Dialer{Timeout: connectTimeout}
	var lconn goLdap.Client
	switch strings.ToLower(u.Scheme) {
	case "ldaps":
		lconn, err = goLdap.DialURL(c.config.Url,
			goLdap.DialWithDialer(dialer),
			goLdap.DialWithTLSConfig(c.tlsConf))
	case "ldap":
		lconn, err = goLdap.DialURL(c.config.Url, goLdap.DialWithDialer(dialer))
	default:
		return nil, fmt.Errorf("unsupported ldap scheme: %s", u.Scheme)
	}
	if err != nil {
		return nil, err
	}
	if c.config.User != "" {
		if err = lconn.Bind(c.config.User, c.config.Pass); err != nil {
			lconn.Close()
			return nil, err
		}
	}
	return lconn, nil
}

func (c *Client) searchWithTimeout(req *goLdap.SearchRequest) (*goLdap.SearchResult, error) {
	c.mu.Lock()
	conn := c.conn
	c.mu.Unlock()
	if conn == nil {
		return nil, errors.New("no ldap connection")
	}

	res, err := c.searchOnce(conn, req)
	if err == nil {
		return res, nil
	}
	log.Warn("LDAP search failed, reconnecting", log.String("message", err.Error()))

	errs := []error{err}
	if cerr := conn.Close(); cerr != nil {
		errs = append(errs, cerr)
	}
	c.mu.Lock()
	if c.conn == conn {
		c.conn = nil
	}
	c.mu.Unlock()

	if rerr := c.connect(); rerr != nil {
		return nil, errors.Join(append(errs, rerr)...)
	}
	c.mu.Lock()
	reconn := c.conn
	c.mu.Unlock()
	res, err = c.searchOnce(reconn, req)
	if err != nil {
		c.mu.Lock()
		if c.conn == reconn {
			c.conn = nil
		}
		c.mu.Unlock()
		return nil, errors.Join(append(errs, err)...)
	}
	return res, nil
}

func (c *Client) searchOnce(conn goLdap.Client, req *goLdap.SearchRequest) (*goLdap.SearchResult, error) {
	type result struct {
		res *goLdap.SearchResult
		err error
	}
	ch := make(chan result, 1)
	go func() {
		res, err := conn.Search(req)
		ch <- result{res: res, err: err}
	}()
	select {
	case r := <-ch:
		return r.res, r.err
	case <-time.After(c.searchTimeout):
		_ = conn.Close()
		return nil, fmt.Errorf("ldap search timed out after %s", c.searchTimeout)
	}
}

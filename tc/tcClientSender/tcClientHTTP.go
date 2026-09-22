package trueConfClient

import (
	pkgTCCommon "axgit.vixiv.ru/snake/arcella-lib/tc"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	log "log/slog"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// https://developers.trueconf.com/api/server/#api-Authorization
//
//	curl --request POST \
//	   --header 'content-type: application/json' \
//	   --url 'https://{{$server_name}}/bridge/api/client/v1/oauth/token' \
//	   --data '{"grant_type":"password", "username": "user", "password":"123", "client_id": "trueconf_server_users"}'

var tokenFieldRe = regexp.MustCompile(`"access_token"\s*:\s*"[^"]*"`)

// safeTokenBody redacts the token field and truncates the response body so
// that an access token is never written to the log.
func safeTokenBody(b []byte) string {
	s := tokenFieldRe.ReplaceAllString(string(b), `"access_token":"<redacted>"`)
	const max = 256
	if len(s) > max {
		return s[:max]
	}
	return s
}

// serverHost normalizes the configured server value for use as a url.URL Host.
func serverHost(server string) string {
	if i := strings.Index(server, "://"); i >= 0 {
		server = server[i+len("://"):]
	}
	return strings.TrimSuffix(server, "/")
}

func (tc *TcClient) getAuthorizationToken() (error, *pkgTCCommon.AuthTokenResponse) {
	atResp := pkgTCCommon.AuthTokenResponse{}
	atReq := pkgTCCommon.AuthTokenRequest{ClientId: "chat_bot", GrantType: "password", Username: tc.config.User, Password: tc.config.Password}
	// log.Debug("request", log.String("json", utils.ToJson(atReq)))
	requestBody, err := json.Marshal(atReq)
	if err != nil {
		return err, nil
	}
	log.Debug("Server", log.String("name", tc.config.Server))
	tokenURL := url.URL{Scheme: "https", Host: serverHost(tc.config.Server), Path: "/bridge/api/client/v1/oauth/token"}
	var resp *http.Response
	for attempt := 0; ; attempt++ {
		req, err := http.NewRequest("POST", tokenURL.String(), bytes.NewReader(requestBody))
		if err != nil {
			return err, nil
		}
		req.Header.Add("content-type", "application/json")
		if resp, err = tc.httpClient.Do(req); err == nil {
			break
		}
		if attempt+1 >= 3 {
			return err, nil
		}
		log.Warn("Token request failed, retrying", log.Int("attempt", attempt+1), log.String("message", err.Error()))
		time.Sleep(time.Duration(attempt+1) * time.Second)
	}
	defer resp.Body.Close()
	resBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		log.Error("io.ReadAll")
		return err, nil
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		log.Error("Token request failed", log.Int("status", resp.StatusCode), log.String("body", safeTokenBody(resBody)))
		return errors.New("token request failed with status " + strconv.Itoa(resp.StatusCode)), nil
	}
	if err = json.Unmarshal(resBody, &atResp); err != nil {
		log.Error("Unmarshal", log.Int("status", resp.StatusCode), log.String("resBody", safeTokenBody(resBody)))
		return err, nil
	}
	ut := time.Unix(atResp.ExpiresAt, 0)
	log.Debug("ExpiresAt", log.String("date", ut.Format("2006-01-02 15:04:05")))
	return nil, &atResp
}

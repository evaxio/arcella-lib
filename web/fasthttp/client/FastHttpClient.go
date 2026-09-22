package client

import (
	appPkgWeb "axgit.vixiv.ru/snake/arcella-lib/web"
	"encoding/json"
	log "log/slog"
	"strconv"

	"github.com/valyala/fasthttp"
)

var headerContentTypeJSON = []byte("application/json")

// https://github.com/valyala/fasthttp/blob/master/examples/client/client.go
type FastHttpClient struct {
	config *ConfigFastHttpClient
	client fasthttp.Client
}

func NewFastHttpClient(config *ConfigFastHttpClient) *FastHttpClient {
	log.Debug("NewFastHttpClient",
		log.Duration("ReadTimeout", config.ReadTimeout),
		log.Duration("WriteTimeout", config.WriteTimeout),
		log.Duration("IdleConnTimeout", config.IdleConnTimeout),
		log.Duration("RequestTimeout", config.RequestTimeout),
	)
	return &FastHttpClient{config: config,
		client: fasthttp.Client{
			ReadTimeout:                   config.ReadTimeout,
			WriteTimeout:                  config.WriteTimeout,
			MaxIdleConnDuration:           config.IdleConnTimeout,
			MaxResponseBodySize:           int(config.MaxResponseBodySize),
			NoDefaultUserAgentHeader:      true, // Don't send: User-Agent: fasthttp
			DisableHeaderNamesNormalizing: true, // If you set the case on your headers correctly you can enable this
			DisablePathNormalizing:        true,
			// increase DNS cache time to an hour instead of default minute
			Dial: (&fasthttp.TCPDialer{
				Concurrency:      4096,
				DNSCacheDuration: config.DNSTimeout,
			}).Dial,
		},
	}
}

func (fh *FastHttpClient) getReqBodyReader(request *appPkgWeb.RequestDTS) ([]byte, error) {
	var requestBody []byte
	switch body := request.Body.(type) {
	case nil:
	case string:
		requestBody = []byte(body)
	default:
		var err error
		if requestBody, err = json.Marshal(body); err != nil {
			return nil, err
		}
	}
	return requestBody, nil
}

func copyResponseHeaders(h *fasthttp.ResponseHeader) map[string][]string {
	if h.Len() == 0 {
		return nil
	}
	headers := make(map[string][]string, h.Len())
	h.VisitAll(func(key, value []byte) {
		headers[string(key)] = append(headers[string(key)], string(value))
	})
	return headers
}

func (fh *FastHttpClient) DORequest(reqData *appPkgWeb.RequestDTS) (appPkgWeb.ResponseDTS, error) {
	response := appPkgWeb.ResponseDTS{}
	//////////////////////////////////////////////////////
	reqEntityBytes, err := fh.getReqBodyReader(reqData)
	if err != nil {
		log.Error("Marshal", log.String("error", err.Error()))
		return response, err
	}

	requestType := reqData.RequestType
	if requestType == "" {
		requestType = fasthttp.MethodGet
	}

	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)
	req.SetRequestURI(reqData.RequestUrl)
	req.Header.SetMethod(requestType)
	if len(reqEntityBytes) > 0 {
		req.Header.SetContentTypeBytes(headerContentTypeJSON)
	}
	req.SetBodyRaw(reqEntityBytes)

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)
	err = fh.client.DoTimeout(req, resp, fh.config.RequestTimeout)

	if err != nil {
		log.Error("DORequest", log.String("error", err.Error()))
		return response, err
	} else {
		response.StatusCode = resp.StatusCode()
		response.Status = strconv.Itoa(resp.StatusCode()) + " " + string(resp.Header.StatusMessage())
		response.Body = string(resp.Body())
		response.Header = copyResponseHeaders(&resp.Header)
	}
	//////////////////////////////////////////////////////
	return response, nil
}

package ucare

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"reflect"
	"time"

	"github.com/uploadcare/uploadcare-go/internal/config"
)

type uploadAPIClient struct {
	authFunc UploadAPIAuthFunc

	conn *http.Client
}

func newUploadAPIClient(creds APICreds, conf *Config) Client {
	c := uploadAPIClient{
		authFunc: simpleUploadAPIAuthFunc(creds),
		conn:     conf.HTTPClient,
	}

	if conf.SignBasedAuthentication {
		c.authFunc = signBasedUploadAPIAuthFunc(creds)
	}

	return &c
}

func (c *uploadAPIClient) NewRequest(
	ctx context.Context,
	endpoint config.Endpoint,
	method string,
	requrl string,
	data ReqEncoder,
) (*http.Request, error) {
	requrl, err := resolveReqURL(endpoint, requrl)
	if err != nil {
		return nil, fmt.Errorf("resolving req url: %w", err)
	}
	req, err := http.NewRequest(method, requrl, nil)
	if err != nil {
		return nil, err
	}

	ctx = context.WithValue(ctx, config.CtxAuthFuncKey, c.authFunc)
	req = req.WithContext(ctx)

	if data != nil {
		req.GetBody = getBodyBuilder(req, data)
		req.Body, err = req.GetBody()
		if err != nil {
			return nil, err
		}
	}

	log.Debugf(
		"created new request: %s %+v %+v",
		req.Method,
		req.URL,
		req.Header,
	)

	return req, nil
}

func (c *uploadAPIClient) Do(
	req *http.Request,
	resdata interface{},
) error {
	tries := 0
try:
	tries++

	if tries > 1 && req.GetBody != nil {
		var err error
		req.Body, err = req.GetBody()
		if err != nil {
			return err
		}
	}

	log.Debugf("making %d request: %s %+v", tries, req.Method, req.URL)

	resp, err := c.conn.Do(req)
	if err != nil {
		return err
	}
	if req.Body != nil {
		defer req.Body.Close()
	}

	log.Debugf("received response: %+v", resp)

	switch resp.StatusCode {
	case 400, 403:
		data, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		resp.Body.Close()
		switch resp.StatusCode {
		case 400:
			return reqValidationErr{respErr{string(data)}}
		case 403:
			return reqForbiddenErr{respErr{string(data)}}
		}
	case 413:
		return ErrFileTooLarge
	case 429:
		if tries > config.MaxThrottleRetries {
			return throttleErr{}
		}
		// retry after is not returned from the upload API
		time.Sleep(5 * time.Second)
		goto try
	default:
	}

	if resdata == nil || reflect.ValueOf(resdata).IsNil() {
		return nil
	}
	err = json.NewDecoder(resp.Body).Decode(&resdata)
	if err != nil {
		return err
	}
	resp.Body.Close()

	return nil
}

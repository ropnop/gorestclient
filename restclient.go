package gorestclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"

	"github.com/pkg/errors"
)

type RestClient interface {
	NewRequest(ctx context.Context, method, relPath string, body any) (*http.Request, error)
	DoRequest(req *http.Request, respDest any) (*http.Response, error)
	GetBaseURL() *url.URL
}

type PrepareRequestFunction func(req *http.Request) error

type ErrorHandlingFunction func(err error, req *http.Request, res *http.Response) (*http.Response, error)

type restClient struct {
	baseURL         *url.URL
	prepareFunc     PrepareRequestFunction
	handleErrorFunc ErrorHandlingFunction
	httpClient      *http.Client
}

var _ RestClient = (*restClient)(nil)

type Option func(client *restClient) error

func WithPrepareRequestFunc(f PrepareRequestFunction) Option {
	return func(c *restClient) error {
		c.prepareFunc = f
		return nil
	}
}

func WithErrorHandlingFunc(f ErrorHandlingFunction) Option {
	return func(c *restClient) error {
		c.handleErrorFunc = f
		return nil
	}
}

func WithHTTPClient(h *http.Client) Option {
	return func(c *restClient) error {
		c.httpClient = h
		return nil
	}
}

var ErrBadStatusCode = errors.New("bad status code")

func defaultErrorHandler(err error, req *http.Request, res *http.Response) (*http.Response, error) {
	if err == nil {
		err = ErrBadStatusCode
	}
	defer res.Body.Close()
	body, err2 := ioutil.ReadAll(res.Body)
	if err2 != nil {
		return res, errors.Wrap(err, fmt.Sprintf("response code: %d", res.StatusCode))
	}

	return res, errors.Wrap(err, fmt.Sprintf("response code: %d, body:\n%s", res.StatusCode, string(body)))

}

func NewRestClient(baseURL string, opts ...Option) (*restClient, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid url: %w", err)
	}
	c := &restClient{baseURL: u}
	for _, opt := range opts {
		err = opt(c)
		if err != nil {
			return nil, err
		}
	}
	if c.httpClient == nil {
		c.httpClient = http.DefaultClient
	}
	if c.handleErrorFunc == nil {
		c.handleErrorFunc = defaultErrorHandler
	}
	return c, nil
}

func (c restClient) NewRequest(ctx context.Context, method, relPath string, body any) (*http.Request, error) {
	rel, err := url.Parse(path.Join(c.baseURL.Path, relPath))
	if err != nil {
		return nil, fmt.Errorf("error parsing path: %w", err)
	}
	u := c.baseURL.ResolveReference(rel)
	var buf io.ReadWriter
	if body != nil {
		buf = new(bytes.Buffer)
		err = json.NewEncoder(buf).Encode(body)
		if err != nil {
			return nil, err
		}
	}
	req, err := http.NewRequestWithContext(ctx, method, u.String(), buf)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	if c.prepareFunc != nil {
		if err = c.prepareFunc(req); err != nil {
			return nil, err
		}
	}
	return req, nil
}

func (c restClient) DoRequest(req *http.Request, respDest any) (*http.Response, error) {
	res, err := c.httpClient.Do(req)
	if err != nil {
		return res, err
	}
	if res.StatusCode >= http.StatusBadRequest {
		if c.handleErrorFunc != nil {
			return c.handleErrorFunc(err, req, res)
		}
	}
	if respDest != nil {
		err = json.NewDecoder(res.Body).Decode(respDest)
		if err != nil {
			return res, fmt.Errorf("error unmarshalling response: %w", err)
		}
	}
	return res, nil
}

func (c restClient) GetBaseURL() *url.URL {
	return c.baseURL
}

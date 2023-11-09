// Package sr provides a schema registry client and a helper type to encode
// values and decode data according to the schema registry wire format.
//
// As mentioned on the Serde type, this package does not provide schema
// auto-discovery and type auto-decoding. To aid in strong typing and validated
// encoding/decoding, you must register IDs and values to how to encode or
// decode them.
//
// The client does not automatically cache schemas, instead, the Serde type is
// used for the actual caching of IDs to how to encode/decode the IDs. The
// Client type itself simply speaks http to your schema registry and returns
// the results.
//
// To read more about the schema registry, see the following:
//
//	https://docs.confluent.io/platform/current/schema-registry/develop/api.html
package sr

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ResponseError is the type returned from the schema registry for errors.
type ResponseError struct {
	// Method is the requested http method.
	Method string `json:"-"`
	// URL is the full path that was requested that resulted in this error.
	URL string `json:"-"`
	// Raw contains the raw response body.
	Raw []byte `json:"-"`

	ErrorCode int    `json:"error_code"`
	Message   string `json:"message"`
}

func (e *ResponseError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return string(e.Raw)
}

// Client talks to a schema registry and contains helper functions to serialize
// and deserialize objects according to schemas.
type Client struct {
	urls      []string
	httpcl    *http.Client
	ua        string
	defParams Param

	basicAuth *struct {
		user string
		pass string
	}
}

// NewClient returns a new schema registry client.
func NewClient(opts ...ClientOpt) (*Client, error) {
	cl := &Client{
		urls:   []string{"http://localhost:8081"},
		httpcl: &http.Client{Timeout: 5 * time.Second},
		ua:     "franz-go",
	}

	for _, opt := range opts {
		opt.apply(cl)
	}

	if len(cl.urls) == 0 {
		return nil, errors.New("unable to create client with no URLs")
	}

	return cl, nil
}

func (cl *Client) get(ctx context.Context, path string, into any) error {
	return cl.do(ctx, http.MethodGet, path, nil, into)
}

func (cl *Client) post(ctx context.Context, path string, v, into any) error {
	return cl.do(ctx, http.MethodPost, path, v, into)
}

func (cl *Client) put(ctx context.Context, path string, v, into any) error {
	return cl.do(ctx, http.MethodPut, path, v, into)
}

func (cl *Client) delete(ctx context.Context, path string, into any) error {
	return cl.do(ctx, http.MethodDelete, path, nil, into)
}

func (cl *Client) do(ctx context.Context, method, path string, v, into any) error {
	urls := cl.urls

start:
	url := fmt.Sprintf("%s%s", urls[0], path)
	urls = urls[1:]

	var reqBody io.Reader
	if v != nil {
		marshaled, err := json.Marshal(v)
		if err != nil {
			return fmt.Errorf("unable to encode body for %s %q: %w", method, url, err)
		}
		reqBody = bytes.NewReader(marshaled)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return fmt.Errorf("unable to create request for %s %q: %v", method, url, err)
	}
	req.Header.Set("Content-Type", "application/vnd.schemaregistry.v1+json")
	req.Header.Set("Accept", "application/vnd.schemaregistry.v1+json")
	req.Header.Set("User-Agent", cl.ua)
	if cl.basicAuth != nil {
		req.SetBasicAuth(cl.basicAuth.user, cl.basicAuth.pass)
	}
	cl.applyParams(ctx, req)

	resp, err := cl.httpcl.Do(req)
	if err != nil {
		if len(urls) == 0 {
			return fmt.Errorf("unable to %s %q: %w", method, url, err)
		}
		goto start
	}

	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return fmt.Errorf("unable to read response body from %s %q: %w", method, url, err)
	}

	if resp.StatusCode >= 300 {
		e := &ResponseError{
			Method: method,
			URL:    url,
			Raw:    body,
		}
		_ = json.Unmarshal(body, e) // best effort
		return e
	}

	if into != nil {
		switch into := into.(type) {
		case *[]byte:
			*into = body // return raw body to caller
		default:
			if err := json.Unmarshal(body, into); err != nil {
				return fmt.Errorf("unable to decode ok response body from %s %q: %w", method, url, err)
			}
		}
	}
	return nil
}

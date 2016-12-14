package triton

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/go-cleanhttp"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/joyent/triton-go/authentication"
)

// Client represents a connection to the Triton API.
type Client struct {
	client      *retryablehttp.Client
	authorizer  []authentication.Signer
	endpoint    string
	accountName string
}

// NewClient is used to construct a Client in order to make API
// requests to the Triton API.
//
// At least one signer must be provided - example signers include
// authentication.PrivateKeySigner and authentication.SSHAgentSigner.
func NewClient(endpoint string, accountName string, signers ...authentication.Signer) (*Client, error) {
	defaultRetryWaitMin := 1 * time.Second
	defaultRetryWaitMax := 5 * time.Minute
	defaultRetryMax     := 32

	httpClient := &http.Client{
		Transport: cleanhttp.DefaultTransport(),
		CheckRedirect: doNotFollowRedirects,
	}

	retryableClient := &retryablehttp.Client{
		HTTPClient:   httpClient,
		Logger:       log.New(os.Stderr, "", log.LstdFlags),
		RetryWaitMin: defaultRetryWaitMin,
		RetryWaitMax: defaultRetryWaitMax,
		RetryMax:     defaultRetryMax,
		CheckRetry:   retryablehttp.DefaultRetryPolicy,
	}

	return &Client{
		client:      retryableClient,
		authorizer:  signers,
		endpoint:    strings.TrimSuffix(endpoint, "/"),
		accountName: accountName,
	}, nil
}

func doNotFollowRedirects(*http.Request, []*http.Request) error {
	return http.ErrUseLastResponse
}

func (c *Client) formatURL(path string) string {
	return fmt.Sprintf("%s%s", c.endpoint, path)
}

func (c *Client) executeRequestURIParams(method, path string, body interface{}, query *url.Values) (io.ReadCloser, error) {
	var requestBody io.ReadSeeker
	if body != nil {
		marshaled, err := json.MarshalIndent(body, "", "    ")
		if err != nil {
			return nil, err
		}
		requestBody = bytes.NewReader(marshaled)
	}

	req, err := retryablehttp.NewRequest(method, c.formatURL(path), requestBody)
	if err != nil {
		return nil, errwrap.Wrapf("Error constructing HTTP request: {{err}}", err)
	}

	dateHeader := time.Now().UTC().Format(time.RFC1123)
	req.Header.Set("date", dateHeader)

	authHeader, err := c.authorizer[0].Sign(dateHeader)
	if err != nil {
		return nil, errwrap.Wrapf("Error signing HTTP request: {{err}}", err)
	}
	req.Header.Set("Authorization", authHeader)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Version", "8")
	req.Header.Set("User-Agent", "triton-go client API")

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	if query != nil {
		req.URL.RawQuery = query.Encode()
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, errwrap.Wrapf("Error executing HTTP request: {{err}}", err)
	}

	if resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices {
		return resp.Body, nil
	}

	tritonError := &TritonError{
		StatusCode: resp.StatusCode,
	}

	errorDecoder := json.NewDecoder(resp.Body)
	if err := errorDecoder.Decode(tritonError); err != nil {
		return nil, errwrap.Wrapf("Error decoding error resopnse: {{err}}", err)
	}
	return nil, tritonError
}

func (c *Client) executeRequest(method, path string, body interface{}) (io.ReadCloser, error) {
	return c.executeRequestURIParams(method, path, body, nil)
}

func (c *Client) executeRequestRaw(method, path string, body interface{}) (*http.Response, error) {
	var requestBody io.ReadSeeker
	if body != nil {
		marshaled, err := json.MarshalIndent(body, "", "    ")
		if err != nil {
			return nil, err
		}
		requestBody = bytes.NewReader(marshaled)
	}

	req, err := retryablehttp.NewRequest(method, c.formatURL(path), requestBody)
	if err != nil {
		return nil, errwrap.Wrapf("Error constructing HTTP request: {{err}}", err)
	}

	dateHeader := time.Now().UTC().Format(time.RFC1123)
	req.Header.Set("date", dateHeader)

	authHeader, err := c.authorizer[0].Sign(dateHeader)
	if err != nil {
		return nil, errwrap.Wrapf("Error signing HTTP request: {{err}}", err)
	}
	req.Header.Set("Authorization", authHeader)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Version", "8")
	req.Header.Set("User-Agent", "triton-go client API")

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, errwrap.Wrapf("Error executing HTTP request: {{err}}", err)
	}

	return resp, nil
}

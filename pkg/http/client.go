package http

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// MethodGet is a constant for the HTTP GET method.
const (
	MethodGet = "GET"
)

// Client is a simple wrapper around http.Client that provides convenience methods
// for making HTTP requests and decoding JSON responses.
type Client struct {
	HttpClient *http.Client
}

// NewHttpClient creates and returns a new instance of Client with a default http.Client.
func NewHttpClient() *Client {
	return &Client{
		HttpClient: &http.Client{},
	}
}

// SendRequest performs an HTTP request with the given method, URL, and headers.
// It returns the response or an error if the request fails or the status is not 200 OK.
func (hc *Client) SendRequest(method string, url string, headers map[string]string) (*http.Response, error) {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := hc.HttpClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch config with status: %d", resp.StatusCode)
	}

	return resp, nil
}

// SendRequestAndDecode performs an HTTP request and decodes the JSON response body into v.
// It uses SendRequest internally and returns an error if the request or decoding fails.
func (hc *Client) SendRequestAndDecode(v any, method string, url string, headers map[string]string) error {
	resp, err := hc.SendRequest(method, url, headers)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		return err
	}

	return nil
}

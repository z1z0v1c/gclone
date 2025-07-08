package http

import (
	"encoding/json"
	"fmt"
	"net/http"
)

const (
	MethodGet = "GET"
)

type Client struct {
	HttpClient *http.Client
}

func NewHttpClient() *Client {
	return &Client{
		HttpClient: &http.Client{},
	}
}

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

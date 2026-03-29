package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Client interface {
	DoRequest(
		ctx context.Context,
		method, path string,
		body any,
		out any,
	) error
}

type httpClient struct {
	baseURL string
	apiKey  string
	client  *http.Client
}

func NewClient(domain, apikey string) Client {
	return &httpClient{
		baseURL: domain,
		apiKey:  apikey,
		client: &http.Client{
			Timeout: 600 * time.Second,
		},
	}
}

func (r *httpClient) DoRequest(
	ctx context.Context,
	method, path string,
	body any,
	out any,
) error {
	var buf *bytes.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		buf = bytes.NewReader(b)
	} else {
		buf = bytes.NewReader(nil)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		method,
		r.baseURL+path,
		buf,
	)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-KEY", r.apiKey)

	resp, err := r.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		var mapResp interface{}
		json.NewDecoder(resp.Body).Decode(&mapResp)
		return fmt.Errorf("httpClient error: %s:%v", resp.Status, mapResp)
	}

	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}

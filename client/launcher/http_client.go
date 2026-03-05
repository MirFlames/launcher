package main

import (
	"fmt"
	"net/http"
	"time"
)

func newHTTPClient(timeout time.Duration) *http.Client {
	return &http.Client{Timeout: timeout}
}

// getWithRetry выполняет GET с ретраями при сетевых ошибках и 5xx.
func getWithRetry(url string, timeout time.Duration) (*http.Response, error) {
	return doWithRetry(func() (*http.Request, error) { return http.NewRequest("GET", url, nil) }, timeout)
}

// doWithRetry выполняет запрос с ретраями (exponential backoff).
func doWithRetry(createReq func() (*http.Request, error), timeout time.Duration) (*http.Response, error) {
	client := newHTTPClient(timeout)
	var resp *http.Response
	var lastErr error
	for attempt := 0; attempt < downloadMaxRetries; attempt++ {
		req, err := createReq()
		if err != nil {
			return nil, err
		}
		resp, lastErr = client.Do(req)
		if lastErr == nil && resp != nil && resp.StatusCode == http.StatusOK {
			return resp, nil
		}
		if resp != nil {
			resp.Body.Close()
		}
		if attempt < downloadMaxRetries-1 {
			delay := time.Duration(downloadRetryDelayMs*(1<<attempt)) * time.Millisecond
			time.Sleep(delay)
		}
	}
	if lastErr != nil {
		return nil, lastErr
	}
	if resp != nil {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return nil, fmt.Errorf("не удалось выполнить запрос")
}

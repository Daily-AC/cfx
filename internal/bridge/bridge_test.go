package bridge

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBridgeAuthenticatesAndDeliversOnce(t *testing.T) {
	s := &server{
		job:    Job{ID: "job-1", Type: "whoami"},
		result: make(chan Result, 1),
		token:  "secret",
	}
	handler := s.routes()

	request := func(method, path string, body []byte, token string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, bytes.NewReader(body))
		req.Header.Set("Origin", "chrome-extension://test-extension")
		if token != "" {
			req.Header.Set("X-Cfx-Token", token)
		}
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, req)
		return response
	}

	if got := request(http.MethodGet, "/v1/job", nil, "wrong").Code; got != http.StatusUnauthorized {
		t.Fatalf("wrong token status = %d", got)
	}
	first := request(http.MethodGet, "/v1/job", nil, "secret")
	if first.Code != http.StatusOK {
		t.Fatalf("first job status = %d", first.Code)
	}
	if got := request(http.MethodGet, "/v1/job", nil, "secret").Code; got != http.StatusNoContent {
		t.Fatalf("second job status = %d", got)
	}

	body, _ := json.Marshal(Result{ID: "job-1", OK: true, Data: json.RawMessage(`{"handle":"e0_7"}`)})
	if got := request(http.MethodPost, "/v1/result", body, "secret").Code; got != http.StatusNoContent {
		t.Fatalf("result status = %d", got)
	}
	result := <-s.result
	if !result.OK || string(result.Data) != `{"handle":"e0_7"}` {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestBridgeRejectsNonExtensionOrigin(t *testing.T) {
	s := &server{job: Job{ID: "job-1"}, result: make(chan Result, 1), token: "secret"}
	req := httptest.NewRequest(http.MethodGet, "/v1/job", nil)
	req.Header.Set("Origin", "https://example.com")
	req.Header.Set("X-Cfx-Token", "secret")
	response := httptest.NewRecorder()
	s.routes().ServeHTTP(response, req)
	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d", response.Code)
	}
}

func TestBridgeWakePageAndPreflight(t *testing.T) {
	s := &server{job: Job{ID: "job-1"}, result: make(chan Result, 1), token: "secret"}
	handler := s.routes()

	wake := httptest.NewRecorder()
	handler.ServeHTTP(wake, httptest.NewRequest(http.MethodGet, "/wake", nil))
	if wake.Code != http.StatusOK {
		t.Fatalf("wake status = %d", wake.Code)
	}

	preflightRequest := httptest.NewRequest(http.MethodOptions, "/v1/job", nil)
	preflightRequest.Header.Set("Origin", "chrome-extension://test-extension")
	preflight := httptest.NewRecorder()
	handler.ServeHTTP(preflight, preflightRequest)
	if preflight.Code != http.StatusNoContent {
		t.Fatalf("preflight status = %d", preflight.Code)
	}
	if got := preflight.Header().Get("Access-Control-Allow-Methods"); got != "GET, POST, OPTIONS" {
		t.Fatalf("allowed methods = %q", got)
	}
}

func TestBridgeAcceptsHostPermissionRequestWithoutOrigin(t *testing.T) {
	s := &server{job: Job{ID: "job-1", Type: "whoami"}, result: make(chan Result, 1), token: "secret"}
	req := httptest.NewRequest(http.MethodGet, "/v1/job", nil)
	req.Header.Set("X-Cfx-Token", "secret")
	response := httptest.NewRecorder()
	s.routes().ServeHTTP(response, req)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
}

package clients

import (
	"fmt"
	"net/http"
	"sync"
)

type MockRoundTripper struct {
	responses []*http.Response
	requests  []*http.Request
	lock      sync.Locker
}

func NewMockRoundTripper() *MockRoundTripper {
	return &MockRoundTripper{lock: &sync.Mutex{}}
}

func (t *MockRoundTripper) GetLastRequest() *http.Request {
	t.lock.Lock()
	defer t.lock.Unlock()
	if len(t.requests) == 0 {
		return nil
	}
	return t.requests[len(t.requests)-1]
}

func (t *MockRoundTripper) AddResponse(response *http.Response) {
	t.lock.Lock()
	defer t.lock.Unlock()
	t.responses = append(t.responses, response)
}

func (t *MockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	t.lock.Lock()
	defer t.lock.Unlock()
	if len(t.responses) == 0 {
		return nil, fmt.Errorf("no response from server")
	}
	response := t.responses[0]
	t.responses = t.responses[1:]
	t.requests = append(t.requests, req)
	return response, nil
}

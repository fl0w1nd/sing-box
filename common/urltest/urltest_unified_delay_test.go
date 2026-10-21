package urltest

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	N "github.com/sagernet/sing/common/network"
)

func TestUnifiedDelayKeepsFirstResultWhenServerClosesConnection(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Connection", "close")
		writer.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	_, err := URLTest(ContextWithIsUnifiedDelay(t.Context()), server.URL, N.SystemDialer)
	if err != nil {
		t.Fatalf("first successful probe was discarded: %v", err)
	}
}

func TestUnifiedDelayKeepsFirstResultWhenSecondProbeStalls(t *testing.T) {
	var requests atomic.Int32
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if requests.Add(1) > 1 {
			select {
			case <-release:
			case <-request.Context().Done():
			}
		}
		writer.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	defer close(release)

	ctx, cancel := context.WithTimeout(ContextWithIsUnifiedDelay(t.Context()), time.Second)
	defer cancel()
	_, err := URLTest(ctx, server.URL, N.SystemDialer)
	if err != nil {
		t.Fatalf("first successful probe was discarded: %v", err)
	}
	if ctx.Err() != nil {
		t.Fatal("result arrived after the caller deadline")
	}
}

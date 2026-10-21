//go:build with_tailscale

package tailscale

import (
	"context"
	"testing"
	"time"

	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing/service"
	"github.com/sagernet/tailscale/envknob"
)

type forceRelayCache struct {
	adapter.CacheFile
	enabled bool
	loaded  bool
}

func (c *forceRelayCache) LoadTailscaleForceRelay() (bool, bool, error) {
	return c.enabled, c.loaded, nil
}

func (c *forceRelayCache) StoreTailscaleForceRelay(enabled bool) error {
	c.enabled, c.loaded = enabled, true
	return nil
}

func resetForceRelay(t *testing.T) {
	t.Cleanup(func() {
		forceRelay.access.Lock()
		setForceRelayLocked(forceRelay.initial)
		forceRelay.access.Unlock()
	})
}

func TestSetForceRelaySwitchesMagicsockKnob(t *testing.T) {
	resetForceRelay(t)
	cache := new(forceRelayCache)
	ctx, cancel := context.WithCancel(t.Context())
	updates := make(chan bool, 4)
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = SubscribeForceRelay(ctx, func(enabled bool) { updates <- enabled })
	}()
	if initial := <-updates; initial != forceRelay.initial {
		t.Fatalf("unexpected initial state %v", initial)
	}
	for _, enabled := range []bool{!forceRelay.initial, forceRelay.initial} {
		if err := SetForceRelay(cache, enabled); err != nil {
			t.Fatal(err)
		}
		if ForceRelay() != enabled || envknob.Bool(forceRelayEnv) != enabled || cache.enabled != enabled {
			t.Fatalf("force relay was not applied: %v", enabled)
		}
		select {
		case update := <-updates:
			if update != enabled {
				t.Fatalf("unexpected update %v, want %v", update, enabled)
			}
		case <-time.After(time.Second):
			t.Fatal("missing update")
		}
	}
	cancel()
	<-done
}

func TestLoadForceRelayFromCache(t *testing.T) {
	resetForceRelay(t)
	cache := &forceRelayCache{enabled: !forceRelay.initial, loaded: true}
	endpoint := &Endpoint{ctx: service.ContextWith[adapter.CacheFile](context.Background(), adapter.CacheFile(cache))}
	if err := endpoint.loadForceRelay(); err != nil {
		t.Fatal(err)
	}
	if ForceRelay() != cache.enabled || envknob.Bool(forceRelayEnv) != cache.enabled {
		t.Fatal("stored state was not applied")
	}
	cache.loaded = false
	if err := endpoint.loadForceRelay(); err != nil {
		t.Fatal(err)
	}
	if ForceRelay() != forceRelay.initial {
		t.Fatal("missing state did not fall back to the environment")
	}
}

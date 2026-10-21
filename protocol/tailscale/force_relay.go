//go:build with_tailscale

package tailscale

import (
	"context"
	"strconv"
	"sync"

	"github.com/sagernet/sing-box/adapter"
	E "github.com/sagernet/sing/common/exceptions"
	"github.com/sagernet/sing/common/observable"
	"github.com/sagernet/sing/service"
	"github.com/sagernet/tailscale/envknob"
	"github.com/sagernet/tailscale/wgengine/magicsock"
)

// forceRelayEnv is magicsock's own switch: while it is set, every rebind hands
// magicsock a blocking UDP conn, so peers are only reached through DERP. The
// knob is process-wide, so force relay applies to all Tailscale endpoints.
const forceRelayEnv = "TS_DEBUG_ALWAYS_USE_DERP"

var forceRelay = struct {
	access sync.Mutex
	// initial is the value from the process environment, used while the cache
	// file has no stored state.
	initial  bool
	enabled  bool
	conns    map[*magicsock.Conn]struct{}
	observer *observable.Observer[struct{}]
}{
	initial:  envknob.Bool(forceRelayEnv),
	enabled:  envknob.Bool(forceRelayEnv),
	conns:    make(map[*magicsock.Conn]struct{}),
	observer: observable.NewObserver(observable.NewSubscriber[struct{}](1), 1),
}

// loadForceRelay applies the stored state before magicsock binds its sockets.
func (t *Endpoint) loadForceRelay() error {
	cacheFile := service.FromContext[adapter.CacheFile](t.ctx)
	if cacheFile == nil {
		return nil
	}
	enabled, loaded, err := cacheFile.LoadTailscaleForceRelay()
	if err != nil {
		return E.Cause(err, "load force relay state")
	}
	forceRelay.access.Lock()
	defer forceRelay.access.Unlock()
	if !loaded {
		enabled = forceRelay.initial
	}
	setForceRelayLocked(enabled)
	return nil
}

// registerForceRelay lets later changes rebind this endpoint's magicsock.
func (t *Endpoint) registerForceRelay(scope *adapter.Scope, magicConn *magicsock.Conn) {
	forceRelay.access.Lock()
	forceRelay.conns[magicConn] = struct{}{}
	forceRelay.access.Unlock()
	scope.Add(func() error {
		forceRelay.access.Lock()
		delete(forceRelay.conns, magicConn)
		forceRelay.access.Unlock()
		return nil
	})
}

func setForceRelayLocked(enabled bool) {
	if forceRelay.enabled == enabled {
		return
	}
	envknob.Setenv(forceRelayEnv, strconv.FormatBool(enabled))
	forceRelay.enabled = enabled
	for magicConn := range forceRelay.conns {
		magicConn.Rebind()
		if !enabled {
			magicConn.ReSTUN("force-relay-disabled")
		}
	}
	forceRelay.observer.Emit(struct{}{})
}

// SetForceRelay switches every Tailscale endpoint in the process between DERP
// only and normal path selection, and stores the state in cacheFile if set.
func SetForceRelay(cacheFile adapter.CacheFile, enabled bool) error {
	forceRelay.access.Lock()
	defer forceRelay.access.Unlock()
	if cacheFile != nil {
		err := cacheFile.StoreTailscaleForceRelay(enabled)
		if err != nil {
			return E.Cause(err, "persist force relay state")
		}
	}
	setForceRelayLocked(enabled)
	return nil
}

func ForceRelay() bool {
	forceRelay.access.Lock()
	defer forceRelay.access.Unlock()
	return forceRelay.enabled
}

// SubscribeForceRelay calls listener with the current state and then on every
// change, until ctx is done.
func SubscribeForceRelay(ctx context.Context, listener func(bool)) error {
	subscription, done, err := forceRelay.observer.Subscribe()
	if err != nil {
		return err
	}
	defer forceRelay.observer.UnSubscribe(subscription)
	current := ForceRelay()
	listener(current)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-done:
			return nil
		case <-subscription:
			updated := ForceRelay()
			if updated == current {
				continue
			}
			current = updated
			listener(updated)
		}
	}
}

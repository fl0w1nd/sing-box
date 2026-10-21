package outbound

import (
	"slices"

	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing/common"
	E "github.com/sagernet/sing/common/exceptions"
)

var _ adapter.DynamicOutboundManager = (*Manager)(nil)

// staticOutbounds excludes dynamic outbounds, whose lifecycle belongs to the
// caller of SwapDynamic. Callers hold m.access.
func (m *Manager) staticOutbounds() []adapter.Outbound {
	if len(m.dynamicTags) == 0 {
		return m.outbounds
	}
	return common.Filter(m.outbounds, func(it adapter.Outbound) bool {
		return !m.dynamicTags[it.Tag()]
	})
}

func (m *Manager) SwapDynamic(tag string, previous adapter.Outbound, next adapter.Outbound) error {
	if next != nil && next.Tag() != tag {
		return E.New("dynamic outbound tag mismatch: ", next.Tag(), " != ", tag)
	}
	m.access.Lock()
	defer m.access.Unlock()
	current, loaded := m.outboundByTag[tag]
	if previous == nil {
		if loaded {
			return E.New("outbound tag is already used by outbound/", current.Type(), "[", tag, "]")
		}
		if endpoint, isEndpoint := m.endpoint.Get(tag); isEndpoint {
			return E.New("outbound tag is already used by endpoint/", endpoint.Type(), "[", tag, "]")
		}
	} else if !loaded || current != previous || !m.dynamicTags[tag] {
		return E.New("dynamic outbound was replaced: ", tag)
	}
	if m.dynamicTags == nil {
		m.dynamicTags = make(map[string]bool)
	}
	// Readers keep the slice returned by Outbounds, so it is never modified in place.
	switch {
	case next == nil:
		m.outbounds = slices.DeleteFunc(slices.Clone(m.outbounds), func(it adapter.Outbound) bool {
			return it == previous
		})
		delete(m.outboundByTag, tag)
		delete(m.dynamicTags, tag)
	case previous == nil:
		m.outbounds = append(slices.Clip(m.outbounds), next)
		m.outboundByTag[tag] = next
		m.dynamicTags[tag] = true
	default:
		outbounds := slices.Clone(m.outbounds)
		outbounds[slices.Index(outbounds, previous)] = next
		m.outbounds = outbounds
		m.outboundByTag[tag] = next
	}
	return nil
}

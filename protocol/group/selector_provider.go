package group

import "github.com/sagernet/sing-box/adapter"

type selectorState struct {
	tags      []string
	outbounds map[string]adapter.Outbound
}

// rebuild refreshes the members and selects, in order, the explicit selection,
// the default, the previous selection's tag and the first member.
func (s *Selector) rebuild() {
	s.updateAccess.Lock()
	defer s.updateAccess.Unlock()
	outbounds := s.members.outbounds()
	state := &selectorState{tags: outboundTags(outbounds), outbounds: make(map[string]adapter.Outbound, len(outbounds))}
	for _, outbound := range outbounds {
		state.outbounds[outbound.Tag()] = outbound
	}
	s.state.Store(state)
	previous := s.selected.Load()
	// The default comes before the previous selection, which may be a fallback
	// picked while the default's provider was not loaded yet.
	candidates := []string{s.preferredTag.Load(), s.defaultTag}
	if previous != nil {
		candidates = append(candidates, previous.Tag())
	}
	if len(state.tags) > 0 {
		candidates = append(candidates, state.tags[0])
	}
	var next adapter.Outbound
	for _, tag := range candidates {
		if outbound, loaded := state.outbounds[tag]; loaded {
			next = outbound
			break
		}
	}
	s.selected.Store(next)
	if previous == next {
		return
	}
	if previous != nil {
		s.interruptGroup.Interrupt(s.interruptExternalConnections)
	}
	if s.history != nil {
		s.history.NotifyUpdated()
	}
}

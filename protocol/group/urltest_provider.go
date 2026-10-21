package group

import (
	"slices"

	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing/common"
	N "github.com/sagernet/sing/common/network"
)

func (s *URLTest) rebuild() {
	s.rebuildAccess.Lock()
	defer s.rebuildAccess.Unlock()
	outbounds := s.members.outbounds()
	s.tags.Store(outboundTags(outbounds))
	if s.group != nil {
		s.group.ReplaceOutbounds(outbounds)
	}
}

// ReplaceOutbounds swaps the members and keeps each selection on the same tag.
// A selection whose tag is gone, or whose replacement lacks the network, is
// chosen again by the next update check.
func (g *URLTestGroup) ReplaceOutbounds(outbounds []adapter.Outbound) {
	g.updateAccess.Lock()
	previousOutbounds := g.outbounds.Swap(outbounds)
	var selectionReplaced bool
	for network, selected := range map[string]*common.TypedValue[adapter.Outbound]{
		N.NetworkTCP: &g.selectedOutboundTCP,
		N.NetworkUDP: &g.selectedOutboundUDP,
	} {
		current := selected.Load()
		if current == nil {
			continue
		}
		replacement := common.Find(outbounds, func(it adapter.Outbound) bool {
			return it.Tag() == current.Tag() && common.Contains(it.Network(), network)
		})
		if replacement != current {
			selected.Store(replacement)
			selectionReplaced = true
		}
	}
	if selectionReplaced {
		g.interruptGroup.Interrupt(g.interruptExternalConnections)
	}
	g.updateAccess.Unlock()
	g.performUpdateCheck()
	if slices.Equal(previousOutbounds, outbounds) {
		return
	}
	g.history.NotifyUpdated()
	if g.started.Load() {
		go g.CheckOutbounds(g.ctx, true)
	}
}

package adapter

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing/common/x/list"
)

var ErrProviderUpdating = errors.New("outbound provider is updating")

type OutboundProviderSubscriptionInfo struct {
	Upload     int64
	Download   int64
	Total      int64
	ExpireTime time.Time
}

type OutboundProviderCallback func()

type OutboundProvider interface {
	Lifecycle
	Type() string
	Tag() string
	Outbounds() []Outbound
	UpdatedAt() time.Time
	Updating() bool
	LastError() error
	SubscriptionInfo() OutboundProviderSubscriptionInfo
	Update(ctx context.Context) error
	// RegisterCallback is called synchronously after every change of outbounds or
	// status. Replaced outbounds are closed only after callbacks return.
	RegisterCallback(callback OutboundProviderCallback) *list.Element[OutboundProviderCallback]
	UnregisterCallback(element *list.Element[OutboundProviderCallback])
}

type ProviderRegistry interface {
	option.ProviderOptionsRegistry
	Create(ctx context.Context, logger log.ContextLogger, tag string, providerType string, options any) (OutboundProvider, error)
}

type ProviderManager interface {
	Lifecycle
	Providers() []OutboundProvider
	Get(tag string) (OutboundProvider, bool)
	Create(ctx context.Context, logger log.ContextLogger, tag string, providerType string, options any) error
}

type SavedOutboundProvider struct {
	Content          []byte                           `json:"content"`
	LastUpdated      time.Time                        `json:"last_updated"`
	LastEtag         string                           `json:"last_etag,omitempty"`
	SubscriptionInfo OutboundProviderSubscriptionInfo `json:"subscription_info"`
}

func (s *SavedOutboundProvider) MarshalBinary() ([]byte, error) {
	return json.Marshal(s)
}

func (s *SavedOutboundProvider) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, s)
}

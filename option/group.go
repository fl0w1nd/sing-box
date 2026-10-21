package option

import "github.com/sagernet/sing/common/json/badoption"

type SelectorOutboundOptions struct {
	Outbounds                 []string                   `json:"outbounds,omitempty" reference:"outbound"`
	Providers                 badoption.Listable[string] `json:"providers,omitempty" reference:"provider"`
	UseAllProviders           bool                       `json:"use_all_providers,omitempty"`
	Include                   *badoption.Regexp          `json:"include,omitempty"`
	Exclude                   *badoption.Regexp          `json:"exclude,omitempty"`
	Default                   string                     `json:"default,omitempty" reference:"outbound"`
	InterruptExistConnections bool                       `json:"interrupt_exist_connections,omitempty"`
}

type URLTestOutboundOptions struct {
	Outbounds                 []string                   `json:"outbounds,omitempty" reference:"outbound"`
	Providers                 badoption.Listable[string] `json:"providers,omitempty" reference:"provider"`
	UseAllProviders           bool                       `json:"use_all_providers,omitempty"`
	Include                   *badoption.Regexp          `json:"include,omitempty"`
	Exclude                   *badoption.Regexp          `json:"exclude,omitempty"`
	URL                       string                     `json:"url,omitempty"`
	Interval                  badoption.Duration         `json:"interval,omitempty"`
	Tolerance                 uint16                     `json:"tolerance,omitempty"`
	IdleTimeout               badoption.Duration         `json:"idle_timeout,omitempty"`
	InterruptExistConnections bool                       `json:"interrupt_exist_connections,omitempty"`
}

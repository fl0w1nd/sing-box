package option

import (
	"context"
	"reflect"

	"github.com/sagernet/sing-box/schema"
	E "github.com/sagernet/sing/common/exceptions"
	"github.com/sagernet/sing/common/json"
	"github.com/sagernet/sing/common/json/badjson"
	"github.com/sagernet/sing/common/json/badoption"
	"github.com/sagernet/sing/service"
)

type ProviderOptionsRegistry interface {
	OptionTypes() []string
	CreateOptions(providerType string) (any, bool)
}

type _Provider struct {
	Type    string `json:"type"`
	Tag     string `json:"tag"`
	Options any    `json:"-"`
}

type Provider _Provider

func (p *Provider) MarshalJSONContext(ctx context.Context) ([]byte, error) {
	return badjson.MarshallObjectsContext(ctx, (*_Provider)(p), p.Options)
}

func (p *Provider) UnmarshalJSONContext(ctx context.Context, content []byte) error {
	err := json.UnmarshalContext(ctx, content, (*_Provider)(p))
	if err != nil {
		return err
	}
	registry := service.FromContext[ProviderOptionsRegistry](ctx)
	if registry == nil {
		return E.New("missing provider options registry in context")
	}
	options, loaded := registry.CreateOptions(p.Type)
	if !loaded {
		return E.New("unknown provider type: ", p.Type)
	}
	err = badjson.UnmarshallExcludedContext(ctx, content, (*_Provider)(p), options)
	if err != nil {
		return err
	}
	p.Options = options
	return nil
}

func (p Provider) DescribeSchema(builder schema.Builder) (*schema.Node, error) {
	return builder.Define("Provider", func() (*schema.Node, error) {
		registry := service.FromContext[ProviderOptionsRegistry](builder.Context())
		if registry == nil {
			return nil, E.New("missing provider options registry in context")
		}
		union, err := registryUnion(builder, registry, nil, true)
		if err != nil {
			return nil, err
		}
		for _, variant := range union.OneOf {
			variant.Required = append(variant.Required, "tag")
		}
		return union, nil
	})
}

type RemoteProviderOptions struct {
	URL                   string                 `json:"url"`
	UserAgent             string                 `json:"user_agent,omitempty"`
	Headers               badoption.HTTPHeader   `json:"headers,omitempty"`
	HTTPClient            string                 `json:"http_client" reference:"http_client"`
	UpdateInterval        badoption.Duration     `json:"update_interval,omitempty"`
	TagPrefix             *string                `json:"tag_prefix,omitempty"`
	Include               *badoption.Regexp      `json:"include,omitempty"`
	Exclude               *badoption.Regexp      `json:"exclude,omitempty"`
	OverrideDialerOptions *DialerOptionsOverride `json:"override_dialer_options,omitempty"`
}

func (o RemoteProviderOptions) DescribeSchema(builder schema.Builder) (*schema.Node, error) {
	node := schema.StrictObject()
	err := builder.FlattenStruct(node, reflect.TypeFor[RemoteProviderOptions]())
	if err != nil {
		return nil, err
	}
	node.Required = []string{"url", "http_client"}
	return node, nil
}

type DialerOptionsOverride struct {
	DialerOptions
	rawMessage json.RawMessage
}

func (o DialerOptionsOverride) MarshalJSONContext(ctx context.Context) ([]byte, error) {
	if o.rawMessage != nil {
		return o.rawMessage, nil
	}
	return json.MarshalContext(ctx, o.DialerOptions)
}

func (o *DialerOptionsOverride) UnmarshalJSONContext(ctx context.Context, content []byte) error {
	var options DialerOptions
	err := json.UnmarshalContextDisallowUnknownFields(ctx, content, &options)
	if err != nil {
		return err
	}
	o.DialerOptions = options
	o.rawMessage = append(o.rawMessage[:0], content...)
	return nil
}

func (o DialerOptionsOverride) DescribeSchema(builder schema.Builder) (*schema.Node, error) {
	node := schema.StrictObject()
	err := builder.FlattenStruct(node, reflect.TypeFor[DialerOptions]())
	if err != nil {
		return nil, err
	}
	return node, nil
}

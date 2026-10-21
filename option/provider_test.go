package option_test

import (
	stdJSON "encoding/json"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/sagernet/sing-box/include"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing-box/schema"
	"github.com/sagernet/sing/common/json"
)

func TestRemoteProviderAndProviderGroupsDecode(t *testing.T) {
	ctx := include.Context(t.Context())
	content := []byte(`{
  "providers": [{
    "type": "remote",
    "tag": "source",
    "url": "https://provider.invalid/outbounds.json",
    "user_agent": "test-agent",
    "headers": {"X-Test": "value"},
    "http_client": "provider-client",
    "update_interval": "1h",
    "tag_prefix": "",
    "include": "allowed",
    "exclude": "blocked",
    "override_dialer_options": {
      "tcp_fast_open": false,
      "domain_resolver": "local-dns"
    }
  }],
  "http_clients": [{
    "tag": "provider-client",
    "detour": "download-select"
  }],
  "outbounds": [
    {"type": "direct", "tag": "provider-direct"},
    {"type": "selector", "tag": "download-select", "outbounds": ["provider-direct"]},
    {"type": "selector", "tag": "select", "providers": ["source"], "include": "source/allowed", "exclude": "blocked"},
    {"type": "urltest", "tag": "auto", "use_all_providers": true, "include": "source/allowed", "exclude": "blocked"}
  ]
}`)
	options, err := json.UnmarshalExtendedContext[option.Options](ctx, content)
	if err != nil {
		t.Fatal(err)
	}
	if len(options.Providers) != 1 || options.Providers[0].Tag != "source" {
		t.Fatalf("unexpected providers: %+v", options.Providers)
	}
	remoteOptions, loaded := options.Providers[0].Options.(*option.RemoteProviderOptions)
	if !loaded || remoteOptions.URL != "https://provider.invalid/outbounds.json" {
		t.Fatalf("unexpected remote provider options: %+v", options.Providers[0].Options)
	}
	if remoteOptions.UserAgent != "test-agent" || remoteOptions.Headers.Build().Get("X-Test") != "value" || remoteOptions.HTTPClient != "provider-client" {
		t.Fatalf("unexpected provider request options: %+v", remoteOptions)
	}
	if remoteOptions.TagPrefix == nil || *remoteOptions.TagPrefix != "" {
		t.Fatalf("unexpected provider tag prefix: %+v", remoteOptions.TagPrefix)
	}
	if time.Duration(remoteOptions.UpdateInterval) != time.Hour || remoteOptions.Include.Build().String() != "allowed" || remoteOptions.Exclude.Build().String() != "blocked" {
		t.Fatalf("unexpected provider update options: %+v", remoteOptions)
	}
	if remoteOptions.OverrideDialerOptions == nil || remoteOptions.OverrideDialerOptions.DomainResolver == nil || remoteOptions.OverrideDialerOptions.DomainResolver.Server != "local-dns" {
		t.Fatalf("unexpected provider dialer options override: %+v", remoteOptions.OverrideDialerOptions)
	}
	encoded, err := json.MarshalContext(ctx, options)
	if err != nil {
		t.Fatal(err)
	}
	var encodedOptions struct {
		Providers []struct {
			OverrideDialerOptions map[string]any `json:"override_dialer_options"`
		} `json:"providers"`
	}
	if err = stdJSON.Unmarshal(encoded, &encodedOptions); err != nil {
		t.Fatal(err)
	}
	if tcpFastOpen, loaded := encodedOptions.Providers[0].OverrideDialerOptions["tcp_fast_open"]; !loaded || tcpFastOpen != false {
		t.Fatalf("explicit false override was not preserved: %+v", encodedOptions.Providers[0].OverrideDialerOptions)
	}
	selectorOptions := options.Outbounds[2].Options.(*option.SelectorOutboundOptions)
	if len(selectorOptions.Providers) != 1 || selectorOptions.Providers[0] != "source" || selectorOptions.Include.Build().String() != "source/allowed" || selectorOptions.Exclude.Build().String() != "blocked" {
		t.Fatalf("unexpected selector providers: %+v", selectorOptions.Providers)
	}
	urlTestOptions := options.Outbounds[3].Options.(*option.URLTestOutboundOptions)
	if !urlTestOptions.UseAllProviders || urlTestOptions.Include.Build().String() != "source/allowed" || urlTestOptions.Exclude.Build().String() != "blocked" {
		t.Fatalf("unexpected urltest provider options: %+v", urlTestOptions)
	}
}

func TestProviderSchemaRequiredFields(t *testing.T) {
	ctx := include.Context(t.Context())
	content, err := schema.Generate(ctx, reflect.TypeFor[option.Options]())
	if err != nil {
		t.Fatal(err)
	}
	var document struct {
		Defs map[string]struct {
			OneOf []struct {
				Required   []string `json:"required"`
				Properties map[string]struct {
					Type string `json:"type"`
				} `json:"properties"`
			} `json:"oneOf"`
		} `json:"$defs"`
	}
	if err = stdJSON.Unmarshal(content, &document); err != nil {
		t.Fatal(err)
	}
	variants := document.Defs["Provider"].OneOf
	if len(variants) != 1 {
		t.Fatalf("unexpected provider schema variants: %d", len(variants))
	}
	for _, field := range []string{"type", "tag", "url", "http_client"} {
		if !slices.Contains(variants[0].Required, field) {
			t.Fatalf("provider schema does not require %s: %v", field, variants[0].Required)
		}
	}
	for _, field := range []string{"include", "exclude"} {
		if variants[0].Properties[field].Type != "string" {
			t.Fatalf("provider schema field %s is not a string", field)
		}
	}
}

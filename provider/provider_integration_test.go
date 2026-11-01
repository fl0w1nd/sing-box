package provider_test

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"

	box "github.com/sagernet/sing-box"
	"github.com/sagernet/sing-box/adapter"
	boxOutbound "github.com/sagernet/sing-box/adapter/outbound"
	"github.com/sagernet/sing-box/include"
	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing/common/json"
	M "github.com/sagernet/sing/common/metadata"
	N "github.com/sagernet/sing/common/network"
)

const integrationOutboundType = "provider-integration-outbound"

type integrationOutboundOptions struct {
	Configuration string `json:"configuration"`
}

type integrationOutbound struct {
	boxOutbound.Adapter
}

func (o *integrationOutbound) DialContext(context.Context, string, M.Socksaddr) (net.Conn, error) {
	return nil, errors.New("test outbound has no dialer")
}

func (o *integrationOutbound) ListenPacket(context.Context, M.Socksaddr) (net.PacketConn, error) {
	return nil, errors.New("test outbound has no packet listener")
}

func TestProviderBoxLifecycleAndCacheRestore(t *testing.T) {
	var contentAccess sync.RWMutex
	content := `{"outbounds":[{"tag":"node","type":"provider-integration-outbound","configuration":"one"}],"endpoints":[]}`
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		contentAccess.RLock()
		defer contentAccess.RUnlock()
		writer.Header().Set("ETag", `"test"`)
		_, _ = writer.Write([]byte(content))
	}))
	cachePath := filepath.Join(t.TempDir(), "cache.db")
	config := fmt.Sprintf(`{
  "log": {"disabled": true},
  "http_clients": [{"tag":"provider-client","detour":"download-select"}],
  "outbounds": [
    {"type":"direct","tag":"direct"},
    {"type":"selector","tag":"download-select","outbounds":["direct"],"default":"direct"},
    {"type":"selector","tag":"select","providers":["source"],"include":"source/node","exclude":"^$"},
    {"type":"selector","tag":"select-all","use_all_providers":true},
    {"type":"urltest","tag":"auto","providers":["source"],"interval":"1h","idle_timeout":"1h"}
  ],
  "providers": [{"type":"remote","tag":"source","url":%s,"http_client":"provider-client","include":"node","exclude":"^$"}],
  "route": {"final":"direct"},
  "experimental": {"cache_file":{"enabled":true,"path":%s}}
}`, strconv.Quote(server.URL), strconv.Quote(cachePath))

	newBox := func() *box.Box {
		outboundRegistry := include.OutboundRegistry()
		boxOutbound.Register[integrationOutboundOptions](outboundRegistry, integrationOutboundType, func(_ context.Context, _ adapter.Router, _ log.ContextLogger, tag string, options integrationOutboundOptions) (adapter.Outbound, error) {
			if options.Configuration == "" {
				return nil, errors.New("missing test outbound configuration")
			}
			return &integrationOutbound{Adapter: boxOutbound.NewAdapter(integrationOutboundType, tag, []string{N.NetworkTCP, N.NetworkUDP}, nil)}, nil
		})
		ctx := box.ProviderContext(context.Background(), include.ProviderRegistry())
		ctx = box.Context(ctx, include.InboundRegistry(), outboundRegistry, include.EndpointRegistry(), include.DNSTransportRegistry(), include.ServiceRegistry(), include.CertificateProviderRegistry())
		options, err := json.UnmarshalExtendedContext[option.Options](ctx, []byte(config))
		if err != nil {
			t.Fatal(err)
		}
		instance, err := box.New(box.Options{Context: ctx, Options: options})
		if err != nil {
			t.Fatal(err)
		}
		if err = instance.Start(); err != nil {
			instance.Close()
			t.Fatal(err)
		}
		return instance
	}

	firstBox := newBox()
	provider, loaded := firstBox.Provider().Get("source")
	if !loaded || len(provider.Outbounds()) != 1 {
		t.Fatalf("unexpected provider state: loaded=%v outbounds=%d", loaded, len(provider.Outbounds()))
	}
	firstOutbound := provider.Outbounds()[0]
	selectorOutbound, loaded := firstBox.Outbound().Outbound("select")
	if !loaded {
		t.Fatal("provider selector was not registered")
	}
	selector := selectorOutbound.(adapter.OutboundGroup)
	if selected := selector.Selected(N.NetworkTCP); selected == nil || selected.Tag() != "source/node" {
		t.Fatalf("unexpected selector choice: %v", selected)
	}
	for _, tag := range []string{"select-all", "auto"} {
		groupOutbound, found := firstBox.Outbound().Outbound(tag)
		if !found {
			t.Fatalf("provider group %s was not registered", tag)
		}
		group := groupOutbound.(adapter.OutboundGroup)
		if members := group.All(); len(members) != 1 || members[0] != "source/node" {
			t.Fatalf("unexpected %s members: %v", tag, members)
		}
	}
	if err := provider.Update(t.Context()); err != nil {
		t.Fatal(err)
	}
	if provider.Outbounds()[0] != firstOutbound {
		t.Fatal("unchanged HTTP update replaced the outbound")
	}

	contentAccess.Lock()
	content = `{"outbounds":[{"tag":"node","type":"provider-integration-outbound","configuration":"two"}]}`
	contentAccess.Unlock()
	if err := provider.Update(t.Context()); err != nil {
		t.Fatal(err)
	}
	replacement := provider.Outbounds()[0]
	if replacement == firstOutbound {
		t.Fatal("changed HTTP update preserved the old outbound")
	}
	if registered, _ := firstBox.Outbound().Outbound("source/node"); registered != replacement {
		t.Fatal("outbound manager does not serve the replacement")
	}
	if selected := selector.Selected(N.NetworkTCP); selected != replacement {
		t.Fatalf("selector did not follow the replacement: %v", selected)
	}
	if err := firstBox.Close(); err != nil {
		t.Fatal(err)
	}
	server.Close()

	secondBox := newBox()
	defer secondBox.Close()
	restored, loaded := secondBox.Provider().Get("source")
	if !loaded || len(restored.Outbounds()) != 1 || restored.Outbounds()[0].Tag() != "source/node" {
		t.Fatalf("provider cache was not restored: loaded=%v", loaded)
	}
}

func TestProviderConvertsSubscription(t *testing.T) {
	links := strings.Join([]string{
		"ss://" + base64.RawURLEncoding.EncodeToString([]byte("aes-128-gcm:pass")) + "@ss.example.com:8388#SS",
		"ss://" + base64.RawURLEncoding.EncodeToString([]byte("aes-128-gcm:pass")) + "@st.example.com:443/?plugin=shadow-tls%3Bhost%3Dwww.example.com%3Bpassword%3Dst%3Bversion%3D2#SS%20ShadowTLS",
		"trojan://pass@trojan.example.com:443?sni=trojan.example.com#Trojan",
		"ssr://unsupported",
	}, "\n")
	content := base64.StdEncoding.EncodeToString([]byte(links))
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		_, _ = writer.Write([]byte(content))
	}))
	defer server.Close()
	config := fmt.Sprintf(`{
  "log": {"disabled": true},
  "http_clients": [{"tag":"provider-client"}],
  "outbounds": [
    {"type":"direct","tag":"direct"},
    {"type":"selector","tag":"select","providers":["source"]}
  ],
  "providers": [{"type":"remote","tag":"source","url":%s,"http_client":"provider-client","exclude":"_shadowtls$|^Trojan$"}],
  "route": {"final":"direct"},
  "experimental": {"cache_file":{"enabled":true,"path":%s}}
}`, strconv.Quote(server.URL), strconv.Quote(filepath.Join(t.TempDir(), "cache.db")))
	ctx := box.ProviderContext(context.Background(), include.ProviderRegistry())
	ctx = box.Context(ctx, include.InboundRegistry(), include.OutboundRegistry(), include.EndpointRegistry(), include.DNSTransportRegistry(), include.ServiceRegistry(), include.CertificateProviderRegistry())
	options, err := json.UnmarshalExtendedContext[option.Options](ctx, []byte(config))
	if err != nil {
		t.Fatal(err)
	}
	instance, err := box.New(box.Options{Context: ctx, Options: options})
	if err != nil {
		t.Fatal(err)
	}
	defer instance.Close()
	if err = instance.Start(); err != nil {
		t.Fatal(err)
	}
	provider, _ := instance.Provider().Get("source")
	var tags []string
	for _, outbound := range provider.Outbounds() {
		tags = append(tags, outbound.Type()+"/"+outbound.Tag())
	}
	// The ShadowTLS node is kept although it is excluded, since a kept node
	// detours through it.
	expected := []string{"shadowsocks/source/SS", "shadowsocks/source/SS ShadowTLS", "shadowtls/source/SS ShadowTLS_shadowtls"}
	if !slices.Equal(tags, expected) {
		t.Fatalf("unexpected provider outbounds: %v", tags)
	}
	if err = provider.LastError(); err != nil {
		t.Fatal(err)
	}
	outbounds := provider.Outbounds()
	if err = provider.Update(t.Context()); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(outbounds, provider.Outbounds()) {
		t.Fatal("an unchanged subscription replaced its outbounds")
	}
}

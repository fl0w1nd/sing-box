package remote

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/sagernet/sing-box/adapter"
	boxEndpoint "github.com/sagernet/sing-box/adapter/endpoint"
	boxOutbound "github.com/sagernet/sing-box/adapter/outbound"
	"github.com/sagernet/sing-box/common/dialer"
	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing/common/json"
	"github.com/sagernet/sing/common/json/badoption"
	M "github.com/sagernet/sing/common/metadata"
	N "github.com/sagernet/sing/common/network"
	"github.com/sagernet/sing/service"
)

const testOutboundType = "test-outbound"

type testOutboundOptions struct {
	option.DialerOptions
	Configuration string `json:"configuration"`
	Reject        bool   `json:"reject,omitempty"`
}

type testTransport struct {
	access    sync.Mutex
	requests  []*http.Request
	responses []*http.Response
	roundTrip func(*http.Request) (*http.Response, error)
}

func (t *testTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	t.access.Lock()
	t.requests = append(t.requests, request.Clone(request.Context()))
	if t.roundTrip != nil {
		roundTrip := t.roundTrip
		t.access.Unlock()
		return roundTrip(request)
	}
	response := t.responses[0]
	t.responses = t.responses[1:]
	t.access.Unlock()
	return response, nil
}

func (t *testTransport) CloseIdleConnections() {}
func (t *testTransport) Reset()                {}

type testOutbound struct {
	boxOutbound.Adapter
	options   testOutboundOptions
	upstream  N.Dialer
	failStart bool
	closed    bool
}

func (o *testOutbound) Start(stage adapter.StartStage, scope *adapter.Scope) error {
	if stage != adapter.StartStateStart {
		return nil
	}
	if o.failStart {
		return errors.New("start failed")
	}
	scope.Add(func() error {
		o.closed = true
		return nil
	})
	return nil
}

func (o *testOutbound) DialContext(ctx context.Context, network string, destination M.Socksaddr) (net.Conn, error) {
	if o.closed {
		return nil, errors.New("closed outbound")
	}
	if o.upstream != nil {
		return o.upstream.DialContext(ctx, network, destination)
	}
	client, server := net.Pipe()
	go func() {
		_, _ = server.Write([]byte(o.options.Configuration))
		_ = server.Close()
	}()
	return client, nil
}

func (o *testOutbound) ListenPacket(context.Context, M.Socksaddr) (net.PacketConn, error) {
	return nil, errors.New("packet listening is unavailable")
}

func newTestProvider(t *testing.T) *Provider {
	t.Helper()
	registry := boxOutbound.NewRegistry()
	boxOutbound.Register[testOutboundOptions](registry, testOutboundType, func(ctx context.Context, _ adapter.Router, _ log.ContextLogger, tag string, options testOutboundOptions) (adapter.Outbound, error) {
		if options.Configuration == "" || options.Reject {
			return nil, errors.New("invalid test outbound configuration")
		}
		outbound := &testOutbound{
			Adapter:   boxOutbound.NewAdapter(testOutboundType, tag, []string{N.NetworkTCP, N.NetworkUDP}, nil),
			options:   options,
			failStart: options.Configuration == "fail-start",
		}
		if options.Detour != "" {
			// Like real outbounds, resolve the detour lazily on first dial.
			outbound.upstream = dialer.NewDetour(service.FromContext[adapter.OutboundManager](ctx), options.Detour, true)
		}
		return outbound, nil
	})
	manager := boxOutbound.NewManager(registry, boxEndpoint.NewManager(boxEndpoint.NewRegistry()), "")
	ctx := service.ContextWith[option.OutboundOptionsRegistry](t.Context(), registry)
	ctx = service.ContextWith[adapter.OutboundManager](ctx, manager)
	logFactory := log.NewNOPFactory()
	provider := &Provider{
		ctx:              ctx,
		cancel:           func() {},
		logger:           logFactory.NewLogger("provider-test"),
		tag:              "provider",
		outboundRegistry: registry,
		outboundManager:  manager,
		logFactory:       logFactory,
	}
	provider.current.Store(&snapshot{})
	return provider
}

func createStatic(t *testing.T, provider *Provider, tag string) adapter.Outbound {
	t.Helper()
	err := provider.outboundManager.Create(provider.ctx, nil, nil, tag, testOutboundType, &testOutboundOptions{Configuration: "static"})
	if err != nil {
		t.Fatal(err)
	}
	outbound, _ := provider.outboundManager.Outbound(tag)
	return outbound
}

func readOutbound(t *testing.T, provider *Provider, tag string) string {
	t.Helper()
	outbound, loaded := provider.outboundManager.Outbound(tag)
	if !loaded {
		t.Fatalf("outbound %s is not registered", tag)
	}
	conn, err := outbound.DialContext(t.Context(), N.NetworkTCP, M.Socksaddr{})
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	content, err := io.ReadAll(conn)
	if err != nil {
		t.Fatal(err)
	}
	return string(content)
}

func TestApplyPreservesUnchangedOutbound(t *testing.T) {
	provider := newTestProvider(t)
	content := []byte(`{
  "outbounds": [{
    "tag": "node",
    "type": "test-outbound",
    "configuration": "one"
  }],
  "endpoints": [],
  "future_field": true
}`)
	if err := provider.apply(content); err != nil {
		t.Fatal(err)
	}
	first := provider.Outbounds()[0]
	if first.Tag() != "provider/node" {
		t.Fatalf("unexpected tag: %s", first.Tag())
	}
	if err := provider.apply(content); err != nil {
		t.Fatal(err)
	}
	if provider.Outbounds()[0] != first {
		t.Fatal("unchanged outbound instance was replaced")
	}

	changed := []byte(`{"outbounds":[{"tag":"node","type":"test-outbound","configuration":"two"}]}`)
	if err := provider.apply(changed); err != nil {
		t.Fatal(err)
	}
	replacement := provider.Outbounds()[0]
	if replacement == first {
		t.Fatal("changed outbound instance was preserved")
	}
	if !first.(*testOutbound).closed {
		t.Fatal("replaced outbound was not closed")
	}
	if registered, _ := provider.outboundManager.Outbound("provider/node"); registered != replacement {
		t.Fatal("manager does not serve the replacement")
	}
	if outbounds := provider.outboundManager.Outbounds(); len(outbounds) != 1 || outbounds[0] != replacement {
		t.Fatalf("unexpected manager outbounds: %+v", outbounds)
	}
}

func TestApplyRefreshesChainedOutbounds(t *testing.T) {
	provider := newTestProvider(t)
	content := func(configuration string) []byte {
		return []byte(`{"outbounds":[
			{"tag":"B","type":"test-outbound","configuration":"chain","detour":"A"},
			{"tag":"A","type":"test-outbound","configuration":"` + configuration + `"}
		]}`)
	}
	if err := provider.apply(content("one")); err != nil {
		t.Fatal(err)
	}
	chain, _ := provider.outboundManager.Outbound("provider/B")
	if got := readOutbound(t, provider, "provider/B"); got != "one" {
		t.Fatalf("initial chain used %q", got)
	}
	if err := provider.apply(content("two")); err != nil {
		t.Fatal(err)
	}
	if got := readOutbound(t, provider, "provider/B"); got != "two" {
		t.Fatalf("refreshed chain used %q", got)
	}
	refreshed, _ := provider.outboundManager.Outbound("provider/B")
	if refreshed == chain {
		t.Fatal("dependent outbound kept the replaced detour instance")
	}
	if err := provider.apply(content("two")); err != nil {
		t.Fatal(err)
	}
	if current, _ := provider.outboundManager.Outbound("provider/B"); current != refreshed {
		t.Fatal("unchanged chain was rebuilt")
	}
}

func TestApplyKeepsDetourOfRetainedDependent(t *testing.T) {
	provider := newTestProvider(t)
	content := func(detour, dependent string) []byte {
		return []byte(`{"outbounds":[
			{"tag":"A","type":"test-outbound","configuration":"` + detour + `"},
			{"tag":"B","type":"test-outbound","configuration":"` + dependent + `","detour":"A"}
		]}`)
	}
	if err := provider.apply(content("one", "chain")); err != nil {
		t.Fatal(err)
	}
	oldDetour, _ := provider.outboundManager.Outbound("provider/A")
	oldDependent, _ := provider.outboundManager.Outbound("provider/B")
	if err := provider.apply(content("two", "fail-start")); err == nil {
		t.Fatal("expected partial update error")
	}
	if current, _ := provider.outboundManager.Outbound("provider/B"); current != oldDependent {
		t.Fatal("failed dependent did not keep its previous instance")
	}
	if got := readOutbound(t, provider, "provider/B"); got != "one" {
		t.Fatalf("retained dependent used %q", got)
	}
	if err := provider.apply(content("two", "chain")); err != nil {
		t.Fatal(err)
	}
	if got := readOutbound(t, provider, "provider/B"); got != "two" {
		t.Fatalf("rebuilt dependent used %q", got)
	}
	if !oldDetour.(*testOutbound).closed || !oldDependent.(*testOutbound).closed {
		t.Fatal("replaced chain was not closed")
	}
}

func TestApplyPinsDetourOfRetainedDependent(t *testing.T) {
	provider := newTestProvider(t)
	content := func(detour, dependent string) []byte {
		return []byte(`{"outbounds":[
			{"tag":"A","type":"test-outbound","configuration":"` + detour + `"},
			{"tag":"B","type":"test-outbound","configuration":"` + dependent + `","detour":"A"}
		]}`)
	}
	if err := provider.apply(content("one", "chain")); err != nil {
		t.Fatal(err)
	}
	// B is first dialed after A was replaced; its detour must still be the A it
	// was started on, which stays open while B is kept.
	for _, detour := range []string{"two", "three"} {
		if err := provider.apply(content(detour, "fail-start")); err == nil {
			t.Fatal("expected partial update error")
		}
		if got := readOutbound(t, provider, "provider/A"); got != detour {
			t.Fatalf("detour was not updated: %q", got)
		}
		if got := readOutbound(t, provider, "provider/B"); got != "one" {
			t.Fatalf("retained dependent used %q", got)
		}
	}
}

func TestApplyKeepsDependentOfUndecodableDetour(t *testing.T) {
	provider := newTestProvider(t)
	content := func(detourType, other string) []byte {
		return []byte(`{"outbounds":[
			{"tag":"A","type":"` + detourType + `","configuration":"one"},
			{"tag":"B","type":"test-outbound","configuration":"chain","detour":"A"},
			{"tag":"C","type":"test-outbound","configuration":"` + other + `"}
		]}`)
	}
	if err := provider.apply(content("test-outbound", "one")); err != nil {
		t.Fatal(err)
	}
	oldDependent, _ := provider.outboundManager.Outbound("provider/B")
	if err := provider.apply(content("unknown", "two")); err == nil {
		t.Fatal("expected partial update error")
	}
	if got := readOutbound(t, provider, "provider/C"); got != "two" {
		t.Fatalf("independent node was not updated: %q", got)
	}
	if current, _ := provider.outboundManager.Outbound("provider/B"); current != oldDependent {
		t.Fatal("dependent of an undecodable detour was not kept")
	}
	if got := readOutbound(t, provider, "provider/B"); got != "one" {
		t.Fatalf("retained dependent used %q", got)
	}
}

func TestApplyRejectsCircularChain(t *testing.T) {
	provider := newTestProvider(t)
	err := provider.apply([]byte(`{"outbounds":[
		{"tag":"A","type":"test-outbound","configuration":"one","detour":"B"},
		{"tag":"B","type":"test-outbound","configuration":"two","detour":"A"}
	]}`))
	if err == nil || !strings.Contains(err.Error(), "circular provider detour") {
		t.Fatalf("circular chain was accepted: %v", err)
	}
	if len(provider.Outbounds()) != 0 {
		t.Fatalf("circular chain changed the provider snapshot: %v", provider.Outbounds())
	}
}

func TestApplyOverridesDialerOptions(t *testing.T) {
	provider := newTestProvider(t)
	createStatic(t, provider, "local-detour")
	var override option.DialerOptionsOverride
	err := json.UnmarshalContext(provider.ctx, []byte(`{
  "detour": "local-detour",
  "tcp_fast_open": false,
  "domain_resolver": "local-dns"
}`), &override)
	if err != nil {
		t.Fatal(err)
	}
	provider.options.OverrideDialerOptions = &override
	content := []byte(`{"outbounds":[{
  "tag":"node",
  "type":"test-outbound",
  "configuration":"one",
  "detour":"remote-detour",
  "tcp_fast_open":true,
  "connect_timeout":"5s",
  "domain_resolver":{"server":"remote-dns","strategy":"ipv6_only"}
}]}`)
	if err = provider.apply(content); err != nil {
		t.Fatal(err)
	}
	created := provider.Outbounds()[0].(*testOutbound)
	if created.options.Detour != "local-detour" {
		t.Fatalf("detour override was not applied: %q", created.options.Detour)
	}
	if created.options.TCPFastOpen {
		t.Fatal("explicit false override was not applied")
	}
	if time.Duration(created.options.ConnectTimeout) != 5*time.Second {
		t.Fatalf("unmentioned dialer option was not preserved: %s", time.Duration(created.options.ConnectTimeout))
	}
	if created.options.DomainResolver == nil || created.options.DomainResolver.Server != "local-dns" {
		t.Fatalf("domain resolver override was not applied: %+v", created.options.DomainResolver)
	}
	if created.options.DomainResolver.Strategy != 0 {
		t.Fatalf("nested domain resolver was merged instead of replaced: %+v", created.options.DomainResolver)
	}

	unchangedEffectiveOptions := []byte(`{"outbounds":[{
  "tag":"node",
  "type":"test-outbound",
  "configuration":"one",
  "detour":"local-detour",
  "tcp_fast_open":false,
  "connect_timeout":"5s",
  "domain_resolver":"another-remote-dns"
}]}`)
	if err = provider.apply(unchangedEffectiveOptions); err != nil {
		t.Fatal(err)
	}
	if provider.Outbounds()[0] != created {
		t.Fatal("changes hidden by dialer overrides replaced the outbound")
	}
}

func TestApplyAllowsEmptyTagPrefix(t *testing.T) {
	provider := newTestProvider(t)
	emptyPrefix := ""
	provider.options.TagPrefix = &emptyPrefix
	content := []byte(`{"outbounds":[{"tag":"node","type":"test-outbound","configuration":"one"}]}`)
	if err := provider.apply(content); err != nil {
		t.Fatal(err)
	}
	if tag := provider.Outbounds()[0].Tag(); tag != "node" {
		t.Fatalf("unexpected tag: %s", tag)
	}
}

func TestApplySkipsStaticTagCollision(t *testing.T) {
	provider := newTestProvider(t)
	emptyPrefix := ""
	provider.options.TagPrefix = &emptyPrefix
	static := createStatic(t, provider, "node")

	err := provider.apply([]byte(`{"outbounds":[
  {"tag":"node","type":"test-outbound","configuration":"collision"},
  {"tag":"other","type":"test-outbound","configuration":"valid"}
]}`))
	if err != nil {
		t.Fatal(err)
	}
	if current, _ := provider.outboundManager.Outbound("node"); current != static {
		t.Fatal("static outbound was replaced")
	}
	outbounds := provider.Outbounds()
	if len(outbounds) != 1 || outbounds[0].Tag() != "other" {
		t.Fatalf("unexpected provider outbounds: %+v", outbounds)
	}
}

func TestApplyRetainsOldNodeOnDecodeFailure(t *testing.T) {
	provider := newTestProvider(t)
	valid := []byte(`{"outbounds":[{"tag":"node","type":"test-outbound","configuration":"valid"}]}`)
	if err := provider.apply(valid); err != nil {
		t.Fatal(err)
	}
	old := provider.Outbounds()[0]
	err := provider.apply([]byte(`{"outbounds":[{"tag":"node","type":"unknown"}]}`))
	if err == nil {
		t.Fatal("expected partial update error")
	}
	if provider.Outbounds()[0] != old {
		t.Fatal("old outbound was not retained")
	}
	if err = provider.apply([]byte(`{"outbounds":[]}`)); err == nil || provider.Outbounds()[0] != old {
		t.Fatal("empty update did not preserve the old snapshot")
	}
	if err = provider.apply([]byte(`{"outbounds":`)); err == nil || provider.Outbounds()[0] != old {
		t.Fatal("malformed update did not preserve the old snapshot")
	}
	if old.(*testOutbound).closed {
		t.Fatal("retained outbound was closed")
	}
}

func TestApplyUsesOutboundConstructionValidation(t *testing.T) {
	provider := newTestProvider(t)
	initial := []byte(`{"outbounds":[{"tag":"existing","type":"test-outbound","configuration":"valid"}]}`)
	if err := provider.apply(initial); err != nil {
		t.Fatal(err)
	}
	existing := provider.Outbounds()[0]
	update := []byte(`{"outbounds":[
  {"tag":"existing","type":"test-outbound","configuration":"changed","reject":true},
  {"tag":"added","type":"test-outbound","configuration":"valid"}
]}`)
	err := provider.apply(update)
	if err == nil {
		t.Fatal("expected outbound construction error")
	}
	outbounds := provider.Outbounds()
	if len(outbounds) != 2 || outbounds[0] != existing || outbounds[1].Tag() != "provider/added" {
		t.Fatalf("unexpected partial publication: %+v", outbounds)
	}
}

func TestApplyClosesOutboundThatFailsToStart(t *testing.T) {
	provider := newTestProvider(t)
	err := provider.apply([]byte(`{"outbounds":[
  {"tag":"visible","type":"test-outbound","configuration":"one"},
  {"tag":"broken","type":"test-outbound","configuration":"fail-start"}
]}`))
	if err == nil || !strings.Contains(err.Error(), "start failed") {
		t.Fatalf("unexpected apply error: %v", err)
	}
	outbounds := provider.Outbounds()
	if len(outbounds) != 1 || outbounds[0].Tag() != "provider/visible" {
		t.Fatalf("unexpected published outbounds: %+v", outbounds)
	}
	if _, loaded := provider.outboundManager.Outbound("provider/broken"); loaded {
		t.Fatal("outbound that failed to start was registered")
	}
}

func TestCloseRemovesOutbounds(t *testing.T) {
	provider := newTestProvider(t)
	static := createStatic(t, provider, "static")
	if err := provider.apply([]byte(`{"outbounds":[{"tag":"node","type":"test-outbound","configuration":"one"}]}`)); err != nil {
		t.Fatal(err)
	}
	node := provider.Outbounds()[0]
	notified := 0
	provider.RegisterCallback(func() { notified++ })
	if err := provider.close(); err != nil {
		t.Fatal(err)
	}
	if !node.(*testOutbound).closed || len(provider.Outbounds()) != 0 || notified == 0 {
		t.Fatal("close did not retire provider outbounds")
	}
	if outbounds := provider.outboundManager.Outbounds(); len(outbounds) != 1 || outbounds[0] != static {
		t.Fatalf("unexpected manager outbounds after close: %+v", outbounds)
	}
}

func TestParseSubscriptionInfo(t *testing.T) {
	info, loaded := parseSubscriptionInfo("upload=1; download=2; total=3; expire=1893456000")
	if !loaded || info.Upload != 1 || info.Download != 2 || info.Total != 3 || info.ExpireTime.Unix() != 1893456000 {
		t.Fatalf("unexpected subscription info: %+v", info)
	}
}

func TestApplyTagRulesAndFiltering(t *testing.T) {
	provider := newTestProvider(t)
	include := badoption.Regexp(*regexp.MustCompile(`^(keep|drop|3)$`))
	exclude := badoption.Regexp(*regexp.MustCompile(`^drop$`))
	provider.options.Include = &include
	provider.options.Exclude = &exclude
	content := []byte(`{"outbounds":[
  {"tag":"ignored","type":"direct"},
  {"tag":"keep","type":"test-outbound","configuration":"one"},
  {"tag":" keep ","type":"test-outbound","configuration":"two"},
  {"tag":"","type":"test-outbound","configuration":"three"},
  {"tag":"drop","type":"test-outbound","configuration":"four"}
]}`)
	if err := provider.apply(content); err != nil {
		t.Fatal(err)
	}
	outbounds := provider.Outbounds()
	wanted := []string{"provider/keep", "provider/keep #2", "provider/3"}
	if len(outbounds) != len(wanted) {
		t.Fatalf("unexpected outbound count: %d", len(outbounds))
	}
	for index, tag := range wanted {
		if outbounds[index].Tag() != tag {
			t.Fatalf("unexpected tag at %d: %q", index, outbounds[index].Tag())
		}
	}
}

func TestRemoteUpdateETagHeadersAndNotModified(t *testing.T) {
	provider := newTestProvider(t)
	provider.options.URL = "https://example.com/subscription.json"
	provider.options.UserAgent = "provider-test"
	provider.options.Headers = badoption.HTTPHeader{"X-Test": {"value"}}
	transport := &testTransport{responses: []*http.Response{
		{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Header: http.Header{
				"Etag":                  {`"r1"`},
				"Subscription-Userinfo": {"upload=1; download=2; total=3; expire=1893456000"},
			},
			Body: io.NopCloser(strings.NewReader(`{"outbounds":[{"tag":"node","type":"test-outbound","configuration":"one"}]}`)),
		},
		{StatusCode: http.StatusNotModified, Status: "304 Not Modified", Header: make(http.Header), Body: http.NoBody},
	}}
	provider.transport = transport
	if err := provider.Update(t.Context()); err != nil {
		t.Fatal(err)
	}
	first := provider.Outbounds()[0]
	if transport.requests[0].Header.Get("User-Agent") != "provider-test" || transport.requests[0].Header.Get("X-Test") != "value" {
		t.Fatalf("unexpected request headers: %+v", transport.requests[0].Header)
	}
	if info := provider.SubscriptionInfo(); info.Upload != 1 || info.Download != 2 || info.Total != 3 {
		t.Fatalf("unexpected subscription info: %+v", info)
	}
	if err := provider.Update(t.Context()); err != nil {
		t.Fatal(err)
	}
	if transport.requests[1].Header.Get("If-None-Match") != `"r1"` {
		t.Fatalf("unexpected If-None-Match: %q", transport.requests[1].Header.Get("If-None-Match"))
	}
	if provider.Outbounds()[0] != first {
		t.Fatal("304 response replaced outbound instance")
	}
}

func TestRemoteUpdateFollowsRedirect(t *testing.T) {
	provider := newTestProvider(t)
	provider.options.URL = "https://example.com/subscription.json"
	provider.transport = &testTransport{responses: []*http.Response{
		{
			StatusCode: http.StatusFound,
			Status:     "302 Found",
			Header:     http.Header{"Location": {"https://cdn.example.com/subscription.json"}},
			Body:       http.NoBody,
		},
		{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"outbounds":[{"tag":"node","type":"test-outbound","configuration":"one"}]}`)),
		},
	}}
	if err := provider.Update(t.Context()); err != nil {
		t.Fatal(err)
	}
	if got := readOutbound(t, provider, "provider/node"); got != "one" {
		t.Fatalf("unexpected outbound after redirect: %q", got)
	}
}

func TestRemoteUpdateKeepsLastCompleteCacheOnPartialSuccess(t *testing.T) {
	provider := newTestProvider(t)
	provider.options.URL = "https://example.com/subscription.json"
	completeContent := `{"outbounds":[{"tag":"existing","type":"test-outbound","configuration":"one"}]}`
	partialContent := `{"outbounds":[
  {"tag":"existing","type":"test-outbound","configuration":"two","reject":true},
  {"tag":"added","type":"test-outbound","configuration":"three"}
]}`
	provider.transport = &testTransport{responses: []*http.Response{
		{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Header:     http.Header{"Etag": {`"complete"`}},
			Body:       io.NopCloser(strings.NewReader(completeContent)),
		},
		{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Header:     http.Header{"Etag": {`"partial"`}},
			Body:       io.NopCloser(strings.NewReader(partialContent)),
		},
	}}
	if err := provider.Update(t.Context()); err != nil {
		t.Fatal(err)
	}
	old := provider.Outbounds()[0]
	if err := provider.Update(t.Context()); err == nil {
		t.Fatal("expected partial update error")
	}
	outbounds := provider.Outbounds()
	if len(outbounds) != 2 || outbounds[0] != old || outbounds[1].Tag() != "provider/added" {
		t.Fatalf("unexpected partial publication: %+v", outbounds)
	}
	provider.stateAccess.RLock()
	defer provider.stateAccess.RUnlock()
	if string(provider.lastContent) != completeContent || provider.lastEtag != `"complete"` {
		t.Fatalf("partial update replaced complete cache state: content=%s etag=%s", provider.lastContent, provider.lastEtag)
	}
}

func TestConcurrentUpdateReturnsUpdating(t *testing.T) {
	provider := newTestProvider(t)
	provider.options.URL = "https://example.com/subscription.json"
	entered := make(chan struct{})
	release := make(chan struct{})
	provider.transport = &testTransport{roundTrip: func(*http.Request) (*http.Response, error) {
		close(entered)
		<-release
		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"outbounds":[{"tag":"node","type":"test-outbound","configuration":"one"}]}`)),
		}, nil
	}}
	result := make(chan error, 1)
	go func() { result <- provider.Update(t.Context()) }()
	<-entered
	if err := provider.Update(t.Context()); !errors.Is(err, adapter.ErrProviderUpdating) {
		t.Fatalf("unexpected concurrent update error: %v", err)
	}
	close(release)
	if err := <-result; err != nil {
		t.Fatal(err)
	}
}

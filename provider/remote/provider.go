package remote

import (
	"context"
	"crypto/sha256"
	stdJSON "encoding/json"
	"errors"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sagernet/sing-box/adapter"
	boxProvider "github.com/sagernet/sing-box/adapter/provider"
	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing-box/option"
	E "github.com/sagernet/sing/common/exceptions"
	F "github.com/sagernet/sing/common/format"
	"github.com/sagernet/sing/common/json"
	"github.com/sagernet/sing/common/json/badjson"
	"github.com/sagernet/sing/common/x/list"
	"github.com/sagernet/sing/service"
)

const (
	defaultUpdateInterval = 24 * time.Hour
	requestTimeout        = 30 * time.Second
	minRetryDelay         = time.Minute
	maximumContentLength  = 16 << 20
)

func Register(registry *boxProvider.Registry) {
	boxProvider.Register[option.RemoteProviderOptions](registry, C.ProviderTypeRemote, New)
}

// node is a started provider outbound. Its scope owns the outbound's lifecycle.
type node struct {
	fingerprint [sha256.Size]byte
	outbound    adapter.Outbound
	scope       *adapter.Scope
	// detour is the provider node this one was started on. It stays open while
	// this node is in use, even after its tag moved to a newer instance.
	detour *node
}

type snapshot struct {
	nodes     []node
	outbounds []adapter.Outbound
}

type Provider struct {
	ctx              context.Context
	cancel           context.CancelFunc
	logger           log.ContextLogger
	tag              string
	options          option.RemoteProviderOptions
	updateInterval   time.Duration
	outboundRegistry adapter.OutboundRegistry
	outboundManager  adapter.DynamicOutboundManager
	router           adapter.Router
	logFactory       log.Factory
	httpManager      adapter.HTTPClientManager
	cacheFile        adapter.CacheFile
	transport        adapter.HTTPTransport

	// updateAccess serializes updates with each other and with close.
	updateAccess sync.Mutex
	loopGroup    sync.WaitGroup
	current      atomic.Pointer[snapshot]

	stateAccess      sync.RWMutex
	updating         bool
	lastUpdated      time.Time
	lastEtag         string
	lastContent      []byte
	lastError        error
	subscriptionInfo adapter.OutboundProviderSubscriptionInfo

	callbackAccess sync.Mutex
	callbacks      list.List[adapter.OutboundProviderCallback]
}

func New(ctx context.Context, logger log.ContextLogger, tag string, options option.RemoteProviderOptions) (adapter.OutboundProvider, error) {
	if options.URL == "" {
		return nil, E.New("provider URL is required")
	}
	if options.HTTPClient == "" {
		return nil, E.New("provider HTTP client is required")
	}
	updateInterval := time.Duration(options.UpdateInterval)
	if updateInterval <= 0 {
		updateInterval = defaultUpdateInterval
	}
	updateInterval = max(updateInterval, time.Minute)
	outboundManager, _ := service.FromContext[adapter.OutboundManager](ctx).(adapter.DynamicOutboundManager)
	ctx, cancel := context.WithCancel(ctx)
	provider := &Provider{
		ctx:              ctx,
		cancel:           cancel,
		logger:           logger,
		tag:              tag,
		options:          options,
		updateInterval:   updateInterval,
		outboundRegistry: service.FromContext[adapter.OutboundRegistry](ctx),
		outboundManager:  outboundManager,
		router:           service.FromContext[adapter.Router](ctx),
		logFactory:       service.FromContext[log.Factory](ctx),
		httpManager:      service.FromContext[adapter.HTTPClientManager](ctx),
		cacheFile:        service.FromContext[adapter.CacheFile](ctx),
	}
	provider.current.Store(&snapshot{})
	if provider.outboundRegistry == nil || provider.outboundManager == nil || provider.router == nil || provider.logFactory == nil || provider.httpManager == nil {
		cancel()
		return nil, E.New("provider dependencies are unavailable")
	}
	return provider, nil
}

func (p *Provider) Type() string { return C.ProviderTypeRemote }
func (p *Provider) Tag() string  { return p.tag }

func (p *Provider) Start(stage adapter.StartStage, scope *adapter.Scope) error {
	if stage != adapter.StartStateStart {
		return nil
	}
	scope.Add(p.close)
	transport, err := p.httpManager.ResolveTransport(p.ctx, p.logger, option.HTTPClientOptions{Tag: p.options.HTTPClient})
	if err != nil {
		return E.Cause(err, "create provider HTTP transport")
	}
	p.transport = transport
	if !p.loadCache() {
		err = p.Update(p.ctx)
		if err != nil {
			if len(p.current.Load().nodes) == 0 {
				return E.Cause(err, "initial provider update")
			}
			p.logger.Warn("initial provider update partially succeeded: ", err)
		}
	}
	p.loopGroup.Go(p.loopUpdate)
	return nil
}

func (p *Provider) close() error {
	p.cancel()
	p.loopGroup.Wait()
	p.updateAccess.Lock()
	defer p.updateAccess.Unlock()
	empty := &snapshot{}
	old := p.current.Swap(empty)
	p.notify()
	return p.retire(old, empty)
}

func (p *Provider) Outbounds() []adapter.Outbound {
	return p.current.Load().outbounds
}

func (p *Provider) UpdatedAt() time.Time {
	p.stateAccess.RLock()
	defer p.stateAccess.RUnlock()
	return p.lastUpdated
}

func (p *Provider) Updating() bool {
	p.stateAccess.RLock()
	defer p.stateAccess.RUnlock()
	return p.updating
}

func (p *Provider) LastError() error {
	p.stateAccess.RLock()
	defer p.stateAccess.RUnlock()
	return p.lastError
}

func (p *Provider) SubscriptionInfo() adapter.OutboundProviderSubscriptionInfo {
	p.stateAccess.RLock()
	defer p.stateAccess.RUnlock()
	return p.subscriptionInfo
}

func (p *Provider) RegisterCallback(callback adapter.OutboundProviderCallback) *list.Element[adapter.OutboundProviderCallback] {
	p.callbackAccess.Lock()
	defer p.callbackAccess.Unlock()
	return p.callbacks.PushBack(callback)
}

func (p *Provider) UnregisterCallback(element *list.Element[adapter.OutboundProviderCallback]) {
	p.callbackAccess.Lock()
	defer p.callbackAccess.Unlock()
	p.callbacks.Remove(element)
}

func (p *Provider) notify() {
	p.callbackAccess.Lock()
	callbacks := p.callbacks.Array()
	p.callbackAccess.Unlock()
	for _, callback := range callbacks {
		callback()
	}
}

func (p *Provider) loadCache() bool {
	if p.cacheFile == nil {
		return false
	}
	saved := p.cacheFile.LoadOutboundProvider(p.tag)
	if saved == nil {
		return false
	}
	p.updateAccess.Lock()
	defer p.updateAccess.Unlock()
	err := p.apply(saved.Content)
	if len(p.current.Load().nodes) == 0 {
		return false
	}
	p.stateAccess.Lock()
	p.lastContent = saved.Content
	p.lastUpdated = saved.LastUpdated
	// Without the ETag the next update downloads the content again and retries
	// the nodes that failed here.
	if err == nil {
		p.lastEtag = saved.LastEtag
	}
	p.subscriptionInfo = saved.SubscriptionInfo
	p.lastError = err
	p.stateAccess.Unlock()
	p.notify()
	return true
}

func (p *Provider) Update(ctx context.Context) error {
	if !p.updateAccess.TryLock() {
		return adapter.ErrProviderUpdating
	}
	defer p.updateAccess.Unlock()
	err := p.ctx.Err()
	if err != nil {
		return err
	}
	p.stateAccess.Lock()
	p.updating = true
	p.stateAccess.Unlock()
	p.notify()
	err = p.fetch(ctx)
	p.stateAccess.Lock()
	p.updating = false
	p.lastError = err
	p.stateAccess.Unlock()
	p.notify()
	return err
}

func (p *Provider) fetch(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()
	defer context.AfterFunc(p.ctx, cancel)()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, p.options.URL, nil)
	if err != nil {
		return err
	}
	p.stateAccess.RLock()
	lastEtag := p.lastEtag
	p.stateAccess.RUnlock()
	if lastEtag != "" {
		request.Header.Set("If-None-Match", lastEtag)
	}
	userAgent := p.options.UserAgent
	if userAgent == "" {
		userAgent = "sing-box " + C.Version
	}
	request.Header.Set("User-Agent", userAgent)
	for name, values := range p.options.Headers.Build() {
		request.Header[name] = append([]string(nil), values...)
	}
	response, err := (&http.Client{Transport: p.transport}).Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	info, hasInfo := parseSubscriptionInfo(response.Header.Get("subscription-userinfo"))
	var (
		content []byte
		etag    string
	)
	switch response.StatusCode {
	case http.StatusNotModified:
		if len(p.current.Load().nodes) == 0 {
			return errors.New("provider returned 304 without cached content")
		}
	case http.StatusOK:
		content, err = readLimited(response.Body)
		if err != nil {
			return err
		}
		err = p.apply(content)
		etag = response.Header.Get("ETag")
	default:
		return E.New("unexpected HTTP status: ", response.Status)
	}
	p.stateAccess.Lock()
	p.lastUpdated = time.Now()
	if hasInfo {
		p.subscriptionInfo = info
	}
	// Only a fully applied document reproduces its snapshot after restart; a
	// partial snapshot keeps instances from the previous content.
	save := err == nil
	if save && content != nil {
		p.lastContent = content
		p.lastEtag = etag
	}
	saved := &adapter.SavedOutboundProvider{
		Content:          p.lastContent,
		LastUpdated:      p.lastUpdated,
		LastEtag:         p.lastEtag,
		SubscriptionInfo: p.subscriptionInfo,
	}
	p.stateAccess.Unlock()
	if save && p.cacheFile != nil {
		saveErr := p.cacheFile.SaveOutboundProvider(p.tag, saved)
		if saveErr != nil {
			p.logger.Error("save outbound provider cache: ", saveErr)
		}
	}
	return err
}

// apply replaces the running nodes with the nodes in content. Nodes that cannot
// be started keep their previous instance, so a partial result is still applied.
func (p *Provider) apply(content []byte) error {
	document, err := p.parse(content)
	if err != nil {
		return err
	}
	old := p.current.Load()
	order, err := document.order(old)
	if err != nil {
		return errors.Join(document.err, err)
	}
	next, err := p.build(document, order, old)
	err = errors.Join(document.err, err)
	if len(next.nodes) == 0 {
		return errors.Join(err, errors.New("provider contains no valid proxy outbounds"))
	}
	p.current.Store(next)
	p.notify()
	return errors.Join(err, p.retire(old, next))
}

type candidate struct {
	option      option.Outbound
	fingerprint [sha256.Size]byte
	detour      string
}

type document struct {
	// tags lists every node the document defines, in order, including nodes
	// that failed to decode.
	tags       []string
	candidates []candidate
	indexByTag map[string]int
	// err collects nodes that failed to decode; their previous instances are kept.
	err error
}

func (p *Provider) parse(content []byte) (*document, error) {
	var raw struct {
		Outbounds []stdJSON.RawMessage `json:"outbounds"`
	}
	err := stdJSON.Unmarshal(content, &raw)
	if err != nil {
		return nil, E.Cause(err, "decode provider document")
	}
	tagPrefix := p.tag + "/"
	if p.options.TagPrefix != nil {
		tagPrefix = *p.options.TagPrefix
	}
	type entry struct {
		raw stdJSON.RawMessage
		tag string
	}
	entries := make([]entry, 0, len(raw.Outbounds))
	// A detour names the first node with that tag; later ones are renamed "tag #n".
	localTags := make(map[string]string)
	counts := make(map[string]int)
	for index, rawOutbound := range raw.Outbounds {
		var metadata struct {
			Type string `json:"type"`
			Tag  string `json:"tag"`
		}
		if stdJSON.Unmarshal(rawOutbound, &metadata) != nil {
			continue
		}
		tag := strings.TrimSpace(metadata.Tag)
		if tag == "" {
			tag = strconv.Itoa(index)
		}
		if p.filtered(tag) || skippedOutboundType(metadata.Type) {
			continue
		}
		counts[tag]++
		fullTag := tagPrefix + tag
		if counts[tag] > 1 {
			fullTag += " #" + strconv.Itoa(counts[tag])
		} else {
			localTags[tag] = fullTag
		}
		entries = append(entries, entry{rawOutbound, fullTag})
	}
	result := &document{indexByTag: make(map[string]int, len(entries))}
	seen := make(map[string]bool, len(entries))
	for _, item := range entries {
		if seen[item.tag] {
			return nil, E.New("duplicate provider outbound tag: ", item.tag)
		}
		seen[item.tag] = true
		result.tags = append(result.tags, item.tag)
		outboundCandidate, decodeErr := p.decode(item.raw, item.tag, localTags)
		if decodeErr != nil {
			result.err = errors.Join(result.err, decodeErr)
			continue
		}
		result.indexByTag[item.tag] = len(result.candidates)
		result.candidates = append(result.candidates, outboundCandidate)
	}
	return result, nil
}

func (p *Provider) decode(raw stdJSON.RawMessage, tag string, localTags map[string]string) (candidate, error) {
	outboundOptions, err := json.UnmarshalExtendedContext[option.Outbound](p.ctx, raw)
	if err != nil {
		return candidate{}, E.Cause(err, "decode provider outbound ", tag)
	}
	var detour string
	if wrapper, isWrapper := outboundOptions.Options.(option.DialerOptionsWrapper); isWrapper {
		dialerOptions := wrapper.TakeDialerOptions()
		if p.options.OverrideDialerOptions != nil {
			dialerOptions, err = overrideDialerOptions(p.ctx, dialerOptions, p.options.OverrideDialerOptions)
			if err != nil {
				return candidate{}, E.Cause(err, "override dialer options for provider outbound ", tag)
			}
		}
		if localTag, loaded := localTags[dialerOptions.Detour]; loaded {
			dialerOptions.Detour = localTag
		}
		detour = dialerOptions.Detour
		wrapper.ReplaceDialerOptions(dialerOptions)
	}
	outboundOptions.Tag = tag
	canonical, err := json.MarshalContext(p.ctx, &outboundOptions)
	if err != nil {
		return candidate{}, E.Cause(err, "fingerprint provider outbound ", tag)
	}
	return candidate{option: outboundOptions, fingerprint: sha256.Sum256(canonical), detour: detour}, nil
}

// order sorts candidates so that every node comes after its detour.
func (d *document) order(old *snapshot) ([]int, error) {
	// Provider tags, true for those this document defines.
	providerTags := make(map[string]bool, len(d.tags)+len(old.nodes))
	for _, item := range old.nodes {
		providerTags[item.outbound.Tag()] = false
	}
	for _, tag := range d.tags {
		providerTags[tag] = true
	}
	const (
		visiting = 1
		visited  = 2
	)
	state := make([]uint8, len(d.candidates))
	result := make([]int, 0, len(d.candidates))
	var visit func(index int) error
	visit = func(index int) error {
		switch state[index] {
		case visited:
			return nil
		case visiting:
			return E.New("circular provider detour involving ", d.candidates[index].option.Tag)
		}
		state[index] = visiting
		detour := d.candidates[index].detour
		if dependency, loaded := d.indexByTag[detour]; loaded {
			err := visit(dependency)
			if err != nil {
				return err
			}
		} else if defined, isProviderTag := providerTags[detour]; isProviderTag && !defined {
			return E.New("provider detour node not found: ", detour)
		}
		state[index] = visited
		result = append(result, index)
		return nil
	}
	for index := range d.candidates {
		err := visit(index)
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

// build starts changed nodes in dependency order and publishes each one as soon
// as it is started, so a node's detour is always registered before the node.
func (p *Provider) build(document *document, order []int, old *snapshot) (*snapshot, error) {
	oldByTag := make(map[string]node, len(old.nodes))
	for _, item := range old.nodes {
		oldByTag[item.outbound.Tag()] = item
	}
	processed := make(map[string]node, len(document.tags))
	failed := make(map[string]bool)
	keepPrevious := func(tag string) {
		failed[tag] = true
		if previous, loaded := oldByTag[tag]; loaded {
			processed[tag] = previous
		}
	}
	for _, tag := range document.tags {
		if _, decoded := document.indexByTag[tag]; !decoded {
			keepPrevious(tag)
		}
	}
	var result error
	for _, index := range order {
		outboundCandidate := document.candidates[index]
		tag := outboundCandidate.option.Tag
		previous, existed := oldByTag[tag]
		if failed[outboundCandidate.detour] {
			result = errors.Join(result, E.New("provider detour node failed: ", tag, " -> ", outboundCandidate.detour))
			keepPrevious(tag)
			continue
		}
		var dependency *node
		if item, isLocal := processed[outboundCandidate.detour]; isLocal {
			// A node is recreated whenever its detour is, since it is bound to
			// that detour instance.
			dependency = &item
			outboundCandidate.fingerprint = sha256.Sum256(append(outboundCandidate.fingerprint[:], dependency.fingerprint[:]...))
		}
		current, loaded := p.outboundManager.Outbound(tag)
		if loaded && (!existed || current != previous.outbound) {
			p.logger.Warn("skip provider outbound ", tag, ": tag is already used by outbound/", current.Type(), "[", current.Tag(), "]")
			failed[tag] = true
			continue
		}
		if existed && previous.fingerprint == outboundCandidate.fingerprint {
			processed[tag] = previous
			continue
		}
		created, err := p.startNode(outboundCandidate, dependency)
		if err != nil {
			result = errors.Join(result, E.Cause(err, "create provider outbound ", tag))
			keepPrevious(tag)
			continue
		}
		err = p.outboundManager.SwapDynamic(tag, previous.outbound, created.outbound)
		if err != nil {
			result = errors.Join(result, err, created.scope.Close())
			failed[tag] = true
			continue
		}
		processed[tag] = created
	}
	next := &snapshot{nodes: make([]node, 0, len(document.tags)), outbounds: make([]adapter.Outbound, 0, len(document.tags))}
	for _, tag := range document.tags {
		if item, loaded := processed[tag]; loaded {
			next.nodes = append(next.nodes, item)
			next.outbounds = append(next.outbounds, item.outbound)
		}
	}
	return next, result
}

func (p *Provider) startNode(outboundCandidate candidate, dependency *node) (node, error) {
	outboundOptions := outboundCandidate.option
	tag := outboundOptions.Tag
	scope := adapter.NewScope(p.ctx, p.logger)
	ctx := scope.Context()
	if dependency != nil {
		// A detour dialer looks its tag up on first dial, when the tag may already
		// point to a newer instance; pin it to the instance retire keeps open.
		ctx = service.ContextWith[adapter.OutboundManager](service.ExtendContext(ctx), &detourOutboundManager{
			OutboundManager: p.outboundManager,
			detour:          dependency.outbound,
		})
	}
	outbound, err := p.outboundRegistry.CreateOutbound(
		adapter.WithContext(ctx, &adapter.InboundContext{Outbound: tag}),
		p.router,
		p.logFactory.NewLogger(F.ToString("outbound/", outboundOptions.Type, "[", tag, "]")),
		tag,
		outboundOptions.Type,
		outboundOptions.Options,
	)
	if err != nil {
		return node{}, errors.Join(err, scope.Close())
	}
	if lifecycle, isLifecycle := outbound.(adapter.Lifecycle); isLifecycle {
		name := "outbound/" + outboundOptions.Type + "[" + tag + "]"
		for _, stage := range adapter.ListStartStages {
			err = scope.Start(name, lifecycle, stage)
			if err != nil {
				return node{}, errors.Join(err, scope.Close())
			}
		}
	}
	return node{fingerprint: outboundCandidate.fingerprint, outbound: outbound, scope: scope, detour: dependency}, nil
}

// detourOutboundManager resolves the detour tag of a provider node to the
// instance the node was started on.
type detourOutboundManager struct {
	adapter.OutboundManager
	detour adapter.Outbound
}

func (m *detourOutboundManager) Outbound(tag string) (adapter.Outbound, bool) {
	if tag == m.detour.Tag() {
		return m.detour, true
	}
	return m.OutboundManager.Outbound(tag)
}

// retire closes nodes of old that next no longer uses, after callbacks have
// moved groups to next. A node in use keeps the detour chain it was started on.
func (p *Provider) retire(old *snapshot, next *snapshot) error {
	nextTags := make(map[string]bool, len(next.nodes))
	// skip holds scopes still in use by next or already closed.
	skip := make(map[*adapter.Scope]bool)
	for _, item := range next.nodes {
		nextTags[item.outbound.Tag()] = true
		for current := &item; current != nil; current = current.detour {
			skip[current.scope] = true
		}
	}
	var result error
	for _, item := range old.nodes {
		tag := item.outbound.Tag()
		if !nextTags[tag] {
			result = errors.Join(result, p.outboundManager.SwapDynamic(tag, item.outbound, nil))
		}
		for current := &item; current != nil; current = current.detour {
			if skip[current.scope] {
				continue
			}
			skip[current.scope] = true
			err := current.scope.Close()
			if err != nil {
				result = errors.Join(result, E.Cause(err, "close provider outbound ", current.outbound.Tag()))
			}
		}
	}
	return result
}

func (p *Provider) loopUpdate() {
	delay := max(time.Until(p.UpdatedAt().Add(p.updateInterval)), 0)
	timer := time.NewTimer(delay)
	defer timer.Stop()
	var retryDelay time.Duration
	for {
		select {
		case <-p.ctx.Done():
			return
		case <-timer.C:
		}
		err := p.Update(p.ctx)
		if err != nil && !errors.Is(err, adapter.ErrProviderUpdating) && p.ctx.Err() == nil {
			p.logger.Error("update outbound provider: ", err)
			// Retry sooner, e.g. when the network was not up yet.
			retryDelay = min(max(retryDelay*2, minRetryDelay), p.updateInterval)
			timer.Reset(retryDelay)
			continue
		}
		retryDelay = 0
		timer.Reset(p.updateInterval)
	}
}

func (p *Provider) filtered(tag string) bool {
	if p.options.Exclude != nil && p.options.Exclude.Build().MatchString(tag) {
		return true
	}
	return p.options.Include != nil && !p.options.Include.Build().MatchString(tag)
}

func overrideDialerOptions(ctx context.Context, current option.DialerOptions, override *option.DialerOptionsOverride) (option.DialerOptions, error) {
	currentContent, err := json.MarshalContext(ctx, current)
	if err != nil {
		return option.DialerOptions{}, E.Cause(err, "marshal current dialer options")
	}
	overrideContent, err := json.MarshalContext(ctx, override)
	if err != nil {
		return option.DialerOptions{}, E.Cause(err, "marshal dialer options override")
	}
	var currentObject badjson.JSONObject
	err = currentObject.UnmarshalJSONContext(ctx, currentContent)
	if err != nil {
		return option.DialerOptions{}, E.Cause(err, "decode current dialer options")
	}
	var overrideObject badjson.JSONObject
	err = overrideObject.UnmarshalJSONContext(ctx, overrideContent)
	if err != nil {
		return option.DialerOptions{}, E.Cause(err, "decode dialer options override")
	}
	for _, entry := range overrideObject.Entries() {
		currentObject.Put(entry.Key, entry.Value)
	}
	mergedContent, err := currentObject.MarshalJSONContext(ctx)
	if err != nil {
		return option.DialerOptions{}, E.Cause(err, "marshal merged dialer options")
	}
	var merged option.DialerOptions
	err = json.UnmarshalContextDisallowUnknownFields(ctx, mergedContent, &merged)
	if err != nil {
		return option.DialerOptions{}, E.Cause(err, "decode merged dialer options")
	}
	return merged, nil
}

func readLimited(reader io.Reader) ([]byte, error) {
	content, err := io.ReadAll(io.LimitReader(reader, maximumContentLength+1))
	if err != nil {
		return nil, err
	}
	if len(content) > maximumContentLength {
		return nil, errors.New("provider response exceeds 16 MiB")
	}
	return content, nil
}

var subscriptionInfoRegex = regexp.MustCompile(`(?i)(upload|download|total|expire)\s*=\s*(-?\d+)`)

func parseSubscriptionInfo(value string) (adapter.OutboundProviderSubscriptionInfo, bool) {
	var info adapter.OutboundProviderSubscriptionInfo
	matches := subscriptionInfoRegex.FindAllStringSubmatch(value, -1)
	if len(matches) == 0 {
		return info, false
	}
	for _, match := range matches {
		number, err := strconv.ParseInt(match[2], 10, 64)
		if err != nil {
			continue
		}
		switch strings.ToLower(match[1]) {
		case "upload":
			info.Upload = number
		case "download":
			info.Download = number
		case "total":
			info.Total = number
		case "expire":
			info.ExpireTime = time.Unix(number, 0)
		}
	}
	return info, true
}

func skippedOutboundType(outboundType string) bool {
	switch outboundType {
	case C.TypeDirect, C.TypeBlock, C.TypeDNS, C.TypeSelector, C.TypeURLTest:
		return true
	default:
		return false
	}
}

var _ adapter.OutboundProvider = (*Provider)(nil)

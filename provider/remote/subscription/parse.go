// This file is part of a Go port of Sub-Store (https://github.com/sub-store-org/Sub-Store),
// licensed under the GNU Affero General Public License v3.0. See NOTICE in this directory.

package subscription

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"slices"
	"strings"
)

// core/proxy-utils/index.js: parse, tryParse and lastParse;
// core/proxy-utils/parsers/index.js: Clash_All.

var parsers = append(append([]parser(nil), uriParsers...), parser{"Clash Parser", clashAllTest, clashAllParse})

func parse(raw string) []*Object {
	raw = preprocess(raw)
	var proxies []*Object
	var lastParser *parser
	for line := range strings.SplitSeq(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		success := false
		if lastParser != nil {
			if proxy, ok := tryParse(lastParser, line); ok {
				proxies = append(proxies, lastParse(proxy))
				success = true
			}
		}
		if !success {
			for i := range parsers {
				if proxy, ok := tryParse(&parsers[i], line); ok {
					proxies = append(proxies, lastParse(proxy))
					lastParser = &parsers[i]
					break
				}
			}
		}
	}
	result := proxies[:0]
	for _, proxy := range proxies {
		if proxy.Get("type") == "hysteria2" && truthy(proxy.Get("obfs")) && !truthy(proxy.Get("obfs-password")) {
			continue
		}
		result = append(result, proxy)
	}
	return result
}

func tryParse(p *parser, line string) (proxy *Object, ok bool) {
	defer func() {
		if recover() != nil {
			proxy, ok = nil, false
		}
	}()
	if !safeMatch(p, line) {
		return nil, false
	}
	proxy = p.parse(line)
	return proxy, proxy != nil
}

func safeMatch(p *parser, line string) (matched bool) {
	defer func() {
		if recover() != nil {
			matched = false
		}
	}()
	return p.test(line)
}

// Clash_All

var clashTypes = []string{
	"zerotier", "shadowquic", "gost-relay", "openvpn", "tailscale", "easytier", "trusttunnel", "h2-connect", "naive",
	"anytls", "mieru", "masque", "masque-surge", "sudoku", "juicity", "ss", "ssr", "vmess", "socks5", "http", "snell",
	"trojan", "tuic", "vless", "hysteria", "hysteria2", "wireguard", "ssh", "direct",
}

func clashLoad(line string) any {
	value, err := jsonParse(line)
	if err != nil {
		value, err = yamlLoad(line)
		if err != nil {
			panic(err)
		}
	}
	return value
}

func clashAllTest(line string) bool {
	proxy, isObject := clashLoad(line).(*Object)
	return isObject && truthy(proxy.Get("type"))
}

func clashAllParse(line string) *Object {
	proxy, isObject := clashLoad(line).(*Object)
	if !isObject || !includesString(proxy.Get("type"), clashTypes...) {
		throw("Clash does not support proxy with type: " + jsString(proxy.Get("type")))
	}
	if includesString(proxy.Get("type"), "vmess", "vless") && truthy(proxy.Get("servername")) {
		proxy.Set("sni", proxy.Get("servername"))
		proxy.Delete("servername")
	}
	if truthy(proxy.Get("server-cert-fingerprint")) {
		proxy.Set("tls-fingerprint", proxy.Get("server-cert-fingerprint"))
	}
	if truthy(proxy.Get("fingerprint")) {
		proxy.Set("tls-fingerprint", proxy.Get("fingerprint"))
	}
	if truthy(proxy.Get("dialer-proxy")) {
		proxy.Set("underlying-proxy", proxy.Get("dialer-proxy"))
	}
	if truthy(proxy.Get("benchmark-url")) {
		proxy.Set("test-url", proxy.Get("benchmark-url"))
	}
	if truthy(proxy.Get("benchmark-timeout")) {
		proxy.Set("test-timeout", proxy.Get("benchmark-timeout"))
	}
	if proxy.Get("type") == "vmess" {
		proxy.Set("cipher", normalizeVmessSecurity(proxy.Get("cipher")))
	}
	return proxy
}

// lastParse

var (
	hopIntervalRangeRegex = regexp.MustCompile(`^(\d+)\s*-\s*(\d+)$`)
	leadingBracketRegex   = regexp.MustCompile(`^\[`)
	trailingBracketRegex  = regexp.MustCompile(`\]$`)
	slashRegex            = regexp.MustCompile(`/`)
)

func formatTransportPath(path any) any {
	switch p := path.(type) {
	case string, float64:
		s := strings.TrimSpace(jsString(p))
		if s == "" {
			return "/"
		}
		if !strings.HasPrefix(s, "/") {
			return "/" + s
		}
		return s
	}
	return path
}

func normalizeOptsKeys(value any) {
	o, isObject := value.(*Object)
	if !isObject {
		return
	}
	for _, key := range o.Keys() {
		normalizedKey := strings.ToLower(key)
		if key != normalizedKey {
			if !o.Has(normalizedKey) {
				o.Set(normalizedKey, o.Get(key))
			}
			o.Delete(key)
		}
	}
}

// opts returns proxy[`${proxy.network}-opts`].
func networkOpts(proxy *Object) *Object {
	return proxy.Obj(jsString(proxy.Get("network")) + "-opts")
}

func networkOptsKey(proxy *Object) string {
	return jsString(proxy.Get("network")) + "-opts"
}

func lastParse(proxy *Object) *Object {
	for _, key := range proxy.Keys() {
		normalizedKey := strings.ToLower(key)
		if !strings.HasSuffix(normalizedKey, "-opts") {
			continue
		}
		if key != normalizedKey {
			if !proxy.Has(normalizedKey) {
				proxy.Set(normalizedKey, proxy.Get(key))
			}
			proxy.Delete(key)
		}
		normalizeOptsKeys(proxy.Get(normalizedKey))
	}
	if realityOpts := proxy.Obj("reality-opts"); truthy(realityOpts.Get("public-key")) && !realityOpts.Has("support-x25519mlkem768") &&
		includesString(strings.TrimSpace(jsString(or(proxy.Get("_loon_tls_profile"), ""))), "safari-ios-26", "chrome147") {
		realityOpts.Set("support-x25519mlkem768", true)
	}
	udp := proxy.Get("udp")
	if s, isString := udp.(string); isString {
		udp = strings.ToLower(s)
	}
	proxy.Set("udp", !includes(udp, false, float64(0), "0", "false", "off"))
	if cipher, isString := proxy.Get("cipher").(string); isString {
		proxy.Set("cipher", strings.ToLower(cipher))
	}
	if password, isNumber := proxy.Get("password").(float64); isNumber {
		proxy.Set("password", numberToString(password))
	}
	if !isNullish(proxy.Get("hop-interval")) {
		hopInterval := strings.TrimSpace(jsString(proxy.Get("hop-interval")))
		if match := hopIntervalRangeRegex.FindStringSubmatch(hopInterval); match != nil {
			hopIntervalMin := parseInt(match[1])
			hopIntervalMax := parseInt(match[2])
			if hopIntervalMin > 0 && hopIntervalMin <= hopIntervalMax {
				proxy.Set("hop-interval", hopIntervalMin)
				proxy.Set("hop-interval-max", hopIntervalMax)
			} else {
				proxy.Delete("hop-interval")
				proxy.Delete("hop-interval-max")
			}
		} else if digitsOnlyRegex.MatchString(hopInterval) {
			parsedHopInterval := parseInt(hopInterval)
			if parsedHopInterval > 0 {
				proxy.Set("hop-interval", parsedHopInterval)
				proxy.Delete("hop-interval-max")
			} else {
				proxy.Delete("hop-interval")
				proxy.Delete("hop-interval-max")
			}
		} else {
			proxy.Delete("hop-interval")
			proxy.Delete("hop-interval-max")
		}
	}
	if proxy.Get("type") == "ss" && proxy.Get("cipher") == "none" && !truthy(proxy.Get("password")) {
		proxy.Set("password", "")
	}
	if truthy(proxy.Get("interface")) {
		proxy.Set("interface-name", proxy.Get("interface"))
		proxy.Delete("interface")
	}
	if isValidPortNumber(proxy.Get("port")) {
		proxy.Set("port", parseInt(proxy.Get("port")))
	}
	if truthy(proxy.Get("server")) {
		server := strings.TrimSpace(jsString(proxy.Get("server")))
		server = leadingBracketRegex.ReplaceAllString(server, "")
		server = trailingBracketRegex.ReplaceAllString(server, "")
		proxy.Set("server", server)
	}
	if includesString(proxy.Get("type"), "vmess", "vless", "trojan", "anytls") && truthy(proxy.Get("shadow-tls-opts")) {
		shadowTLSOpts := proxy.Get("shadow-tls-opts")
		proxy.Set("plugin", "shadow-tls")
		proxy.Set("plugin-opts", obj(
			"host", proxy.Get("sni"),
			"password", getPath(shadowTLSOpts, "password"),
			"version", getPath(shadowTLSOpts, "version"),
		))
		proxy.Delete("shadow-tls-opts")
	}
	if proxy.Get("type") == "snell" && getPath(proxy.Get("obfs-opts"), "mode") == "shadow-tls" && !truthy(proxy.Get("plugin")) {
		obfsOpts := proxy.Get("obfs-opts")
		proxy.Set("plugin", "shadow-tls")
		proxy.Set("plugin-opts", obj(
			"host", getPath(obfsOpts, "host"),
			"password", getPath(obfsOpts, "password"),
			"version", getPath(obfsOpts, "version"),
			"alpn", getPath(obfsOpts, "alpn"),
		))
		proxy.Delete("obfs-opts")
	}
	if proxy.Get("plugin") == "shadow-tls" && truthy(proxy.Get("plugin-opts")) {
		pluginOpts, isObject := proxy.Get("plugin-opts").(*Object)
		if truthy(proxy.Get("alpn")) && isObject && !truthy(pluginOpts.Get("alpn")) {
			pluginOpts.Set("alpn", proxy.Get("alpn"))
		}
		proxy.Delete("alpn")
	}
	if proxy.Get("network") == "ws" {
		if !truthy(proxy.Get("ws-opts")) && (truthy(proxy.Get("ws-path")) || truthy(proxy.Get("ws-headers"))) {
			wsOpts := newObject()
			if truthy(proxy.Get("ws-path")) {
				wsOpts.Set("path", proxy.Get("ws-path"))
			}
			if truthy(proxy.Get("ws-headers")) {
				wsOpts.Set("headers", proxy.Get("ws-headers"))
			}
			proxy.Set("ws-opts", wsOpts)
		}
		proxy.Delete("ws-path")
		proxy.Delete("ws-headers")
	}
	if opts := networkOpts(proxy); opts != nil {
		transportPath := opts.Get("path")
		if array, isArray := transportPath.([]any); isArray {
			formatted := make([]any, len(array))
			for i, item := range array {
				formatted[i] = formatTransportPath(item)
			}
			opts.Set("path", formatted)
		} else if !isNullish(transportPath) {
			opts.Set("path", formatTransportPath(transportPath))
		}
	}
	if proxy.Get("type") == "trojan" {
		proxy.Set("network", or(proxy.Get("network"), "tcp"))
	}
	if proxy.Get("type") == "vmess" {
		proxy.Set("network", or(proxy.Get("network"), "tcp"))
		proxy.Set("cipher", or(proxy.Get("cipher"), "none"))
		proxy.Set("alterId", or(proxy.Get("alterId"), float64(0)))
	}
	if proxy.Get("type") == "vless" {
		proxy.Set("network", or(proxy.Get("network"), "tcp"))
	}
	if includesString(proxy.Get("type"), "vmess", "vless") && isNullish(proxy.Get("packet-encoding")) {
		if truthy(proxy.Get("xudp")) {
			proxy.Set("packet-encoding", "xudp")
		} else if truthy(proxy.Get("packet-addr")) {
			proxy.Set("packet-encoding", "packetaddr")
		}
	}
	if includesString(proxy.Get("type"), "trojan", "tuic", "hysteria", "hysteria2", "juicity", "anytls", "trusttunnel", "h2-connect", "naive", "masque", "shadowquic") {
		proxy.Set("tls", true)
	}
	if truthy(proxy.Get("network")) {
		if opts := networkOpts(proxy); opts != nil {
			if headers := opts.Obj("headers"); headers != nil {
				if proxy.Get("network") != "h2" && truthy(headers.Get("host")) && !truthy(headers.Get("Host")) {
					headers.Set("Host", headers.Get("host"))
					headers.Delete("host")
				}
			}
		}
	}
	if proxy.Get("network") == "h2" {
		h2Opts := proxy.Obj("h2-opts")
		host := coalesce(h2Opts.Get("host"), getPath(h2Opts.Get("headers"), "host"), getPath(h2Opts.Get("headers"), "Host"), nil)
		path := h2Opts.Get("path")
		if truthy(host) {
			if _, isArray := host.([]any); !isArray {
				host = []any{host}
			}
			h2Opts.Set("host", host)
		}
		if headers := h2Opts.Get("headers"); truthy(headers) {
			if headersObject, isObject := headers.(*Object); isObject {
				headersObject.Delete("host")
				headersObject.Delete("Host")
			}
			if objectKeyCount(headers) == 0 {
				h2Opts.Delete("headers")
			}
		}
		if array, isArray := path.([]any); isArray {
			h2Opts.Set("path", arrayAt(array, 0))
		}
	}
	if !truthy(proxy.Get("tls")) && includesString(proxy.Get("network"), "ws", "http") &&
		!truthy(getPath(networkOpts(proxy), "headers.Host")) && !isIP(proxy.Get("server")) {
		key := networkOptsKey(proxy)
		opts, isObject := proxy.Get(key).(*Object)
		if !truthy(proxy.Get(key)) {
			opts = newObject()
			proxy.Set(key, opts)
		} else if !isObject {
			throw("TypeError: cannot set headers")
		}
		headers, isObject := opts.Get("headers").(*Object)
		if !truthy(opts.Get("headers")) {
			headers = newObject()
			opts.Set("headers", headers)
		} else if !isObject {
			throw("TypeError: cannot set Host")
		}
		if includesString(proxy.Get("type"), "vmess", "vless") && proxy.Get("network") == "http" {
			headers.Set("Host", []any{proxy.Get("server")})
		} else {
			headers.Set("Host", proxy.Get("server"))
		}
	}
	if includesString(proxy.Get("type"), "vmess", "vless") && proxy.Get("network") == "http" {
		opts := networkOpts(proxy)
		transportPath := opts.Get("path")
		transportHost := getPath(opts, "headers.Host")
		if truthy(transportHost) {
			if _, isArray := transportHost.([]any); !isArray {
				opts.Obj("headers").Set("Host", []any{transportHost})
			}
		}
		if truthy(transportPath) {
			if _, isArray := transportPath.([]any); !isArray {
				opts.Set("path", []any{transportPath})
			}
		}
	}
	if truthy(proxy.Get("tls")) && !truthy(proxy.Get("sni")) && proxy.Get("sni") != "" {
		if truthy(proxy.Get("network")) {
			var transportHost any
			if proxy.Get("network") == "h2" {
				transportHost = getPath(proxy.Get("h2-opts"), "host")
			} else {
				transportHost = getPath(networkOpts(proxy), "headers.Host")
			}
			if array, isArray := transportHost.([]any); isArray {
				transportHost = arrayAt(array, 0)
			}
			if truthy(transportHost) {
				proxy.Set("sni", transportHost)
			}
		}
		if !truthy(proxy.Get("sni")) && !isIP(proxy.Get("server")) {
			proxy.Set("sni", proxy.Get("server"))
		}
	}
	if truthy(proxy.Get("ports")) {
		proxy.Set("ports", slashRegex.ReplaceAllString(jsString(proxy.Get("ports")), ","))
	} else {
		proxy.Delete("ports")
	}
	if proxy.Get("type") == "hysteria2" && truthy(proxy.Get("obfs")) && proxy.Get("obfs") != "salamander" && !truthy(proxy.Get("obfs-password")) {
		proxy.Set("obfs-password", proxy.Get("obfs"))
		proxy.Set("obfs", "salamander")
	}
	if proxy.Get("type") == "hysteria2" && !truthy(proxy.Get("obfs-password")) && truthy(proxy.Get("obfs_password")) {
		proxy.Set("obfs-password", proxy.Get("obfs_password"))
		proxy.Delete("obfs_password")
	}
	if proxy.Get("type") == "vless" {
		if realityOpts := proxy.Get("reality-opts"); truthy(realityOpts) && objectKeyCount(realityOpts) == 0 {
			proxy.Delete("reality-opts")
		}
		if grpcOpts := proxy.Get("grpc-opts"); truthy(grpcOpts) && objectKeyCount(grpcOpts) == 0 {
			proxy.Delete("grpc-opts")
		}
		if (!truthy(proxy.Get("reality-opts")) && !truthy(proxy.Get("flow"))) || includes(proxy.Get("flow"), "null", jsNull) {
			proxy.Delete("flow")
		}
		if proxy.Get("network") == "http" {
			if !truthy(getPath(networkOpts(proxy), "path")) {
				key := networkOptsKey(proxy)
				if !truthy(proxy.Get(key)) {
					proxy.Set(key, newObject())
				}
				mustObject(proxy.Get(key)).Set("path", []any{"/"})
			}
		}
	}
	if _, isString := proxy.Get("name").(string); !isString {
		name := proxy.Get("name")
		if testString(digitsOnlyRegex, name) {
			proxy.Set("name", jsString(name))
		} else {
			proxy.Set("name", bufferToString(proxy))
		}
	}
	if includesString(proxy.Get("network"), "ws", "http", "h2") && proxy.Get("type") != "masque" {
		key := networkOptsKey(proxy)
		if includesString(proxy.Get("network"), "ws", "h2") && !truthy(getPath(proxy.Get(key), "path")) {
			opts, isObject := proxy.Get(key).(*Object)
			if !truthy(proxy.Get(key)) {
				opts = newObject()
				proxy.Set(key, opts)
			} else if !isObject {
				throw("TypeError: cannot set path")
			}
			opts.Set("path", "/")
		} else if proxy.Get("network") == "http" && !httpPathHasValue(getPath(proxy.Get(key), "path")) {
			opts, isObject := proxy.Get(key).(*Object)
			if !truthy(proxy.Get(key)) {
				opts = newObject()
				proxy.Set(key, opts)
			} else if !isObject {
				throw("TypeError: cannot set path")
			}
			opts.Set("path", []any{"/"})
		}
	}
	if proxy.Get("type") == "anytls" && truthy(proxy.Get("disable-reuse")) {
		proxy.Set("reuse", false)
	}
	if includes(proxy.Get("sni"), "", "off") {
		proxy.Set("disable-sni", true)
	}
	caStr := proxy.Get("ca_str")
	if truthy(proxy.Get("ca-str")) {
		caStr = proxy.Get("ca-str")
	} else if truthy(caStr) {
		proxy.Delete("ca_str")
		proxy.Set("ca-str", caStr)
	}
	if !truthy(proxy.Get("tls-fingerprint")) && truthy(caStr) {
		proxy.Set("tls-fingerprint", generateFingerprint(jsString(caStr)))
	}
	if proxy.Get("type") == "tuic" {
		if _, isArray := proxy.Get("alpn").([]any); !isArray {
			proxy.Set("alpn", []any{or(proxy.Get("alpn"), "h3")})
		}
		proxy.Set("congestion-controller", or(proxy.Get("congestion-controller"), "cubic"))
		proxy.Set("udp-relay-mode", or(proxy.Get("udp-relay-mode"), "native"))
	}
	return proxy
}

// httpPathHasValue is Array.isArray(path) && !path.every((i) => !i).
func httpPathHasValue(path any) bool {
	array, isArray := path.([]any)
	if !isArray {
		return false
	}
	return slices.ContainsFunc(array, truthy)
}

// bufferToString converts a non-string name like Buffer.from(...).toString().
func bufferToString(proxy *Object) string {
	name := proxy.Get("name")
	data := name
	if o, isObject := name.(*Object); isObject && truthy(o.Get("data")) {
		data = o.Get("data")
	}
	if array, isArray := data.([]any); isArray {
		content := make([]byte, len(array))
		for i, item := range array {
			content[i] = byte(int64(jsNumber(item)))
		}
		return strings.ToValidUTF8(string(content), "�")
	}
	return jsString(proxy.Get("type")) + " " + jsString(proxy.Get("server")) + ":" + jsString(proxy.Get("port"))
}

var (
	pemHeaderRegex = regexp.MustCompile(`^[\s\S]*-----BEGIN [^-]+-----`)
	pemFooterRegex = regexp.MustCompile(`-----END [^-]+-----[\s\S]*$`)
	whitespace     = regexp.MustCompile(`\s+`)
)

// generateFingerprint is utils/rs.js generateFingerprint.
func generateFingerprint(caStr string) string {
	if !strings.Contains(caStr, "-----BEGIN ") {
		throw("can't find PEM header")
	}
	body := pemHeaderRegex.ReplaceAllString(caStr, "")
	body = pemFooterRegex.ReplaceAllString(body, "")
	body = whitespace.ReplaceAllString(body, "")
	sum := sha256.Sum256(base64DecodeBytes(body))
	hexString := strings.ToUpper(hex.EncodeToString(sum[:]))
	parts := make([]string, 0, len(hexString)/2)
	for i := 0; i+2 <= len(hexString); i += 2 {
		parts = append(parts, hexString[i:i+2])
	}
	return strings.Join(parts, ":")
}

// objectKeyCount is Object.keys(value).length for a non-nullish value.
func objectKeyCount(value any) int {
	switch v := value.(type) {
	case *Object:
		return v.Len()
	case []any:
		return len(v)
	case string:
		return len([]rune(v))
	}
	return 0
}

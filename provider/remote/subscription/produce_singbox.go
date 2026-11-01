// This file is part of a Go port of Sub-Store (https://github.com/sub-store-org/Sub-Store),
// licensed under the GNU Affero General Public License v3.0. See NOTICE in this directory.

package subscription

import (
	"encoding/base64"
	"encoding/hex"
	"math"
	"regexp"
	"strings"
)

// core/proxy-utils/producers/sing-box.js, called without options.
//
// Not ported: the WireGuard and Tailscale parsers, which produce endpoints, and
// the SSR parser, which the producer only uses with "include-unsupported-proxy".

var singBoxIPVersions = map[string]string{
	"ipv4":        "ipv4_only",
	"ipv6":        "ipv6_only",
	"v4-only":     "ipv4_only",
	"v6-only":     "ipv6_only",
	"ipv4-prefer": "prefer_ipv4",
	"ipv6-prefer": "prefer_ipv6",
	"prefer-v4":   "prefer_ipv4",
	"prefer-v6":   "prefer_ipv6",
}

func ipVersionParser(proxy *Object, parsedProxy *Object) {
	strategy, loaded := singBoxIPVersions[jsString(proxy.Get("ip-version"))]
	if truthy(proxy.Get("_dns_server")) && loaded {
		parsedProxy.Set("domain_resolver", obj(
			"server", proxy.Get("_dns_server"),
			"strategy", strategy,
		))
	}
}

func domainResolverParser(proxy *Object, parsedProxy *Object) {
	resolver := proxy.Get("_domain_resolver")
	if !truthy(resolver) {
		return
	}
	result := newObject()
	if existing, isObject := parsedProxy.Get("domain_resolver").(*Object); isObject {
		result = existing.Clone()
	}
	if s, isString := resolver.(string); isString {
		result.Set("server", s)
	} else if o, isObject := resolver.(*Object); isObject {
		for _, key := range o.Keys() {
			result.Set(key, o.Get(key))
		}
	}
	parsedProxy.Set("domain_resolver", result)
}

func detourParser(proxy *Object, parsedProxy *Object) {
	parsedProxy.Set("detour", or(proxy.Get("dialer-proxy"), proxy.Get("detour")))
}

func networkParser(proxy *Object, parsedProxy *Object) {
	if includesString(proxy.Get("_network"), "tcp", "udp") {
		parsedProxy.Set("network", proxy.Get("_network"))
		return
	}
	if proxy.Get("udp") == false {
		parsedProxy.Set("network", "tcp")
	}
}

func tfoParser(proxy *Object, parsedProxy *Object) {
	parsedProxy.Set("tcp_fast_open", false)
	if truthy(proxy.Get("tfo")) || truthy(proxy.Get("tcp_fast_open")) || truthy(proxy.Get("tcp-fast-open")) {
		parsedProxy.Set("tcp_fast_open", true)
	}
	if !truthy(parsedProxy.Get("tcp_fast_open")) {
		parsedProxy.Delete("tcp_fast_open")
	}
}

func smuxParser(smux any, parsedProxy *Object) {
	if !truthy(smux) || !truthy(getPath(smux, "enabled")) {
		return
	}
	multiplex := obj("enabled", true)
	parsedProxy.Set("multiplex", multiplex)
	multiplex.Set("protocol", getPath(smux, "protocol"))
	if value := getPath(smux, "max-connections"); truthy(value) {
		multiplex.Set("max_connections", parseInt(value))
	}
	if value := getPath(smux, "max-streams"); truthy(value) {
		multiplex.Set("max_streams", parseInt(value))
	}
	if value := getPath(smux, "min-streams"); truthy(value) {
		multiplex.Set("min_streams", parseInt(value))
	}
	if truthy(getPath(smux, "padding")) {
		multiplex.Set("padding", true)
	}
	up := getPath(smux, "brutal-opts.up")
	down := getPath(smux, "brutal-opts.down")
	if truthy(up) || truthy(down) {
		brutal := obj("enabled", true)
		multiplex.Set("brutal", brutal)
		if truthy(up) {
			brutal.Set("up_mbps", parseInt(up))
		}
		if truthy(down) {
			brutal.Set("down_mbps", parseInt(down))
		}
	}
}

// destructure is `const { key = defaultValue } = o`: the default applies only
// to undefined.
func destructure(o *Object, key string, defaultValue any) any {
	value := o.Get(key)
	if value == nil {
		return defaultValue
	}
	return value
}

// objectKeys is Object.keys for the values the producers pass to it.
func objectKeys(value any) []string {
	switch v := value.(type) {
	case *Object:
		return v.Keys()
	case string:
		keys := make([]string, 0, len(v))
		for i := range []rune(v) {
			keys = append(keys, numberToString(float64(i)))
		}
		return keys
	case []any:
		keys := make([]string, len(v))
		for i := range v {
			keys[i] = numberToString(float64(i))
		}
		return keys
	case nil, nullType:
		throw("TypeError: Cannot convert undefined or null to object")
	}
	return nil
}

// property is value[key] for the values the producers index.
func property(value any, key string) any {
	switch v := value.(type) {
	case *Object:
		return v.Get(key)
	case string:
		runes := []rune(v)
		if index := jsNumber(key); isSafeInteger(index) && index >= 0 && int(index) < len(runes) {
			return string(runes[int(index)])
		}
	case []any:
		if index := jsNumber(key); isSafeInteger(index) && index >= 0 && int(index) < len(v) {
			return v[int(index)]
		}
	case nil, nullType:
		throw("TypeError: Cannot read properties of " + jsString(value))
	}
	return nil
}

// headerValues normalizes a header map into arrays of strings.
func headerValues(headers any) *Object {
	result := newObject()
	for _, key := range objectKeys(headers) {
		value := property(headers, key)
		if value == "" {
			continue
		}
		array, isArray := value.([]any)
		if !isArray {
			array = []any{jsString(value)}
		}
		if len(array) > 0 {
			result.Set(key, array)
		}
	}
	return result
}

func wsParser(proxy *Object, parsedProxy *Object) {
	transport := obj("type", "ws", "headers", newObject())
	if wsOpts := proxy.Get("ws-opts"); truthy(wsOpts) {
		opts, _ := wsOpts.(*Object)
		wsPath := destructure(opts, "path", "")
		wsHeaders := destructure(opts, "headers", newObject())
		maxEarlyData := opts.Get("max-early-data")
		transport.Set("early_data_header_name", opts.Get("early-data-header-name"))
		if truthy(maxEarlyData) {
			transport.Set("max_early_data", parseInt(maxEarlyData))
		} else {
			transport.Set("max_early_data", nil)
		}
		if wsPath != "" {
			transport.Set("path", jsString(wsPath))
		}
		if len(objectKeys(wsHeaders)) > 0 {
			headers := headerValues(wsHeaders)
			wsHost := headers.Get("Host")
			hostArray, isArray := wsHost.([]any)
			if !isArray {
				throw("TypeError: Cannot read properties of undefined (reading 'length')")
			}
			if len(hostArray) == 1 {
				for item := range strings.SplitSeq("Host:"+jsString(hostArray[0]), "\n") {
					key, value := splitPair(item, ":")
					valueString, isString := value.(string)
					if !isString {
						throw("TypeError: Cannot read properties of undefined (reading 'trim')")
					}
					if strings.TrimSpace(valueString) == "" {
						continue
					}
					headers.Set(strings.TrimSpace(key), stringsToArray(strings.Split(strings.TrimSpace(valueString), ",")))
				}
			}
			transport.Set("headers", headers)
		}
	}
	if path := transport.Get("path"); truthy(path) {
		ed, maxEarlyData := getSafeIntegerPathQueryParam(path, "ed")
		if ed != "" {
			extracted, _ := extractPathQueryParam(path, "ed")
			transport.Set("path", extracted)
			transport.Set("early_data_header_name", "Sec-WebSocket-Protocol")
			transport.Set("max_early_data", maxEarlyData)
		}
	}
	headers := transport.Obj("headers")
	if truthy(getPath(parsedProxy, "tls.insecure")) {
		parsedProxy.Obj("tls").Set("server_name", property(property(headers, "Host"), "0"))
	}
	if truthy(getPath(proxy, "ws-opts.v2ray-http-upgrade")) {
		transport.Set("type", "httpupgrade")
		if host := headers.Get("Host"); truthy(host) {
			transport.Set("host", property(host, "0"))
			headers.Delete("Host")
		}
		if truthy(transport.Get("max_early_data")) {
			transport.Delete("max_early_data")
		}
		if truthy(transport.Get("early_data_header_name")) {
			transport.Delete("early_data_header_name")
		}
	}
	collapseSingleValues(headers)
	parsedProxy.Set("transport", transport)
}

// collapseSingleValues replaces one-item arrays with the item.
func collapseSingleValues(headers *Object) {
	for _, key := range headers.Keys() {
		if array, isArray := headers.Get(key).([]any); isArray && len(array) == 1 {
			headers.Set(key, array[0])
		}
	}
}

// splitTrim is `${value}`.split(",").map((i) => i.trim()).
func splitTrim(value any) []any {
	return trimStrings(strings.Split(jsString(value), ","), false)
}

func h1Parser(proxy *Object, parsedProxy *Object) {
	transport := obj("type", "http", "headers", newObject())
	headers := transport.Obj("headers")
	if httpOpts := proxy.Get("http-opts"); truthy(httpOpts) {
		opts, _ := httpOpts.(*Object)
		method := destructure(opts, "method", "")
		h1Path := destructure(opts, "path", "")
		h1Headers := destructure(opts, "headers", newObject())
		if method != "" {
			transport.Set("method", method)
		}
		if array, isArray := h1Path.([]any); isArray {
			transport.Set("path", jsString(arrayAt(array, 0)))
		} else if h1Path != "" {
			transport.Set("path", jsString(h1Path))
		}
		for _, key := range objectKeys(h1Headers) {
			value := property(h1Headers, key)
			if value == "" {
				continue
			}
			if strings.ToLower(key) == "host" {
				host, isArray := value.([]any)
				if !isArray {
					host = splitTrim(value)
				}
				if len(host) > 0 {
					transport.Set("host", host)
				}
				continue
			}
			array, isArray := value.([]any)
			if !isArray {
				array = splitTrim(value)
			}
			if len(array) > 0 {
				headers.Set(key, array)
			}
		}
	}
	if host := proxy.Get("http-host"); truthy(host) && host != "" {
		hostArray, isArray := host.([]any)
		if !isArray {
			hostArray = splitTrim(host)
		}
		if len(hostArray) > 0 {
			transport.Set("host", hostArray)
		}
	}
	if path := proxy.Get("http-path"); truthy(path) && path != "" {
		if array, isArray := path.([]any); isArray {
			transport.Set("path", jsString(arrayAt(array, 0)))
		} else {
			transport.Set("path", jsString(path))
		}
	}
	if truthy(getPath(parsedProxy, "tls.insecure")) {
		parsedProxy.Obj("tls").Set("server_name", property(transport.Get("host"), "0"))
	}
	if host, isArray := transport.Get("host").([]any); isArray && len(host) == 1 {
		transport.Set("host", host[0])
	}
	collapseSingleValues(headers)
	parsedProxy.Set("transport", transport)
}

func h2Parser(proxy *Object, parsedProxy *Object) {
	transport := obj("type", "http")
	if h2Opts := proxy.Get("h2-opts"); truthy(h2Opts) {
		opts, _ := h2Opts.(*Object)
		host := destructure(opts, "host", "")
		path := destructure(opts, "path", "")
		if path != "" {
			transport.Set("path", jsString(path))
		}
		if host != "" {
			hostArray, isArray := host.([]any)
			if !isArray {
				hostArray = splitTrim(host)
			}
			if len(hostArray) > 0 {
				transport.Set("host", hostArray)
			}
		}
	}
	if host := proxy.Get("h2-host"); truthy(host) && host != "" {
		hostArray, isArray := host.([]any)
		if !isArray {
			hostArray = splitTrim(host)
		}
		if len(hostArray) > 0 {
			transport.Set("host", hostArray)
		}
	}
	if path := proxy.Get("h2-path"); truthy(path) && path != "" {
		transport.Set("path", jsString(path))
	}
	parsedProxy.Obj("tls").Set("enabled", true)
	if truthy(getPath(parsedProxy, "tls.insecure")) {
		parsedProxy.Obj("tls").Set("server_name", property(transport.Get("host"), "0"))
	}
	host := transport.Get("host")
	if isNullish(host) {
		throw("TypeError: Cannot read properties of undefined (reading 'length')")
	}
	if array, isArray := host.([]any); isArray && len(array) == 1 {
		transport.Set("host", array[0])
	}
	parsedProxy.Set("transport", transport)
}

func grpcParser(proxy *Object, parsedProxy *Object) {
	transport := obj("type", "grpc")
	if grpcOpts := proxy.Get("grpc-opts"); truthy(grpcOpts) {
		serviceName := property(grpcOpts, "grpc-service-name")
		if !isNullish(serviceName) && serviceName != "" {
			transport.Set("service_name", jsString(serviceName))
		}
	}
	parsedProxy.Set("transport", transport)
}

var (
	escapedCRLFRegex = regexp.MustCompile(`\\r\\n`)
	escapedLFRegex   = regexp.MustCompile(`\\n`)
	lineBreakRegex   = regexp.MustCompile(`\r?\n`)
	pemBeginRegex    = regexp.MustCompile(`^-----BEGIN [A-Za-z0-9 -]+-----$`)
	colonRegex       = regexp.MustCompile(`:`)
	sha256HexRegex   = regexp.MustCompile(`(?i)^[0-9a-f]{64}$`)
)

func normalizePemLines(value any, label string) any {
	items, isArray := value.([]any)
	if !isArray {
		items = []any{value}
	}
	var lines []any
	for _, item := range items {
		normalized := strings.TrimSpace(jsString(item))
		normalized = escapedCRLFRegex.ReplaceAllString(normalized, "\n")
		normalized = escapedLFRegex.ReplaceAllString(normalized, "\n")
		if normalized == "" {
			continue
		}
		for _, line := range lineBreakRegex.Split(normalized, -1) {
			if trimmed := strings.TrimSpace(line); trimmed != "" {
				lines = append(lines, trimmed)
			}
		}
	}
	if len(lines) == 0 {
		return nil
	}
	for _, line := range lines {
		if pemBeginRegex.MatchString(line.(string)) {
			return lines
		}
	}
	return append(append([]any{"-----BEGIN " + label + "-----"}, lines...), "-----END "+label+"-----")
}

var singBoxUtlsFingerprints = []string{"chrome", "firefox", "edge", "safari", "360", "qq", "ios", "android", "random", "randomized"}

func getSingBoxUtlsFingerprint(value any) any {
	fingerprint := strings.ToLower(strings.TrimSpace(jsString(or(value, ""))))
	if includesString(fingerprint, singBoxUtlsFingerprints...) {
		return fingerprint
	}
	return nil
}

func hasLength(value any) bool {
	switch v := value.(type) {
	case string:
		return v != ""
	case []any:
		return len(v) > 0
	}
	return false
}

func tlsCertificateParser(proxy *Object, parsedProxy *Object) {
	tls := parsedProxy.Obj("tls")
	if truthy(proxy.Get("ca")) {
		tls.Set("certificate_path", jsString(proxy.Get("ca")))
	}
	if truthy(proxy.Get("ca_str")) {
		tls.Set("certificate", []any{proxy.Get("ca_str")})
	}
	if truthy(proxy.Get("ca-str")) {
		tls.Set("certificate", []any{proxy.Get("ca-str")})
	}
	if truthy(proxy.Get("_certificate")) {
		tls.Set("certificate", proxy.Get("_certificate"))
	}
	if truthy(proxy.Get("_certificate_path")) {
		tls.Set("certificate_path", proxy.Get("_certificate_path"))
	}
	if truthy(getPath(tls, "reality.enabled")) {
		return
	}
	if truthy(proxy.Get("_certificate_public_key_sha256")) {
		tls.Set("certificate_public_key_sha256", proxy.Get("_certificate_public_key_sha256"))
	}
	hasCertificate := hasLength(tls.Get("certificate")) || truthy(tls.Get("certificate_path"))
	if !isNullish(proxy.Get("_certificate_sha256")) {
		tls.Set("certificate_sha256", proxy.Get("_certificate_sha256"))
	} else if (truthy(proxy.Get("fingerprint")) || truthy(proxy.Get("tls-fingerprint"))) && !hasCertificate &&
		!hasLength(tls.Get("certificate_public_key_sha256")) {
		hexString := colonRegex.ReplaceAllString(strings.TrimSpace(jsString(or(proxy.Get("fingerprint"), proxy.Get("tls-fingerprint")))), "")
		if !sha256HexRegex.MatchString(hexString) {
			throw("Platform sing-box: invalid SHA-256 certificate fingerprint for proxy " + jsString(proxy.Get("name")))
		}
		content, _ := hex.DecodeString(hexString)
		tls.Set("certificate_sha256", []any{base64.StdEncoding.EncodeToString(content)})
	}
	if hasCertificate && (hasLength(tls.Get("certificate_sha256")) || hasLength(tls.Get("certificate_public_key_sha256"))) {
		throw("Platform sing-box: certificate_sha256 or certificate_public_key_sha256 conflicts with certificate or certificate_path for proxy " + jsString(proxy.Get("name")))
	}
}

func tlsParser(proxy *Object, parsedProxy *Object) {
	tls := parsedProxy.Obj("tls")
	if truthy(proxy.Get("tls")) {
		tls.Set("enabled", true)
	}
	if truthy(proxy.Get("servername")) {
		tls.Set("server_name", proxy.Get("servername"))
	}
	if truthy(proxy.Get("peer")) {
		tls.Set("server_name", proxy.Get("peer"))
	}
	if truthy(proxy.Get("sni")) {
		tls.Set("server_name", proxy.Get("sni"))
	}
	if truthy(proxy.Get("skip-cert-verify")) || truthy(proxy.Get("insecure")) {
		tls.Set("insecure", true)
	}
	if truthy(proxy.Get("disable-sni")) {
		tls.Set("disable_sni", true)
	}
	switch alpn := proxy.Get("alpn").(type) {
	case string:
		tls.Set("alpn", []any{alpn})
	case []any:
		tls.Set("alpn", alpn)
	}
	if realityOpts := proxy.Get("reality-opts"); truthy(realityOpts) {
		reality := obj("enabled", true)
		if publicKey := property(realityOpts, "public-key"); truthy(publicKey) {
			reality.Set("public_key", publicKey)
		}
		if shortID := property(realityOpts, "short-id"); truthy(shortID) {
			reality.Set("short_id", shortID)
		}
		tls.Set("reality", reality)
		tls.Set("utls", obj("enabled", true))
	}
	if !includesString(proxy.Get("type"), "hysteria", "hysteria2", "tuic") && truthy(proxy.Get("client-fingerprint")) {
		if fingerprint := getSingBoxUtlsFingerprint(proxy.Get("client-fingerprint")); truthy(fingerprint) {
			utls := newObject()
			if existing, isObject := tls.Get("utls").(*Object); isObject {
				utls = existing.Clone()
			}
			utls.Set("enabled", true)
			utls.Set("fingerprint", fingerprint)
			tls.Set("utls", utls)
		}
	}
	if ech, isObject := proxy.Get("_ech").(*Object); isObject {
		tls.Set("ech", ech)
	} else if echOpts, isObject := proxy.Get("ech-opts").(*Object); isObject {
		echConfig, isObject := tls.Get("ech").(*Object)
		if !truthy(tls.Get("ech")) {
			echConfig = newObject()
			tls.Set("ech", echConfig)
		} else if !isObject {
			throw("TypeError: invalid tls.ech")
		}
		echConfig.Set("enabled", echOpts.Get("enable"))
		switch config := echOpts.Get("config").(type) {
		case string, []any:
			if normalized := normalizePemLines(config, "ECH CONFIGS"); normalized != nil {
				echConfig.Set("config", normalized)
			}
		}
		echConfig.Set("query_server_name", echOpts.Get("query-server-name"))
		echConfig.Set("config_path", echOpts.Get("config-path"))
		echConfig.Set("fragment", echOpts.Get("fragment"))
		echConfig.Set("fragment_fallback_delay", echOpts.Get("fragment-fallback-delay"))
		echConfig.Set("record_fragment", echOpts.Get("record-fragment"))
	}
	if curvePreferences, isArray := proxy.Get("_curve_preferences").([]any); isArray {
		tls.Set("curve_preferences", curvePreferences)
	}
	if truthy(proxy.Get("_fragment")) {
		tls.Set("fragment", true)
	}
	if truthy(proxy.Get("_fragment_fallback_delay")) {
		tls.Set("fragment_fallback_delay", proxy.Get("_fragment_fallback_delay"))
	}
	if truthy(proxy.Get("_record_fragment")) {
		tls.Set("record_fragment", true)
	}
	for _, field := range []string{"client_certificate", "client_certificate_path", "client_key", "client_key_path"} {
		if value := proxy.Get("_" + field); truthy(value) {
			tls.Set(field, value)
		}
	}
	if !truthy(tls.Get("enabled")) {
		parsedProxy.Delete("tls")
	} else {
		tlsCertificateParser(proxy, parsedProxy)
	}
}

func checkPort(parsedProxy *Object) {
	port, _ := parsedProxy.Get("server_port").(float64)
	if port < 0 || port > 65535 {
		throw("invalid port")
	}
}

func newTLS(proxy *Object, enabled bool) *Object {
	return obj("enabled", enabled, "server_name", proxy.Get("server"), "insecure", false)
}

func base(proxy *Object, outboundType string) *Object {
	return obj(
		"tag", proxy.Get("name"),
		"type", outboundType,
		"server", proxy.Get("server"),
		"server_port", parseInt(proxy.Get("port")),
	)
}

func commonDialParsers(proxy *Object, parsedProxy *Object) {
	ipVersionParser(proxy, parsedProxy)
	domainResolverParser(proxy, parsedProxy)
}

func udpFragmentParser(proxy *Object, parsedProxy *Object) {
	if truthy(proxy.Get("fast-open")) {
		parsedProxy.Set("udp_fragment", true)
	}
}

func udpOverTCPParser(proxy *Object, parsedProxy *Object) {
	if truthy(proxy.Get("uot")) {
		parsedProxy.Set("udp_over_tcp", true)
	}
	if truthy(proxy.Get("udp-over-tcp")) {
		version := float64(2)
		if value := proxy.Get("udp-over-tcp-version"); !truthy(value) || value == float64(1) {
			version = 1
		}
		parsedProxy.Set("udp_over_tcp", obj("enabled", true, "version", version))
	}
}

func sshParser(proxy *Object) *Object {
	parsedProxy := base(proxy, "ssh")
	checkPort(parsedProxy)
	if truthy(proxy.Get("username")) {
		parsedProxy.Set("user", proxy.Get("username"))
	}
	if truthy(proxy.Get("password")) {
		parsedProxy.Set("password", proxy.Get("password"))
	}
	if privateKey := or(proxy.Get("private-key"), proxy.Get("privateKey")); truthy(privateKey) {
		key := "private_key_path"
		if strings.Contains(jsString(privateKey), "PRIVATE KEY") {
			key = "private_key"
		}
		parsedProxy.Set(key, privateKey)
	}
	if truthy(proxy.Get("private-key-passphrase")) {
		parsedProxy.Set("private_key_passphrase", proxy.Get("private-key-passphrase"))
	}
	if fingerprint := proxy.Get("server-fingerprint"); truthy(fingerprint) {
		parsedProxy.Set("host_key", []any{fingerprint})
		fingerprintString, isString := fingerprint.(string)
		if !isString {
			throw("TypeError: server-fingerprint.split is not a function")
		}
		parsedProxy.Set("host_key_algorithms", []any{strings.Split(fingerprintString, " ")[0]})
	}
	if truthy(proxy.Get("host-key")) {
		parsedProxy.Set("host_key", proxy.Get("host-key"))
	}
	if truthy(proxy.Get("host-key-algorithms")) {
		parsedProxy.Set("host_key_algorithms", proxy.Get("host-key-algorithms"))
	}
	udpFragmentParser(proxy, parsedProxy)
	tfoParser(proxy, parsedProxy)
	detourParser(proxy, parsedProxy)
	commonDialParsers(proxy, parsedProxy)
	return parsedProxy
}

func httpParser(proxy *Object) *Object {
	parsedProxy := base(proxy, "http")
	parsedProxy.Set("tls", newTLS(proxy, false))
	checkPort(parsedProxy)
	if truthy(proxy.Get("username")) {
		parsedProxy.Set("username", proxy.Get("username"))
	}
	if truthy(proxy.Get("password")) {
		parsedProxy.Set("password", proxy.Get("password"))
	}
	if headers := proxy.Get("headers"); truthy(headers) {
		result := newObject()
		for _, key := range objectKeys(headers) {
			result.Set(key, jsString(property(headers, key)))
		}
		if result.Len() > 0 {
			parsedProxy.Set("headers", result)
		}
	}
	udpFragmentParser(proxy, parsedProxy)
	tfoParser(proxy, parsedProxy)
	detourParser(proxy, parsedProxy)
	tlsParser(proxy, parsedProxy)
	commonDialParsers(proxy, parsedProxy)
	return parsedProxy
}

func socks5Parser(proxy *Object) *Object {
	parsedProxy := base(proxy, "socks")
	parsedProxy.Set("version", "5")
	checkPort(parsedProxy)
	if truthy(proxy.Get("username")) {
		parsedProxy.Set("username", proxy.Get("username"))
	}
	if truthy(proxy.Get("password")) {
		parsedProxy.Set("password", proxy.Get("password"))
	}
	udpOverTCPParser(proxy, parsedProxy)
	udpFragmentParser(proxy, parsedProxy)
	networkParser(proxy, parsedProxy)
	tfoParser(proxy, parsedProxy)
	detourParser(proxy, parsedProxy)
	commonDialParsers(proxy, parsedProxy)
	return parsedProxy
}

func getShadowTLSTag(proxy *Object) string {
	return jsString(proxy.Get("name")) + "_shadowtls"
}

func getShadowTLSPluginOpts(proxy *Object) any {
	if proxy.Get("plugin") == "shadow-tls" && truthy(proxy.Get("plugin-opts")) {
		return proxy.Get("plugin-opts")
	}
	if proxy.Get("type") == "snell" && getPath(proxy.Get("obfs-opts"), "mode") == "shadow-tls" {
		obfsOpts := proxy.Get("obfs-opts")
		return obj(
			"host", getPath(obfsOpts, "host"),
			"password", getPath(obfsOpts, "password"),
			"version", getPath(obfsOpts, "version"),
			"alpn", getPath(obfsOpts, "alpn"),
		)
	}
	return nil
}

func normalizeALPN(alpn any) any {
	switch a := alpn.(type) {
	case string:
		return trimStrings(strings.Split(a, ","), true)
	case []any:
		return a
	}
	return nil
}

func shadowTLSParser(proxy *Object) (*Object, *Object) {
	pluginOpts := getShadowTLSPluginOpts(proxy)
	ssPart := obj(
		"tag", proxy.Get("name"),
		"type", "shadowsocks",
		"method", proxy.Get("cipher"),
		"password", proxy.Get("password"),
		"detour", getShadowTLSTag(proxy),
	)
	udpOverTCPParser(proxy, ssPart)
	networkParser(proxy, ssPart)
	smuxParser(proxy.Get("smux"), ssPart)
	return ssPart, shadowTLSOutboundParser(proxy, pluginOpts)
}

func shadowTLSOutboundParser(proxy *Object, pluginOpts any) *Object {
	if !truthy(pluginOpts) {
		throw("shadow-tls plugin options are missing")
	}
	fingerprint := getSingBoxUtlsFingerprint(proxy.Get("client-fingerprint"))
	stPart := obj(
		"tag", getShadowTLSTag(proxy),
		"type", "shadowtls",
		"server", proxy.Get("server"),
		"server_port", parseInt(proxy.Get("port")),
		"version", property(pluginOpts, "version"),
		"password", property(pluginOpts, "password"),
		"tls", obj(
			"enabled", true,
			"server_name", property(pluginOpts, "host"),
		),
	)
	tls := stPart.Obj("tls")
	if truthy(proxy.Get("skip-cert-verify")) {
		tls.Set("insecure", true)
	}
	if truthy(fingerprint) {
		tls.Set("utls", obj("enabled", true, "fingerprint", fingerprint))
	}
	checkPort(stPart)
	alpn := coalesce(normalizeALPN(property(pluginOpts, "alpn")), normalizeALPN(proxy.Get("alpn")))
	if truthy(alpn) {
		tls.Set("alpn", alpn)
	}
	tlsCertificateParser(proxy, stPart)
	if proxy.Get("fast-open") == true {
		stPart.Set("udp_fragment", true)
	}
	tfoParser(proxy, stPart)
	detourParser(proxy, stPart)
	commonDialParsers(proxy, stPart)
	return stPart
}

func ssParser(proxy *Object) *Object {
	parsedProxy := base(proxy, "shadowsocks")
	parsedProxy.Set("method", proxy.Get("cipher"))
	parsedProxy.Set("password", proxy.Get("password"))
	checkPort(parsedProxy)
	udpOverTCPParser(proxy, parsedProxy)
	udpFragmentParser(proxy, parsedProxy)
	networkParser(proxy, parsedProxy)
	tfoParser(proxy, parsedProxy)
	detourParser(proxy, parsedProxy)
	smuxParser(proxy.Get("smux"), parsedProxy)
	commonDialParsers(proxy, parsedProxy)
	if truthy(proxy.Get("plugin")) {
		var optArr []string
		if proxy.Get("plugin") == "obfs" {
			parsedProxy.Set("plugin", "obfs-local")
			parsedProxy.Set("plugin_opts", "")
			if truthy(proxy.Get("obfs-host")) {
				mustObject(proxy.Get("plugin-opts")).Set("host", proxy.Get("obfs-host"))
			}
			pluginOpts := proxy.Get("plugin-opts")
			for _, key := range objectKeys(pluginOpts) {
				switch key {
				case "mode":
					optArr = append(optArr, "obfs="+jsString(property(pluginOpts, "mode")))
				case "host":
					optArr = append(optArr, "obfs-host="+jsString(property(pluginOpts, "host")))
				default:
					optArr = append(optArr, key+"="+jsString(property(pluginOpts, key)))
				}
			}
		}
		if proxy.Get("plugin") == "v2ray-plugin" {
			parsedProxy.Set("plugin", "v2ray-plugin")
			if truthy(proxy.Get("ws-host")) {
				mustObject(proxy.Get("plugin-opts")).Set("host", proxy.Get("ws-host"))
			}
			if truthy(proxy.Get("ws-path")) {
				mustObject(proxy.Get("plugin-opts")).Set("path", proxy.Get("ws-path"))
			}
			pluginOpts := proxy.Get("plugin-opts")
			for _, key := range objectKeys(pluginOpts) {
				switch key {
				case "tls":
					if truthy(property(pluginOpts, "tls")) {
						optArr = append(optArr, "tls")
					}
				case "host":
					optArr = append(optArr, "host="+jsString(property(pluginOpts, "host")))
				case "path":
					optArr = append(optArr, "path="+jsString(property(pluginOpts, "path")))
				case "headers":
					headers := property(pluginOpts, "headers")
					if headers == nil {
						optArr = append(optArr, "headers=undefined")
					} else {
						content, err := marshalValue(headers)
						if err != nil {
							panic(err)
						}
						optArr = append(optArr, "headers="+string(content))
					}
				case "mux":
					mux := normalizePluginMuxValue(property(pluginOpts, "mux"))
					if truthy(mux) {
						parsedProxy.Set("multiplex", obj("enabled", true))
					}
					optArr = append(optArr, "mux="+jsString(mux))
				default:
					optArr = append(optArr, key+"="+jsString(property(pluginOpts, key)))
				}
			}
		}
		parsedProxy.Set("plugin_opts", strings.Join(optArr, ";"))
	}
	return parsedProxy
}

func getSnellVersion(version any) any {
	if isNullish(version) {
		return nil
	}
	normalized := strings.TrimSpace(jsString(version))
	if !digitsOnlyRegex.MatchString(normalized) {
		return math.NaN()
	}
	return parseInt(normalized)
}

func snellParser(proxy *Object) *Object {
	version := getSnellVersion(proxy.Get("version"))
	shadowTLSPluginOpts := getShadowTLSPluginOpts(proxy)
	if version != nil && !includes(version, float64(4), float64(5), float64(6)) {
		throw("Platform sing-box does not support snell version " + jsString(proxy.Get("version")))
	}
	outputVersion := version
	if version == float64(5) {
		outputVersion = float64(4)
	}
	parsedProxy := base(proxy, "snell")
	parsedProxy.Set("psk", proxy.Get("psk"))
	checkPort(parsedProxy)
	if outputVersion != nil {
		parsedProxy.Set("version", outputVersion)
	}
	if truthy(proxy.Get("_userkey")) {
		parsedProxy.Set("userkey", proxy.Get("_userkey"))
	}
	if outputVersion == float64(6) {
		if truthy(proxy.Get("mode")) {
			parsedProxy.Set("mode", proxy.Get("mode"))
		}
	} else {
		obfsOpts := proxy.Get("obfs-opts")
		if mode := getPath(obfsOpts, "mode"); truthy(mode) && mode != "shadow-tls" {
			parsedProxy.Set("obfs_mode", mode)
		}
		if host := getPath(obfsOpts, "host"); truthy(host) && getPath(obfsOpts, "mode") != "shadow-tls" {
			parsedProxy.Set("obfs_host", host)
		}
	}
	if truthy(proxy.Get("reuse")) && (version == nil || !jsLess(version, 4)) {
		parsedProxy.Set("reuse", true)
	}
	networkParser(proxy, parsedProxy)
	if truthy(shadowTLSPluginOpts) {
		parsedProxy.Set("detour", getShadowTLSTag(proxy))
		parsedProxy.Delete("server")
		parsedProxy.Delete("server_port")
	} else {
		udpFragmentParser(proxy, parsedProxy)
		tfoParser(proxy, parsedProxy)
		detourParser(proxy, parsedProxy)
		commonDialParsers(proxy, parsedProxy)
	}
	return parsedProxy
}

func vmessVlessPacketEncodingParser(proxy *Object, parsedProxy *Object) {
	if packetEncoding := proxy.Get("packet-encoding"); !isNullish(packetEncoding) {
		normalized := strings.ToLower(strings.TrimSpace(jsString(packetEncoding)))
		if includesString(normalized, "", "packetaddr", "xudp") {
			parsedProxy.Set("packet_encoding", normalized)
		}
	} else if truthy(proxy.Get("xudp")) {
		parsedProxy.Set("packet_encoding", "xudp")
	} else if truthy(proxy.Get("packet-addr")) {
		parsedProxy.Set("packet_encoding", "packetaddr")
	}
}

func vmessParser(proxy *Object) *Object {
	parsedProxy := base(proxy, "vmess")
	parsedProxy.Set("uuid", proxy.Get("uuid"))
	parsedProxy.Set("security", normalizeVmessSecurity(proxy.Get("cipher")))
	parsedProxy.Set("alter_id", parseInt(proxy.Get("alterId")))
	parsedProxy.Set("tls", newTLS(proxy, false))
	checkPort(parsedProxy)
	vmessVlessPacketEncodingParser(proxy, parsedProxy)
	if globalPadding := proxy.Get("global-padding"); !isNullish(globalPadding) {
		parsedProxy.Set("global_padding", truthy(globalPadding))
	}
	if authenticatedLength := proxy.Get("authenticated-length"); !isNullish(authenticatedLength) {
		parsedProxy.Set("authenticated_length", truthy(authenticatedLength))
	}
	udpFragmentParser(proxy, parsedProxy)
	transportParser(proxy, parsedProxy)
	networkParser(proxy, parsedProxy)
	tfoParser(proxy, parsedProxy)
	detourParser(proxy, parsedProxy)
	tlsParser(proxy, parsedProxy)
	smuxParser(proxy.Get("smux"), parsedProxy)
	commonDialParsers(proxy, parsedProxy)
	return parsedProxy
}

func transportParser(proxy *Object, parsedProxy *Object) {
	switch proxy.Get("network") {
	case "ws":
		wsParser(proxy, parsedProxy)
	case "h2":
		h2Parser(proxy, parsedProxy)
	case "http":
		h1Parser(proxy, parsedProxy)
	case "grpc":
		grpcParser(proxy, parsedProxy)
	}
}

func vlessParser(proxy *Object) *Object {
	parsedProxy := base(proxy, "vless")
	parsedProxy.Set("uuid", proxy.Get("uuid"))
	parsedProxy.Set("tls", newTLS(proxy, false))
	checkPort(parsedProxy)
	vmessVlessPacketEncodingParser(proxy, parsedProxy)
	udpFragmentParser(proxy, parsedProxy)
	if !isNullish(proxy.Get("flow")) {
		parsedProxy.Set("flow", proxy.Get("flow"))
	}
	transportParser(proxy, parsedProxy)
	networkParser(proxy, parsedProxy)
	tfoParser(proxy, parsedProxy)
	detourParser(proxy, parsedProxy)
	smuxParser(proxy.Get("smux"), parsedProxy)
	tlsParser(proxy, parsedProxy)
	commonDialParsers(proxy, parsedProxy)
	return parsedProxy
}

func trojanParser(proxy *Object) *Object {
	parsedProxy := base(proxy, "trojan")
	parsedProxy.Set("password", proxy.Get("password"))
	parsedProxy.Set("tls", newTLS(proxy, true))
	checkPort(parsedProxy)
	udpFragmentParser(proxy, parsedProxy)
	switch proxy.Get("network") {
	case "grpc":
		grpcParser(proxy, parsedProxy)
	case "ws":
		wsParser(proxy, parsedProxy)
	}
	networkParser(proxy, parsedProxy)
	tfoParser(proxy, parsedProxy)
	detourParser(proxy, parsedProxy)
	tlsParser(proxy, parsedProxy)
	smuxParser(proxy.Get("smux"), parsedProxy)
	commonDialParsers(proxy, parsedProxy)
	return parsedProxy
}

func naiveParser(proxy *Object) *Object {
	parsedProxy := base(proxy, "naive")
	parsedProxy.Set("tls", newTLS(proxy, true))
	checkPort(parsedProxy)
	if truthy(proxy.Get("username")) {
		parsedProxy.Set("username", proxy.Get("username"))
	}
	if truthy(proxy.Get("password")) {
		parsedProxy.Set("password", proxy.Get("password"))
	}
	udpOverTCPParser(proxy, parsedProxy)
	if concurrency := parseInt(proxy.Get("insecure-concurrency")); isSafeInteger(concurrency) && concurrency >= 0 {
		parsedProxy.Set("insecure_concurrency", concurrency)
	}
	if truthy(proxy.Get("extra-headers")) {
		parsedProxy.Set("extra_headers", proxy.Get("extra-headers"))
	}
	if truthy(proxy.Get("quic")) {
		parsedProxy.Set("quic", true)
	}
	if truthy(proxy.Get("quic-congestion-control")) {
		parsedProxy.Set("quic_congestion_control", proxy.Get("quic-congestion-control"))
	}
	udpFragmentParser(proxy, parsedProxy)
	tfoParser(proxy, parsedProxy)
	detourParser(proxy, parsedProxy)
	tlsParser(proxy, parsedProxy)
	smuxParser(proxy.Get("smux"), parsedProxy)
	commonDialParsers(proxy, parsedProxy)
	if tls := parsedProxy.Obj("tls"); truthy(tls.Get("insecure")) {
		tls.Delete("insecure")
	}
	return parsedProxy
}

var (
	bandwidthRegex     = regexp.MustCompile(`^[0-9]+[ \t]*[KMGT]*[Bb]ps$`)
	portsSeparator     = regexp.MustCompile(`\s*,\s*`)
	portsRangeSep      = regexp.MustCompile(`\s*-\s*`)
	hopIntervalDigitRe = regexp.MustCompile(`^\d+$`)
)

func hopIntervalParser(proxy *Object, parsedProxy *Object) {
	if hopInterval := proxy.Get("hop-interval"); truthy(hopInterval) {
		if testString(hopIntervalDigitRe, hopInterval) {
			parsedProxy.Set("hop_interval", jsString(hopInterval)+"s")
		} else {
			parsedProxy.Set("hop_interval", hopInterval)
		}
	}
}

func serverPortsParser(proxy *Object, parsedProxy *Object) {
	ports := proxy.Get("ports")
	if !truthy(ports) {
		return
	}
	portsString, isString := ports.(string)
	if !isString {
		throw("TypeError: ports.split is not a function")
	}
	var serverPorts []any
	for _, port := range portsSeparator.Split(portsString, -1) {
		portRange := portsRangeSep.ReplaceAllString(port, ":")
		if strings.Contains(portRange, ":") {
			serverPorts = append(serverPorts, portRange)
		} else {
			serverPorts = append(serverPorts, portRange+":"+portRange)
		}
	}
	parsedProxy.Set("server_ports", serverPorts)
}

func hysteriaParser(proxy *Object) *Object {
	parsedProxy := base(proxy, "hysteria")
	parsedProxy.Set("disable_mtu_discovery", false)
	parsedProxy.Set("tls", newTLS(proxy, true))
	checkPort(parsedProxy)
	hopIntervalParser(proxy, parsedProxy)
	serverPortsParser(proxy, parsedProxy)
	if truthy(proxy.Get("auth_str")) {
		parsedProxy.Set("auth_str", jsString(proxy.Get("auth_str")))
	}
	if truthy(proxy.Get("auth-str")) {
		parsedProxy.Set("auth_str", jsString(proxy.Get("auth-str")))
	}
	udpFragmentParser(proxy, parsedProxy)
	if up := jsString(proxy.Get("up")); bandwidthRegex.MatchString(up) && !strings.HasSuffix(up, "Mbps") {
		parsedProxy.Set("up", up)
	} else {
		parsedProxy.Set("up_mbps", parseInt(up))
	}
	if down := jsString(proxy.Get("down")); bandwidthRegex.MatchString(down) && !strings.HasSuffix(down, "Mbps") {
		parsedProxy.Set("down", down)
	} else {
		parsedProxy.Set("down_mbps", parseInt(down))
	}
	if truthy(proxy.Get("obfs")) {
		parsedProxy.Set("obfs", proxy.Get("obfs"))
	}
	for _, key := range []string{"recv_window_conn", "recv-window-conn", "recv_window", "recv-window"} {
		if value := proxy.Get(key); truthy(value) {
			parsedProxy.Set(strings.ReplaceAll(key, "-", "_"), value)
		}
	}
	if disable := proxy.Get("disable_mtu_discovery"); truthy(disable) {
		if b, isBool := disable.(bool); isBool {
			parsedProxy.Set("disable_mtu_discovery", b)
		} else if disable == float64(1) {
			parsedProxy.Set("disable_mtu_discovery", true)
		}
	}
	networkParser(proxy, parsedProxy)
	tlsParser(proxy, parsedProxy)
	detourParser(proxy, parsedProxy)
	tfoParser(proxy, parsedProxy)
	smuxParser(proxy.Get("smux"), parsedProxy)
	commonDialParsers(proxy, parsedProxy)
	return parsedProxy
}

func hysteria2Parser(proxy *Object) *Object {
	parsedProxy := base(proxy, "hysteria2")
	parsedProxy.Set("password", proxy.Get("password"))
	obfs := newObject()
	parsedProxy.Set("obfs", obfs)
	parsedProxy.Set("tls", newTLS(proxy, true))
	checkPort(parsedProxy)
	hopIntervalParser(proxy, parsedProxy)
	serverPortsParser(proxy, parsedProxy)
	if truthy(proxy.Get("up")) {
		parsedProxy.Set("up_mbps", parseInt(proxy.Get("up")))
	}
	if truthy(proxy.Get("down")) {
		parsedProxy.Set("down_mbps", parseInt(proxy.Get("down")))
	}
	if includesString(proxy.Get("obfs"), "salamander", "gecko") {
		obfs.Set("type", proxy.Get("obfs"))
	}
	if proxy.Get("obfs") == "gecko" {
		minRaw := proxy.Get("obfs-min-packet-size")
		maxRaw := proxy.Get("obfs-max-packet-size")
		hasMin := !isNullish(minRaw) && jsString(minRaw) != ""
		hasMax := !isNullish(maxRaw) && jsString(maxRaw) != ""
		if hasMin || hasMax {
			var minPacketSize, rawMaxPacketSize, maxPacketSize any
			if hasMin {
				minPacketSize = parseSafeIntegerValue(strings.TrimSpace(jsString(minRaw)))
			}
			if hasMax {
				rawMaxPacketSize = parseSafeIntegerValue(strings.TrimSpace(jsString(maxRaw)))
			}
			maxPacketSize = rawMaxPacketSize
			if rawMaxPacketSize != nil {
				maxPacketSize = math.Min(rawMaxPacketSize.(float64), 2048)
			}
			effectiveMin := coalesce(minPacketSize, float64(512)).(float64)
			effectiveMax := coalesce(maxPacketSize, float64(1200)).(float64)
			if (hasMin && (minPacketSize == nil || minPacketSize.(float64) <= 0)) ||
				(hasMax && (rawMaxPacketSize == nil || rawMaxPacketSize.(float64) <= 0)) ||
				effectiveMax < effectiveMin {
				// Sub-Store logs the invalid packet size and omits it.
			} else {
				if hasMin {
					obfs.Set("min_packet_size", minPacketSize)
				}
				if hasMax {
					obfs.Set("max_packet_size", maxPacketSize)
				}
			}
		}
	}
	if truthy(proxy.Get("obfs-password")) {
		obfs.Set("password", proxy.Get("obfs-password"))
	}
	if !truthy(obfs.Get("type")) {
		parsedProxy.Delete("obfs")
	}
	if truthy(proxy.Get("bbr-profile")) {
		parsedProxy.Set("bbr_profile", proxy.Get("bbr-profile"))
	}
	if truthy(proxy.Get("disable-chrome-parrot")) {
		parsedProxy.Set("disable_chrome_parrot", true)
	}
	networkParser(proxy, parsedProxy)
	tlsParser(proxy, parsedProxy)
	tfoParser(proxy, parsedProxy)
	detourParser(proxy, parsedProxy)
	smuxParser(proxy.Get("smux"), parsedProxy)
	commonDialParsers(proxy, parsedProxy)
	return parsedProxy
}

func tuic5Parser(proxy *Object) *Object {
	parsedProxy := base(proxy, "tuic")
	parsedProxy.Set("uuid", proxy.Get("uuid"))
	parsedProxy.Set("password", proxy.Get("password"))
	parsedProxy.Set("tls", newTLS(proxy, true))
	checkPort(parsedProxy)
	udpFragmentParser(proxy, parsedProxy)
	if congestion := proxy.Get("congestion-controller"); truthy(congestion) && congestion != "cubic" {
		parsedProxy.Set("congestion_control", congestion)
	}
	if relayMode := proxy.Get("udp-relay-mode"); truthy(relayMode) && relayMode != "native" {
		parsedProxy.Set("udp_relay_mode", relayMode)
	}
	if truthy(proxy.Get("reduce-rtt")) {
		parsedProxy.Set("zero_rtt_handshake", true)
	}
	if truthy(proxy.Get("udp-over-stream")) {
		parsedProxy.Set("udp_over_stream", true)
	}
	if heartbeat := proxy.Get("heartbeat-interval"); truthy(heartbeat) {
		parsedProxy.Set("heartbeat", jsString(heartbeat)+"ms")
	}
	networkParser(proxy, parsedProxy)
	tfoParser(proxy, parsedProxy)
	detourParser(proxy, parsedProxy)
	tlsParser(proxy, parsedProxy)
	smuxParser(proxy.Get("smux"), parsedProxy)
	commonDialParsers(proxy, parsedProxy)
	return parsedProxy
}

func anytlsParser(proxy *Object) *Object {
	parsedProxy := base(proxy, "anytls")
	parsedProxy.Set("password", proxy.Get("password"))
	parsedProxy.Set("tls", newTLS(proxy, true))
	if metadata := proxy.Get("client-metadata"); truthy(metadata) {
		parsedProxy.Set("client_metadata", jsString(metadata))
	}
	if value := proxy.Get("idle-session-check-interval"); testString(digitsOnlyRegex, value) {
		parsedProxy.Set("idle_session_check_interval", jsString(value)+"s")
	}
	if value := proxy.Get("idle-session-timeout"); testString(digitsOnlyRegex, value) {
		parsedProxy.Set("idle_session_timeout", jsString(value)+"s")
	}
	if value := proxy.Get("min-idle-session"); testString(digitsOnlyRegex, value) {
		parsedProxy.Set("min_idle_session", parseInt(value))
	}
	if disableReuse := proxy.Get("disable-reuse"); !isNullish(disableReuse) {
		parsedProxy.Set("disable_reuse", truthy(disableReuse))
	}
	detourParser(proxy, parsedProxy)
	tlsParser(proxy, parsedProxy)
	commonDialParsers(proxy, parsedProxy)
	return parsedProxy
}

type shadowTLSBackup struct {
	plugin     any
	pluginOpts any
	obfsOpts   any
}

// produceSingBox is singbox_Producer().produce(proxies), returning the
// outbounds.
func produceSingBox(proxies []*Object) []*Object {
	originalShadowTLS := make(map[*Object]shadowTLSBackup)
	for _, proxy := range proxies {
		if proxy.Get("plugin") == "shadow-tls" && truthy(proxy.Get("plugin-opts")) {
			originalShadowTLS[proxy] = shadowTLSBackup{
				plugin:     proxy.Get("plugin"),
				pluginOpts: cloneJSON(proxy.Get("plugin-opts")),
				obfsOpts:   cloneJSONOrUndefined(proxy.Get("obfs-opts")),
			}
		}
	}
	for _, proxy := range proxies {
		produceClashMetaInternal(proxy)
	}
	var list []*Object
	for _, proxy := range proxies {
		outbounds, err := produceSingBoxProxy(proxy, originalShadowTLS)
		if err == nil {
			list = append(list, outbounds...)
		}
	}
	return list
}

func cloneJSONOrUndefined(value any) any {
	if !truthy(value) {
		return nil
	}
	return cloneJSON(value)
}

func produceSingBoxProxy(proxy *Object, originalShadowTLS map[*Object]shadowTLSBackup) (list []*Object, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			list = nil
			err = recoveredError(recovered)
		}
	}()
	if shadowTLS, loaded := originalShadowTLS[proxy]; loaded {
		proxy.Set("plugin", shadowTLS.plugin)
		proxy.Set("plugin-opts", shadowTLS.pluginOpts)
		if truthy(shadowTLS.obfsOpts) {
			proxy.Set("obfs-opts", shadowTLS.obfsOpts)
		} else {
			proxy.Delete("obfs-opts")
		}
	}
	shadowTLSPluginOpts := getShadowTLSPluginOpts(proxy)
	shadowTLSVersion := getPath(shadowTLSPluginOpts, "version")
	shadowTLSEnabled := truthy(shadowTLSPluginOpts) && (truthy(getPath(shadowTLSPluginOpts, "password")) ||
		(!isNullish(shadowTLSVersion) && jsNumber(shadowTLSVersion) != 0))
	var streamShadowTLSOutbound *Object
	if shadowTLSEnabled && includesString(proxy.Get("type"), "vmess", "vless", "trojan") {
		if truthy(proxy.Get("reality-opts")) {
			throw("Platform sing-box cannot chain ShadowTLS with Reality")
		}
		if includesString(proxy.Get("type"), "vmess", "vless") && proxy.Get("network") == "h2" {
			throw("Platform sing-box cannot chain ShadowTLS with network h2")
		}
		if proxy.Get("type") == "vless" && proxy.Get("flow") == "xtls-rprx-vision" {
			throw("Platform sing-box cannot chain ShadowTLS with flow xtls-rprx-vision")
		}
		var parsedVersion float64
		if s, isString := shadowTLSVersion.(string); isString && strings.TrimSpace(s) == "" {
			parsedVersion = math.NaN()
		} else {
			parsedVersion = jsNumber(coalesce(shadowTLSVersion, float64(0)))
		}
		version := parsedVersion
		if parsedVersion == 0 {
			version = 2
		}
		if !includes(version, float64(1), float64(2), float64(3)) {
			throw("Platform sing-box does not support shadow-tls version " + jsString(shadowTLSVersion))
		}
		pluginOpts := mustObject(shadowTLSPluginOpts).Clone()
		pluginOpts.Set("version", version)
		streamShadowTLSOutbound = shadowTLSOutboundParser(proxy, pluginOpts)
	}
	if proxy.Get("type") == "anytls" && shadowTLSEnabled {
		throw("Platform sing-box cannot replace AnyTLS TLS with ShadowTLS")
	}
	if proxy.Get("network") == "xhttp" {
		throw("Platform sing-box does not support network: xhttp")
	}
	switch proxy.Get("type") {
	case "ssh":
		list = append(list, sshParser(proxy))
	case "http":
		list = append(list, httpParser(proxy))
	case "socks5":
		if truthy(proxy.Get("tls")) {
			throw("Platform sing-box does not support proxy type: socks5 with tls")
		}
		list = append(list, socks5Parser(proxy))
	case "ss":
		if proxy.Get("plugin") == "shadow-tls" {
			ssPart, stPart := shadowTLSParser(proxy)
			list = append(list, ssPart, stPart)
		} else {
			list = append(list, ssParser(proxy))
		}
	case "snell":
		list = append(list, snellParser(proxy))
		if pluginOpts := getShadowTLSPluginOpts(proxy); truthy(pluginOpts) {
			list = append(list, shadowTLSOutboundParser(proxy, pluginOpts))
		}
	case "vmess":
		if !truthy(proxy.Get("network")) || includesString(proxy.Get("network"), "tcp", "ws", "grpc", "h2", "http") {
			list = append(list, vmessParser(proxy))
		} else {
			throw("Platform sing-box does not support proxy type: vmess with network " + jsString(proxy.Get("network")))
		}
	case "vless":
		if encryption := proxy.Get("encryption"); truthy(encryption) && encryption != "none" {
			throw("VLESS encryption is not supported")
		}
		if flow := proxy.Get("flow"); !truthy(flow) || flow == "xtls-rprx-vision" {
			list = append(list, vlessParser(proxy))
		} else {
			throw("Platform sing-box does not support proxy type: vless with flow " + jsString(flow))
		}
	case "trojan":
		if truthy(proxy.Get("flow")) {
			throw("Platform sing-box does not support proxy type: trojan with flow " + jsString(proxy.Get("flow")))
		}
		list = append(list, trojanParser(proxy))
	case "naive":
		list = append(list, naiveParser(proxy))
	case "hysteria":
		list = append(list, hysteriaParser(proxy))
	case "hysteria2":
		list = append(list, hysteria2Parser(proxy))
	case "tuic":
		if token := proxy.Get("token"); !truthy(token) || jsLength(token) == 0 {
			list = append(list, tuic5Parser(proxy))
		} else {
			throw("Platform sing-box does not support proxy type: TUIC v4")
		}
	case "anytls":
		list = append(list, anytlsParser(proxy))
	default:
		// wireguard and tailscale become endpoints, which providers do not load.
		throw("Platform sing-box does not support proxy type: " + jsString(proxy.Get("type")))
	}
	if streamShadowTLSOutbound != nil {
		outbound := list[0]
		outbound.Set("detour", getShadowTLSTag(proxy))
		outbound.Delete("tls")
		list = append(list, streamShadowTLSOutbound)
	}
	return list, nil
}

func recoveredError(recovered any) error {
	if err, isError := recovered.(error); isError {
		return err
	}
	return jsError(jsString(recovered))
}

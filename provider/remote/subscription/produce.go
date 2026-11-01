// This file is part of a Go port of Sub-Store (https://github.com/sub-store-org/Sub-Store),
// licensed under the GNU Affero General Public License v3.0. See NOTICE in this directory.

package subscription

import (
	"strings"
)

// core/proxy-utils/index.js: produce, with targetPlatform "sing-box" and no
// options; core/proxy-utils/producers/clashmeta.js with type "internal" and
// "include-unsupported-proxy", as the sing-box producer calls it.

func produce(proxies []*Object) []*Object {
	filtered := proxies[:0:0]
	for _, proxy := range proxies {
		if supported, isObject := proxy.Get("supported").(*Object); isObject && supported.Get("sing-box") == false {
			continue
		}
		if hasRootProxyHeaders(proxy) && includesString(proxy.Get("type"), "http", "h2-connect", "trusttunnel") && proxy.Get("type") != "http" {
			continue
		}
		if isShadowsocksOverTLS(proxy) {
			continue
		}
		if proxy.Get("type") == "vless" {
			if realityOpts := proxy.Get("reality-opts"); truthy(realityOpts) && !isNotBlank(getPath(realityOpts, "public-key")) {
				continue
			}
		}
		filtered = append(filtered, proxy)
	}
	for _, proxy := range filtered {
		proxy.Set("_resolved", proxy.Get("resolved"))
		if !isNotBlank(proxy.Get("name")) {
			proxy.Set("name", jsString(proxy.Get("type"))+" "+jsString(proxy.Get("server"))+":"+jsString(proxy.Get("port")))
		}
		if truthy(proxy.Get("ports")) {
			ports := slashRegex.ReplaceAllString(jsString(proxy.Get("ports")), ",")
			proxy.Set("ports", ports)
			if !truthy(proxy.Get("port")) {
				proxy.Set("port", getRandomPort(ports))
			}
		}
	}
	return produceSingBox(filtered)
}

func hasRootProxyHeaders(proxy *Object) bool {
	headers := proxy.Get("headers")
	switch headers.(type) {
	case *Object, []any:
		return objectKeyCount(headers) > 0
	}
	return false
}

func isShadowsocksOverTLS(proxy *Object) bool {
	network := proxy.Get("network")
	if s, isString := network.(string); isString {
		network = strings.ToLower(strings.TrimSpace(s))
	}
	return proxy.Get("type") == "ss" && proxy.Get("tls") == true && !isPresentPath(proxy, "plugin") &&
		(!isPresentPath(proxy, "network") || network == "tcp")
}

func normalizePluginMuxValue(mux any) any {
	switch m := mux.(type) {
	case bool:
		if m {
			return float64(1)
		}
		return float64(0)
	case string:
		normalized := strings.ToLower(strings.TrimSpace(m))
		switch {
		case normalized == "true":
			return float64(1)
		case normalized == "false":
			return float64(0)
		case digitsOnlyRegex.MatchString(normalized):
			return parseInt(normalized)
		}
	}
	return mux
}

func restoreShadowTLSOpts(target *Object, serverNameKey string) any {
	if target == nil || target.Get("plugin") != "shadow-tls" || !truthy(target.Get("plugin-opts")) {
		return nil
	}
	opts := target.Get("plugin-opts")
	version := getPath(opts, "version")
	enabled := truthy(getPath(opts, "password")) || (!isNullish(version) && jsNumber(version) != 0)
	target.Set("shadow-tls-opts", obj(
		"password", getPath(opts, "password"),
		"version", version,
	))
	if host := getPath(opts, "host"); !isNullish(host) {
		target.Set(serverNameKey, host)
	}
	if alpn := getPath(opts, "alpn"); !isNullish(alpn) {
		target.Set("alpn", alpn)
	}
	target.Delete("plugin")
	target.Delete("plugin-opts")
	return enabled
}

func restoreShadowTLSProxyOpts(proxy *Object) {
	if includesString(proxy.Get("type"), "vmess", "vless", "trojan", "anytls") {
		restored := restoreShadowTLSOpts(proxy, "sni")
		if truthy(restored) && includesString(proxy.Get("type"), "vmess", "vless") {
			proxy.Set("tls", true)
		}
	}
}

func getMihomoShadowTLSOpts(proxy *Object) any {
	if proxy.Get("plugin") == "shadow-tls" && truthy(proxy.Get("plugin-opts")) {
		return proxy.Get("plugin-opts")
	}
	if proxy.Get("type") == "snell" && getPath(proxy.Get("obfs-opts"), "mode") == "shadow-tls" {
		return proxy.Get("obfs-opts")
	}
	return nil
}

var clashIPVersions = map[string]string{
	"dual":      "dual",
	"v4-only":   "ipv4",
	"v6-only":   "ipv6",
	"prefer-v4": "ipv4-prefer",
	"prefer-v6": "ipv6-prefer",
}

func produceClashMetaInternal(proxy *Object) {
	restoreShadowTLSProxyOpts(proxy)
	if truthy(proxy.Get("reality-opts")) && !truthy(proxy.Get("client-fingerprint")) {
		proxy.Set("client-fingerprint", "chrome")
	}
	switch proxy.Get("type") {
	case "vmess":
		if isPresentPath(proxy, "aead") {
			if truthy(proxy.Get("aead")) {
				proxy.Set("alterId", float64(0))
			}
			proxy.Delete("aead")
		}
		if isPresentPath(proxy, "sni") {
			proxy.Set("servername", proxy.Get("sni"))
			proxy.Delete("sni")
		}
		proxy.Set("cipher", normalizeVmessSecurity(proxy.Get("cipher")))
	case "tuic":
		if isPresentPath(proxy, "alpn") {
			if _, isArray := proxy.Get("alpn").([]any); !isArray {
				proxy.Set("alpn", []any{proxy.Get("alpn")})
			}
		}
		if isPresentPath(proxy, "tfo") && !isPresentPath(proxy, "fast-open") {
			proxy.Set("fast-open", proxy.Get("tfo"))
		}
		if !truthy(proxy.Get("token")) || jsLength(proxy.Get("token")) == 0 {
			if !isPresentPath(proxy, "version") {
				proxy.Set("version", float64(5))
			}
		}
	case "hysteria":
		if isPresentPath(proxy, "auth_str") && !isPresentPath(proxy, "auth-str") {
			proxy.Set("auth-str", proxy.Get("auth_str"))
		}
		if isPresentPath(proxy, "alpn") {
			if _, isArray := proxy.Get("alpn").([]any); !isArray {
				proxy.Set("alpn", []any{proxy.Get("alpn")})
			}
		}
		if isPresentPath(proxy, "tfo") && !isPresentPath(proxy, "fast-open") {
			proxy.Set("fast-open", proxy.Get("tfo"))
		}
	case "snell":
		if jsLess(proxy.Get("version"), 3) {
			proxy.Delete("udp")
		}
	case "vless":
		if isPresentPath(proxy, "sni") {
			proxy.Set("servername", proxy.Get("sni"))
			proxy.Delete("sni")
		}
	case "anytls":
		if !isNullish(proxy.Get("reuse")) && !truthy(proxy.Get("reuse")) {
			proxy.Set("disable-reuse", true)
			proxy.Delete("reuse")
		}
	}
	if isPresentPath(proxy, "plugin-opts.mux") {
		proxy.Obj("plugin-opts").Set("mux", truthy(normalizePluginMuxValue(getPath(proxy, "plugin-opts.mux"))))
	}
	if proxy.Get("type") == "snell" {
		if shadowTLSOpts := getMihomoShadowTLSOpts(proxy); truthy(shadowTLSOpts) {
			obfsOpts := obj(
				"mode", "shadow-tls",
				"host", getPath(shadowTLSOpts, "host"),
				"password", getPath(shadowTLSOpts, "password"),
				"version", getPath(shadowTLSOpts, "version"),
			)
			if alpn := getPath(shadowTLSOpts, "alpn"); truthy(alpn) {
				obfsOpts.Set("alpn", alpn)
			}
			proxy.Set("obfs-opts", obfsOpts)
			proxy.Delete("plugin")
			proxy.Delete("plugin-opts")
		}
	}
	if includesString(proxy.Get("type"), "vmess", "vless") && proxy.Get("network") == "http" {
		if isPresentPath(proxy, "http-opts.path") {
			if path := getPath(proxy, "http-opts.path"); !isArray(path) {
				proxy.Obj("http-opts").Set("path", []any{path})
			}
		}
		if isPresentPath(proxy, "http-opts.headers.Host") {
			if host := getPath(proxy, "http-opts.headers.Host"); !isArray(host) {
				mustObject(getPath(proxy, "http-opts.headers")).Set("Host", []any{host})
			}
		}
	}
	if includesString(proxy.Get("type"), "vmess", "vless") && proxy.Get("network") == "h2" {
		if isPresentPath(proxy, "h2-opts.path") {
			if path, isArray := getPath(proxy, "h2-opts.path").([]any); isArray {
				proxy.Obj("h2-opts").Set("path", arrayAt(path, 0))
			}
		}
		host := coalesce(getPath(proxy, "h2-opts.host"), getPath(proxy, "h2-opts.headers.host"), getPath(proxy, "h2-opts.headers.Host"), nil)
		if isPresentPath(proxy, "h2-opts.host") || isPresentPath(proxy, "h2-opts.headers.host") || isPresentPath(proxy, "h2-opts.headers.Host") {
			if !isArray(host) {
				host = []any{host}
			}
			mustObject(proxy.Get("h2-opts")).Set("host", host)
		}
		if headers := getPath(proxy, "h2-opts.headers"); truthy(headers) {
			if headersObject, isObject := headers.(*Object); isObject {
				headersObject.Delete("host")
				headersObject.Delete("Host")
			}
			if objectKeyCount(headers) == 0 {
				proxy.Obj("h2-opts").Delete("headers")
			}
		}
	}
	if proxy.Get("network") == "ws" {
		key := networkOptsKey(proxy)
		opts, isObject := proxy.Get(key).(*Object)
		if !truthy(proxy.Get(key)) {
			opts = newObject()
			proxy.Set(key, opts)
		} else if !isObject {
			throw("TypeError: invalid ws-opts")
		}
		if !truthy(opts.Get("path")) {
			opts.Set("path", "/")
		}
		normalizeWebSocketEarlyDataPath(opts)
	}
	if truthy(getPath(proxy, "plugin-opts.tls")) {
		if isPresentPath(proxy, "skip-cert-verify") {
			pluginOpts := proxy.Obj("plugin-opts")
			pluginOpts.Set("skip-cert-verify", or(pluginOpts.Get("skip-cert-verify"), proxy.Get("skip-cert-verify")))
		}
	}
	if includesString(proxy.Get("type"), "trojan", "tuic", "hysteria", "hysteria2", "juicity", "anytls", "trusttunnel", "naive", "masque", "shadowquic") {
		proxy.Delete("tls")
	}
	if truthy(proxy.Get("tls-fingerprint")) {
		proxy.Set("fingerprint", proxy.Get("tls-fingerprint"))
	}
	proxy.Delete("tls-fingerprint")
	if truthy(proxy.Get("underlying-proxy")) {
		proxy.Set("dialer-proxy", proxy.Get("underlying-proxy"))
	}
	proxy.Delete("underlying-proxy")
	if isPresentPath(proxy, "tls") {
		if _, isBool := proxy.Get("tls").(bool); !isBool {
			proxy.Delete("tls")
		}
	}
	for _, key := range []string{"subName", "collectionName", "id", "resolved", "no-resolve", "ip-cidr", "ipv6-cidr"} {
		proxy.Delete(key)
	}
	if proxy.Get("network") == "grpc" {
		if opts := networkOpts(proxy); truthy(proxy.Get(networkOptsKey(proxy))) && opts != nil {
			opts.Delete("_grpc-type")
			opts.Delete("_grpc-authority")
		}
	}
	if ipVersion, isString := proxy.Get("ip-version").(string); isString && ipVersion != "" {
		if mapped, loaded := clashIPVersions[ipVersion]; loaded {
			proxy.Set("ip-version", mapped)
		}
	}
}

func isArray(value any) bool {
	_, isArray := value.([]any)
	return isArray
}

// jsLength is value.length for strings and arrays.
func jsLength(value any) float64 {
	switch v := value.(type) {
	case string:
		return float64(len([]rune(v)))
	case []any:
		return float64(len(v))
	}
	return -1
}

// jsLess is value < number with JavaScript coercion.
func jsLess(value any, number float64) bool {
	if isNullish(value) && value == nil {
		return false
	}
	return jsNumber(value) < number
}

// cloneJSON is JSON.parse(JSON.stringify(value)).
func cloneJSON(value any) any {
	switch v := value.(type) {
	case *Object:
		clone := newObject()
		for _, key := range v.Keys() {
			item := v.Get(key)
			if item == nil {
				continue
			}
			clone.Set(key, cloneJSON(item))
		}
		return clone
	case []any:
		clone := make([]any, len(v))
		for i, item := range v {
			if item == nil {
				clone[i] = jsNull
			} else {
				clone[i] = cloneJSON(item)
			}
		}
		return clone
	case float64:
		if isNaN(v) {
			return jsNull
		}
		return v
	default:
		return value
	}
}

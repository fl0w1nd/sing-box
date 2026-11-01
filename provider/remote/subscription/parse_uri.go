// This file is part of a Go port of Sub-Store (https://github.com/sub-store-org/Sub-Store),
// licensed under the GNU Affero General Public License v3.0. See NOTICE in this directory.

package subscription

import (
	"regexp"
	"strings"
)

// core/proxy-utils/parsers/index.js: URI parsers.
//
// Parsers panic where Sub-Store throws, including the TypeErrors raised when
// it reads a property of undefined; the parse loop treats the line as failed.
//
// Not ported: URI_SSR and URI_WireGuard, since sing-box has no SSR outbound and
// WireGuard is an endpoint, and VLESS xhttp, which sing-box does not support.

type parser struct {
	name  string
	test  func(line string) bool
	parse func(line string) *Object
}

func throw(message string) {
	panic(jsError(message))
}

// and is a && b.
func and(a, b any) any {
	if !truthy(a) {
		return a
	}
	return b
}

// splitCapture returns line.split(re)[2] for a regex with one capture group:
// the text between the first and the second match.
func splitCapture(re *regexp.Regexp, line string) any {
	matches := re.FindAllStringIndex(line, 2)
	if len(matches) == 0 {
		return nil
	}
	end := len(line)
	if len(matches) > 1 {
		end = matches[1][0]
	}
	return line[matches[0][1]:end]
}

// splitFirst returns s.split(separator)[1].
func splitFirst(s string, separator string) string {
	parts := split(s, separator)
	if len(parts) < 2 {
		throw("TypeError: Cannot read properties of undefined")
	}
	return parts[1]
}

// splitPair is `const [key, value] = s.split("=")`: only the first two parts.
func splitPair(s string, separator string) (string, any) {
	parts := split(s, separator)
	return parts[0], at(parts, 1)
}

var (
	uriFragmentRegex = regexp.MustCompile(`^(.*?)(?:#(.*?))?$`)
	digitsOnlyRegex  = regexp.MustCompile(`^\d+$`)
	trueOrOneRegex   = regexp.MustCompile(`(?i)(TRUE)|1`)
)

func splitURIFragment(raw string) (string, any) {
	match := regexExec(uriFragmentRegex, raw)
	content := group(match, 1)
	if match[2] == nil {
		return content, nil
	}
	return content, mustDecodeURIComponent(match[2])
}

func decodeShadowsocksUserInfo(raw string) string {
	before, after, ok := strings.Cut(raw, ":")
	if ok {
		return mustDecodeURIComponent(before) + ":" + mustDecodeURIComponent(after)
	}
	decoded := mustDecodeURIComponent(raw)
	if strings.Contains(decoded, ":") {
		return decoded
	}
	return base64Decode(decoded)
}

func isNumericEarlyData(value any) bool {
	if isNullish(value) || !digitsOnlyRegex.MatchString(jsString(value)) {
		return false
	}
	return isSafeInteger(parseInt(value))
}

func extractEarlyDataFromPath(path any) (any, string) {
	ed := getPathQueryParam(path, "ed")
	if !isNumericEarlyData(ed) {
		return path, ""
	}
	extracted, _ := extractPathQueryParam(path, "ed")
	return extracted, ed
}

func parseEarlyDataSize(value any) float64 {
	raw := jsString(value)
	if !digitsOnlyRegex.MatchString(raw) {
		throw("bad WebSocket max early data size: " + raw)
	}
	parsed := parseInt(raw)
	if !isSafeInteger(parsed) {
		throw("bad WebSocket max early data size: " + raw)
	}
	return parsed
}

func splitURIHostList(host any) any {
	switch h := host.(type) {
	case []any:
		var result []any
		for _, item := range h {
			if list, isArray := splitURIHostList(item).([]any); isArray {
				result = append(result, list...)
			}
		}
		if result == nil {
			result = []any{}
		}
		return result
	case string:
		hosts := trimStrings(strings.Split(h, ","), true)
		if len(hosts) > 0 {
			return hosts
		}
		return nil
	default:
		if isNullish(host) {
			return nil
		}
		return []any{host}
	}
}

// parseQuery parses "a=b&c=d" with `const [key, valueRaw] = addon.split("=")`
// and decodeURIComponent(valueRaw).
func parseQuery(query string) *Object {
	params := newObject()
	for addon := range strings.SplitSeq(query, "&") {
		if addon == "" {
			continue
		}
		key, valueRaw := splitPair(addon, "=")
		params.Set(key, mustDecodeURIComponent(valueRaw))
	}
	return params
}

var uriParsers = []parser{
	{"URI PROXY Parser", uriProxyTest, uriProxyParse},
	{"URI SOCKS Parser", uriSocksTest, uriSocksParse},
	{"URI SS Parser", uriSSTest, uriSSParse},
	{"URI VMess Parser", uriVMessTest, uriVMessParse},
	{"URI VLESS Parser", uriVLESSTest, func(line string) *Object { return uriVLESSParse(line, "vless") }},
	{"URI TUIC Parser", uriTUICTest, uriTUICParse},
	{"URI Hysteria Parser", uriHysteriaTest, uriHysteriaParse},
	{"URI Hysteria2 Parser", uriHysteria2Test, uriHysteria2Parse},
	{"URI Trojan Parser", uriTrojanTest, uriTrojanParse},
	{"URI AnyTLS Parser", uriAnyTLSTest, uriAnyTLSParse},
}

// URI_PROXY

var (
	uriProxyTestRegex = regexp.MustCompile(`^(socks5\+tls|socks5|http|https)://`)
	uriProxyRegex     = regexp.MustCompile(`^(socks5|http|http)(\+tls|s)?://(?:(.*?):(.*?)@)?(.*?)(?::(\d+?))?/?(\?.*?)?(?:#(.*?))?$`)
)

func uriProxyTest(line string) bool {
	return uriProxyTestRegex.MatchString(line)
}

func uriProxyParse(line string) *Object {
	match := regexExec(uriProxyRegex, line)
	if match == nil {
		throw("TypeError: line.match(...) is not iterable")
	}
	proxyType, tls, username, password, server, port, name := match[1], match[2], match[3], match[4], match[5], match[6], match[8]
	var portValue float64
	if truthy(port) {
		portValue = parseInt(port)
	} else if truthy(tls) {
		portValue = 443
	} else if proxyType == "http" {
		portValue = 80
	} else {
		throw("port is not present in line: " + line)
	}
	var proxyName any
	if !isNullish(name) {
		proxyName = mustDecodeURIComponent(name)
	} else {
		proxyName = jsString(proxyType) + " " + jsString(server) + ":" + numberToString(portValue)
	}
	var decodedUsername, decodedPassword any
	if !isNullish(username) {
		decodedUsername = mustDecodeURIComponent(username)
	}
	if !isNullish(password) {
		decodedPassword = mustDecodeURIComponent(password)
	}
	return obj(
		"name", proxyName,
		"type", proxyType,
		"tls", truthy(tls),
		"server", server,
		"port", portValue,
		"username", decodedUsername,
		"password", decodedPassword,
	)
}

// URI_SOCKS

var (
	uriSocksTestRegex = regexp.MustCompile(`^socks://`)
	uriSocksRegex     = regexp.MustCompile(`^(socks)?://(?:(.*)@)?(.*?)(?::(\d+?))?(\?.*?)?(?:#(.*?))?$`)
)

func uriSocksTest(line string) bool {
	return uriSocksTestRegex.MatchString(line)
}

func uriSocksParse(line string) *Object {
	match := regexExec(uriSocksRegex, line)
	if match == nil {
		throw("TypeError: line.match(...) is not iterable")
	}
	proxyType, auth, server, port, name := match[1], match[2], match[3], match[4], match[6]
	if !truthy(port) {
		throw("port is not present in line: " + line)
	}
	portValue := parseInt(port)
	var username, password any
	if truthy(auth) {
		parsed := split(base64Decode(mustDecodeURIComponent(auth)), ":")
		username = parsed[0]
		password = at(parsed, 1)
	}
	var proxyName any
	if !isNullish(name) {
		proxyName = mustDecodeURIComponent(name)
	} else {
		proxyName = jsString(proxyType) + " " + jsString(server) + ":" + numberToString(portValue)
	}
	return obj(
		"name", proxyName,
		"type", "socks5",
		"server", server,
		"port", portValue,
		"username", username,
		"password", password,
	)
}

// URI_SS

var (
	ssTestRegex            = regexp.MustCompile(`^ss://`)
	ssServerAndPortRegex   = regexp.MustCompile(`@([^/?]*)(/|\?|$)`)
	ssQuerySplitRegex      = regexp.MustCompile(`^(.*)(\?.*)$`)
	ssV2rayPluginTestRegex = regexp.MustCompile(`(&|\?)v2ray-plugin=`)
	ssV2rayPluginRegex     = regexp.MustCompile(`(&|\?)v2ray-plugin=(.*?)(&|$)`)
	ssLegacyUserInfoRegex  = regexp.MustCompile(`(^.*)@`)
	ssLegacyServerRegex    = regexp.MustCompile(`@([^/@]*)(/|$)`)
	ssQueryRegex           = regexp.MustCompile(`(\?.*)$`)
	ssPortRegex            = regexp.MustCompile(`\d+`)
	ssUserInfoRegex        = regexp.MustCompile(`(^.*?):(.*$)`)
	ssPluginRegex          = regexp.MustCompile(`[?&]plugin=([^&]+)`)
	ssShadowTLSRegex       = regexp.MustCompile(`[?&]shadow-tls=([^&]+)`)
	ssGostRegex            = regexp.MustCompile(`[?&]gost=([^&]+)`)
	ssUOTRegex             = regexp.MustCompile(`(?i)(&|\?)uot=(1|true)`)
	ssTFORegex             = regexp.MustCompile(`(?i)(&|\?)tfo=(1|true)`)
	escapedEqualsRegex     = regexp.MustCompile(`\\=`)
	leadingQuestionRegex   = regexp.MustCompile(`^\?`)
)

func uriSSTest(line string) bool {
	return ssTestRegex.MatchString(line)
}

func uriSSParse(line string) *Object {
	content, name := splitURIFragment(splitFirst(line, "ss://"))
	proxy := obj("type", "ss")
	serverAndPortArray := regexExec(ssServerAndPortRegex, content)
	userInfoStr := any(decodeShadowsocksUserInfo(split(content, "@")[0]))
	query := ""
	if serverAndPortArray == nil {
		if strings.Contains(content, "?") {
			parsed := regexExec(ssQuerySplitRegex, content)
			content = group(parsed, 1)
			query = group(parsed, 2)
		}
		content = base64Decode(content)
		if query != "" {
			if ssV2rayPluginTestRegex.MatchString(query) {
				parsed := regexExec(ssV2rayPluginRegex, query)
				v2rayPlugin := group(parsed, 2)
				if v2rayPlugin != "" {
					proxy.Set("plugin", "v2ray-plugin")
					pluginOpts, err := jsonParse(base64Decode(v2rayPlugin))
					if err != nil {
						panic(err)
					}
					proxy.Set("plugin-opts", pluginOpts)
				}
			}
			content = content + query
		}
		userInfoStr = nil
		if match := regexExec(ssLegacyUserInfoRegex, content); match != nil {
			userInfoStr = match[1]
		}
		serverAndPortArray = regexExec(ssLegacyServerRegex, content)
	} else if strings.Contains(content, "?") {
		query = group(regexExec(ssQueryRegex, content), 1)
	}
	params := parseQuery(leadingQuestionRegex.ReplaceAllString(query, ""))
	proxy.Set("tls", and(params.Get("security"), params.Get("security") != "none"))
	proxy.Set("skip-cert-verify", truthy(params.Get("allowInsecure")))
	proxy.Set("sni", or(params.Get("sni"), params.Get("peer")))
	proxy.Set("client-fingerprint", params.Get("fp"))
	if truthy(params.Get("alpn")) {
		proxy.Set("alpn", stringsToArray(split(mustDecodeURIComponent(params.Get("alpn")), ",")))
	} else {
		proxy.Set("alpn", nil)
	}
	if truthy(params.Get("ws")) {
		proxy.Set("network", "ws")
		setPath(proxy, "ws-opts.path", params.Get("wspath"))
	}
	if truthy(params.Get("type")) {
		httpupgrade := false
		httpUpgradeEd := ""
		pathEarlyData := ""
		network := jsString(params.Get("type"))
		if network == "httpupgrade" {
			network = "ws"
			httpupgrade = true
		}
		proxy.Set("network", network)
		if network == "grpc" {
			proxy.Set(network+"-opts", obj(
				"grpc-service-name", params.Get("serviceName"),
				"_grpc-type", params.Get("mode"),
				"_grpc-authority", params.Get("authority"),
			))
		} else {
			if truthy(params.Get("path")) {
				transportPath := params.Get("path")
				if network == "ws" {
					var ed string
					transportPath, ed = extractEarlyDataFromPath(transportPath)
					if httpupgrade {
						httpUpgradeEd = ed
					} else {
						pathEarlyData = ed
					}
				}
				setPath(proxy, network+"-opts.path", transportPath)
			}
			if truthy(params.Get("host")) {
				setPath(proxy, network+"-opts.headers.Host", mustDecodeURIComponent(params.Get("host")))
			}
			if httpupgrade {
				if httpUpgradeEd == "" && isNumericEarlyData(params.Get("ed")) {
					httpUpgradeEd = jsString(params.Get("ed"))
				}
				setPath(proxy, network+"-opts.v2ray-http-upgrade", true)
				if httpUpgradeEd != "" {
					setPath(proxy, network+"-opts.v2ray-http-upgrade-fast-open", true)
					setPath(proxy, network+"-opts._v2ray-http-upgrade-ed", httpUpgradeEd)
				}
			} else if network == "ws" && pathEarlyData != "" {
				setPath(proxy, network+"-opts.max-early-data", parseEarlyDataSize(pathEarlyData))
				setPath(proxy, network+"-opts.early-data-header-name", "Sec-WebSocket-Protocol")
			}
		}
		if params.Get("security") == "reality" {
			opts := newObject()
			if truthy(params.Get("pbk")) {
				opts.Set("public-key", params.Get("pbk"))
			}
			if truthy(params.Get("sid")) {
				opts.Set("short-id", params.Get("sid"))
			}
			if truthy(params.Get("spx")) {
				opts.Set("_spider-x", params.Get("spx"))
			}
			if truthy(params.Get("mode")) {
				proxy.Set("_mode", params.Get("mode"))
			}
			if truthy(params.Get("extra")) {
				proxy.Set("_extra", params.Get("extra"))
			}
			if opts.Len() > 0 {
				setPath(proxy, "reality-opts", opts)
			}
		}
	}
	udp := params.Get("udp")
	if s, isString := udp.(string); isString {
		udp = strings.ToLower(s)
	}
	proxy.Set("udp", !includes(udp, false, float64(0), "0", "false", "off"))

	if serverAndPortArray == nil {
		throw("TypeError: Cannot read properties of null")
	}
	serverAndPort := group(serverAndPortArray, 1)
	portIndex := strings.LastIndex(serverAndPort, ":")
	if portIndex < 0 {
		proxy.Set("server", "")
	} else {
		proxy.Set("server", serverAndPort[:portIndex])
	}
	if port := ssPortRegex.FindString(serverAndPort[portIndex+1:]); port != "" {
		proxy.Set("port", port)
	} else {
		proxy.Set("port", nil)
	}
	userInfoString, isString := userInfoStr.(string)
	if !isString {
		throw("TypeError: Cannot read properties of undefined")
	}
	if userInfo := regexExec(ssUserInfoRegex, userInfoString); userInfo != nil {
		proxy.Set("cipher", userInfo[1])
		proxy.Set("password", userInfo[2])
	} else {
		proxy.Set("cipher", nil)
		proxy.Set("password", nil)
	}

	pluginMatch := regexExec(ssPluginRegex, content)
	shadowTLSMatch := regexExec(ssShadowTLSRegex, content)
	gostMatch := regexExec(ssGostRegex, content)
	if pluginMatch != nil {
		pluginParams := newObject()
		for item := range strings.SplitSeq("plugin="+mustDecodeURIComponent(pluginMatch[1]), ";") {
			before, after, ok := strings.Cut(item, "=")
			if !ok {
				if item != "" {
					pluginParams.Set(item, true)
				}
				continue
			}
			key := before
			value := escapedEqualsRegex.ReplaceAllString(after, "=")
			if key != "" {
				pluginParams.Set(key, or(value, true))
			}
		}
		switch pluginParams.Get("plugin") {
		case "obfs-local", "simple-obfs":
			proxy.Set("plugin", "obfs")
			proxy.Set("plugin-opts", obj(
				"mode", pluginParams.Get("obfs"),
				"host", getIfNotBlank(pluginParams.Get("obfs-host"), nil),
			))
		case "v2ray-plugin":
			proxy.Set("plugin", "v2ray-plugin")
			var mux any
			if testString(digitsOnlyRegex, pluginParams.Get("mux")) {
				mux = parseInt(pluginParams.Get("mux"))
			}
			proxy.Set("plugin-opts", obj(
				"mode", or(getIfNotBlank(pluginParams.Get("obfs"), nil), getIfNotBlank(pluginParams.Get("mode"), nil), "websocket"),
				"host", or(getIfNotBlank(pluginParams.Get("obfs-host"), nil), getIfNotBlank(pluginParams.Get("host"), nil)),
				"path", getIfNotBlank(pluginParams.Get("path"), nil),
				"tls", getIfPresent(pluginParams.Get("tls"), nil),
				"sni", getIfPresent(pluginParams.Get("sni"), nil),
				"skip-cert-verify", includes(pluginParams.Get("skip-cert-verify"), "1", "true", float64(1), true),
				"mux", mux,
			))
		case "shadow-tls":
			proxy.Set("plugin", "shadow-tls")
			version := getIfNotBlank(pluginParams.Get("version"), nil)
			var versionValue any
			if truthy(version) {
				versionValue = parseInt(version)
			}
			proxy.Set("plugin-opts", obj(
				"host", getIfNotBlank(pluginParams.Get("host"), nil),
				"password", getIfNotBlank(pluginParams.Get("password"), nil),
				"version", versionValue,
			))
		default:
			throw("Unsupported plugin option: " + jsString(pluginParams.Get("plugin")))
		}
	}
	if shadowTLSMatch != nil {
		parsed, err := jsonParse(base64Decode(shadowTLSMatch[1]))
		if err != nil {
			panic(err)
		}
		shadowTLSParams := mustObject(parsed)
		version := getIfNotBlank(shadowTLSParams.Get("version"), nil)
		address := getIfNotBlank(shadowTLSParams.Get("address"), nil)
		port := getIfNotBlank(shadowTLSParams.Get("port"), nil)
		var versionValue any
		if truthy(version) {
			versionValue = parseInt(version)
		}
		proxy.Set("plugin", "shadow-tls")
		proxy.Set("plugin-opts", obj(
			"host", getIfNotBlank(shadowTLSParams.Get("host"), nil),
			"password", getIfNotBlank(shadowTLSParams.Get("password"), nil),
			"version", versionValue,
		))
		if truthy(address) {
			proxy.Set("server", address)
		}
		if truthy(port) {
			proxy.Set("port", parseInt(port))
		}
	}
	if gostMatch != nil {
		parsed, err := jsonParse(base64Decode(mustDecodeURIComponent(gostMatch[1])))
		if err != nil {
			panic(err)
		}
		gostParams := mustObject(parsed)
		address := getIfNotBlank(gostParams.Get("address"), nil)
		port := getIfNotBlank(gostParams.Get("port"), nil)
		route := getIfNotBlank(gostParams.Get("route"), nil)
		var normalizedRoute any
		if routeString, isString := route.(string); isString {
			normalizedRoute = strings.ToLower(strings.TrimSpace(routeString))
		}
		mode := route
		if includes(normalizedRoute, "ws", "wss", "websocket") {
			mode = "websocket"
		}
		proxy.Set("plugin", "gost-plugin")
		pluginOpts := obj(
			"mode", mode,
			"host", getIfNotBlank(gostParams.Get("host"), nil),
			"path", getIfNotBlank(gostParams.Get("path"), nil),
		)
		if normalizedRoute == "wss" {
			pluginOpts.Set("tls", true)
		}
		proxy.Set("plugin-opts", pluginOpts)
		if truthy(address) {
			proxy.Set("server", address)
		}
		if truthy(port) {
			proxy.Set("port", parseInt(port))
		}
	}
	if ssUOTRegex.MatchString(query) {
		proxy.Set("udp-over-tcp", true)
	}
	if ssTFORegex.MatchString(query) {
		proxy.Set("tfo", true)
	}
	if !isNullish(name) {
		proxy.Set("name", name)
	} else {
		proxy.Set("name", "SS "+jsString(proxy.Get("server"))+":"+jsString(proxy.Get("port")))
	}
	return proxy
}

func mustObject(value any) *Object {
	o, isObject := value.(*Object)
	if !isObject {
		throw("TypeError: Cannot read properties of non-object")
	}
	return o
}

// URI_VMess

var (
	vmessTestRegex         = regexp.MustCompile(`^vmess://`)
	vmessUserInfoRegex     = regexp.MustCompile(`^vmess://[^/?#]+@`)
	vmessQueryTrimRegex    = regexp.MustCompile(`\?.*?$`)
	vmessQuantumultRegex   = regexp.MustCompile(`=\s*vmess`)
	quotedRegex            = regexp.MustCompile(`^"(.*)"$`)
	vmessShadowrocketRegex = regexp.MustCompile(`(^[^?]+?)/?\?(.*)$`)
	vmessShadowrocketBody  = regexp.MustCompile(`(^[^:]+?):([^:]+?)@(.*):(\d+)$`)
)

func uriVMessTest(line string) bool {
	return vmessTestRegex.MatchString(line)
}

func uriVMessParse(line string) *Object {
	if vmessUserInfoRegex.MatchString(line) {
		return uriVLESSParse(line, "vmess")
	}
	lineWithoutFragment, fragmentName := splitURIFragment(splitFirst(line, "vmess://"))
	content := base64Decode(vmessQueryTrimRegex.ReplaceAllString(lineWithoutFragment, ""))
	if vmessQuantumultRegex.MatchString(content) {
		partitions := trimStrings(strings.Split(content, ","), false)
		params := newObject()
		for _, part := range partitions {
			partString := part.(string)
			if strings.Contains(partString, "=") {
				key, value := splitPair(partString, "=")
				params.Set(strings.TrimSpace(key), strings.TrimSpace(jsString(value)))
			}
		}
		if len(partitions) < 5 {
			throw("TypeError: Cannot read properties of undefined")
		}
		uuidMatch := regexExec(quotedRegex, partitions[4].(string))
		if uuidMatch == nil {
			throw("TypeError: Cannot read properties of null")
		}
		var skipCertVerify any
		if !isNullish(params.Get("tls-verification")) {
			skipCertVerify = !truthy(params.Get("tls-verification"))
		}
		proxy := obj(
			"name", strings.TrimSpace(split(partitions[0].(string), "=")[0]),
			"type", "vmess",
			"server", partitions[1],
			"port", partitions[2],
			"cipher", normalizeVmessSecurity(getIfNotBlank(partitions[3], "auto")),
			"uuid", uuidMatch[1],
			"tls", params.Get("obfs") == "wss",
			"udp", getIfPresent(params.Get("udp-relay"), nil),
			"tfo", getIfPresent(params.Get("fast-open"), nil),
			"skip-cert-verify", skipCertVerify,
		)
		if !isNullish(params.Get("obfs")) {
			if params.Get("obfs") == "ws" || params.Get("obfs") == "wss" {
				proxy.Set("network", "ws")
				// Sub-Store sets proxy["ws-opts"].path before creating "ws-opts".
				throw("TypeError: Cannot set properties of undefined (setting 'path')")
			}
			throw("Unsupported obfs: " + jsString(params.Get("obfs")))
		}
		if isNotBlank(fragmentName) {
			proxy.Set("name", fragmentName)
		}
		return proxy
	}

	var params *Object
	parsed, err := jsonParse(content)
	if err == nil {
		params, _ = parsed.(*Object)
		if params == nil {
			if isNullish(parsed) {
				throw("TypeError: Cannot read properties of null")
			}
			params = newObject()
		}
	} else {
		params = newObject()
		match := regexExec(vmessShadowrocketRegex, lineWithoutFragment)
		if match == nil {
			throw("TypeError: not iterable")
		}
		content = base64Decode(match[1])
		for addon := range strings.SplitSeq(group(match, 2), "&") {
			key, valueRaw := splitPair(addon, "=")
			value := mustDecodeURIComponent(valueRaw)
			if !strings.Contains(value, ",") {
				params.Set(key, value)
			} else {
				params.Set(key, stringsToArray(strings.Split(value, ",")))
			}
		}
		body := regexExec(vmessShadowrocketBody, content)
		if body == nil {
			throw("TypeError: not iterable")
		}
		params.Set("scy", body[1])
		params.Set("id", body[2])
		params.Set("port", body[4])
		params.Set("add", body[3])
	}
	server := params.Get("add")
	port := parseInt(getIfPresent(params.Get("port"), nil))
	var skipCertVerify any
	if !isNullish(params.Get("verify_cert")) {
		skipCertVerify = !truthy(params.Get("verify_cert"))
	}
	proxy := obj(
		"name", coalesce(params.Get("ps"), params.Get("remarks"), params.Get("remark"), "VMess "+jsString(server)+":"+numberToString(port)),
		"type", "vmess",
		"server", server,
		"port", port,
		"cipher", normalizeVmessSecurity(params.Get("scy")),
		"uuid", params.Get("id"),
		"alterId", parseInt(getIfPresent(coalesce(params.Get("aid"), params.Get("alterId"), nil), float64(0))),
		"tls", includes(params.Get("tls"), "tls", true, float64(1), "1"),
		"skip-cert-verify", skipCertVerify,
	)
	if !truthy(proxy.Get("skip-cert-verify")) && !isNullish(params.Get("allowInsecure")) {
		proxy.Set("skip-cert-verify", testString(trueOrOneRegex, params.Get("allowInsecure")))
	}
	if truthy(proxy.Get("tls")) {
		if truthy(params.Get("sni")) {
			proxy.Set("sni", params.Get("sni"))
		} else if truthy(params.Get("peer")) {
			proxy.Set("sni", params.Get("peer"))
		}
	}
	httpupgrade := false
	net := params.Get("net")
	switch {
	case net == "ws" || params.Get("obfs") == "websocket":
		proxy.Set("network", "ws")
	case params.Get("obfs") == "http" || params.Get("type") == "http":
		proxy.Set("network", "http")
	case net == "http":
		proxy.Set("network", "h2")
	case includesString(net, "grpc", "kcp", "quic"):
		proxy.Set("network", net)
	case net == "httpupgrade" || proxy.Get("network") == "httpupgrade":
		proxy.Set("network", "ws")
		httpupgrade = true
	case net == "h2" || proxy.Get("network") == "h2":
		proxy.Set("network", "h2")
	}
	if truthy(proxy.Get("network")) {
		network := jsString(proxy.Get("network"))
		transportHost := coalesce(params.Get("host"), params.Get("obfsParam"), nil)
		if parsedObfs, err := jsonParse(jsString(transportHost)); err == nil {
			if o, isObject := parsedObfs.(*Object); isObject && truthy(o.Get("Host")) {
				transportHost = o.Get("Host")
			}
		}
		transportPath := params.Get("path")
		httpUpgradeEd := ""
		pathEarlyData := ""
		if network == "ws" && truthy(transportPath) {
			var ed string
			transportPath, ed = extractEarlyDataFromPath(transportPath)
			if httpupgrade {
				httpUpgradeEd = ed
			} else {
				pathEarlyData = ed
			}
		}
		if network == "ws" {
			transportPath = or(transportPath, "/")
		}
		switch network {
		case "http":
			if truthy(transportHost) {
				hostString, isString := transportHost.(string)
				if !isString {
					throw("TypeError: transportHost.split is not a function")
				}
				transportHost = strings.TrimSpace(strings.Split(hostString, ",")[0])
			}
			if truthy(transportPath) {
				if array, isArray := transportPath.([]any); isArray {
					transportPath = arrayAt(array, 0)
				}
			} else {
				transportPath = "/"
			}
		case "h2":
			if !truthy(transportPath) {
				transportPath = "/"
			}
		}
		if truthy(transportPath) || truthy(transportHost) || includesString(network, "kcp", "quic") {
			switch network {
			case "grpc":
				proxy.Set(network+"-opts", obj(
					"grpc-service-name", getIfNotBlank(transportPath, nil),
					"_grpc-type", getIfNotBlank(params.Get("type"), nil),
					"_grpc-authority", getIfNotBlank(params.Get("authority"), nil),
				))
			case "kcp", "quic":
				proxy.Set(network+"-opts", obj(
					"_"+network+"-type", getIfNotBlank(params.Get("type"), nil),
					"_"+network+"-host", getIfNotBlank(getIfNotBlank(transportHost, nil), nil),
					"_"+network+"-path", getIfNotBlank(transportPath, nil),
				))
			default:
				opts := obj("path", getIfNotBlank(transportPath, nil))
				normalizedTransportHost := getIfNotBlank(transportHost, nil)
				if network == "h2" {
					if h2Hosts := splitURIHostList(normalizedTransportHost); h2Hosts != nil {
						opts.Set("host", h2Hosts)
					}
				} else {
					opts.Set("headers", obj("Host", normalizedTransportHost))
				}
				if httpupgrade {
					opts.Set("v2ray-http-upgrade", true)
					if httpUpgradeEd == "" && isNumericEarlyData(params.Get("ed")) {
						httpUpgradeEd = jsString(params.Get("ed"))
					}
					if httpUpgradeEd != "" {
						opts.Set("v2ray-http-upgrade-fast-open", true)
						opts.Set("_v2ray-http-upgrade-ed", httpUpgradeEd)
					}
				} else if network == "ws" && pathEarlyData != "" {
					opts.Set("max-early-data", parseEarlyDataSize(pathEarlyData))
					opts.Set("early-data-header-name", "Sec-WebSocket-Protocol")
				}
				proxy.Set(network+"-opts", opts)
			}
		} else {
			proxy.Delete("network")
		}
	}
	proxy.Set("client-fingerprint", params.Get("fp"))
	if truthy(params.Get("alpn")) {
		alpn, isString := params.Get("alpn").(string)
		if !isString {
			throw("TypeError: params.alpn.split is not a function")
		}
		proxy.Set("alpn", stringsToArray(strings.Split(alpn, ",")))
	} else {
		proxy.Set("alpn", nil)
	}
	if isNotBlank(fragmentName) {
		proxy.Set("name", fragmentName)
	}
	return proxy
}

func arrayAt(array []any, index int) any {
	if index < len(array) {
		return array[index]
	}
	return nil
}

// URI_VLESS

var (
	vlessTestRegex        = regexp.MustCompile(`^vless://`)
	vlessRegex            = regexp.MustCompile(`^(.*?)@(.*?):(\d+)/?(\?(.*?))?(?:#(.*?))?$`)
	vlessShadowrocketLine = regexp.MustCompile(`^(.*?)(\?.*?$)`)
	vlessUUIDPrefixRegex  = regexp.MustCompile(`^.*?:`)
	trueOrOneRegexStrict  = regexp.MustCompile(`(?i)TRUE|1`)
)

func uriVLESSTest(line string) bool {
	return vlessTestRegex.MatchString(line)
}

func uriVLESSParse(line string, protocol string) *Object {
	line = splitFirst(line, protocol+"://")
	isShadowrocket := false
	parsed := regexExec(vlessRegex, line)
	if parsed == nil {
		match := regexExec(vlessShadowrocketLine, line)
		if match == nil {
			throw("TypeError: not iterable")
		}
		line = base64Decode(match[1]) + group(match, 2)
		parsed = regexExec(vlessRegex, line)
		if parsed == nil {
			throw("TypeError: not iterable")
		}
		isShadowrocket = true
	}
	uuid := group(parsed, 1)
	server := parsed[2]
	addons := group(parsed, 5)
	name := parsed[6]
	if isShadowrocket {
		uuid = vlessUUIDPrefixRegex.ReplaceAllString(uuid, "")
	}
	port := parseInt(parsed[3])
	uuid = mustDecodeURIComponent(uuid)
	if !isNullish(name) {
		name = mustDecodeURIComponent(name)
	}
	proxy := obj(
		"type", protocol,
		"name", name,
		"server", server,
		"port", port,
		"uuid", uuid,
		"udp", true,
	)
	params := newObject()
	for addon := range strings.SplitSeq(addons, "&") {
		if addon == "" {
			continue
		}
		key, value, _ := strings.Cut(addon, "=")
		params.Set(key, mustDecodeURIComponent(value))
	}
	protocolName := "VLESS"
	if protocol == "vmess" {
		protocolName = "VMess"
	}
	proxy.Set("name", coalesce(name, params.Get("remarks"), params.Get("remark"), protocolName+" "+jsString(server)+":"+numberToString(port)))
	proxy.Set("tls", and(params.Get("security"), params.Get("security") != "none"))
	if truthy(params.Get("pbk")) {
		params.Set("security", "reality")
	}
	if isShadowrocket && testString(trueOrOneRegexStrict, params.Get("tls")) {
		proxy.Set("tls", true)
		params.Set("security", coalesce(params.Get("security"), "reality"))
	}
	proxy.Set("sni", or(params.Get("sni"), params.Get("peer")))
	proxy.Set("flow", params.Get("flow"))
	if !truthy(proxy.Get("flow")) && isShadowrocket && truthy(params.Get("xtls")) {
		switch jsString(params.Get("xtls")) {
		case "1":
			proxy.Set("flow", "xtls-rprx-direct")
		case "2":
			proxy.Set("flow", "xtls-rprx-vision")
		}
	}
	proxy.Set("client-fingerprint", params.Get("fp"))
	if truthy(params.Get("alpn")) {
		proxy.Set("alpn", stringsToArray(strings.Split(params.Get("alpn").(string), ",")))
	} else {
		proxy.Set("alpn", nil)
	}
	proxy.Set("skip-cert-verify", testString(trueOrOneRegex, params.Get("allowInsecure")))
	proxy.Set("_echConfigList", getIfPresent(params.Get("ech"), nil))
	if echOpts := buildMihomoEchOptsFromXrayFields(params.Get("ech")); echOpts != nil {
		proxy.Set("ech-opts", echOpts)
	}
	proxy.Set("tls-fingerprint", getIfPresent(params.Get("pcs"), nil))
	var vcn any
	if vcnString, isString := params.Get("vcn").(string); isString {
		vcn = trimStrings(strings.Split(vcnString, ","), true)
	}
	proxy.Set("_vcn", vcn)
	if vcnArray, isArray := vcn.([]any); isArray {
		proxy.Set("name-cert-verify", arrayAt(vcnArray, 0))
	} else {
		proxy.Set("name-cert-verify", nil)
	}
	proxy.Set("_h2", testString(trueOrOneRegex, params.Get("h2")))
	switch strings.ToLower(strings.TrimSpace(jsString(or(params.Get("packetEncoding"), "")))) {
	case "none":
		proxy.Set("packet-encoding", "")
	case "packet":
		proxy.Set("packet-encoding", "packetaddr")
	default:
		proxy.Set("packet-encoding", "xudp")
	}
	if params.Get("security") == "reality" {
		opts := newObject()
		if truthy(params.Get("pbk")) {
			opts.Set("public-key", params.Get("pbk"))
			if includesString(params.Get("support-x25519mlkem768"), "1", "t", "T", "true", "TRUE", "True") {
				opts.Set("support-x25519mlkem768", true)
			}
		}
		if truthy(params.Get("sid")) {
			opts.Set("short-id", params.Get("sid"))
		}
		if truthy(params.Get("spx")) {
			opts.Set("_spider-x", params.Get("spx"))
		}
		if opts.Len() > 0 {
			proxy.Set("reality-opts", opts)
		}
	}
	httpupgrade := false
	network := jsString(or(params.Get("type"), "tcp"))
	if network == "tcp" && params.Get("headerType") == "http" {
		network = "http"
	} else if network == "http" {
		network = "h2"
	} else if network == "httpupgrade" {
		network = "ws"
		httpupgrade = true
	}
	if !truthy(params.Get("type")) && isShadowrocket && truthy(params.Get("obfs")) {
		network = jsString(params.Get("obfs"))
		if network == "none" {
			network = "tcp"
		}
	}
	if network == "websocket" {
		network = "ws"
	}
	proxy.Set("network", network)
	if network == "xhttp" {
		throw("Platform sing-box does not support network: xhttp")
	}
	if network != "" && network != "tcp" && network != "none" {
		opts := newObject()
		pathEarlyData := ""
		host := coalesce(params.Get("host"), params.Get("obfsParam"), nil)
		if truthy(host) {
			if truthy(params.Get("obfsParam")) {
				if headers, err := jsonParse(jsString(host)); err == nil {
					opts.Set("headers", headers)
				} else {
					opts.Set("headers", obj("Host", host))
				}
			} else {
				opts.Set("headers", obj("Host", host))
			}
			headers, _ := opts.Get("headers").(*Object)
			h2Host := coalesce(headers.Get("Host"), headers.Get("host"), nil)
			if network == "h2" && truthy(h2Host) {
				opts.Set("host", splitURIHostList(h2Host))
				headers.Delete("Host")
				headers.Delete("host")
				if headers.Len() == 0 {
					opts.Delete("headers")
				}
			}
		}
		if truthy(params.Get("serviceName")) {
			opts.Set(network+"-service-name", params.Get("serviceName"))
			if network == "grpc" && truthy(params.Get("authority")) {
				opts.Set("_grpc-authority", params.Get("authority"))
			}
		} else if isShadowrocket && truthy(params.Get("path")) {
			if !includesString(network, "ws", "http", "h2") {
				opts.Set(network+"-service-name", params.Get("path"))
				params.Delete("path")
			}
		}
		if truthy(params.Get("path")) {
			transportPath := params.Get("path")
			if network == "ws" {
				transportPath, pathEarlyData = extractEarlyDataFromPath(transportPath)
			}
			opts.Set("path", transportPath)
		} else if network == "h2" {
			opts.Set("path", "/")
		}
		if network == "http" && truthy(params.Get("method")) {
			opts.Set("method", params.Get("method"))
		}
		if network == "grpc" {
			opts.Set("_grpc-type", or(params.Get("mode"), "gun"))
		}
		if httpupgrade {
			opts.Set("v2ray-http-upgrade", true)
		}
		earlyDataRaw := or(pathEarlyData, params.Get("ed"))
		if truthy(earlyDataRaw) {
			maxEarlyData := parseEarlyDataSize(earlyDataRaw)
			if httpupgrade {
				opts.Set("v2ray-http-upgrade-fast-open", true)
				opts.Set("_v2ray-http-upgrade-ed", jsString(earlyDataRaw))
			} else if network == "ws" {
				opts.Set("max-early-data", maxEarlyData)
				opts.Set("early-data-header-name", or(params.Get("eh"), "Sec-WebSocket-Protocol"))
			}
		}
		if truthy(params.Get("eh")) && (network == "ws" || httpupgrade) {
			opts.Set("early-data-header-name", params.Get("eh"))
		}
		if opts.Len() > 0 {
			proxy.Set(network+"-opts", opts)
		}
		if network == "kcp" {
			if truthy(params.Get("seed")) {
				proxy.Set("seed", params.Get("seed"))
			}
			proxy.Set("headerType", or(params.Get("headerType"), "none"))
		}
		if truthy(params.Get("extra")) {
			proxy.Set("_extra", params.Get("extra"))
		}
		if truthy(params.Get("mode")) {
			proxy.Set("_mode", params.Get("mode"))
		}
	}
	if protocol == "vmess" {
		proxy.Set("cipher", normalizeVmessSecurity(params.Get("encryption")))
		proxy.Set("alterId", float64(0))
		proxy.Delete("flow")
	} else if truthy(params.Get("encryption")) {
		proxy.Set("encryption", params.Get("encryption"))
	}
	if truthy(params.Get("pqv")) {
		proxy.Set("_pqv", params.Get("pqv"))
	}
	if truthy(params.Get("fm")) {
		if finalmask, err := jsonParse(jsString(params.Get("fm"))); err == nil && isPlainObject(finalmask) {
			proxy.Set("_finalmask", finalmask)
		} else {
			proxy.Set("_finalmask", params.Get("fm"))
		}
	}
	return proxy
}

// URI_AnyTLS

var (
	anytlsTestRegex = regexp.MustCompile(`^anytls://`)
	anytlsRegex     = regexp.MustCompile(`^(.*?)@(.*?)(?::(\d+))?/?(?:\?(.*?))?(?:#(.*?))?$`)
	anytlsSplit     = regexp.MustCompile(`anytls://`)
	underscoreRegex = regexp.MustCompile(`_`)
)

func uriAnyTLSTest(line string) bool {
	return anytlsTestRegex.MatchString(line)
}

func uriAnyTLSParse(line string) *Object {
	parsed := uriVLESSParse(strings.Replace(line, "anytls", "vless", 1), "vless")
	parts := anytlsSplit.Split(line, -1)
	if len(parts) < 2 {
		throw("TypeError: Cannot read properties of undefined")
	}
	match := regexExec(anytlsRegex, parts[1])
	if match == nil {
		throw("TypeError: not iterable")
	}
	port := parseInt(match[3])
	if isNaN(port) {
		port = 443
	}
	password := mustDecodeURIComponent(match[1])
	name := match[5]
	if !isNullish(name) {
		name = mustDecodeURIComponent(name)
	}
	name = coalesce(name, "AnyTLS "+jsString(match[2])+":"+numberToString(port))
	proxy := parsed.Clone()
	proxy.Set("uuid", nil)
	proxy.Set("type", "anytls")
	proxy.Set("name", name)
	proxy.Set("server", match[2])
	proxy.Set("port", port)
	proxy.Set("password", password)
	for addon := range strings.SplitSeq(group(match, 4), "&") {
		if addon == "" {
			continue
		}
		key, rawValue := splitPair(addon, "=")
		key = underscoreRegex.ReplaceAllString(key, "-")
		value := mustDecodeURIComponent(rawValue)
		switch {
		case key == "alpn":
			if value != "" {
				proxy.Set(key, stringsToArray(strings.Split(value, ",")))
			} else {
				proxy.Set(key, nil)
			}
		case key == "insecure":
			proxy.Set("skip-cert-verify", trueOrOneRegex.MatchString(value))
		case key == "udp":
			proxy.Set(key, trueOrOneRegex.MatchString(value))
		case !proxy.Has(key):
			proxy.Set(key, value)
		}
	}
	if proxy.Get("network") == "tcp" && !truthy(proxy.Get("reality-opts")) {
		proxy.Delete("network")
		proxy.Delete("security")
	}
	return proxy
}

// URI_Hysteria2

var (
	hysteria2TestRegex  = regexp.MustCompile(`^(hysteria2|hy2)://`)
	hysteria2SplitRegex = regexp.MustCompile(`(hysteria2|hy2)://`)
	hysteria2Regex      = regexp.MustCompile(`^(.*?)@(.*?)(:((\d+(-\d+)?)([,;]\d+(-\d+)?)*))?/?(\?(.*?))?(?:#(.*?))?$`)
)

func uriHysteria2Test(line string) bool {
	return hysteria2TestRegex.MatchString(line)
}

func uriHysteria2Parse(line string) *Object {
	rest, isString := splitCapture(hysteria2SplitRegex, line).(string)
	if !isString {
		throw("TypeError: Cannot read properties of undefined")
	}
	match := regexExec(hysteria2Regex, rest)
	if match == nil {
		throw("TypeError: not iterable")
	}
	server := match[2]
	portString := match[4]
	addons := group(match, 10)
	name := match[11]
	var ports any
	var port float64
	if testString(digitsOnlyRegex, portString) {
		port = parseInt(portString)
		if isNaN(port) {
			port = 443
		}
	} else if truthy(portString) {
		ports = portString
		port = getRandomPort(portString.(string))
	} else {
		port = 443
	}
	password := mustDecodeURIComponent(match[1])
	if !isNullish(name) {
		name = mustDecodeURIComponent(name)
	}
	name = coalesce(name, "Hysteria2 "+jsString(server)+":"+numberToString(port))
	proxy := obj(
		"type", "hysteria2",
		"name", name,
		"server", server,
		"port", port,
		"ports", ports,
		"password", password,
	)
	params := parseQuery(addons)
	proxy.Set("sni", params.Get("sni"))
	if !truthy(proxy.Get("sni")) && truthy(params.Get("peer")) {
		proxy.Set("sni", params.Get("peer"))
	}
	if truthy(params.Get("obfs")) && params.Get("obfs") != "none" {
		proxy.Set("obfs", params.Get("obfs"))
	}
	if truthy(params.Get("mport")) {
		proxy.Set("ports", params.Get("mport"))
	}
	proxy.Set("obfs-password", params.Get("obfs-password"))
	proxy.Set("skip-cert-verify", testString(trueOrOneRegex, params.Get("insecure")))
	proxy.Set("tfo", testString(trueOrOneRegex, params.Get("fastopen")))
	proxy.Set("tls-fingerprint", params.Get("pinSHA256"))
	hopInterval := or(params.Get("hop-interval"), params.Get("hop_interval"))
	if !isNullish(hopInterval) {
		proxy.Set("hop-interval", hopInterval)
	}
	if keepalive := params.Get("keepalive"); testString(digitsOnlyRegex, keepalive) {
		proxy.Set("keepalive", parseInt(keepalive))
	}
	if truthy(params.Get("upmbps")) {
		proxy.Set("up", params.Get("upmbps"))
	}
	if truthy(params.Get("downmbps")) {
		proxy.Set("down", params.Get("downmbps"))
	}
	if echOpts := buildMihomoEchOptsFromXrayFields(params.Get("ech")); echOpts != nil {
		proxy.Set("ech-opts", echOpts)
	}
	return proxy
}

// URI_Hysteria

var (
	hysteriaTestRegex  = regexp.MustCompile(`^(hysteria|hy)://`)
	hysteriaSplitRegex = regexp.MustCompile(`(hysteria|hy)://`)
	hysteriaRegex      = regexp.MustCompile(`^(.*?)(:(\d+))?/?(\?(.*?))?(?:#(.*?))?$`)
)

func uriHysteriaTest(line string) bool {
	return hysteriaTestRegex.MatchString(line)
}

func uriHysteriaParse(line string) *Object {
	rest, isString := splitCapture(hysteriaSplitRegex, line).(string)
	if !isString {
		throw("TypeError: Cannot read properties of undefined")
	}
	match := regexExec(hysteriaRegex, rest)
	server := match[1]
	port := parseInt(match[3])
	if isNaN(port) {
		port = 443
	}
	name := match[6]
	if !isNullish(name) {
		name = mustDecodeURIComponent(name)
	}
	name = coalesce(name, "Hysteria "+jsString(server)+":"+numberToString(port))
	proxy := obj(
		"type", "hysteria",
		"name", name,
		"server", server,
		"port", port,
	)
	params := newObject()
	for addon := range strings.SplitSeq(group(match, 5), "&") {
		if addon == "" {
			continue
		}
		key, rawValue := splitPair(addon, "=")
		key = strings.Replace(key, "_", "-", 1)
		value := mustDecodeURIComponent(rawValue)
		switch {
		case key == "alpn":
			if value != "" {
				proxy.Set(key, stringsToArray(strings.Split(value, ",")))
			} else {
				proxy.Set(key, nil)
			}
		case key == "insecure":
			proxy.Set("skip-cert-verify", trueOrOneRegex.MatchString(value))
		case key == "auth":
			proxy.Set("auth-str", value)
		case key == "mport":
			proxy.Set("ports", value)
		case key == "obfsParam":
			proxy.Set("obfs", value)
		case key == "upmbps":
			proxy.Set("up", value)
		case key == "downmbps":
			proxy.Set("down", value)
		case key == "obfs":
			proxy.Set("_obfs", value)
		case key == "fast-open" || key == "peer":
			params.Set(key, value)
		case !proxy.Has(key):
			proxy.Set(key, value)
		}
	}
	if !truthy(proxy.Get("sni")) && truthy(params.Get("peer")) {
		proxy.Set("sni", params.Get("peer"))
	}
	if !truthy(proxy.Get("fast-open")) && truthy(params.Get("fastopen")) {
		proxy.Set("fast-open", true)
	}
	if !truthy(proxy.Get("protocol")) {
		proxy.Set("protocol", "udp")
	}
	return proxy
}

// URI_TUIC

var (
	tuicTestRegex = regexp.MustCompile(`^tuic://`)
	tuicSplit     = regexp.MustCompile(`tuic://`)
	tuicRegex     = regexp.MustCompile(`^(.*?)@(.*?)(?::(\d+))?/?(?:\?(.*?))?(?:#(.*?))?$`)
)

func uriTUICTest(line string) bool {
	return tuicTestRegex.MatchString(line)
}

func uriTUICParse(line string) *Object {
	parts := tuicSplit.Split(line, -1)
	if len(parts) < 2 {
		throw("TypeError: Cannot read properties of undefined")
	}
	match := regexExec(tuicRegex, parts[1])
	if match == nil {
		throw("TypeError: not iterable")
	}
	auth := mustDecodeURIComponent(match[1])
	authParts := strings.Split(auth, ":")
	uuid := authParts[0]
	password := strings.Join(authParts[1:], ":")
	port := parseInt(match[3])
	if isNaN(port) {
		port = 443
	}
	password = mustDecodeURIComponent(password)
	name := match[5]
	if !isNullish(name) {
		name = mustDecodeURIComponent(name)
	}
	name = coalesce(name, "TUIC "+jsString(match[2])+":"+numberToString(port))
	proxy := obj(
		"type", "tuic",
		"name", name,
		"server", match[2],
		"port", port,
		"password", password,
		"uuid", uuid,
	)
	for addon := range strings.SplitSeq(group(match, 4), "&") {
		if addon == "" {
			continue
		}
		key, rawValue := splitPair(addon, "=")
		key = underscoreRegex.ReplaceAllString(key, "-")
		value := mustDecodeURIComponent(rawValue)
		switch {
		case key == "alpn":
			if value != "" {
				proxy.Set(key, stringsToArray(strings.Split(value, ",")))
			} else {
				proxy.Set(key, nil)
			}
		case key == "allow-insecure" || key == "insecure":
			proxy.Set("skip-cert-verify", trueOrOneRegex.MatchString(value))
		case key == "fast-open":
			proxy.Set("tfo", true)
		case key == "disable-sni" || key == "reduce-rtt":
			proxy.Set(key, trueOrOneRegex.MatchString(value))
		case key == "congestion-control":
			proxy.Set("congestion-controller", value)
			proxy.Delete(key)
		case !proxy.Has(key):
			proxy.Set(key, value)
		}
	}
	return proxy
}

// URI_Trojan and parsers/peggy/trojan-uri.js

var (
	trojanTestRegex   = regexp.MustCompile(`^trojan://`)
	trojanNameRegex   = regexp.MustCompile(`#(.+)`)
	trojanPortRegex   = regexp.MustCompile(`^(trojan://.*?@.*?)(:(\d+))?/?(\?.*?)?$`)
	trojanURLRegex    = regexp.MustCompile(`^trojan://([^@]+)@(\[[^\]]+\]|[^/?#]+):(\d+)(?:/)?(?:\?([^#]*))?(?:#(.*))?$`)
	bracketTrimRegex  = regexp.MustCompile(`^\[|\]$`)
	trojanToBoolRegex = regexp.MustCompile(`(?i)(TRUE)|1`)
)

func uriTrojanTest(line string) bool {
	return trojanTestRegex.MatchString(line)
}

func uriTrojanParse(line string) *Object {
	newLine := line
	var name any
	if indexes := trojanNameRegex.FindStringSubmatchIndex(line); indexes != nil {
		newLine = line[:indexes[0]]
		name = line[indexes[2]:indexes[3]]
	}
	matched := regexExec(trojanPortRegex, newLine)
	if matched == nil || matched[2] == nil {
		if matched == nil {
			throw("TypeError: Cannot read properties of null")
		}
		prefix := matched[1].(string)
		newLine = jsReplace(newLine, prefix, prefix+":443")
	}
	proxy := parseTrojan(newLine)
	if isNotBlank(name) {
		if decoded, err := decodeURIComponent(name); err == nil {
			proxy.Set("name", decoded)
		}
	}
	return proxy
}

// jsReplace is String.prototype.replace with a string pattern.
func jsReplace(s string, pattern string, replacement string) string {
	before, after, ok := strings.Cut(s, pattern)
	if !ok {
		return s
	}
	var expanded strings.Builder
	for i := 0; i < len(replacement); i++ {
		if replacement[i] != '$' || i+1 >= len(replacement) {
			expanded.WriteByte(replacement[i])
			continue
		}
		switch replacement[i+1] {
		case '$':
			expanded.WriteByte('$')
		case '&':
			expanded.WriteString(pattern)
		case '`':
			expanded.WriteString(before)
		case '\'':
			expanded.WriteString(after)
		default:
			expanded.WriteByte('$')
			continue
		}
		i++
	}
	return before + expanded.String() + after
}

func trojanDecode(value string) string {
	decoded, err := decodeURIComponent(value)
	if err != nil {
		return value
	}
	return decoded
}

func trojanToBool(value any) any {
	if isNullish(value) {
		return nil
	}
	return testString(trojanToBoolRegex, value)
}

func parseTrojan(url string) *Object {
	proxy := newObject()
	match := regexExec(trojanURLRegex, url)
	if match == nil {
		throw("Invalid trojan url")
	}
	password, server, port, query, name := match[1].(string), match[2].(string), match[3].(string), match[4], match[5]
	normalizedServer := bracketTrimRegex.ReplaceAllString(server, "")
	if (strings.HasPrefix(server, "[") || strings.Contains(server, ":")) && !isIPv6(normalizedServer) {
		throw("Invalid server: " + server)
	}
	proxy.Set("type", "trojan")
	proxy.Set("password", trojanDecode(password))
	proxy.Set("server", server)
	portValue := jsNumber(port)
	proxy.Set("port", portValue)
	if !isSafeInteger(portValue) || portValue < 1 || portValue > 65535 {
		throw("Invalid port: " + port)
	}
	if truthy(name) {
		proxy.Set("name", trojanDecode(name.(string)))
	} else {
		proxy.Set("name", server+":"+port)
	}
	params := newObject()
	if truthy(query) {
		for item := range strings.SplitSeq(query.(string), "&") {
			before, after, ok := strings.Cut(item, "=")
			if !ok {
				params.Set(item, true)
			} else {
				params.Set(before, trojanDecode(after))
			}
		}
	}
	proxy.Set("skip-cert-verify", trojanToBool(params.Get("allowInsecure")))
	proxy.Set("sni", or(params.Get("sni"), params.Get("peer")))
	proxy.Set("client-fingerprint", params.Get("fp"))
	proxy.Set("tls-fingerprint", params.Get("pcs"))
	var vcn any
	if !isNullish(params.Get("vcn")) {
		vcnString, isString := params.Get("vcn").(string)
		if !isString {
			throw("TypeError: params.vcn?.split is not a function")
		}
		vcn = trimStrings(strings.Split(vcnString, ","), true)
	}
	proxy.Set("_vcn", vcn)
	if vcnArray, isArray := vcn.([]any); isArray {
		proxy.Set("name-cert-verify", arrayAt(vcnArray, 0))
	} else {
		proxy.Set("name-cert-verify", nil)
	}
	if truthy(params.Get("alpn")) {
		alpn, isString := params.Get("alpn").(string)
		if !isString {
			throw("TypeError: params.alpn.split is not a function")
		}
		proxy.Set("alpn", stringsToArray(strings.Split(alpn, ",")))
	}
	if truthy(trojanToBool(params.Get("ws"))) {
		proxy.Set("network", "ws")
		setPath(proxy, "ws-opts.path", params.Get("wspath"))
	}
	if truthy(params.Get("type")) {
		httpupgrade := false
		httpUpgradeEd := ""
		pathEarlyData := ""
		network := jsString(params.Get("type"))
		if network == "httpupgrade" {
			network = "ws"
			httpupgrade = true
		}
		proxy.Set("network", network)
		if network == "grpc" {
			proxy.Set("grpc-opts", obj(
				"grpc-service-name", params.Get("serviceName"),
				"_grpc-type", params.Get("mode"),
				"_grpc-authority", params.Get("authority"),
			))
		} else {
			if truthy(params.Get("path")) {
				path := params.Get("path")
				if network == "ws" {
					ed := getPathQueryParam(path, "ed")
					if parseSafeIntegerValue(ed) != nil {
						path, _ = extractPathQueryParam(path, "ed")
						if httpupgrade {
							httpUpgradeEd = ed
						} else {
							pathEarlyData = ed
						}
					}
				}
				setPath(proxy, network+"-opts.path", path)
			}
			if truthy(params.Get("host")) {
				setPath(proxy, network+"-opts.headers.Host", params.Get("host"))
			}
			if httpupgrade {
				if httpUpgradeEd == "" && parseSafeIntegerValue(params.Get("ed")) != nil {
					httpUpgradeEd = jsString(params.Get("ed"))
				}
				setPath(proxy, "ws-opts.v2ray-http-upgrade", true)
				if httpUpgradeEd != "" {
					setPath(proxy, "ws-opts.v2ray-http-upgrade-fast-open", true)
					setPath(proxy, "ws-opts._v2ray-http-upgrade-ed", httpUpgradeEd)
				}
			} else if network == "ws" && pathEarlyData != "" {
				setPath(proxy, "ws-opts.max-early-data", parseSafeIntegerValue(pathEarlyData))
				setPath(proxy, "ws-opts.early-data-header-name", "Sec-WebSocket-Protocol")
			}
		}
		if params.Get("security") == "reality" {
			opts := newObject()
			if truthy(params.Get("pbk")) {
				opts.Set("public-key", params.Get("pbk"))
			}
			if truthy(params.Get("sid")) {
				opts.Set("short-id", params.Get("sid"))
			}
			if truthy(params.Get("spx")) {
				opts.Set("_spider-x", params.Get("spx"))
			}
			if truthy(params.Get("mode")) {
				proxy.Set("_mode", params.Get("mode"))
			}
			if truthy(params.Get("extra")) {
				proxy.Set("_extra", params.Get("extra"))
			}
			if opts.Len() > 0 {
				setPath(proxy, "reality-opts", opts)
			}
		}
	}
	proxy.Set("udp", trojanToBool(params.Get("udp")))
	proxy.Set("tfo", trojanToBool(params.Get("tfo")))
	return proxy
}

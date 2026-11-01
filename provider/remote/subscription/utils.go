// This file is part of a Go port of Sub-Store (https://github.com/sub-store-org/Sub-Store),
// licensed under the GNU Affero General Public License v3.0. See NOTICE in this directory.

package subscription

import (
	"regexp"
	"slices"
	"strings"
)

// utils/index.js

var (
	ipv4OctetRegex  = regexp.MustCompile(`^(25[0-5]|(2[0-4]|1\d|[1-9]|)\d)$`)
	ipv6Regex       = regexp.MustCompile(`^(([0-9a-fA-F]{1,4}:){7,7}[0-9a-fA-F]{1,4}|([0-9a-fA-F]{1,4}:){1,7}:|([0-9a-fA-F]{1,4}:){1,6}:[0-9a-fA-F]{1,4}|([0-9a-fA-F]{1,4}:){1,5}(:[0-9a-fA-F]{1,4}){1,2}|([0-9a-fA-F]{1,4}:){1,4}(:[0-9a-fA-F]{1,4}){1,3}|([0-9a-fA-F]{1,4}:){1,3}(:[0-9a-fA-F]{1,4}){1,4}|([0-9a-fA-F]{1,4}:){1,2}(:[0-9a-fA-F]{1,4}){1,5}|[0-9a-fA-F]{1,4}:((:[0-9a-fA-F]{1,4}){1,6})|:((:[0-9a-fA-F]{1,4}){1,7}|:)|fe80:(:[0-9a-fA-F]{0,4}){0,4}%[0-9a-zA-Z]{1,}|::(ffff(:0{1,4}){0,1}:){0,1}((25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])\.){3,3}(25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])|([0-9a-fA-F]{1,4}:){1,4}:((25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])\.){3,3}(25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9]))$`)
	portNumberRegex = regexp.MustCompile(`^((6553[0-5])|(655[0-2][0-9])|(65[0-4][0-9]{2})|(6[0-4][0-9]{3})|([1-5][0-9]{4})|([0-5]{0,5})|([0-9]{1,4}))$`)
	digitsRegex     = regexp.MustCompile(`^\d+$`)
)

func isIPv4(value any) bool {
	parts := strings.Split(jsString(value), ".")
	if len(parts) != 4 {
		return false
	}
	for _, part := range parts {
		if !ipv4OctetRegex.MatchString(part) {
			return false
		}
	}
	return true
}

func isIPv6(value any) bool {
	return testString(ipv6Regex, value)
}

func isIP(value any) bool {
	return isIPv4(value) || isIPv6(value)
}

func isValidPortNumber(value any) bool {
	return testString(portNumberRegex, value)
}

// getRandomPort picks a port from a port hopping list. Sub-Store picks one at
// random; the port keeps the first port so that the converted document only
// changes when the subscription does.
func getRandomPort(portString string) float64 {
	parts := regexp.MustCompile(`,|/`).Split(portString, -1)
	part := parts[0]
	if strings.Contains(part, "-") {
		bounds := split(part, "-")
		return jsCeil(jsNumber(bounds[0]))
	}
	return jsNumber(part)
}

func jsCeil(v float64) float64 {
	if isNaN(v) {
		return v
	}
	if v == float64(int64(v)) {
		return v
	}
	if v > 0 {
		return float64(int64(v) + 1)
	}
	return float64(int64(v))
}

// core/proxy-utils/transport-path.js

func decodeQueryComponent(value string) string {
	decoded, err := decodeURIComponent(strings.ReplaceAll(value, "+", "%20"))
	if err != nil {
		return value
	}
	return decoded
}

func splitQueryPart(part string) (string, string) {
	key, value, found := strings.Cut(part, "=")
	if !found {
		return decodeQueryComponent(part), ""
	}
	return decodeQueryComponent(key), decodeQueryComponent(value)
}

// parseSafeIntegerValue returns the integer, or nil.
func parseSafeIntegerValue(value any) any {
	if !digitsRegex.MatchString(jsString(value)) {
		return nil
	}
	parsed := parseInt(value)
	if !isSafeInteger(parsed) {
		return nil
	}
	return parsed
}

func extractPathQueryParam(rawPath any, paramName string) (string, string) {
	path := ""
	if !isNullish(rawPath) {
		path = jsString(rawPath)
	}
	basePath, query, found := strings.Cut(path, "?")
	if !found {
		return path, ""
	}
	var keptParts []string
	value := ""
	for part := range strings.SplitSeq(query, "&") {
		if part == "" {
			continue
		}
		key, partValue := splitQueryPart(part)
		if key == paramName {
			if value == "" && partValue != "" {
				value = partValue
			}
			continue
		}
		keptParts = append(keptParts, part)
	}
	if len(keptParts) > 0 {
		return basePath + "?" + strings.Join(keptParts, "&"), value
	}
	return basePath, value
}

func getPathQueryParam(rawPath any, paramName string) string {
	path := ""
	if !isNullish(rawPath) {
		path = jsString(rawPath)
	}
	_, query, found := strings.Cut(path, "?")
	if !found {
		return ""
	}
	for part := range strings.SplitSeq(query, "&") {
		if part == "" {
			continue
		}
		key, value := splitQueryPart(part)
		if key == paramName && value != "" {
			return value
		}
	}
	return ""
}

// getSafeIntegerPathQueryParam returns the raw value and the parsed integer, or
// "" and nil.
func getSafeIntegerPathQueryParam(rawPath any, paramName string) (string, any) {
	value := getPathQueryParam(rawPath, paramName)
	parsed := parseSafeIntegerValue(value)
	if parsed == nil {
		return "", nil
	}
	return value, parsed
}

func normalizeWebSocketEarlyDataPath(wsOpts *Object) {
	if wsOpts == nil {
		return
	}
	networkPath := wsOpts.Get("path")
	ed, maxEarlyData := getSafeIntegerPathQueryParam(networkPath, "ed")
	if truthy(wsOpts.Get("v2ray-http-upgrade")) {
		if ed != "" {
			path, _ := extractPathQueryParam(networkPath, "ed")
			wsOpts.Set("path", path)
			wsOpts.Set("v2ray-http-upgrade-fast-open", true)
			existing := wsOpts.Get("_v2ray-http-upgrade-ed")
			if isNullish(existing) || jsString(existing) == "" {
				wsOpts.Set("_v2ray-http-upgrade-ed", ed)
			}
		}
		wsOpts.Delete("early-data-header-name")
		wsOpts.Delete("max-early-data")
		return
	}
	if ed == "" {
		return
	}
	path, _ := extractPathQueryParam(networkPath, "ed")
	wsOpts.Set("path", path)
	if isNullish(wsOpts.Get("early-data-header-name")) {
		wsOpts.Set("early-data-header-name", "Sec-WebSocket-Protocol")
	}
	if isNullish(wsOpts.Get("max-early-data")) {
		wsOpts.Set("max-early-data", maxEarlyData)
	}
}

// core/proxy-utils/vmess-security.js

var vmessSecurityCommonValues = []string{"auto", "none", "zero", "aes-128-gcm", "chacha20-poly1305"}

func canonicalizeVmessSecurity(security string) string {
	if security == "chacha20-ietf-poly1305" {
		return "chacha20-poly1305"
	}
	return security
}

// normalizeVmessSecurity is normalizeVmessSecurity with the default arguments.
func normalizeVmessSecurity(security any) string {
	normalized := ""
	if !isNullish(security) {
		normalized = strings.ToLower(strings.TrimSpace(jsString(security)))
	}
	if normalized == "" {
		return "auto"
	}
	if slices.Contains(vmessSecurityCommonValues, normalized) {
		return canonicalizeVmessSecurity(normalized)
	}
	canonical := canonicalizeVmessSecurity(normalized)
	for _, value := range vmessSecurityCommonValues {
		if canonicalizeVmessSecurity(value) == canonical {
			return canonical
		}
	}
	return "auto"
}

// core/proxy-utils/ech-utils.js

const echDNSField = "_dns"

func buildMihomoEchOptsFromXrayFields(echConfigList any) *Object {
	if !isNotBlank(echConfigList) {
		return nil
	}
	list := echConfigList.(string)
	echOpts := obj("enable", true)
	if !strings.Contains(list, "://") {
		echOpts.Set("config", list)
		return echOpts
	}
	parts := strings.Split(list, "+")
	switch {
	case len(parts) == 1 && isNotBlank(parts[0]):
		echOpts.Set(echDNSField, parts[0])
	case len(parts) == 2 && isNotBlank(parts[0]) && isNotBlank(parts[1]):
		echOpts.Set(echDNSField, parts[1])
		echOpts.Set("query-server-name", parts[0])
	default:
		return nil
	}
	return echOpts
}

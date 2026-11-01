// This file is part of a Go port of Sub-Store (https://github.com/sub-store-org/Sub-Store),
// licensed under the GNU Affero General Public License v3.0. See NOTICE in this directory.

package subscription

import (
	"regexp"
	"strings"
)

// core/proxy-utils/preprocessors/index.js

type preprocessor struct {
	name  string
	test  func(raw string) bool
	parse func(raw string) string
}

var preprocessors = []preprocessor{
	{"HTML", htmlTest, func(string) string { return "" }},
	{"Clash Pre-processor", clashTest, clashParse},
	{"Base64 Pre-processor", base64EncodedTest, base64EncodedParse},
	{"SSD Pre-processor", ssdTest, ssdParse},
	{"Full Config Preprocessor", fullConfigTest, fullConfigParse},
	{"Fallback Base64 Pre-processor", base64IsValid2, base64EncodedParse},
}

func preprocess(raw string) string {
	for _, processor := range preprocessors {
		result, ok := tryPreprocess(processor, raw)
		if ok {
			return result
		}
	}
	return raw
}

func tryPreprocess(processor preprocessor, raw string) (result string, ok bool) {
	defer func() {
		if recover() != nil {
			result, ok = "", false
		}
	}()
	if !processor.test(raw) {
		return "", false
	}
	return processor.parse(raw), true
}

var (
	shortIDRegex        = regexp.MustCompile(`short-id:([ \t]*[^#\n,}]*)`)
	quotedValueRegex    = regexp.MustCompile(`^(?:'.*'|".*")$`)
	htmlRegex           = regexp.MustCompile(`^<!DOCTYPE html>`)
	schemeLineRegex     = regexp.MustCompile(`(?im)^\w+://\w+`)
	decodedLineRegex    = regexp.MustCompile(`(?m)^\w+(://|\s*?=\s*?)\w+`)
	fullConfigRegex     = regexp.MustCompile(`(?m)^(\[server_local\]|\[Proxy\])`)
	fullConfigSection   = regexp.MustCompile(`(?im)^\[server_local|Proxy\]([\s\S]+?)^\[.+?\](\r?\n|$)`)
	base64EncodedKeys   = []string{"dm1lc3M", "c3NyOi8v", "c29ja3M6Ly", "dHJvamFu", "c3M6Ly", "c3NkOi8v", "c2hhZG93", "aHR0c", "dmxlc3M=", "aHlzdGVyaWEy", "aHkyOi8v", "d2lyZWd1YXJkOi8v", "d2c6Ly8=", "dHVpYzovLw=="}
	proxiesKeywordRegex = regexp.MustCompile(`proxies`)
)

// normalizeClashYaml quotes REALITY short-id values so that YAML does not read
// them as numbers.
func normalizeClashYaml(raw string) string {
	if !strings.Contains(raw, "proxies:") || !strings.Contains(raw, "short-id:") {
		return raw
	}
	content, err := yamlLoad(raw)
	if err != nil {
		return raw
	}
	document, isObject := content.(*Object)
	if !isObject {
		return raw
	}
	proxies, isArray := document.Get("proxies").([]any)
	if !isArray || len(proxies) == 0 {
		return raw
	}
	return shortIDRegex.ReplaceAllStringFunc(raw, func(matched string) string {
		value := shortIDRegex.FindStringSubmatch(matched)[1]
		afterTrim := strings.TrimSpace(value)
		switch {
		case afterTrim == "":
			return `short-id: ""`
		case quotedValueRegex.MatchString(afterTrim) && afterTrim[0] == afterTrim[len(afterTrim)-1]:
			return "short-id: " + afterTrim
		case afterTrim == "null":
			return "short-id: " + afterTrim
		default:
			return `short-id: "` + afterTrim + `"`
		}
	})
}

func htmlTest(raw string) bool {
	return htmlRegex.MatchString(raw)
}

func clashTest(raw string) bool {
	if !proxiesKeywordRegex.MatchString(raw) {
		return false
	}
	content, err := yamlLoad(raw)
	if err != nil {
		panic(err)
	}
	document, isObject := content.(*Object)
	if !isObject {
		panic(jsError("TypeError: not an object"))
	}
	_, proxiesIsArray := document.Get("proxies").([]any)
	_, groupsIsArray := document.Get("proxy-groups").([]any)
	return proxiesIsArray || groupsIsArray
}

func clashParse(raw string) string {
	content, err := yamlLoad(normalizeClashYaml(raw))
	if err != nil {
		panic(err)
	}
	proxies, _ := content.(*Object).Get("proxies").([]any)
	var builder strings.Builder
	for _, proxy := range proxies {
		line, err := marshalValue(proxy)
		if err != nil {
			panic(err)
		}
		builder.Write(line)
		builder.WriteByte('\n')
	}
	return builder.String()
}

func base64EncodedTest(raw string) bool {
	if !base64IsValid(raw) || schemeLineRegex.MatchString(raw) {
		return false
	}
	for _, key := range base64EncodedKeys {
		if strings.Contains(raw, key) {
			return true
		}
	}
	return false
}

func base64EncodedParse(raw string) string {
	decoded := base64Decode(raw)
	if !decodedLineRegex.MatchString(decoded) {
		return raw
	}
	return decoded
}

func base64IsValid2(raw string) bool {
	return base64IsValid(raw)
}

func ssdTest(raw string) bool {
	return strings.HasPrefix(raw, "ssd://")
}

func ssdParse(raw string) string {
	content, err := jsonParse(base64Decode(at(split(raw, "ssd://"), 1)))
	if err != nil {
		panic(err)
	}
	info, isObject := content.(*Object)
	if !isObject {
		panic(jsError("TypeError: invalid SSD document"))
	}
	port := info.Get("port")
	method := info.Get("encryption")
	password := info.Get("password")
	servers, isArray := info.Get("servers").([]any)
	if !isArray {
		panic(jsError("TypeError: invalid SSD servers"))
	}
	output := make([]string, len(servers))
	for i, item := range servers {
		server, isObject := item.(*Object)
		if !isObject {
			panic(jsError("TypeError: invalid SSD server"))
		}
		if truthy(server.Get("encryption")) {
			method = server.Get("encryption")
		}
		if truthy(server.Get("password")) {
			password = server.Get("password")
		}
		userinfo := base64Encode(jsString(method) + ":" + jsString(password))
		hostname := server.Get("server")
		if truthy(server.Get("port")) {
			port = server.Get("port")
		}
		var tag any = float64(i)
		if truthy(server.Get("remarks")) {
			tag = server.Get("remarks")
		}
		plugin := ""
		if truthy(server.Get("plugin_options")) {
			plugin = "/?plugin=" + encodeURIComponent(jsString(server.Get("plugin"))+";"+jsString(server.Get("plugin_options")))
		}
		output[i] = "ss://" + userinfo + "@" + jsString(hostname) + ":" + jsString(port) + plugin + "#" + jsString(tag)
	}
	return strings.Join(output, "\n")
}

func fullConfigTest(raw string) bool {
	return fullConfigRegex.MatchString(raw)
}

func fullConfigParse(raw string) string {
	match := fullConfigSection.FindStringSubmatch(raw)
	if match != nil && match[1] != "" {
		return match[1]
	}
	return raw
}

// This file is part of a Go port of Sub-Store (https://github.com/sub-store-org/Sub-Store),
// licensed under the GNU Affero General Public License v3.0. See NOTICE in this directory.

package subscription

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"math"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"
)

// The port keeps the JavaScript data model of Sub-Store so that the code can be
// compared with upstream function by function:
//
//   - Go nil is JavaScript undefined, null is jsNull.
//   - Numbers are float64, NaN included.
//   - Objects keep their key insertion order and may hold keys whose value is
//     undefined, since Sub-Store iterates over such keys.

type nullType struct{}

var jsNull = nullType{}

// Object is an insertion-ordered JavaScript object.
type Object struct {
	keys   []string
	values map[string]any
}

func newObject() *Object {
	return &Object{values: make(map[string]any)}
}

// obj builds an object from alternating keys and values, like an object literal.
// Keys with undefined values are kept, as in JavaScript.
func obj(entries ...any) *Object {
	o := newObject()
	for i := 0; i+1 < len(entries); i += 2 {
		o.Set(entries[i].(string), entries[i+1])
	}
	return o
}

func (o *Object) Get(key string) any {
	if o == nil {
		return nil
	}
	return o.values[key]
}

func (o *Object) Has(key string) bool {
	if o == nil {
		return false
	}
	_, loaded := o.values[key]
	return loaded
}

func (o *Object) Set(key string, value any) {
	if _, loaded := o.values[key]; !loaded {
		o.keys = append(o.keys, key)
	}
	o.values[key] = value
}

func (o *Object) Delete(key string) {
	if o == nil {
		return
	}
	if _, loaded := o.values[key]; !loaded {
		return
	}
	delete(o.values, key)
	for i, k := range o.keys {
		if k == key {
			o.keys = append(o.keys[:i:i], o.keys[i+1:]...)
			break
		}
	}
}

// Keys returns the keys in JavaScript property order: integer-like keys first
// in ascending order, then the other keys in insertion order.
func (o *Object) Keys() []string {
	if o == nil {
		return nil
	}
	var indexKeys []string
	var otherKeys []string
	for _, key := range o.keys {
		if isArrayIndex(key) {
			indexKeys = append(indexKeys, key)
		} else {
			otherKeys = append(otherKeys, key)
		}
	}
	if len(indexKeys) == 0 {
		return append([]string(nil), otherKeys...)
	}
	sortIndexKeys(indexKeys)
	return append(indexKeys, otherKeys...)
}

func (o *Object) Len() int {
	if o == nil {
		return 0
	}
	return len(o.keys)
}

// Obj returns the object stored at key, or nil.
func (o *Object) Obj(key string) *Object {
	value, _ := o.Get(key).(*Object)
	return value
}

func (o *Object) Clone() *Object {
	if o == nil {
		return nil
	}
	clone := newObject()
	for _, key := range o.keys {
		clone.Set(key, o.values[key])
	}
	return clone
}

func isArrayIndex(key string) bool {
	if key == "" || len(key) > 10 || (len(key) > 1 && key[0] == '0') {
		return false
	}
	for i := 0; i < len(key); i++ {
		if key[i] < '0' || key[i] > '9' {
			return false
		}
	}
	value, err := strconv.ParseUint(key, 10, 64)
	return err == nil && value < 1<<32-1
}

func sortIndexKeys(keys []string) {
	for i := 1; i < len(keys); i++ {
		for j := i; j > 0; j-- {
			a, _ := strconv.ParseUint(keys[j-1], 10, 64)
			b, _ := strconv.ParseUint(keys[j], 10, 64)
			if a <= b {
				break
			}
			keys[j-1], keys[j] = keys[j], keys[j-1]
		}
	}
}

func (o *Object) MarshalJSON() ([]byte, error) {
	var buffer bytes.Buffer
	buffer.WriteByte('{')
	first := true
	for _, key := range o.Keys() {
		value := o.values[key]
		if value == nil {
			continue
		}
		if !first {
			buffer.WriteByte(',')
		}
		first = false
		keyContent, err := json.Marshal(key)
		if err != nil {
			return nil, err
		}
		buffer.Write(keyContent)
		buffer.WriteByte(':')
		valueContent, err := marshalValue(value)
		if err != nil {
			return nil, err
		}
		buffer.Write(valueContent)
	}
	buffer.WriteByte('}')
	return buffer.Bytes(), nil
}

// marshalValue encodes a value like JSON.stringify.
func marshalValue(value any) ([]byte, error) {
	switch v := value.(type) {
	case nil, nullType:
		return []byte("null"), nil
	case float64:
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return []byte("null"), nil
		}
		return []byte(numberToString(v)), nil
	case []any:
		var buffer bytes.Buffer
		buffer.WriteByte('[')
		for i, item := range v {
			if i > 0 {
				buffer.WriteByte(',')
			}
			content, err := marshalValue(item)
			if err != nil {
				return nil, err
			}
			buffer.Write(content)
		}
		buffer.WriteByte(']')
		return buffer.Bytes(), nil
	case *Object:
		return v.MarshalJSON()
	default:
		var buffer bytes.Buffer
		encoder := json.NewEncoder(&buffer)
		encoder.SetEscapeHTML(false)
		err := encoder.Encode(v)
		if err != nil {
			return nil, err
		}
		return bytes.TrimRight(buffer.Bytes(), "\n"), nil
	}
}

// jsonParse is JSON.parse. Object keys keep their document order.
func jsonParse(content string) (any, error) {
	decoder := json.NewDecoder(strings.NewReader(content))
	value, err := decodeJSONValue(decoder)
	if err != nil {
		return nil, err
	}
	if decoder.More() {
		return nil, errInvalidJSON
	}
	_, err = decoder.Token()
	if err == nil {
		return nil, errInvalidJSON
	}
	return value, nil
}

type jsError string

func (e jsError) Error() string { return string(e) }

const errInvalidJSON = jsError("invalid JSON")

func decodeJSONValue(decoder *json.Decoder) (any, error) {
	token, err := decoder.Token()
	if err != nil {
		return nil, err
	}
	switch t := token.(type) {
	case json.Delim:
		switch t {
		case '{':
			o := newObject()
			for decoder.More() {
				keyToken, err := decoder.Token()
				if err != nil {
					return nil, err
				}
				key, isString := keyToken.(string)
				if !isString {
					return nil, errInvalidJSON
				}
				value, err := decodeJSONValue(decoder)
				if err != nil {
					return nil, err
				}
				o.Set(key, value)
			}
			_, err = decoder.Token()
			return o, err
		case '[':
			result := []any{}
			for decoder.More() {
				value, err := decodeJSONValue(decoder)
				if err != nil {
					return nil, err
				}
				result = append(result, value)
			}
			_, err = decoder.Token()
			return result, err
		}
		return nil, errInvalidJSON
	case nil:
		return jsNull, nil
	default:
		return t, nil
	}
}

func isNullish(value any) bool {
	return value == nil || value == jsNull
}

// truthy is JavaScript ToBoolean.
func truthy(value any) bool {
	switch v := value.(type) {
	case nil, nullType:
		return false
	case bool:
		return v
	case float64:
		return v != 0 && !math.IsNaN(v)
	case string:
		return v != ""
	default:
		return true
	}
}

// jsString is JavaScript String(value), as used by template literals.
func jsString(value any) string {
	switch v := value.(type) {
	case nil:
		return "undefined"
	case nullType:
		return "null"
	case string:
		return v
	case bool:
		if v {
			return "true"
		}
		return "false"
	case float64:
		return numberToString(v)
	case []any:
		parts := make([]string, len(v))
		for i, item := range v {
			if !isNullish(item) {
				parts[i] = jsString(item)
			}
		}
		return strings.Join(parts, ",")
	case *Object:
		return "[object Object]"
	default:
		return ""
	}
}

func numberToString(v float64) string {
	switch {
	case math.IsNaN(v):
		return "NaN"
	case math.IsInf(v, 1):
		return "Infinity"
	case math.IsInf(v, -1):
		return "-Infinity"
	case v == 0:
		return "0"
	}
	abs := math.Abs(v)
	if abs >= 1e21 || abs < 1e-6 {
		s := strconv.FormatFloat(v, 'e', -1, 64)
		mantissa, exponent, _ := strings.Cut(s, "e")
		if exponent[0] == '+' {
			return mantissa + "e+" + strings.TrimLeft(exponent[1:], "0")
		}
		return mantissa + "e-" + strings.TrimLeft(exponent[1:], "0")
	}
	return strconv.FormatFloat(v, 'f', -1, 64)
}

var parseIntRegex = regexp.MustCompile(`^[+-]?\d+`)

// parseInt is parseInt(String(value), 10).
func parseInt(value any) float64 {
	s := strings.TrimLeft(jsString(value), " \t\n\v\f\r\u00a0\ufeff\u2028\u2029")
	match := parseIntRegex.FindString(s)
	if match == "" {
		return math.NaN()
	}
	number, err := strconv.ParseFloat(match, 64)
	if err != nil {
		return math.NaN()
	}
	return number
}

// jsNumber is Number(value).
func jsNumber(value any) float64 {
	switch v := value.(type) {
	case nil:
		return math.NaN()
	case nullType:
		return 0
	case bool:
		if v {
			return 1
		}
		return 0
	case float64:
		return v
	case string:
		s := strings.TrimSpace(v)
		if s == "" {
			return 0
		}
		if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
			number, err := strconv.ParseUint(s[2:], 16, 64)
			if err != nil {
				return math.NaN()
			}
			return float64(number)
		}
		if s == "Infinity" || s == "+Infinity" {
			return math.Inf(1)
		}
		if s == "-Infinity" {
			return math.Inf(-1)
		}
		for _, c := range s {
			if !(c >= '0' && c <= '9' || c == '.' || c == 'e' || c == 'E' || c == '+' || c == '-') {
				return math.NaN()
			}
		}
		number, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return math.NaN()
		}
		return number
	case []any:
		switch len(v) {
		case 0:
			return 0
		case 1:
			return jsNumber(jsString(v[0]))
		}
		return math.NaN()
	default:
		return math.NaN()
	}
}

func isNaN(v float64) bool { return math.IsNaN(v) }

func isSafeInteger(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0) && v == math.Trunc(v) && math.Abs(v) <= 1<<53-1
}

// looseEquals is the subset of == used with Array.prototype.includes, which
// uses SameValueZero: strict equality, except NaN equals NaN.
func sameValueZero(a, b any) bool {
	switch av := a.(type) {
	case float64:
		bv, isNumber := b.(float64)
		return isNumber && (av == bv || math.IsNaN(av) && math.IsNaN(bv))
	case string:
		bv, isString := b.(string)
		return isString && av == bv
	case bool:
		bv, isBool := b.(bool)
		return isBool && av == bv
	case nil:
		return b == nil
	case nullType:
		return b == jsNull
	default:
		return a == b
	}
}

// includes is [items...].includes(value).
func includes(value any, items ...any) bool {
	for _, item := range items {
		if sameValueZero(item, value) {
			return true
		}
	}
	return false
}

func includesString(value any, items ...string) bool {
	s, isString := value.(string)
	if !isString {
		return false
	}
	return slices.Contains(items, s)
}

// split is String.prototype.split with a string separator.
func split(s string, separator string) []string {
	if separator == "" {
		var result []string
		for _, r := range s {
			result = append(result, string(r))
		}
		return result
	}
	return strings.Split(s, separator)
}

// at returns parts[index] or undefined.
func at(parts []string, index int) any {
	if index < len(parts) {
		return parts[index]
	}
	return nil
}

// regexExec returns the groups of the first match like RegExp.prototype.exec,
// with undefined for groups that did not participate, or nil on no match.
func regexExec(re *regexp.Regexp, s string) []any {
	indexes := re.FindStringSubmatchIndex(s)
	if indexes == nil {
		return nil
	}
	result := make([]any, len(indexes)/2)
	for i := range result {
		if indexes[2*i] < 0 {
			continue
		}
		result[i] = s[indexes[2*i]:indexes[2*i+1]]
	}
	return result
}

// group returns a match group as a string, empty for undefined.
func group(match []any, index int) string {
	if index < len(match) {
		if s, isString := match[index].(string); isString {
			return s
		}
	}
	return ""
}

var errURIMalformed = jsError("URIError: URI malformed")

// decodeURIComponent is the JavaScript function of the same name.
func decodeURIComponent(value any) (string, error) {
	s := jsString(value)
	if !strings.Contains(s, "%") {
		return s, nil
	}
	var result strings.Builder
	for i := 0; i < len(s); {
		if s[i] != '%' {
			result.WriteByte(s[i])
			i++
			continue
		}
		var buffer []byte
		for i < len(s) && s[i] == '%' {
			if i+2 >= len(s) || !isHex(s[i+1]) || !isHex(s[i+2]) {
				return "", errURIMalformed
			}
			b, _ := strconv.ParseUint(s[i+1:i+3], 16, 8)
			buffer = append(buffer, byte(b))
			i += 3
			if len(buffer) > 0 && buffer[0] < 0x80 {
				break
			}
			if utf8.FullRune(buffer) {
				break
			}
		}
		if !utf8.Valid(buffer) {
			return "", errURIMalformed
		}
		result.Write(buffer)
	}
	return result.String(), nil
}

// mustDecodeURIComponent panics with a JavaScript-like error on malformed
// input, which the parse loop turns into a failed line.
func mustDecodeURIComponent(value any) string {
	s, err := decodeURIComponent(value)
	if err != nil {
		panic(err)
	}
	return s
}

func isHex(c byte) bool {
	return c >= '0' && c <= '9' || c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F'
}

var (
	base64TidyRegex  = regexp.MustCompile(`[^A-Za-z0-9+/]`)
	base64SpaceRegex = regexp.MustCompile(`\s+`)
	base64TrailRegex = regexp.MustCompile(`={0,2}$`)
	base64StdRegex   = regexp.MustCompile(`[^\s0-9a-zA-Z+/]`)
	base64URLRegex   = regexp.MustCompile(`[^\s0-9a-zA-Z\-_]`)
)

// base64Decode is Base64.decode of js-base64 on Node.js: URL-safe characters
// are accepted, other invalid characters are dropped and invalid UTF-8 becomes
// U+FFFD.
func base64Decode(value any) string {
	s := jsString(value)
	s = strings.NewReplacer("-", "+", "_", "/").Replace(s)
	s = base64TidyRegex.ReplaceAllString(s, "")
	if len(s)%4 == 1 {
		s = s[:len(s)-1]
	}
	content, _ := base64.RawStdEncoding.DecodeString(s)
	return strings.ToValidUTF8(string(content), "�")
}

// base64IsValid is Base64.isValid of js-base64.
func base64IsValid(value any) bool {
	s, isString := value.(string)
	if !isString {
		return false
	}
	s = base64SpaceRegex.ReplaceAllString(s, "")
	s = base64TrailRegex.ReplaceAllString(s, "")
	return !base64StdRegex.MatchString(s) || !base64URLRegex.MatchString(s)
}

func stringsToArray(items []string) []any {
	result := make([]any, len(items))
	for i, item := range items {
		result[i] = item
	}
	return result
}

// mapStrings maps an array with String(item).trim() and drops empty items when
// filter is set.
func trimStrings(items []string, filter bool) []any {
	result := make([]any, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if filter && item == "" {
			continue
		}
		result = append(result, item)
	}
	return result
}

// getPath is lodash _.get with a dotted path.
func getPath(value any, path string) any {
	for segment := range strings.SplitSeq(path, ".") {
		o, isObject := value.(*Object)
		if !isObject {
			return nil
		}
		value = o.Get(segment)
	}
	return value
}

// setPath is lodash _.set with a dotted path made of object keys.
func setPath(o *Object, path string, value any) {
	segments := strings.Split(path, ".")
	current := o
	for _, segment := range segments[:len(segments)-1] {
		next, isObject := current.Get(segment).(*Object)
		if !isObject {
			next = newObject()
			current.Set(segment, next)
		}
		current = next
	}
	current.Set(segments[len(segments)-1], value)
}

// isPresentPath is isPresent(obj, attr) of producers/utils.js.
func isPresentPath(o *Object, path string) bool {
	return !isNullish(getPath(o, path))
}

// isNotBlank is isNotBlank of utils.
func isNotBlank(value any) bool {
	s, isString := value.(string)
	return isString && strings.TrimSpace(s) != ""
}

// getIfNotBlank is getIfNotBlank of utils.
func getIfNotBlank(value any, defaultValue any) any {
	if isNotBlank(value) {
		return value
	}
	return defaultValue
}

// getIfPresent is getIfPresent of utils.
func getIfPresent(value any, defaultValue any) any {
	if !isNullish(value) {
		return value
	}
	return defaultValue
}

// or is a || b.
func or(values ...any) any {
	for _, value := range values[:len(values)-1] {
		if truthy(value) {
			return value
		}
	}
	return values[len(values)-1]
}

// coalesce is a ?? b.
func coalesce(values ...any) any {
	for _, value := range values[:len(values)-1] {
		if !isNullish(value) {
			return value
		}
	}
	return values[len(values)-1]
}

// isPlainObject is isPlainObject of utils.
func isPlainObject(value any) bool {
	_, isObject := value.(*Object)
	return isObject
}

// testString is regex.test(value), which converts value to a string first.
func testString(re *regexp.Regexp, value any) bool {
	return re.MatchString(jsString(value))
}

func base64Encode(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}

// encodeURIComponent is the JavaScript function of the same name.
func encodeURIComponent(s string) string {
	const unreserved = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_.!~*'()"
	var builder strings.Builder
	for i := 0; i < len(s); i++ {
		if strings.IndexByte(unreserved, s[i]) >= 0 {
			builder.WriteByte(s[i])
		} else {
			builder.WriteString("%" + strings.ToUpper(strconv.FormatUint(uint64(s[i])|0x100, 16)[1:]))
		}
	}
	return builder.String()
}

// base64DecodeBytes decodes leniently, like Node.js Buffer.from(s, "base64").
func base64DecodeBytes(s string) []byte {
	s = strings.NewReplacer("-", "+", "_", "/").Replace(s)
	if index := strings.IndexByte(s, '='); index >= 0 {
		s = s[:index]
	}
	s = base64TidyRegex.ReplaceAllString(s, "")
	if len(s)%4 == 1 {
		s = s[:len(s)-1]
	}
	content, _ := base64.RawStdEncoding.DecodeString(s)
	return content
}

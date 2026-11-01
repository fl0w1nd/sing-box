// This file is part of a Go port of Sub-Store (https://github.com/sub-store-org/Sub-Store),
// licensed under the GNU Affero General Public License v3.0. See NOTICE in this directory.

// Package subscription converts proxy subscriptions that are not sing-box
// configurations into sing-box outbounds.
//
// It is a port of the Sub-Store conversion to the sing-box target: Base64 and
// plain share link lists and Clash/Mihomo YAML are parsed into Sub-Store proxy
// objects and produced as sing-box outbounds with the same output as Sub-Store.
// UPSTREAM.json records the Sub-Store commit the port follows.
package subscription

import (
	"bytes"
	"strings"
)

// Convert converts a subscription into a sing-box document with an outbounds
// array. Nodes that sing-box cannot use are skipped, as Sub-Store does.
func Convert(content []byte) (result []byte, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			result, err = nil, recoveredError(recovered)
		}
	}()
	raw := strings.ToValidUTF8(string(content), "�")
	outbounds := produce(parse(raw))
	var buffer bytes.Buffer
	buffer.WriteString(`{"outbounds":[`)
	for i, outbound := range outbounds {
		removeUnsupportedFields(outbound)
		if i > 0 {
			buffer.WriteByte(',')
		}
		encoded, err := outbound.MarshalJSON()
		if err != nil {
			return nil, err
		}
		buffer.Write(encoded)
	}
	buffer.WriteString("]}")
	return buffer.Bytes(), nil
}

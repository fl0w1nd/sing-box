// This file is part of a Go port of Sub-Store (https://github.com/sub-store-org/Sub-Store),
// licensed under the GNU Affero General Public License v3.0. See NOTICE in this directory.

package subscription

// unsupportedFields lists, by outbound type, fields that Sub-Store produces for
// sing-box but sing-box does not have. They are left out, so that the outbound
// still loads instead of failing on an unknown field. This is the only place
// where the output differs from Sub-Store; the tests remove the same fields
// from the Sub-Store output.
var unsupportedFields = map[string][]string{
	// Sub-Store maps the Mihomo AnyTLS reuse option to disable_reuse.
	"anytls": {"disable_reuse"},
}

func removeUnsupportedFields(outbound *Object) {
	outboundType, _ := outbound.Get("type").(string)
	for _, field := range unsupportedFields[outboundType] {
		outbound.Delete(field)
	}
}

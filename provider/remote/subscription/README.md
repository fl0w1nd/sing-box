# subscription

Converts subscriptions that are not sing-box documents into sing-box outbounds
for the remote provider: Base64 or plain share link lists (`ss`, `vmess`,
`vless`, `trojan`, `hysteria`, `hysteria2`/`hy2`, `tuic`, `anytls`, `socks`,
`http`/`https`), SSD and Clash/Mihomo YAML.

The package is a Go port of the sing-box conversion of
[Sub-Store](https://github.com/sub-store-org/Sub-Store) (`parse` and
`produce(proxies, "sing-box")`), and produces the same outbounds, except for
the fields in `unsupported.go`. Sub-Store is
licensed under the GNU Affero General Public License v3.0, see `NOTICE`.

Not ported: Surge, Loon and Quantumult X formats, and WireGuard and Tailscale,
which Sub-Store produces as endpoints.

## Layout

The files follow the Sub-Store modules they port, and keep its JavaScript
semantics (`js.go`: ordered objects, `undefined`/`null`, truthiness,
`parseInt`, js-base64), so that a change in Sub-Store maps to the matching Go
function:

| Go | Sub-Store `backend/src/` |
| --- | --- |
| `preprocess.go` | `core/proxy-utils/preprocessors/index.js` |
| `parse_uri.go` | `core/proxy-utils/parsers/index.js` (URI parsers), `parsers/peggy/trojan-uri.js` |
| `parse.go` | `core/proxy-utils/index.js` (`parse`, `lastParse`), `parsers/index.js` (`Clash_All`) |
| `produce.go` | `core/proxy-utils/index.js` (`produce`), `producers/clashmeta.js`, `producers/utils.js` |
| `produce_singbox.go` | `core/proxy-utils/producers/sing-box.js` |
| `utils.go` | `utils/index.js`, `transport-path.js`, `vmess-security.js`, `ech-utils.js` |
| `yaml.go` | `utils/yaml.js` |

`UPSTREAM.json` records the Sub-Store commit the port follows and the hashes of
these files.

## Tests

- `TestConvertMatchesSubStore`: every `testdata/cases/*.txt` is converted and
  compared with `*.json` next to it, which Sub-Store produced.
- `TestConvertedOutboundsMatchSingBoxOptions`: every converted outbound must
  decode, with unknown fields rejected, as an outbound of this sing-box. After
  merging sing-box upstream, a renamed, removed or retyped field that Sub-Store
  produces fails here.

The output differs from Sub-Store in one way: fields that Sub-Store produces
but sing-box does not have, listed in `unsupported.go`, are left out so that
the outbound still loads. `TestConvertMatchesSubStore` removes them from the
Sub-Store output too, and fails once Sub-Store stops producing one.

## Following Sub-Store

Requires git, Node.js and pnpm:

```bash
node provider/remote/subscription/testdata/harness/sync.js
go test ./provider/remote/subscription/
git diff provider/remote/subscription/testdata/cases/
```

`sync.js` clones Sub-Store master (`--ref` for another revision), regenerates
the expected outputs and lists the followed files that changed since the pinned
commit, with their diff (exit status 2). Then:

1. Port the changes until `go test` passes. `git diff` of `testdata/cases`
   shows how the sing-box output changed.
2. For a new field or format, add an input to `testdata/cases` and run
   `sync.js` again, so that the tests cover it.
3. Pin the revision with `sync.js --update`.

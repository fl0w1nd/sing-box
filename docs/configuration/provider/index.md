# Outbound Provider

Outbound providers load proxy outbounds from a remote subscription and update referenced Selector and URLTest groups at runtime.

### Complete example

```json
{
  "dns": {
    "servers": [
      {
        "type": "local",
        "tag": "local"
      }
    ]
  },
  // Shared HTTP client used to download the provider.
  "http_clients": [
    {
      // Referenced by the provider's http_client field.
      "tag": "provider-download-client",
      // A static download group available before the provider starts.
      "detour": "provider-download-direct"
    }
  ],
  // Remote outbound providers.
  "providers": [
    {
      // Provider type. Currently fixed to remote.
      "type": "remote",
      // Unique provider tag; imported nodes use airport/original-tag.
      "tag": "airport",
      // Remote subscription URL: sing-box JSON, share links or Clash YAML.
      "url": "https://example.com/subscription.json",
      // Required tag of a shared top-level HTTP client.
      "http_client": "provider-download-client",
      // User-Agent for subscription requests.
      "user_agent": "sing-box",
      // Additional subscription request headers.
      "headers": {
        "Authorization": "Bearer example"
      },
      // Refresh interval; defaults to 24h and has a 1m minimum.
      "update_interval": "1h",
      // Override dial fields on every imported outbound that supports them.
      "override_dialer_options": {
        "domain_resolver": "local"
      },
      // Import original node tags matching this expression; .* includes all nodes.
      "include": ".*",
      // Exclude matching original tags; ^$ keeps every normalized non-empty tag.
      "exclude": "^$"
    }
  ],
  "outbounds": [
    {
      // Explicit direct outbound for the provider download path.
      "type": "direct",
      "tag": "provider-direct"
    },
    {
      // Static Selector dedicated to provider downloads.
      "type": "selector",
      "tag": "provider-download-direct",
      // Contains only static outbounds available before provider startup.
      "outbounds": ["provider-direct"],
      "default": "provider-direct",
      "interrupt_exist_connections": false
    },
    {
      // Dynamic Selector for manually selecting provider nodes.
      "type": "selector",
      "tag": "proxy",
      // Static members; an empty array creates a provider-only group.
      "outbounds": [],
      // Explicit provider references.
      "providers": ["airport"],
      // Uses every provider when true.
      "use_all_providers": false,
      // Match full dynamic tags such as airport/HK 01; .* includes every member.
      "include": ".*",
      // Exclude full dynamic tags; ^$ keeps every normalized member.
      "exclude": "^$",
      // Optional: a default node expressed as a full dynamic tag.
      // "default": "airport/HK 01",
      // Preserve existing connections when provider membership changes.
      "interrupt_exist_connections": false
    },
    {
      // Uses the existing URLTest system to measure and select provider nodes.
      "type": "urltest",
      "tag": "auto",
      "outbounds": [],
      "providers": ["airport"],
      "use_all_providers": false,
      // URLTest dynamic-member filter; .* includes every member.
      "include": ".*",
      // ^$ keeps every normalized member.
      "exclude": "^$",
      // URLTest probe URL.
      "url": "https://www.gstatic.com/generate_204",
      // Test interval.
      "interval": "3m",
      // Required delay improvement in milliseconds before switching.
      "tolerance": 50,
      // Stops periodic tests after this idle duration.
      "idle_timeout": "30m",
      "interrupt_exist_connections": false
    }
  ],
  "route": {
    "final": "proxy"
  },
  "experimental": {
    "cache_file": {
      // Stores raw content, ETag, update time, and subscription traffic data.
      "enabled": true,
      "path": "cache.db"
    }
  }
}
```

Static outbounds start before HTTP clients, and HTTP clients start before providers. `provider-download-direct` therefore contains only the explicitly declared static `provider-direct`; the dynamic `proxy` group receives provider nodes after the first subscription update.

### Provider fields

| Field | Description |
|-------|-------------|
| `type` | Required. Currently `remote`. |
| `tag` | Required unique provider tag. |
| `url` | Required remote subscription URL. |
| `http_client` | Required tag of a shared top-level [HTTP client](/configuration/shared/http-client/). |
| `user_agent` | Subscription request User-Agent; defaults to the current sing-box version. |
| `headers` | Additional subscription request headers. |
| `update_interval` | Update interval; defaults to 24 hours and has a minimum of 1 minute. |
| `tag_prefix` | Prefix added to imported node tags; defaults to `<provider tag>/`. |
| `override_dialer_options` | Dial fields applied to every imported outbound that supports them. Fields present here replace the corresponding subscription fields; omitted fields are preserved. Object-valued fields such as `domain_resolver` are replaced as a whole. |
| `include` / `exclude` | Regular-expression filters for original node tags; `exclude` takes precedence. |

The subscription document uses sing-box JSON. The provider imports its top-level `outbounds` array and ignores other top-level fields. Empty tags use the array index; duplicate tags receive ` #2`, ` #3`, and subsequent suffixes.

A subscription that is not a sing-box JSON document with an `outbounds` array is converted first, with the same result as the [Sub-Store](https://github.com/sub-store-org/Sub-Store) sing-box target:

* Base64-encoded or plain share links: `ss`, `vmess`, `vless`, `trojan`, `hysteria`, `hysteria2`/`hy2`, `tuic`, `anytls`, `socks`/`socks5` and `http`/`https`.
* Clash and Mihomo YAML; only `proxies` is read.
* SSD.

Fields that sing-box does not support, such as `disable_reuse` that Sub-Store produces for AnyTLS, are left out. Converted nodes use their names as tags. Nodes that sing-box cannot use, such as ShadowsocksR or VLESS with XHTTP, are skipped; WireGuard and Tailscale nodes are skipped as well, since sing-box configures them as endpoints. Surge, Loon and Quantumult X formats are not supported. Many subscription services choose the format by `User-Agent`; a Clash or Mihomo `user_agent` often returns more complete node parameters than a share link list.

`include` and `exclude` do not drop a node that a loaded node uses as its `detour`, such as the ShadowTLS node `<name>_shadowtls` that a converted Shadowsocks node with the shadow-tls plugin dials through.

An imported outbound can use another imported node as its `detour`. The provider resolves an original node tag such as `entry` to its full tag (`airport/entry` by default); an explicit full tag also works. Other detour tags keep their original value. Chained nodes start in dependency order, and an upstream node change rebuilds its dependents during a subscription refresh.

For chained proxies, we recommend pointing the downstream node's `detour` to a statically configured Selector or URLTest group that selects upstream provider nodes. If subscription nodes refer to each other directly, users must keep those references valid across updates.

Configuring a provider enables CacheFile for subscription persistence. `cache_file.path` controls its location; the provider still opens it when `cache_file.enabled` is false.

Selector and URLTest group filters apply only to dynamic provider members and match their full provider-prefixed tags. Existing URLTest code owns delay measurement and best-node selection.

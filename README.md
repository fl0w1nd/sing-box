# sing-box Extended

A fork of [sing-box](https://github.com/SagerNet/sing-box), the universal proxy platform, that follows upstream and adds the features below. Releases are tagged `v<upstream version>-extended.<N>`.

## Documentation

Upstream documentation: https://sing-box.sagernet.org

## Extended features

### Outbound providers

Top-level `providers` load proxy outbounds from a remote subscription and keep them updated at runtime, without restarting the core. Subscriptions can be sing-box JSON, share links (plain or Base64), SSD or Clash/Mihomo YAML; other formats are converted like the [Sub-Store](https://github.com/sub-store-org/Sub-Store) sing-box target. Selector and URLTest groups take provider nodes with `providers` or `use_all_providers`, filtered by `include` and `exclude`.

See [Outbound Provider](docs/configuration/provider/index.md) ([中文](docs/configuration/provider/index.zh.md)).

### Unified delay

```json
{
  "experimental": {
    "unified_delay": {
      "enabled": true
    }
  }
}
```

URL tests, including URLTest groups and Clash API delay tests, send a second request over the established connection and report its time, like the Mihomo `unified-delay` option, so the result leaves out handshakes. If the second request fails, the first result is kept.

### Sniff `override_destination`

```json
{
  "action": "sniff",
  "override_destination": true
}
```

The `sniff` route action takes `override_destination` again, which upstream removed with the deprecated inbound sniff fields, so that a proxy resolves the domain on the remote side instead of connecting to the IP from the local DNS. Rules match as without it: domain rules see the sniffed domain and IP rules the original IP. Only when the connection is handed to the outbound is the destination replaced with the sniffed domain, unless the outbound is `direct` or an endpoint, which would resolve it locally again, or the destination was overridden by route options. Unlike upstream before, `direct` keeps the IP. Only a domain read from the connection itself (the TLS or QUIC server name, or the HTTP host) is used: a domain from the DNS reverse mapping is left to rule matching, since it is a guess when several domains share the IP.

On a server, whose clients resolved the IP on their side, `direct` can resolve the sniffed domain again with the server's own DNS. Set `override_direct` as well, typically only for the proxy inbounds:

```json
{
  "inbound": ["vless-in"],
  "action": "sniff",
  "override_destination": true,
  "override_direct": true
}
```

`override_direct` has no effect without `override_destination`. Endpoints always keep the IP.

### 15s initial TCP keepalive

The initial TCP keepalive interval is 15 seconds, as in the Go standard library and Mihomo, instead of 5 minutes upstream, so that carrier-grade NAT does not drop idle connections silently.

### DNS query log

```json
{
  "experimental": {
    "dns_query_log": {
      "enabled": true,
      "path": "dns_query_log",
      "memory_only": false,
      "retention": "24h",
      "max_size": "64MB",
      "memory_size": 1000,
      "flush_interval": "30s"
    }
  }
}
```

Records every query of the DNS router, like the AdGuard Home query log, with what only the core knows: the client (inbound, source address, process), the matched DNS rules, each exchange with a DNS server (server tag, its configured address such as `https://dns.google/dns-query` and detour, or the addresses a `local` or `dhcp` server used, query type, time, answers, rejected by a later rule), cache and optimistic cache hits, FakeIP, and the final answer or error. Queries from `hijack-dns` and DNS inbounds are recorded as `exchange`, lookups for routing and dialing as `lookup`, and optimistic cache refreshes as `background`. Each entry has a status: `upstream`, `cached`, `optimistic`, `fakeip`, `rejected`, `predefined`, `strategy` (an empty answer for a query type the domain strategy excludes, such as AAAA with `ipv4_only`), `local` (any other answer the core made itself, such as for resolver discovery queries) or `error`.

Entries are appended to segment files under `path` every `flush_interval`, or once `memory_size` entries are waiting, starting a new segment every hour or every eighth of `max_size`. Segments older than `retention` are removed, then the oldest segments while the total size exceeds `max_size`. With `memory_only`, only the latest `memory_size` entries are kept in memory and nothing is written. All fields except `enabled` are optional, with the defaults above.

The query log is served by `ExtendedService` and by the `sing-box api dns-log` command:

```bash
sing-box api dns-log -n 50 --search example.com
sing-box api dns-log -f --status upstream --server remote
sing-box api dns-log status
sing-box api dns-log stats --since 24h --bucket 1h
sing-box api dns-log clear
sing-box api dns-cache clear
```

`stats` summarizes the entries a filter selects: counts by status, by method and per time bucket, the top domains of the queries the core received and, apart, of the lookups and background queries it made itself (reverse lookups, under `in-addr.arpa` or `ip6.arpa`, are only counted, since every address is a domain of its own), blocked domains and the rules that blocked them, clients (of the queries the core received, not of the lookups it made for connections) and query types, each DNS server's exchanges, failures, rejections and durations, and the average and 95th percentile duration of all queries and of those sent upstream. Only queries answered by a `reject` rule count as blocked; `predefined` answers, such as hosts or an empty answer to every AAAA query, count only by status. `--method` selects `exchange`, `lookup` or `background` entries. It reads the matching entries on each call.

`dns-cache clear` clears the DNS cache like the Clash API `/cache/dns/flush`: the memory and optimistic cache, the cache stored in the cache file, the platform and system DNS caches, and the reverse mapping.

### ExtendedService

The daemon serves a separate gRPC service next to the upstream `StartedService`, which stays unchanged, for fork features in graphical clients. See [`daemon/extended_service.proto`](daemon/extended_service.proto):

* `GetCapabilities`: the extended features the core supports, with versions.
* `SubscribeOutboundProviders` and `UpdateOutboundProvider`: provider status, including subscription traffic and expiry, and manual updates.
* `SetTailscaleForceRelay` and `SubscribeTailscaleForceRelayStatus`: make every Tailscale endpoint reach peers only through DERP relays, switched at runtime and kept in the cache file.
* `GetDNSQueryLogStatus`, `QueryDNSQueryLog`, `SubscribeDNSQueryLog`, `GetDNSQueryLogStats` and `ClearDNSQueryLog`: the DNS query log, queried with filters and cursor pages, newest first, followed live, or summarized.
* `ClearDNSCache`: clear the DNS cache, without the Clash API.

## License

`provider/remote/subscription` is a port of Sub-Store and is licensed under the GNU Affero General Public License v3.0, see [its NOTICE](provider/remote/subscription/NOTICE).

```
Copyright (C) 2022 by nekohasekai <contact-sagernet@sekai.icu>

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program. If not, see <http://www.gnu.org/licenses/>.

In addition, no derivative work may use the name or imply association
with this application without prior consent.
```
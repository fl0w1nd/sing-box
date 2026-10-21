# 出站提供者

出站提供者从远程 sing-box JSON 文档加载代理出站，并在运行时更新引用它的 Selector 和 URLTest 策略组。

### 完整配置示例

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
  // Provider 下载使用的共享 HTTP 客户端。
  "http_clients": [
    {
      // 共享 HTTP 客户端标签，由 Provider 的 http_client 引用。
      "tag": "provider-download-client",
      // 指向启动时已经可用的静态下载策略组。
      "detour": "provider-download-direct"
    }
  ],
  // 远程 Outbound Provider 列表。
  "providers": [
    {
      // Provider 类型，当前固定为 remote。
      "type": "remote",
      // Provider 唯一标签；导入节点使用 airport/原始标签。
      "tag": "airport",
      // 返回 sing-box JSON 的远程订阅地址。
      "url": "https://example.com/subscription.json",
      // 必填的顶层共享 HTTP 客户端标签。
      "http_client": "provider-download-client",
      // 订阅请求的 User-Agent。
      "user_agent": "sing-box",
      // 订阅请求的附加请求头。
      "headers": {
        "Authorization": "Bearer example"
      },
      // 自动刷新周期；默认 24h，最小 1m。
      "update_interval": "1h",
      // 覆盖所有支持拨号字段的导入出站。
      "override_dialer_options": {
        "domain_resolver": "local"
      },
      // 只导入原始标签命中该正则的节点；.* 表示全部节点。
      "include": ".*",
      // 排除原始标签命中该正则的节点，优先级高于 include；^$ 不匹配规范化后的非空标签。
      "exclude": "^$"
    }
  ],
  "outbounds": [
    {
      // Provider 下载链路使用的显式直连出站。
      "type": "direct",
      "tag": "provider-direct"
    },
    {
      // Provider 下载专用静态 Selector。
      "type": "selector",
      "tag": "provider-download-direct",
      // 只引用 Provider 启动前已经存在的静态出站。
      "outbounds": ["provider-direct"],
      "default": "provider-direct",
      "interrupt_exist_connections": false
    },
    {
      // 手动选择 Provider 节点的动态 Selector。
      "type": "selector",
      "tag": "proxy",
      // 静态成员；空数组表示只使用 Provider 节点。
      "outbounds": [],
      // 引用指定 Provider。
      "providers": ["airport"],
      // true 时使用全部 Provider。
      "use_all_providers": false,
      // 匹配完整动态标签，例如 airport/HK 01；.* 表示全部成员。
      "include": ".*",
      // 排除完整动态标签，优先级高于 include；^$ 保留全部规范化成员。
      "exclude": "^$",
      // 可选：默认节点使用完整动态标签。
      // "default": "airport/HK 01",
      // Provider 成员更新时保留现有连接。
      "interrupt_exist_connections": false
    },
    {
      // 使用官方 URLTest 机制测速并自动选择 Provider 节点。
      "type": "urltest",
      "tag": "auto",
      "outbounds": [],
      "providers": ["airport"],
      "use_all_providers": false,
      // URLTest 组级动态成员过滤；.* 表示全部成员。
      "include": ".*",
      // ^$ 保留全部规范化成员。
      "exclude": "^$",
      // URLTest 探测地址。
      "url": "https://www.gstatic.com/generate_204",
      // 测速周期。
      "interval": "3m",
      // 新节点延迟至少降低该毫秒数时切换。
      "tolerance": 50,
      // 空闲达到该时间后暂停周期测速。
      "idle_timeout": "30m",
      "interrupt_exist_connections": false
    }
  ],
  "route": {
    "final": "proxy"
  },
  "experimental": {
    "cache_file": {
      // 保存原始订阅、ETag、更新时间和订阅流量信息。
      "enabled": true,
      "path": "cache.db"
    }
  }
}
```

启动顺序为静态出站、HTTP 客户端、Provider。`provider-download-direct` 因此只引用显式声明的静态 `provider-direct`；动态 `proxy` 组在首次订阅完成后获得 Provider 节点。

### Provider 字段

| 字段 | 说明 |
|------|------|
| `type` | 必填，当前为 `remote`。 |
| `tag` | 必填的 Provider 唯一标签。 |
| `url` | 必填的远程订阅 URL。 |
| `http_client` | 必填的顶层共享 [HTTP 客户端](/zh/configuration/shared/http-client/) 标签。 |
| `user_agent` | 订阅请求的 User-Agent；默认使用当前 sing-box 版本。 |
| `headers` | 订阅请求的附加请求头。 |
| `update_interval` | 更新周期，默认 24 小时，最小 1 分钟。 |
| `tag_prefix` | 导入节点标签的前缀；默认是 `<Provider 标签>/`。 |
| `override_dialer_options` | 应用于所有支持拨号字段的导入出站。此处出现的字段替换订阅中的对应字段，省略的字段保持原值；`domain_resolver` 等对象字段整体替换。 |
| `include` / `exclude` | 原始节点标签正则过滤；`exclude` 优先。 |

订阅文档使用 sing-box JSON。Provider 导入顶层 `outbounds` 数组，并忽略其他顶层字段。空标签使用数组索引，重复标签追加 ` #2`、` #3`。

导入节点可用 `detour` 引用同一 Provider 的其他节点。原始标签 `entry` 会被解析为完整标签（默认 `airport/entry`），显式写完整标签也可使用；其他 `detour` 标签保持原值。链式节点按依赖顺序启动，上游节点变化时会在订阅刷新中重建依赖它的节点。

链式代理建议让下游节点的 `detour` 指向静态配置的 Selector 或 URLTest 策略组，由策略组选择上游 Provider 节点。直接引用订阅节点时，使用者需自行维护刷新后的引用关系。

配置 Provider 会启用 CacheFile 保存订阅。`cache_file.path` 可指定文件位置；即使 `cache_file.enabled` 为 false，Provider 也会打开该文件。

Selector 和 URLTest 的组级 `include`/`exclude` 只过滤 Provider 动态成员，并匹配带 Provider 前缀的完整标签。URLTest 延迟和最优节点选择统一由现有 URLTest 系统管理。

# GoDNS

一个功能强大的Go语言DNS客户端库，支持多种DNS协议（UDP、TCP、DoT、DoH）和代理配置，特别适合需要并发查询多个DNS服务器的场景。

## 特性

- 🚀 **多协议支持**：UDP、TCP、DNS over TLS (DoT)、DNS over HTTPS (DoH)
- 🌐 **并发查询**：MultiQuery功能可同时查询多个DNS服务器
- 🔒 **代理支持**：支持SOCKS5和HTTP代理
- 🎯 **CDN友好**：特别适合查询CDN域名，获取多个IP地址
- ⚡ **高性能**：并发查询，快速获取结果
- 🛡️ **容错机制**：支持重试和超时配置
- 📦 **易于使用**：简洁的API设计

## 协议代理支持对比

| 协议 | SOCKS5代理 | HTTP代理 | 技术原理 | 实现难度 | 网络兼容性 |
|------|------------|----------|----------|----------|------------|
| **UDP** | ✅ 支持 | ❌ 不支持 | SOCKS5支持UDP转发 | 简单 | 好 |
| **DoT** | ✅ 支持 | ❌ 原生不支持* | TCP连接可通过SOCKS5代理 | 中等 | 一般 |
| **DoH** | ✅ 支持 | ✅ 支持 | 基于HTTP/HTTPS，天然支持HTTP代理 | 简单 | 最好 |

*注：DoT可通过HTTP CONNECT方法实现HTTP代理支持，但需要额外实现。


## 安装

```bash
go get github.com/zan8in/godns
```

## 快速开始

### 基本使用

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"
    
    "github.com/zan8in/godns"
)

func main() {
    // 创建默认客户端
    client := godns.NewDefault()
    
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    
    // 查询A记录
    result, err := client.MultiQueryA(ctx, "example.com")
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("域名: %s\n", result.Domain)
    for _, ip := range result.AllIPs {
        fmt.Printf("IP: %s\n", ip)
    }
}
```

## 详细使用

### 1. 协议配置

#### UDP/TCP 查询
```go
client := godns.New(
    godns.WithProtocol(godns.UDP),
    godns.WithTimeout(3*time.Second),
    godns.WithRetries(2),
)
```

#### DNS over TLS (DoT)
```go
client := godns.New(
    godns.WithProtocol(godns.DoT),
    godns.WithTimeout(5*time.Second),
)
```

#### DNS over HTTPS (DoH)
```go
client := godns.New(
    godns.WithProtocol(godns.DoH),
    godns.WithTimeout(10*time.Second),
)
```

### 2. 自定义DNS服务器

```go
client := godns.New(
    godns.WithServers(
        "8.8.8.8:53",
        "1.1.1.1:53",
        "223.5.5.5:53",
    ),
)
```

### 3. 代理配置

#### SOCKS5代理
```go
client := godns.New(
    godns.WithSOCKS5Proxy("127.0.0.1:1080", nil), // 无认证
)

// 带认证的SOCKS5代理
auth := &godns.ProxyAuth{
    Username: "user",
    Password: "pass",
}
client := godns.New(
    godns.WithSOCKS5Proxy("127.0.0.1:1080", auth),
)
```

#### HTTP代理
```go
client := godns.New(
    godns.WithHTTPProxy("127.0.0.1:8080", nil),
)
```

### 4. 查询不同类型的DNS记录

```go
// A记录
result, err := client.QueryA(ctx, "example.com")

// AAAA记录
result, err := client.QueryAAAA(ctx, "example.com")

// CNAME记录
result, err := client.QueryCNAME(ctx, "www.example.com")

// MX记录
result, err := client.QueryMX(ctx, "example.com")

// TXT记录
result, err := client.QueryTXT(ctx, "example.com")
```

### 5. 并发多服务器查询

```go
// MultiQuery会并发查询所有配置的DNS服务器
result, err := client.MultiQuery(ctx, "example.com", dns.TypeA)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("域名: %s\n", result.Domain)
fmt.Printf("所有IP地址: %v\n", result.AllIPs)

// 查看每个DNS服务器的结果
for _, res := range result.Results {
    fmt.Printf("服务器 %s:\n", res.Server)
    if res.Error != nil {
        fmt.Printf("  错误: %v\n", res.Error)
    } else {
        for _, record := range res.Records {
            fmt.Printf("  %s -> %s (TTL: %d)\n", record.Name, record.Value, record.TTL)
        }
    }
}
```

## 配置选项

| 选项 | 说明 | 默认值 |
|------|------|--------|
| `WithTimeout(duration)` | 设置查询超时时间 | 5秒 |
| `WithRetries(count)` | 设置重试次数 | 3次 |
| `WithProtocol(protocol)` | 设置DNS协议 | UDP |
| `WithServers(servers...)` | 设置DNS服务器列表 | 8.8.8.8:53, 1.1.1.1:53 |
| `WithSOCKS5Proxy(addr, auth)` | 设置SOCKS5代理 | 无 |
| `WithHTTPProxy(addr, auth)` | 设置HTTP代理 | 无 |
| `WithTLSConfig(config)` | 设置TLS配置 | 默认配置 |
| `WithHTTPClient(client)` | 设置HTTP客户端 | 默认客户端 |

## 预配置的DNS服务器

### DoH服务器
```go
var DoHServers = []string{
    "https://dns.alidns.com/dns-query",     // 阿里DoH
    "https://doh.pub/dns-query",            // DoH.Pub
    "https://1.12.12.12/dns-query",        // DNSPod DoH
    "https://120.53.53.53/dns-query",      // DNSPod DoH备用
}
```

### DoT服务器
```go
var DoTServers = []string{
    "8.8.8.8:853",
    "1.1.1.1:853",
}
```

### UDP服务器
```go
var UDPServers = []string{
    "8.8.8.8:53",         // Google DNS
    "1.1.1.1:53",         // Cloudflare DNS
    "114.114.114.114:53", // 114 DNS
    "223.5.5.5:53",       // 阿里DNS
}
```

## 使用场景

### CDN域名解析
对于使用CDN的域名，不同的DNS服务器可能返回不同的IP地址。使用MultiQuery可以获取所有可能的IP地址：

```go
result, err := client.MultiQueryA(ctx, "cdn-domain.com")
if err != nil {
    log.Fatal(err)
}

fmt.Printf("CDN域名 %s 的所有IP地址:\n", result.Domain)
for _, ip := range result.AllIPs {
    fmt.Printf("  %s\n", ip)
}
```

### 网络环境检测
通过查询多个DNS服务器，可以检测网络环境和DNS污染情况：

```go
result, err := client.MultiQuery(ctx, "test-domain.com", dns.TypeA)
if err != nil {
    log.Fatal(err)
}

for _, res := range result.Results {
    if res.Error != nil {
        fmt.Printf("服务器 %s 不可用: %v\n", res.Server, res.Error)
    } else {
        fmt.Printf("服务器 %s 正常\n", res.Server)
    }
}
```

## 错误处理

```go
result, err := client.MultiQuery(ctx, "example.com", dns.TypeA)
if err != nil {
    log.Printf("查询失败: %v", err)
    return
}

// 检查是否有成功的结果
if len(result.AllIPs) == 0 {
    log.Println("没有获取到任何IP地址")
    return
}

// 检查各个服务器的结果
for _, res := range result.Results {
    if res.Error != nil {
        log.Printf("服务器 %s 查询失败: %v", res.Server, res.Error)
    }
}
```

## 性能优化建议

1. **合理设置超时时间**：根据网络环境调整超时时间
2. **控制DNS服务器数量**：过多的服务器会增加查询时间
3. **选择合适的协议**：UDP最快，DoH最安全但较慢
4. **使用代理时增加超时时间**：代理会增加额外的延迟

## 依赖

- [github.com/miekg/dns](https://github.com/miekg/dns) - DNS库
- [golang.org/x/net](https://golang.org/x/net) - 网络扩展包

## 许可证

本项目采用 MIT 许可证。详见 [LICENSE](LICENSE) 文件。

## 贡献

欢迎提交Issue和Pull Request！

## 更新日志

### v1.0.0
- 初始版本发布
- 支持UDP、TCP、DoT、DoH协议
- 支持SOCKS5和HTTP代理
- 支持并发多服务器查询
package godns

import (
	"crypto/tls"
	"net/http"
	"time"
)

var DoHServers = []string{
	"https://dns.alidns.com/dns-query", // 阿里DoH - 保留
	"https://doh.pub/dns-query",        // DoH.Pub - 国内优化
	"https://1.12.12.12/dns-query",     // DNSPod DoH
	"https://120.53.53.53/dns-query",   // DNSPod DoH备用
	"https://1.1.1.1/dns-query",        // Cloudflare - 可选，但在国内可能较慢
}

var DoTServers = []string{
	"223.5.5.5:853",    // 阿里DoT
	"223.6.6.6:853",    // 阿里DoT备用
	"1.12.12.12:853",   // DNSPod DoT
	"120.53.53.53:853", // DNSPod DoT备用
	"8.8.8.8:853",      // Google - 在国内可能不稳定
	"1.1.1.1:853",      // Cloudflare - 在国内可能不稳定
}

var UDPServers = []string{
	"223.5.5.5:53",       // 阿里DNS - 保留
	"223.6.6.6:53",       // 阿里DNS备用
	"114.114.114.114:53", // 114DNS - 保留
	"114.114.115.115:53", // 114DNS备用
	"1.12.12.12:53",      // DNSPod
	"120.53.53.53:53",    // DNSPod备用
	"119.29.29.29:53",    // DNSPod腾讯
	"182.254.116.116:53", // DNSPod腾讯备用
	"8.8.8.8:53",         // Google - 可选，但可能被污染
	"1.1.1.1:53",         // Cloudflare - 可选，但在国内较慢
}

// Client DNS客户端
type Client struct {
	config *Config
}

// Config 配置选项
type Config struct {
	// 基础配置
	Timeout  time.Duration
	Retries  int
	Protocol Protocol

	// 服务器配置
	Servers []string

	// 代理配置
	ProxyType ProxyType
	ProxyAddr string
	ProxyAuth *ProxyAuth

	// TLS配置
	TLSConfig *tls.Config

	// HTTP配置（用于DoH）
	HTTPClient *http.Client
}

// Protocol 协议类型
type Protocol string

const (
	UDP Protocol = "udp"
	TCP Protocol = "tcp"
	DoT Protocol = "dot" // DNS over TLS
	DoH Protocol = "doh" // DNS over HTTPS
)

// ProxyType 代理类型
type ProxyType string

const (
	NoProxy   ProxyType = ""
	SOCKS5    ProxyType = "socks5"
	HTTPProxy ProxyType = "http"
)

// ProxyAuth 代理认证
type ProxyAuth struct {
	Username string
	Password string
}

// Option 配置选项函数
type Option func(*Config)

// NewDefault 创建默认客户端
func NewDefault() *Client {
	return &Client{
		config: &Config{
			Timeout:   5 * time.Second,
			Retries:   3,
			Protocol:  UDP,
			Servers:   UDPServers,
			ProxyType: NoProxy,
		},
	}
}

// New 创建自定义客户端
func New(opts ...Option) *Client {

	config := &Config{
		Timeout:   5 * time.Second,
		Retries:   3,
		Protocol:  UDP,
		Servers:   UDPServers,
		ProxyType: NoProxy,
	}

	for _, opt := range opts {
		opt(config)
	}

	return &Client{config: config}
}

// 配置选项函数
func WithTimeout(timeout time.Duration) Option {
	return func(c *Config) {
		c.Timeout = timeout
	}
}

func WithRetries(retries int) Option {
	return func(c *Config) {
		c.Retries = retries
	}
}

func WithProtocol(protocol Protocol) Option {
	return func(c *Config) {
		c.Protocol = protocol
		switch protocol {
		case DoH:
			c.Servers = DoHServers
		case DoT:
			c.Servers = DoTServers
		default:
			c.Servers = UDPServers
		}
	}
}

func WithServers(servers ...string) Option {
	return func(c *Config) {
		c.Servers = servers
	}
}

func WithSOCKS5Proxy(addr string, auth *ProxyAuth) Option {
	return func(c *Config) {
		c.ProxyType = SOCKS5
		c.ProxyAddr = addr
		c.ProxyAuth = auth
	}
}

func WithHTTPProxy(addr string, auth *ProxyAuth) Option {
	return func(c *Config) {
		c.ProxyType = HTTPProxy
		c.ProxyAddr = addr
		c.ProxyAuth = auth
	}
}

func WithTLSConfig(tlsConfig *tls.Config) Option {
	return func(c *Config) {
		c.TLSConfig = tlsConfig
	}
}

func WithHTTPClient(client *http.Client) Option {
	return func(c *Config) {
		c.HTTPClient = client
	}
}

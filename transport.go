package godns

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/miekg/dns"
	"golang.org/x/net/proxy"
)

// queryUDPTCP UDP/TCP查询
func (c *Client) queryUDPTCP(ctx context.Context, msg *dns.Msg, server string) (*dns.Msg, error) {
	client := &dns.Client{
		Net:     string(c.config.Protocol),
		Timeout: c.config.Timeout,
	}

	var lastErr error
	for i := 0; i <= c.config.Retries; i++ {
		var response *dns.Msg
		var err error

		if c.config.ProxyType != NoProxy {
			// 使用代理连接
			response, err = c.exchangeWithProxy(ctx, msg, server)
		} else {
			// 直接连接
			response, _, err = client.ExchangeContext(ctx, msg, server)
		}

		if err == nil {
			return response, nil
		}
		lastErr = err

		if i < c.config.Retries {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(time.Millisecond * 100 * time.Duration(i+1)):
				// 指数退避重试
			}
		}
	}

	return nil, lastErr
}

// exchangeWithProxy 通过代理进行DNS查询
func (c *Client) exchangeWithProxy(ctx context.Context, msg *dns.Msg, server string) (*dns.Msg, error) {
	proxyDialer, err := c.createDialer()
	if err != nil {
		return nil, fmt.Errorf("failed to create proxy dialer: %v", err)
	}

	// 通过代理建立连接
	conn, err := proxyDialer.Dial("tcp", server)
	if err != nil {
		return nil, fmt.Errorf("failed to dial through proxy: %v", err)
	}
	defer conn.Close()

	// 手动发送DNS查询
	dnsConn := &dns.Conn{Conn: conn}
	err = dnsConn.WriteMsg(msg)
	if err != nil {
		return nil, fmt.Errorf("failed to write DNS message: %v", err)
	}

	// 读取响应
	response, err := dnsConn.ReadMsg()
	if err != nil {
		return nil, fmt.Errorf("failed to read DNS response: %v", err)
	}

	return response, nil
}

// exchangeDoTWithProxy 通过代理进行DoT查询
func (c *Client) exchangeDoTWithProxy(ctx context.Context, msg *dns.Msg, server string, tlsConfig *tls.Config) (*dns.Msg, error) {
	proxyDialer, err := c.createDialer()
	if err != nil {
		return nil, fmt.Errorf("failed to create proxy dialer: %v", err)
	}

	// 通过代理建立连接
	conn, err := proxyDialer.Dial("tcp", server)
	if err != nil {
		return nil, fmt.Errorf("failed to dial through proxy: %v", err)
	}
	defer conn.Close()

	// 升级到TLS连接
	tlsConn := tls.Client(conn, tlsConfig)
	err = tlsConn.Handshake()
	if err != nil {
		return nil, fmt.Errorf("TLS handshake failed: %v", err)
	}

	// 手动发送DNS查询
	dnsConn := &dns.Conn{Conn: tlsConn}
	err = dnsConn.WriteMsg(msg)
	if err != nil {
		return nil, fmt.Errorf("failed to write DNS message: %v", err)
	}

	// 读取响应
	response, err := dnsConn.ReadMsg()
	if err != nil {
		return nil, fmt.Errorf("failed to read DNS response: %v", err)
	}

	return response, nil
}

// queryDoT DNS over TLS查询
func (c *Client) queryDoT(ctx context.Context, msg *dns.Msg, server string) (*dns.Msg, error) {
	tlsConfig := c.config.TLSConfig
	if tlsConfig == nil {
		tlsConfig = &tls.Config{}
	}

	client := &dns.Client{
		Net:       "tcp-tls",
		Timeout:   c.config.Timeout,
		TLSConfig: tlsConfig,
	}

	// 确保端口
	if !strings.Contains(server, ":") {
		server += ":853"
	}

	var lastErr error
	for i := 0; i <= c.config.Retries; i++ {
		var response *dns.Msg
		var err error

		if c.config.ProxyType != NoProxy {
			// 使用代理连接（DoT需要特殊处理）
			response, err = c.exchangeDoTWithProxy(ctx, msg, server, tlsConfig)
		} else {
			// 直接连接
			response, _, err = client.ExchangeContext(ctx, msg, server)
		}

		if err == nil {
			return response, nil
		}
		lastErr = err

		if i < c.config.Retries {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(time.Millisecond * 100 * time.Duration(i+1)):
			}
		}
	}

	return nil, lastErr
}

// queryDoH DNS over HTTPS查询
func (c *Client) queryDoH(ctx context.Context, msg *dns.Msg, server string) (*dns.Msg, error) {
	msgBytes, err := msg.Pack()
	if err != nil {
		return nil, fmt.Errorf("failed to pack DNS message: %v", err)
	}

	// 构建DoH URL
	dohURL := server
	if !strings.HasPrefix(server, "http") {
		dohURL = "https://" + server + "/dns-query"
	}

	// 使用GET方法（RFC 8484）
	u, err := url.Parse(dohURL)
	if err != nil {
		return nil, fmt.Errorf("invalid DoH URL: %v", err)
	}

	q := u.Query()
	q.Set("dns", base64.RawURLEncoding.EncodeToString(msgBytes))
	u.RawQuery = q.Encode()

	httpClient := c.config.HTTPClient
	if httpClient == nil {
		transport := &http.Transport{
			TLSClientConfig: c.config.TLSConfig,
		}

		// 设置代理
		if c.config.ProxyType != NoProxy {
			proxyURL, err := c.getProxyURL()
			if err != nil {
				return nil, fmt.Errorf("failed to get proxy URL: %v", err)
			}
			transport.Proxy = http.ProxyURL(proxyURL)
		}

		httpClient = &http.Client{
			Transport: transport,
			Timeout:   c.config.Timeout,
		}
	}

	var lastErr error
	for i := 0; i <= c.config.Retries; i++ {
		req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create HTTP request: %v", err)
		}

		req.Header.Set("Accept", "application/dns-message")
		req.Header.Set("Content-Type", "application/dns-message")

		resp, err := httpClient.Do(req)
		if err != nil {
			lastErr = err
			if i < c.config.Retries {
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(time.Millisecond * 100 * time.Duration(i+1)):
					continue
				}
			}
			continue
		}

		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("HTTP error: %d %s", resp.StatusCode, resp.Status)
			if i < c.config.Retries {
				continue
			}
			break
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			lastErr = fmt.Errorf("failed to read response body: %v", err)
			if i < c.config.Retries {
				continue
			}
			break
		}

		response := new(dns.Msg)
		if err := response.Unpack(body); err != nil {
			lastErr = fmt.Errorf("failed to unpack DNS response: %v", err)
			if i < c.config.Retries {
				continue
			}
			break
		}

		return response, nil
	}

	return nil, lastErr
}

// createDialer 创建代理拨号器
func (c *Client) createDialer() (proxy.Dialer, error) {
	switch c.config.ProxyType {
	case SOCKS5:
		var auth *proxy.Auth
		if c.config.ProxyAuth != nil {
			auth = &proxy.Auth{
				User:     c.config.ProxyAuth.Username,
				Password: c.config.ProxyAuth.Password,
			}
		}
		return proxy.SOCKS5("tcp", c.config.ProxyAddr, auth, proxy.Direct)
	default:
		return nil, fmt.Errorf("unsupported proxy type for dialer: %s", c.config.ProxyType)
	}
}

// getProxyURL 获取代理URL
func (c *Client) getProxyURL() (*url.URL, error) {
	switch c.config.ProxyType {
	case HTTPProxy:
		proxyURL := c.config.ProxyAddr
		if !strings.HasPrefix(proxyURL, "http") {
			proxyURL = "http://" + proxyURL
		}

		u, err := url.Parse(proxyURL)
		if err != nil {
			return nil, err
		}

		if c.config.ProxyAuth != nil {
			u.User = url.UserPassword(c.config.ProxyAuth.Username, c.config.ProxyAuth.Password)
		}

		return u, nil
	case SOCKS5:
		proxyURL := "socks5://" + c.config.ProxyAddr
		u, err := url.Parse(proxyURL)
		if err != nil {
			return nil, err
		}

		if c.config.ProxyAuth != nil {
			u.User = url.UserPassword(c.config.ProxyAuth.Username, c.config.ProxyAuth.Password)
		}

		return u, nil
	default:
		return nil, fmt.Errorf("unsupported proxy type: %s", c.config.ProxyType)
	}
}

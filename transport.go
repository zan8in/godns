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

	"github.com/miekg/dns"
	"golang.org/x/net/proxy"
)

// queryUDPTCP UDP/TCP查询 - 简化版
func (c *Client) queryUDPTCP(ctx context.Context, msg *dns.Msg, server string) (*dns.Msg, error) {
	client := &dns.Client{
		Net:     string(c.config.Protocol),
		Timeout: c.config.Timeout,
	}

	return c.withRetry(ctx, func() (*dns.Msg, error) {
		if c.config.ProxyType != NoProxy {
			return c.exchangeWithProxy(ctx, msg, server)
		}
		response, _, err := client.ExchangeContext(ctx, msg, server)
		return response, err
	})
}

// queryDoT DoT查询 - 简化版
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

	return c.withRetry(ctx, func() (*dns.Msg, error) {
		if c.config.ProxyType != NoProxy {
			return c.exchangeDoTWithProxy(ctx, msg, server, tlsConfig)
		}
		response, _, err := client.ExchangeContext(ctx, msg, server)
		return response, err
	})
}

// queryDoH DoH查询 - 简化版
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

	return c.withRetry(ctx, func() (*dns.Msg, error) {
		req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create HTTP request: %v", err)
		}

		req.Header.Set("Accept", "application/dns-message")
		req.Header.Set("Content-Type", "application/dns-message")

		resp, err := httpClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("HTTP error: %d %s", resp.StatusCode, resp.Status)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read response body: %v", err)
		}

		response := new(dns.Msg)
		if err := response.Unpack(body); err != nil {
			return nil, fmt.Errorf("failed to unpack DNS response: %v", err)
		}

		return response, nil
	})
}

// exchangeWithProxy 通过代理进行DNS查询
func (c *Client) exchangeWithProxy(ctx context.Context, msg *dns.Msg, server string) (*dns.Msg, error) {
	proxyDialer, err := c.createDialer()
	if err != nil {
		return nil, fmt.Errorf("failed to create proxy dialer: %v", err)
	}

	// 使用 context 控制连接超时
	conn, err := proxyDialer.Dial("tcp", server)
	if err != nil {
		return nil, fmt.Errorf("failed to dial through proxy: %v", err)
	}
	defer conn.Close()

	// 设置连接的读写超时
	if deadline, ok := ctx.Deadline(); ok {
		conn.SetDeadline(deadline)
	}

	// 使用 goroutine 和 channel 来支持 context 取消
	type result struct {
		response *dns.Msg
		err      error
	}

	resultChan := make(chan result, 1)

	go func() {
		// 手动发送DNS查询
		dnsConn := &dns.Conn{Conn: conn}
		err := dnsConn.WriteMsg(msg)
		if err != nil {
			resultChan <- result{nil, fmt.Errorf("failed to write DNS message: %v", err)}
			return
		}

		// 读取响应
		response, err := dnsConn.ReadMsg()
		if err != nil {
			resultChan <- result{nil, fmt.Errorf("failed to read DNS response: %v", err)}
			return
		}

		resultChan <- result{response, nil}
	}()

	// 等待结果或 context 取消
	select {
	case res := <-resultChan:
		return res.response, res.err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
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

	// 设置连接的读写超时
	if deadline, ok := ctx.Deadline(); ok {
		conn.SetDeadline(deadline)
	}

	// 使用 goroutine 和 channel 来支持 context 取消
	type result struct {
		response *dns.Msg
		err      error
	}

	resultChan := make(chan result, 1)

	go func() {
		// 升级到TLS连接
		tlsConn := tls.Client(conn, tlsConfig)
		err := tlsConn.Handshake()
		if err != nil {
			resultChan <- result{nil, fmt.Errorf("TLS handshake failed: %v", err)}
			return
		}

		// 手动发送DNS查询
		dnsConn := &dns.Conn{Conn: tlsConn}
		err = dnsConn.WriteMsg(msg)
		if err != nil {
			resultChan <- result{nil, fmt.Errorf("failed to write DNS message: %v", err)}
			return
		}

		// 读取响应
		response, err := dnsConn.ReadMsg()
		if err != nil {
			resultChan <- result{nil, fmt.Errorf("failed to read DNS response: %v", err)}
			return
		}

		resultChan <- result{response, nil}
	}()

	// 等待结果或 context 取消
	select {
	case res := <-resultChan:
		return res.response, res.err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
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

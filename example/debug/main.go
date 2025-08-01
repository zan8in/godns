package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"time"

	"github.com/miekg/dns"
	"github.com/zan8in/godns"
)

func main() {

	// 示例6: DoT SOCKS5 代理示例
	fmt.Println("\n=== DoT SOCKS5 代理示例 ===")

	// 创建正确的TLS配置
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true, // 跳过证书验证
	}

	client6 := godns.New(
		godns.WithProtocol(godns.DoT),
		godns.WithSOCKS5Proxy("127.0.0.1:20170", nil), // 无认证
		godns.WithTLSConfig(tlsConfig),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	result, err := client6.MultiQuery(ctx, "deepseek.com", dns.TypeA)
	if err != nil {
		log.Printf("SOCKS5 代理查询失败: %v", err)
	} else {
		fmt.Println(result)
		if len(result.AllIPs) == 0 {
			fmt.Printf("  DoT 可能不支持 SOCKS5 代理\n")
		} else {
			fmt.Printf("通过代理查询成功: %s\n", result.Domain)
			for _, ip := range result.AllIPs {
				fmt.Printf("  %s -> %s\n", result.Domain, ip)
			}
		}
	}

}

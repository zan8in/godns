package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/zan8in/godns"
)

func main() {

	// 示例1: 使用默认配置
	fmt.Println("=== 默认配置示例 UDP ===")
	client1 := godns.NewDefault()

	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()

	result, err := client1.MultiQueryA(ctx, "deepseek.com")
	if err != nil {
		log.Printf("查询失败: %v", err)
	} else {
		fmt.Printf("域名: %s\n", result.Domain)
		for _, ip := range result.AllIPs {
			fmt.Printf("  %s -> %s\n", result.Domain, ip)
		}
	}

	// 示例2: 自定义配置
	fmt.Println("\n=== 自定义配置示例 DoT ===")
	client2 := godns.New(
		godns.WithTimeout(3*time.Second),
		godns.WithProtocol(godns.DoT),
		godns.WithRetries(2),
	)

	ctx, cancel = context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	result, err = client2.MultiQueryA(ctx, "deepseek.com")
	if err != nil {
		log.Printf("查询失败: %v", err)
	} else {
		fmt.Printf("域名: %s\n", result.Domain)
		for _, ip := range result.AllIPs {
			fmt.Printf("  %s -> %s\n", result.Domain, ip)
		}
	}

	// 示例3: 自定义配置
	fmt.Println("\n=== 自定义配置示例 DoH ===")
	client3 := godns.New(
		godns.WithTimeout(3*time.Second),
		godns.WithProtocol(godns.DoH),
		godns.WithRetries(2),
	)

	ctx, cancel = context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	result, err = client3.MultiQueryA(ctx, "deepseek.com")
	if err != nil {
		log.Printf("查询失败: %v", err)
	} else {
		fmt.Printf("域名: %s\n", result.Domain)
		for _, ip := range result.AllIPs {
			fmt.Printf("  %s -> %s\n", result.Domain, ip)
		}
	}

	// 示例4: UDP SOCKS5代理示例
	fmt.Println("\n=== UDP SOCKS5代理示例 ===")
	client4 := godns.New(
		godns.WithSOCKS5Proxy("127.0.0.1:20170", nil), // 无认证
	)

	ctx, cancel = context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	result, err = client4.MultiQueryA(ctx, "deepseek.com")
	if err != nil {
		log.Printf("SOCKS5 代理查询失败: %v", err)
	} else {
		fmt.Printf("通过代理查询成功: %s\n", result.Domain)
		for _, ip := range result.AllIPs {
			fmt.Printf("  %s -> %s\n", result.Domain, ip)
		}
	}

	// 示例5: UDP HTTP 代理示例
	fmt.Println("\n=== UDP HTTP 代理示例 ===")
	client5 := godns.New(
		godns.WithHTTPProxy("127.0.0.1:20170", nil), // 无认证
	)

	ctx, cancel = context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	result, err = client5.MultiQueryA(ctx, "deepseek.com")
	if err != nil {
		log.Printf("HTTP 代理查询失败: %v", err)
	} else {
		if len(result.AllIPs) == 0 {
			fmt.Printf("  UDP 可能不支持 HTTP 代理\n")
		} else {
			fmt.Printf("通过代理查询成功: %s\n", result.Domain)
			for _, ip := range result.AllIPs {
				fmt.Printf("  %s -> %s\n", result.Domain, ip)
			}
		}
	}

	// 示例6: DoT SOCKS5 代理示例
	fmt.Println("\n=== DoT SOCKS5 代理示例 ===")
	client6 := godns.New(
		godns.WithProtocol(godns.DoT),
		godns.WithSOCKS5Proxy("127.0.0.1:20170", nil), // 无认证
	)

	ctx, cancel = context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	result, err = client6.MultiQueryA(ctx, "deepseek.com")
	if err != nil {
		log.Printf("SOCKS5 代理查询失败: %v", err)
	} else {
		if len(result.AllIPs) == 0 {
			fmt.Printf("  DoT 可能不支持 SOCKS5 代理\n")
		} else {
			fmt.Printf("通过代理查询成功: %s\n", result.Domain)
			for _, ip := range result.AllIPs {
				fmt.Printf("  %s -> %s\n", result.Domain, ip)
			}
		}
	}

	// 示例7: DoT HTTP 代理示例
	fmt.Println("\n=== DoT HTTP 代理示例 ===")
	client7 := godns.New(
		godns.WithProtocol(godns.DoT),
		godns.WithHTTPProxy("127.0.0.1:20170", nil), // 无认证
	)

	ctx, cancel = context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	result, err = client7.MultiQueryA(ctx, "deepseek.com")
	if err != nil {
		log.Printf("HTTP 代理查询失败: %v", err)
	} else {
		if len(result.AllIPs) == 0 {
			fmt.Printf("  DoT 可能不支持 HTTP 代理\n")
		} else {
			fmt.Printf("通过代理查询成功: %s\n", result.Domain)
			for _, ip := range result.AllIPs {
				fmt.Printf("  %s -> %s\n", result.Domain, ip)
			}
		}
	}

	// 示例8: DoH SOCK5 代理示例
	fmt.Println("\n=== DoH SOCK5 代理示例 ===")
	client8 := godns.New(
		godns.WithProtocol(godns.DoH),
		godns.WithSOCKS5Proxy("127.0.0.1:20170", nil), // 无认证
	)

	ctx, cancel = context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	result, err = client8.MultiQueryA(ctx, "deepseek.com")
	if err != nil {
		log.Printf("SOCK5 代理查询失败: %v", err)
	} else {
		if len(result.AllIPs) == 0 {
			fmt.Printf("  DoH 可能不支持 SOCK5 代理\n")
		} else {
			fmt.Printf("通过代理查询成功: %s\n", result.Domain)
			for _, ip := range result.AllIPs {
				fmt.Printf("  %s -> %s\n", result.Domain, ip)
			}
		}
	}

	// 示例9: DoH HTTP 代理示例
	fmt.Println("\n=== DoH HTTP 代理示例 ===")
	client9 := godns.New(
		godns.WithProtocol(godns.DoH),
		godns.WithHTTPProxy("127.0.0.1:20170", nil), // 无认证
	)

	ctx, cancel = context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	result, err = client9.MultiQueryA(ctx, "deepseek.com")
	if err != nil {
		log.Printf("HTTP 代理查询失败: %v", err)
	} else {
		if len(result.AllIPs) == 0 {
			fmt.Printf("  DoH 可能不支持 HTTP 代理\n")
		} else {
			fmt.Printf("通过代理查询成功: %s\n", result.Domain)
			for _, ip := range result.AllIPs {
				fmt.Printf("  %s -> %s\n", result.Domain, ip)
			}
		}
	}

}

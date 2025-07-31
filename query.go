package godns

import (
    "context"
    "fmt"
    "net"
    "strings"

    "github.com/miekg/dns"
)

// QueryResult 查询结果
type QueryResult struct {
    Domain   string
    Type     uint16
    Records  []Record
    Error    error
    Server   string
}

// Record DNS记录
type Record struct {
    Name  string
    Type  uint16
    TTL   uint32
    Value string
}

// MultiQueryResult 多DNS查询结果
type MultiQueryResult struct {
    Domain  string
    Type    uint16
    Results []QueryResult
    AllIPs  []string // 所有查询到的IP地址
}

// Query 单个DNS查询
func (c *Client) Query(ctx context.Context, domain string, qtype uint16) (*QueryResult, error) {
    if len(c.config.Servers) == 0 {
        return nil, fmt.Errorf("no DNS servers configured")
    }
    
    return c.queryServer(ctx, domain, qtype, c.config.Servers[0])
}

// QueryA 查询A记录
func (c *Client) QueryA(ctx context.Context, domain string) (*QueryResult, error) {
    return c.Query(ctx, domain, dns.TypeA)
}

// QueryAAAA 查询AAAA记录
func (c *Client) QueryAAAA(ctx context.Context, domain string) (*QueryResult, error) {
    return c.Query(ctx, domain, dns.TypeAAAA)
}

// QueryCNAME 查询CNAME记录
func (c *Client) QueryCNAME(ctx context.Context, domain string) (*QueryResult, error) {
    return c.Query(ctx, domain, dns.TypeCNAME)
}

// QueryMX 查询MX记录
func (c *Client) QueryMX(ctx context.Context, domain string) (*QueryResult, error) {
    return c.Query(ctx, domain, dns.TypeMX)
}

// QueryTXT 查询TXT记录
func (c *Client) QueryTXT(ctx context.Context, domain string) (*QueryResult, error) {
    return c.Query(ctx, domain, dns.TypeTXT)
}

// MultiQuery 多DNS服务器查询
func (c *Client) MultiQuery(ctx context.Context, domain string, qtype uint16) (*MultiQueryResult, error) {
    if len(c.config.Servers) == 0 {
        return nil, fmt.Errorf("no DNS servers configured")
    }
    
    result := &MultiQueryResult{
        Domain:  domain,
        Type:    qtype,
        Results: make([]QueryResult, 0, len(c.config.Servers)),
        AllIPs:  make([]string, 0),
    }
    
    // 并发查询所有DNS服务器
    resultChan := make(chan QueryResult, len(c.config.Servers))
    
    for _, server := range c.config.Servers {
        go func(srv string) {
            res, err := c.queryServer(ctx, domain, qtype, srv)
            if res == nil {
                res = &QueryResult{
                    Domain: domain,
                    Type:   qtype,
                    Server: srv,
                    Error:  err,
                }
            }
            resultChan <- *res
        }(server)
    }
    
    // 收集结果
    ipSet := make(map[string]bool)
    for i := 0; i < len(c.config.Servers); i++ {
        res := <-resultChan
        result.Results = append(result.Results, res)
        
        // 收集所有IP地址
        if res.Error == nil {
            for _, record := range res.Records {
                if (qtype == dns.TypeA || qtype == dns.TypeAAAA) && net.ParseIP(record.Value) != nil {
                    if !ipSet[record.Value] {
                        ipSet[record.Value] = true
                        result.AllIPs = append(result.AllIPs, record.Value)
                    }
                }
            }
        }
    }
    
    return result, nil
}

// MultiQueryA 多DNS服务器查询A记录
func (c *Client) MultiQueryA(ctx context.Context, domain string) (*MultiQueryResult, error) {
    return c.MultiQuery(ctx, domain, dns.TypeA)
}

// MultiQueryAAAA 多DNS服务器查询AAAA记录
func (c *Client) MultiQueryAAAA(ctx context.Context, domain string) (*MultiQueryResult, error) {
    return c.MultiQuery(ctx, domain, dns.TypeAAAA)
}

// queryServer 查询指定DNS服务器
func (c *Client) queryServer(ctx context.Context, domain string, qtype uint16, server string) (*QueryResult, error) {
    msg := new(dns.Msg)
    msg.SetQuestion(dns.Fqdn(domain), qtype)
    msg.RecursionDesired = true
    
    var response *dns.Msg
    var err error
    
    switch c.config.Protocol {
    case UDP, TCP:
        response, err = c.queryUDPTCP(ctx, msg, server)
    case DoT:
        response, err = c.queryDoT(ctx, msg, server)
    case DoH:
        response, err = c.queryDoH(ctx, msg, server)
    default:
        return nil, fmt.Errorf("unsupported protocol: %s", c.config.Protocol)
    }
    
    if err != nil {
        return &QueryResult{
            Domain: domain,
            Type:   qtype,
            Server: server,
            Error:  err,
        }, err
    }
    
    records := make([]Record, 0, len(response.Answer))
    for _, rr := range response.Answer {
        record := Record{
            Name: rr.Header().Name,
            Type: rr.Header().Rrtype,
            TTL:  rr.Header().Ttl,
        }
        
        switch v := rr.(type) {
        case *dns.A:
            record.Value = v.A.String()
        case *dns.AAAA:
            record.Value = v.AAAA.String()
        case *dns.CNAME:
            record.Value = v.Target
        case *dns.MX:
            record.Value = fmt.Sprintf("%d %s", v.Preference, v.Mx)
        case *dns.TXT:
            record.Value = strings.Join(v.Txt, " ")
        default:
            record.Value = rr.String()
        }
        
        records = append(records, record)
    }
    
    return &QueryResult{
        Domain:  domain,
        Type:    qtype,
        Records: records,
        Server:  server,
    }, nil
}
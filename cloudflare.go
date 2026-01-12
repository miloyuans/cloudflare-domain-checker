package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/cloudflare/cloudflare-go"
)

// DomainInfo 结构体表示CSV文件中的一行数据。
type DomainInfo struct {
	AccountName       string `csv:"账户名称"`      // 账户名字
	Domain            string `csv:"域"`         // 域
	DomainStatus      string `csv:"域状态"`       // 域的状态 (e.g., active, pending)
	DNSRecordName     string `csv:"域名解析名称"`    // 域名解析名称 (e.g., www, @)
	DNSRecordType     string `csv:"解析类型"`      // 解析类型 (e.g., A, CNAME)
	DNSRecordContent  string `csv:"解析内容"`      // 解析内容 (e.g., IP address, target domain)
	Notes             string `csv:"备注"`        // 备注 (Cloudflare API不直接提供此字段，此处为空白)
	ProxyStatus       string `csv:"代理状态"`      // 代理状态 (是否开启Cloudflare代理)
	TLSEncryptionMode string `csv:"TLS加密模式"`   // TLS的加密模式 (e.g., flexible, full, strict)
	DomainNSInfo      string `csv:"域NS信息"`      // 域的NS信息
}

// ZoneSummary 结构体用于存储单个账户的域名统计信息。
type ZoneSummary struct {
	TotalZones          int            // 总计域名数
	StatusCounts        map[string]int // 按状态分类的域名数量 (e.g., "active": 5, "pending": 2)
	DomainsWithDNSRecords int            // 至少有一个DNS记录的域名数量
}

// GetCloudflareData 从给定的Cloudflare账户中获取域名和DNS记录信息。
// 返回所有DomainInfo记录以及一个ZoneSummary统计报告。
func GetCloudflareData(accountName, apiToken string) ([]DomainInfo, *ZoneSummary, error) {
	// 初始化Cloudflare API客户端
	api, err := cloudflare.NewWithAPIToken(apiToken)
	if err != nil {
		return nil, nil, fmt.Errorf("为账户 '%s' 创建Cloudflare API客户端失败: %w", accountName, err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second) // 增加超时时间，避免API调用过慢
	defer cancel()

	var allDomainInfo []DomainInfo // 存储所有获取到的域名信息
	summary := &ZoneSummary{
		StatusCounts: make(map[string]int),
	}

	// 1. 获取账户下的所有域名 (Zone)
	// 适配旧版API：使用 cloudflare.ListZonesOptions
	zoneOptions := cloudflare.ListZonesOptions{PerPage: 50} // 每页最多50个Zone
	page := 1
	for {
		zoneOptions.Page = page
		// 适配旧版API：ListZonesContext 返回 cloudflare.ZonesResponse 和 error
		// 并且 ListZonesContext 可能需要 AccountID 或其他过滤器，这里我们简化，仅使用分页
		// 如果你想筛选特定账户，你可能需要额外的 AccountID 逻辑来构建 ListZonesOptions 或 WithZoneFilters
		zonesResp, err := api.ListZonesContext(ctx, zoneOptions) // 传递 options struct
		if err != nil {
			return nil, nil, fmt.Errorf("为账户 '%s' 列出域名失败 (第 %d 页): %w", accountName, page, err)
		}

		zones := zonesResp.Result      // 获取 Zones 列表
		resInfo := zonesResp.ResultInfo // 获取分页信息

		if len(zones) == 0 {
			break // 没有更多域名了
		}

		summary.TotalZones += len(zones)

		for _, zone := range zones {
			summary.StatusCounts[string(zone.Status)]++ // 统计域名状态

			// 获取域的NS信息
			nsInfo := "未提供"
			if len(zone.NameServers) > 0 {
				nsInfo = strings.Join(zone.NameServers, ", ")
			}

			// 获取域的TLS加密模式
			tlsMode := "未知"
			// 注意：旧版本 zone.SSL 可能是一个字符串指针或直接是字符串，需要根据具体库版本调整
			if zone.SSL != nil {
				tlsMode = string(*zone.SSL) // 假设它仍然是指针
			} else if zone.SSLSetting != nil { // 有些旧版本可能是 SSLSetting
				tlsMode = zone.SSLSetting.Value
			}


			// 2. 获取每个域下的所有DNS解析记录
			// 适配旧版API：使用 cloudflare.ListDNSRecordsOptions
			dnsRecordOptions := cloudflare.ListDNSRecordsOptions{PerPage: 100} // 每页最多100个DNS记录
			dnsPage := 1
			foundDNSRecordsForZone := false // 标记当前zone是否至少有一个DNS记录
			for {
				dnsRecordOptions.Page = dnsPage
				// 适配旧版API：ListDNSRecords 返回 cloudflare.DNSRecordsResponse 和 error
				dnsRecordsResp, err := api.ListDNSRecords(ctx, zone.ID, dnsRecordOptions) // 传递 zone ID 和 options struct
				if err != nil {
					log.Printf("警告: 无法为账户 '%s' 的域名 '%s' (%s) 列出DNS记录: %v", accountName, zone.Name, zone.ID, err)
					break // 如果获取DNS记录失败，跳过此域的DNS记录处理
				}

				records := dnsRecordsResp.Result      // 获取 DNS 记录列表
				dnsResInfo := dnsRecordsResp.ResultInfo // 获取分页信息

				if len(records) > 0 {
					foundDNSRecordsForZone = true
				}

				for _, record := range records {
					proxyStatus := "否"
					if record.Proxied != nil && *record.Proxied {
						proxyStatus = "是"
					}

					allDomainInfo = append(allDomainInfo, DomainInfo{
						AccountName:       accountName,
						Domain:            zone.Name,
						DomainStatus:      string(zone.Status),
						DNSRecordName:     record.Name,
						DNSRecordType:     string(record.Type),
						DNSRecordContent:  record.Content,
						Notes:             "", // Cloudflare API不直接提供备注字段
						ProxyStatus:       proxyStatus,
						TLSEncryptionMode: tlsMode,
						DomainNSInfo:      nsInfo,
					})
				}
                // 适配旧版API：使用 dnsResInfo 来判断分页
				if dnsResInfo.Page >= dnsResInfo.TotalPages {
					break // 没有更多DNS记录页了
				}
				dnsPage++
			}
			if foundDNSRecordsForZone {
				summary.DomainsWithDNSRecords++
			}
		}

        // 适配旧版API：使用 resInfo 来判断分页
		if resInfo.Page >= resInfo.TotalPages {
			break // 没有更多域名页了
		}
		page++
	}

	return allDomainInfo, summary, nil
}

// 示例：主函数如何调用 GetCloudflareData
func main() {
    // 假设你有 API Token 和一个账户名
    // 实际应用中这些值可能来自环境变量、配置文件或命令行参数
    myAccountName := "My Cloudflare Account" // 给你的账户起个名字
    myAPIToken := os.Getenv("CF_API_TOKEN") // 建议通过环境变量设置

    if myAPIToken == "" {
        log.Fatal("请设置 CF_API_TOKEN 环境变量")
    }

    fmt.Printf("正在从账户 '%s' 获取 Cloudflare 数据...\n", myAccountName)
    domainData, summary, err := GetCloudflareData(myAccountName, myAPIToken)
    if err != nil {
        log.Fatalf("获取 Cloudflare 数据失败: %v", err)
    }

    fmt.Printf("\n--- 数据获取成功 ---\n")
    fmt.Printf("总计域名数: %d\n", summary.TotalZones)
    fmt.Printf("域名状态统计: %+v\n", summary.StatusCounts)
    fmt.Printf("有DNS记录的域名数: %d\n", summary.DomainsWithDNSRecords)
    fmt.Printf("获取到 %d 条 DNS 记录详情。\n", len(domainData))

    // 打印前几条记录作为示例
    if len(domainData) > 0 {
        fmt.Println("\n--- 部分 DNS 记录详情 ---")
        for i, info := range domainData {
            if i >= 5 { // 只打印前5条
                break
            }
            fmt.Printf("  域名: %s, 类型: %s, 内容: %s, 代理: %s\n",
                info.Domain, info.DNSRecordType, info.DNSRecordContent, info.ProxyStatus)
        }
    }
}

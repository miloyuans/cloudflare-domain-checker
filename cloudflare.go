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
	TotalZones      int            // 总计域名数
	StatusCounts    map[string]int // 按状态分类的域名数量 (e.g., "active": 5, "pending": 2)
	DomainsWithDNSRecords int // 至少有一个DNS记录的域名数量
}

// GetCloudflareData 从给定的Cloudflare账户中获取域名和DNS记录信息。
// 返回所有DomainInfo记录以及一个ZoneSummary统计报告。
func GetCloudflareData(accountName, apiToken string) ([]DomainInfo, *ZoneSummary, error) {
	// 初始化Cloudflare API客户端
	api, err := cloudflare.NewWithAPIToken(apiToken)
	if err != nil {
		return nil, nil, fmt.Errorf("为账户 '%s' 创建Cloudflare API客户端失败: %w", accountName, err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second) // 设置API请求超时
	defer cancel()

	var allDomainInfo []DomainInfo // 存储所有获取到的域名信息
	summary := &ZoneSummary{
		StatusCounts: make(map[string]int),
	}

	// 1. 获取账户下的所有域名 (Zone)
	zonesOptions := cloudflare.ListZonesOptions{PerPage: 50} // 每页最多50个Zone
	page := 1
	for {
		zonesOptions.Page = page
		zones, res, err := api.ListZones(ctx, zonesOptions)
		if err != nil {
			return nil, nil, fmt.Errorf("为账户 '%s' 列出域名失败 (第 %d 页): %w", accountName, page, err)
		}

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
			if zone.SSL != nil {
				tlsMode = string(*zone.SSL)
			}

			// 2. 获取每个域下的所有DNS解析记录
			dnsRecordsOptions := cloudflare.ListDNSRecordsOptions{PerPage: 100} // 每页最多100个DNS记录
			dnsPage := 1
			foundDNSRecordsForZone := false // 标记当前zone是否至少有一个DNS记录
			for {
				dnsRecordsOptions.Page = dnsPage
				records, dnsRes, err := api.ListDNSRecords(ctx, cloudflare.ZoneIdentifier(zone.ID), dnsRecordsOptions)
				if err != nil {
					log.Printf("警告: 无法为账户 '%s' 的域名 '%s' (%s) 列出DNS记录: %v", accountName, zone.Name, zone.ID, err)
					break // 如果获取DNS记录失败，跳过此域的DNS记录处理
				}

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

				if dnsRes.ResultInfo.Page >= dnsRes.ResultInfo.TotalPages {
					break // 没有更多DNS记录页了
				}
				dnsPage++
			}
			if foundDNSRecordsForZone {
				summary.DomainsWithDNSRecords++
			}
		}

		if res.ResultInfo.Page >= res.ResultInfo.TotalPages {
			break // 没有更多域名页了
		}
		page++
	}

	return allDomainInfo, summary, nil
}

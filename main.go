package main

import (
	//"fmt" // <-- 如果确实没用到 fmt.Sprintf 或其他 fmt 函数，就删除
	"log"
	"encoding/json"
	//"os"  // <-- 如果确实没用到 os.Exit 或其他 os 函数，就删除
	//"time"// <-- 如果确实没用到 time.Sleep 或其他 time 函数，就删除
)


const (
	configFile  = "config.json"       // 配置文件名
	csvFileName = "cloudflare_domains.csv" // 输出的CSV文件名
)

func main() {
	log.Println("--- Cloudflare 域名查询程序启动 ---")

	// 1. 加载配置
	cfg, err := LoadConfig(configFile)
	if err != nil {
		log.Fatalf("错误: 无法加载配置文件 '%s': %v", configFile, err)
	}
	log.Printf("成功加载配置文件 '%s'。", configFile)

	var allDomainData []DomainInfo                   // 存储所有账户的所有域名信息
	allAccountSummaries := make(map[string]*ZoneSummary) // 存储所有账户的汇总统计
	processedAccounts := 0                           // 成功处理的账户数量

	// 2. 遍历每个Cloudflare账户，获取数据
	for _, account := range cfg.CloudflareAccounts {
		log.Printf("正在从 Cloudflare 账户 '%s' 获取域名信息...", account.Name)
		data, summary, err := GetCloudflareData(account.Name, account.APIToken)
		if err != nil {
			log.Printf("警告: 无法获取账户 '%s' 的数据: %v", account.Name, err)
			continue // 即使某个账户失败，也继续处理下一个账户
		}
		allDomainData = append(allDomainData, data...)
		allAccountSummaries[account.Name] = summary
		processedAccounts++
		log.Printf("账户 '%s' 处理完成。总计找到 %d 个域名，其中有DNS记录的域名 %d 个。", account.Name, summary.TotalZones, summary.DomainsWithDNSRecords)
	}

	if processedAccounts == 0 {
		log.Fatal("错误: 未能成功处理任何 Cloudflare 账户。请检查配置和API令牌。")
	}

	if len(allDomainData) == 0 {
		log.Println("没有找到任何域名解析记录。CSV文件将只包含头部。")
	} else {
		log.Printf("总共找到了 %d 条域名解析记录。", len(allDomainData))
	}

	// 3. 将数据写入CSV文件
	log.Printf("正在将数据写入 CSV 文件: '%s'...", csvFileName)
	err = WriteToCSV(csvFileName, allDomainData)
	if err != nil {
		log.Fatalf("错误: 无法写入 CSV 文件 '%s': %v", csvFileName, err)
	}
	log.Printf("数据已成功写入到 '%s'。", csvFileName)

	// 4. 发送Telegram通知
	if cfg.TelegramConfig.BotToken != "" && cfg.TelegramConfig.ChatID != "" {
		log.Println("正在发送 Telegram 通知...")
		message := "Cloudflare 域名信息每日报告"
		err := SendTelegramNotification(
			cfg.TelegramConfig.BotToken,
			cfg.TelegramConfig.ChatID,
			message,
			csvFileName,
			allAccountSummaries,
		)
		if err != nil {
			log.Printf("警告: 无法发送 Telegram 通知: %v", err)
		} else {
			log.Println("Telegram 通知发送成功。")
		}
	} else {
		log.Println("Telegram 配置不完整（bot_token或chat_id缺失），跳过发送通知。")
	}

	log.Println("--- 程序执行完毕 ---")
}

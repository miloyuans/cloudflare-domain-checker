package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5" // 如果使用的是v6, 需要修改为v6
)

// SendTelegramNotification 发送汇总消息和CSV文件作为附件到Telegram。
func SendTelegramNotification(botToken, chatIDStr, messageText, csvFilePath string, allAccountSummaries map[string]*ZoneSummary) error {
	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		return fmt.Errorf("创建Telegram Bot API客户端失败: %w", err)
	}

	bot.Debug = false // 设置为 true 可以看到更多来自bot库的调试日志

	log.Printf("Telegram机器人已授权，账户名: @%s", bot.Self.UserName)

	chatID, err := strconv.ParseInt(chatIDStr, 10, 64)
	if err != nil {
		return fmt.Errorf("无效的Telegram聊天ID '%s': %w", chatIDStr, err)
	}

	// 1. 构建并发送汇总消息
	summaryMessage := buildSummaryMessage(messageText, allAccountSummaries)
	msg := tgbotapi.NewMessage(chatID, summaryMessage)
	msg.ParseMode = tgbotapi.ModeMarkdownV2 // 使用MarkdownV2模式进行格式化
	_, err = bot.Send(msg)
	if err != nil {
		return fmt.Errorf("发送Telegram汇总消息失败: %w", err)
	}
	log.Println("Telegram汇总消息发送成功。")

	// 2. 发送CSV文件作为文档附件
	file, err := os.Open(csvFilePath)
	if err != nil {
		return fmt.Errorf("打开CSV文件 '%s' 失败，无法作为Telegram附件发送: %w", csvFilePath, err)
	}
	defer file.Close()

	document := tgbotapi.NewDocument(chatID, tgbotapi.FileReader{
		Name:   "cloudflare_domains.csv", // Telegram中显示的文件名
		Reader: file,
		// 修正：移除 Size 字段，最新版本库会自动处理
	})
	document.Caption = "Cloudflare 域名信息汇总报告" // 文件附件的标题
	_, err = bot.Send(document)
	if err != nil {
		return fmt.Errorf("发送CSV文件到Telegram失败: %w", err)
	}
	log.Println("Telegram CSV文件发送成功。")

	return nil
}

// buildSummaryMessage 格式化生成Telegram通知的文本内容。
func buildSummaryMessage(baseMessage string, allAccountSummaries map[string]*ZoneSummary) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("*%s*\n\n", escapeMarkdownV2(baseMessage))) // 标题加粗
	sb.WriteString(fmt.Sprintf("Cloudflare 域名统计报告 \\(%s\\)\n", time.Now().Format("2006-01-02 15:04:05")))
	sb.WriteString("```\n") // 开始预格式化代码块，使排版更整齐

	totalDomainsAcrossAllAccounts := 0
	for accName, summary := range allAccountSummaries {
		sb.WriteString(fmt.Sprintf("账户: %s\n", accName))
		sb.WriteString(fmt.Sprintf("  总计域名数: %d\n", summary.TotalZones))
		totalDomainsAcrossAllAccounts += summary.TotalZones
		sb.WriteString("  按状态分类:\n")
		// 对状态进行排序，以便输出顺序一致
		statuses := make([]string, 0, len(summary.StatusCounts))
		for status := range summary.StatusCounts {
			statuses = append(statuses, status)
		}
		// 可以在此处对 statuses 进行排序，例如 sort.Strings(statuses)

		for _, status := range statuses {
			count := summary.StatusCounts[status]
			sb.WriteString(fmt.Sprintf("    \\- %s: %d 个\n", status, count))
		}
		sb.WriteString(fmt.Sprintf("  有DNS记录的域名数: %d\n", summary.DomainsWithDNSRecords))
		sb.WriteString("\n")
	}
	sb.WriteString(fmt.Sprintf("所有账户总计域名数: %d\n", totalDomainsAcrossAllAccounts))
	sb.WriteString("```\n") // 结束预格式化代码块

	// 对整个消息进行MarkdownV2转义
	return escapeMarkdownV2(sb.String())
}

// escapeMarkdownV2 转义MarkdownV2中具有特殊含义的字符。
// 这是为了防止用户输入或API响应中的特殊字符破坏Markdown格式。
func escapeMarkdownV2(text string) string {
	replacer := strings.NewReplacer(
		"_", "\\_", "*", "\\*", "[", "\\[", "]", "\\]", "(", "\\(", ")", "\\)", "~", "\\~",
		"`", "\\`", ">", "\\>", "#", "\\#", "+", "\\+", "-", "\\-", "=", "\\=", "|", "\\|",
		"{", "\\{", "}", "\\}", ".", "\\.", "!", "\\!",
	)
	return replacer.Replace(text)
}

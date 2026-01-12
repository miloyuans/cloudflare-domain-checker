package main

import (
	"encoding/json"
	"os"
)

// CloudflareAccount 结构体用于存储单个Cloudflare账户的配置信息。
type CloudflareAccount struct {
	Name     string `json:"name"`      // 账户名称
	APIToken string `json:"api_token"` // Cloudflare API Token
}

// TelegramConfig 结构体用于存储Telegram通知的配置信息。
type TelegramConfig struct {
	BotToken string `json:"bot_token"` // Telegram Bot API Token
	ChatID   string `json:"chat_id"`   // Telegram 聊天ID (可以是用户ID或群组ID)
}

// Config 结构体包含整个程序的配置。
type Config struct {
	CloudflareAccounts []CloudflareAccount `json:"cloudflare_accounts"` // 多个Cloudflare账户
	TelegramConfig     TelegramConfig      `json:"telegram_config"`     // Telegram配置
}

// LoadConfig 从指定的文件名加载配置。
func LoadConfig(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

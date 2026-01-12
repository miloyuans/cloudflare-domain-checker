module cloudflare-domain-checker

go 1.22

require (
	github.com/cloudflare/cloudflare-go v0.90.0
	github.com/go-telegram-bot-api/telegram-bot-api/v5 v5.0.2
	golang.org/x/net v0.27.0 // Required by cloudflare-go
)

// 请在运行 `go mod tidy` 后，根据实际生成的 go.sum 内容进行填充
// go.sum 文件内容将由 `go mod tidy` 自动生成，这里不给出示例

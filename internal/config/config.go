package config

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	IMAP struct {
		Host              string   `yaml:"host"`
		Email             string   `yaml:"email"`
		UseTLS            bool     `yaml:"use_tls"`
		Provider          string   `yaml:"provider"`
		OAuthClientID     string   `yaml:"oauth_client_id"`
		OAuthClientSecret string   `yaml:"oauth_client_secret"`
		Folders           []string `yaml:"folders"`
	} `yaml:"imap"`
	MCP struct {
		Endpoint string `yaml:"endpoint"`
		APIKey   string `yaml:"api_key"`
	} `yaml:"mcp"`
	Fetch struct {
		Start     string `yaml:"start"` // YYYY-MM-DD or RFC3339
		End       string `yaml:"end"`   // YYYY-MM-DD or RFC3339
		MaxEmails int    `yaml:"max_emails"`
	} `yaml:"fetch"`
	LLM struct {
		APIBase     string  `yaml:"api_base"`
		APIKey      string  `yaml:"api_key"`
		Model       string  `yaml:"model"`
		Temperature float64 `yaml:"temperature"`
		MaxTokens   int     `yaml:"max_tokens"`
	} `yaml:"llm"`
	Export struct {
		File string `yaml:"file"`
	} `yaml:"export"`
}

// Load 加载配置文件并替换环境变量
func Load(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	// 替换环境变量 ${VAR_NAME} 格式
	content := string(b)
	content = expandEnvVars(content)

	var cfg Config
	if err := yaml.Unmarshal([]byte(content), &cfg); err != nil {
		return nil, fmt.Errorf("parse yaml: %w", err)
	}

	if cfg.IMAP.Email == "" {
		return nil, fmt.Errorf("imap.email is required")
	}

	// 使用MCP协议时，不需要具体的IMAP配置
	// 但仍需要推断邮箱提供商类型用于MCP客户端
	if cfg.IMAP.Provider == "" {
		cfg.IMAP.Provider = inferEmailProvider(cfg.IMAP.Email)
	}

	// 只有在未使用MCP协议时才配置传统IMAP设置
	if cfg.MCP.Endpoint == "" {
		// 自动推断传统IMAP配置（仅在无MCP时使用）
		if cfg.IMAP.Host == "" {
			cfg.IMAP.Host = inferIMAPHost(cfg.IMAP.Email)
			if cfg.IMAP.Host != "" {
				cfg.IMAP.UseTLS = true
			}
		}
	}

	// 默认文件夹配置根据邮箱提供商调整
	if len(cfg.IMAP.Folders) == 0 {
		cfg.IMAP.Folders = getDefaultFolders(cfg.IMAP.Provider)
	}

	// 默认抓取数量
	if cfg.Fetch.MaxEmails <= 0 {
		cfg.Fetch.MaxEmails = 100
	}

	// 默认导出文件
	if cfg.Export.File == "" {
		cfg.Export.File = "emails.csv"
	}

	return &cfg, nil
}

// expandEnvVars 替换 ${VAR_NAME} 格式的环境变量
func expandEnvVars(content string) string {
	re := regexp.MustCompile(`\$\{([^}]+)\}`)
	return re.ReplaceAllStringFunc(content, func(match string) string {
		varName := match[2 : len(match)-1] // 去掉 ${ 和 }
		if value := os.Getenv(varName); value != "" {
			return value
		}
		return match // 如果环境变量不存在，保持原样
	})
}

// inferEmailProvider 根据邮箱地址推断提供商
func inferEmailProvider(email string) string {
	email = strings.ToLower(email)

	if strings.Contains(email, "@gmail.com") || strings.Contains(email, "@googlemail.com") {
		return "gmail"
	}
	if strings.Contains(email, "@outlook.com") || strings.Contains(email, "@hotmail.com") || strings.Contains(email, "@live.com") {
		return "outlook"
	}
	if strings.Contains(email, "@yahoo.com") || strings.Contains(email, "@yahoo.co.") {
		return "yahoo"
	}
	if strings.Contains(email, "@qq.com") || strings.Contains(email, "@163.com") || strings.Contains(email, "@126.com") {
		return "chinese"
	}

	return "custom"
}

// inferIMAPHost 根据邮箱地址推断IMAP主机（仅在无MCP时使用）
func inferIMAPHost(email string) string {
	provider := inferEmailProvider(email)

	switch provider {
	case "gmail":
		return "imap.gmail.com:993"
	case "outlook":
		return "outlook.office365.com:993"
	case "yahoo":
		return "imap.mail.yahoo.com:993"
	default:
		return "" // 需要手动配置
	}
}

// getDefaultFolders 根据邮箱提供商返回默认文件夹
func getDefaultFolders(provider string) []string {
	switch provider {
	case "gmail":
		return []string{"INBOX", "[Gmail]/Sent Mail", "[Gmail]/All Mail"}
	case "outlook":
		return []string{"INBOX", "Sent Items"}
	case "yahoo":
		return []string{"INBOX", "Sent"}
	default:
		return []string{"INBOX"}
	}
}

func hasSuffixInsensitive(s, suf string) bool {
	sLower := strings.ToLower(s)
	sufLower := strings.ToLower(suf)
	return strings.HasSuffix(sLower, sufLower)
}

func ParseDateLoose(s string, def time.Time) time.Time {
	if s == "" {
		return def
	}
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t
	}
	return def
}

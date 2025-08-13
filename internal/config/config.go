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

	// 自动推断 Gmail 配置
	if cfg.IMAP.Host == "" {
		if hasSuffixInsensitive(cfg.IMAP.Email, "@gmail.com") || hasSuffixInsensitive(cfg.IMAP.Email, "@googlemail.com") {
			cfg.IMAP.Host = "imap.gmail.com:993"
			cfg.IMAP.UseTLS = true
		}
	}

	// 默认文件夹
	if len(cfg.IMAP.Folders) == 0 {
		cfg.IMAP.Folders = []string{"INBOX"}
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

package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/YKarmar/JobTracker/internal/types"
)

// MCP协议相关结构体
type MCPRequest struct {
	Jsonrpc string      `json:"jsonrpc"`
	ID      string      `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
}

type MCPResponse struct {
	Jsonrpc string          `json:"jsonrpc"`
	ID      string          `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *MCPError       `json:"error,omitempty"`
}

type MCPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// 邮件提供商枚举
type EmailProvider string

const (
	ProviderGmailMCP EmailProvider = "gmail"
	ProviderOutlook  EmailProvider = "outlook"
	ProviderYahoo    EmailProvider = "yahoo"
	ProviderCustom   EmailProvider = "custom"
)

// MCP邮件客户端配置
type MCPEmailConfig struct {
	Provider    EmailProvider `json:"provider"`
	Email       string        `json:"email"`
	MCPEndpoint string        `json:"mcp_endpoint"`
	APIKey      string        `json:"api_key,omitempty"`
}

// 邮件查询参数
type EmailQuery struct {
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`
	MaxEmails int       `json:"max_emails"`
	Folders   []string  `json:"folders,omitempty"`
	Keywords  []string  `json:"keywords,omitempty"`
}

// MCP邮件客户端
type MCPEmailClient struct {
	config     MCPEmailConfig
	httpClient *http.Client
}

// 登录会话信息
type LoginSession struct {
	SessionID string `json:"session_id"`
	LoginURL  string `json:"login_url"`
	Status    string `json:"status"`
	Message   string `json:"message"`
}

// 创建MCP邮件客户端
func NewMCPEmailClient(config MCPEmailConfig) *MCPEmailClient {
	return &MCPEmailClient{
		config: config,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// 通过MCP协议获取邮件
func (c *MCPEmailClient) FetchEmails(ctx context.Context, query EmailQuery) ([]types.Email, error) {
	// 构建MCP请求
	params := map[string]interface{}{
		"provider":   c.config.Provider,
		"email":      c.config.Email,
		"start_date": query.StartDate.Format(time.RFC3339),
		"end_date":   query.EndDate.Format(time.RFC3339),
		"max_emails": query.MaxEmails,
		"folders":    query.Folders,
		"keywords":   query.Keywords,
	}

	mcpReq := MCPRequest{
		Jsonrpc: "2.0",
		ID:      fmt.Sprintf("fetch_%d", time.Now().Unix()),
		Method:  "email.fetch",
		Params:  params,
	}

	// 发送MCP请求
	reqBody, err := json.Marshal(mcpReq)
	if err != nil {
		return nil, fmt.Errorf("marshal MCP request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.config.MCPEndpoint, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("create HTTP request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("MCP server error: %s", string(body))
	}

	// 解析MCP响应
	var mcpResp MCPResponse
	if err := json.NewDecoder(resp.Body).Decode(&mcpResp); err != nil {
		return nil, fmt.Errorf("decode MCP response: %w", err)
	}

	if mcpResp.Error != nil {
		return nil, fmt.Errorf("MCP error: %s", mcpResp.Error.Message)
	}

	// 解析邮件数据
	var emails []types.Email
	if err := json.Unmarshal(mcpResp.Result, &emails); err != nil {
		return nil, fmt.Errorf("unmarshal emails: %w", err)
	}

	return emails, nil
}

// 触发邮箱登录
func (c *MCPEmailClient) InitiateEmailLogin(ctx context.Context) (*LoginSession, error) {
	params := map[string]interface{}{
		"provider": c.config.Provider,
		"email":    c.config.Email,
	}

	mcpReq := MCPRequest{
		Jsonrpc: "2.0",
		ID:      fmt.Sprintf("login_%d", time.Now().Unix()),
		Method:  "email.login",
		Params:  params,
	}

	reqBody, err := json.Marshal(mcpReq)
	if err != nil {
		return nil, fmt.Errorf("marshal MCP request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.config.MCPEndpoint, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("create HTTP request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	var mcpResp MCPResponse
	if err := json.NewDecoder(resp.Body).Decode(&mcpResp); err != nil {
		return nil, fmt.Errorf("decode MCP response: %w", err)
	}

	if mcpResp.Error != nil {
		return nil, fmt.Errorf("MCP error: %s", mcpResp.Error.Message)
	}

	var session LoginSession
	if err := json.Unmarshal(mcpResp.Result, &session); err != nil {
		return nil, fmt.Errorf("unmarshal login session: %w", err)
	}

	return &session, nil
}

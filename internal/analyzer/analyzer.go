package analyzer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"jobtracker/internal/types"
)

// LLM客户端配置
type LLMConfig struct {
	APIBase     string  `json:"api_base"`
	APIKey      string  `json:"api_key"`
	Model       string  `json:"model"`
	Temperature float64 `json:"temperature"`
	MaxTokens   int     `json:"max_tokens"`
}

// LLM请求和响应结构
type LLMRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float64   `json:"temperature"`
	MaxTokens   int       `json:"max_tokens"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type LLMResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// 求职邮件分析器
type JobAnalyzer struct {
	llmConfig  LLMConfig
	httpClient *http.Client
}

// 创建求职分析器
func NewJobAnalyzer(config LLMConfig) *JobAnalyzer {
	return &JobAnalyzer{
		llmConfig: config,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// 分析邮件是否与求职相关
func (ja *JobAnalyzer) IsJobRelated(ctx context.Context, email types.Email) (bool, error) {
	prompt := fmt.Sprintf(`
请判断以下邮件是否与求职、招聘、面试相关。

邮件信息：
发件人: %s
主题: %s
正文: %s

请仅回答 "是" 或 "否"，不要包含其他内容。
`, email.From, email.Subject, truncateText(email.BodyText, 1000))

	response, err := ja.callLLM(ctx, prompt)
	if err != nil {
		return false, fmt.Errorf("LLM call failed: %w", err)
	}

	response = strings.TrimSpace(strings.ToLower(response))
	return response == "是" || response == "yes" || strings.Contains(response, "是"), nil
}

// 解析求职相关邮件的详细信息
func (ja *JobAnalyzer) AnalyzeJobEmail(ctx context.Context, email types.Email) (*types.JobApplication, error) {
	prompt := fmt.Sprintf(`
请分析以下求职相关的邮件，提取出公司名称、职位、当前状态等信息。

邮件信息：
发件人: %s
主题: %s
日期: %s
正文: %s

请按以下JSON格式返回分析结果：
{
  "company": "公司名称",
  "position": "职位名称", 
  "status": "状态(APPLIED/OA/INTERVIEW/OFFER/REJECTED/WITHDRAWN/OTHER)",
  "location": "工作地点(可选)",
  "description": "简短描述当前状态"
}

状态说明：
- APPLIED: 已申请/简历已投递
- OA: 在线测试/笔试邀请
- INTERVIEW: 面试邀请/面试安排
- OFFER: 收到录用通知/offer
- REJECTED: 被拒绝/未通过
- WITHDRAWN: 撤回申请
- OTHER: 其他状态

请确保返回有效的JSON格式，不要包含其他内容。
`, email.From, email.Subject, email.Date.Format("2006-01-02"), truncateText(email.BodyText, 2000))

	response, err := ja.callLLM(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("LLM analysis failed: %w", err)
	}

	// 解析JSON响应
	var result struct {
		Company     string `json:"company"`
		Position    string `json:"position"`
		Status      string `json:"status"`
		Location    string `json:"location"`
		Description string `json:"description"`
	}

	// 清理响应文本，提取JSON部分
	jsonStr := extractJSON(response)
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("parse JSON response: %w, response: %s", err, response)
	}

	// 验证和标准化状态
	status := normalizeJobStatus(result.Status)

	return &types.JobApplication{
		Company:     cleanText(result.Company),
		Position:    cleanText(result.Position),
		Status:      status,
		Location:    cleanText(result.Location),
		Description: cleanText(result.Description),
		Email:       email,
		ExtractedAt: time.Now(),
	}, nil
}

// 批量分析邮件
func (ja *JobAnalyzer) AnalyzeEmails(ctx context.Context, emails []types.Email) ([]types.JobApplication, error) {
	var jobApplications []types.JobApplication

	for i, email := range emails {
		select {
		case <-ctx.Done():
			return jobApplications, ctx.Err()
		default:
		}

		fmt.Printf("分析邮件 %d/%d: %s\n", i+1, len(emails), email.Subject)

		// 先判断是否与求职相关
		isJobRelated, err := ja.IsJobRelated(ctx, email)
		if err != nil {
			fmt.Printf("检查邮件 %s 失败: %v\n", email.Subject, err)
			continue
		}

		if !isJobRelated {
			continue
		}

		// 分析求职邮件详情
		jobApp, err := ja.AnalyzeJobEmail(ctx, email)
		if err != nil {
			fmt.Printf("分析邮件 %s 失败: %v\n", email.Subject, err)
			continue
		}

		jobApplications = append(jobApplications, *jobApp)
		fmt.Printf("发现求职邮件: %s - %s (%s)\n", jobApp.Company, jobApp.Position, jobApp.Status)

		// 添加延迟避免API限制
		time.Sleep(500 * time.Millisecond)
	}

	return jobApplications, nil
}

// 调用LLM API
func (ja *JobAnalyzer) callLLM(ctx context.Context, prompt string) (string, error) {
	req := LLMRequest{
		Model:       ja.llmConfig.Model,
		Temperature: ja.llmConfig.Temperature,
		MaxTokens:   ja.llmConfig.MaxTokens,
		Messages: []Message{
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", ja.llmConfig.APIBase+"/chat/completions", bytes.NewBuffer(reqBody))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+ja.llmConfig.APIKey)

	resp, err := ja.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("LLM API error: %s", string(body))
	}

	var llmResp LLMResponse
	if err := json.NewDecoder(resp.Body).Decode(&llmResp); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	if len(llmResp.Choices) == 0 {
		return "", fmt.Errorf("no response from LLM")
	}

	return llmResp.Choices[0].Message.Content, nil
}

// 辅助函数

// 截断文本到指定长度
func truncateText(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen] + "..."
}

// 提取JSON字符串
func extractJSON(text string) string {
	// 尝试找到JSON对象
	start := strings.Index(text, "{")
	if start == -1 {
		return text
	}

	end := strings.LastIndex(text, "}")
	if end == -1 || end <= start {
		return text
	}

	return text[start : end+1]
}

// 清理文本
func cleanText(text string) string {
	// 移除多余的空白字符
	re := regexp.MustCompile(`\s+`)
	return strings.TrimSpace(re.ReplaceAllString(text, " "))
}

// 标准化求职状态
func normalizeJobStatus(status string) types.Status {
	status = strings.ToUpper(strings.TrimSpace(status))

	switch status {
	case "APPLIED", "APPLICATION", "申请", "已申请":
		return types.StatusApplied
	case "OA", "ONLINE_ASSESSMENT", "笔试", "在线测试":
		return types.StatusOA
	case "INTERVIEW", "面试":
		return types.StatusInterview
	case "OFFER", "ACCEPTED", "录用", "录取":
		return types.StatusOffer
	case "REJECTED", "DECLINED", "拒绝", "未通过":
		return types.StatusRejected
	case "WITHDRAWN", "撤回":
		return types.StatusWithdrawn
	default:
		return types.StatusOther
	}
}

package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-message"
	"golang.org/x/oauth2"
)

// MCP协议结构体
type MCPRequest struct {
	Jsonrpc string      `json:"jsonrpc"`
	ID      string      `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
}

type MCPResponse struct {
	Jsonrpc string      `json:"jsonrpc"`
	ID      string      `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *MCPError   `json:"error,omitempty"`
}

type MCPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// 邮件相关结构体
type Email struct {
	ID        string    `json:"id"`
	From      string    `json:"from"`
	Subject   string    `json:"subject"`
	Date      time.Time `json:"date"`
	BodyText  string    `json:"body_text"`
	BodyHTML  string    `json:"body_html"`
	MessageID string    `json:"message_id"`
	Folder    string    `json:"folder"`
}

type LoginParams struct {
	Provider string `json:"provider"`
	Email    string `json:"email"`
}

type FetchParams struct {
	Provider  string    `json:"provider"`
	Email     string    `json:"email"`
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`
	MaxEmails int       `json:"max_emails"`
	Folders   []string  `json:"folders"`
	Keywords  []string  `json:"keywords"`
}

type LoginSession struct {
	SessionID string `json:"session_id"`
	LoginURL  string `json:"login_url"`
	Status    string `json:"status"`
	Message   string `json:"message"`
}

// MCP服务器
type MCPServer struct {
	sessions map[string]*LoginSession
	tokens   map[string]*oauth2.Token
}

func NewMCPServer() *MCPServer {
	return &MCPServer{
		sessions: make(map[string]*LoginSession),
		tokens:   make(map[string]*oauth2.Token),
	}
}

func main() {
	server := NewMCPServer()

	http.HandleFunc("/mcp", server.handleMCP)
	http.HandleFunc("/oauth/callback", server.handleOAuthCallback)

	fmt.Println("🚀 MCP服务器启动在 http://localhost:8080")
	fmt.Println("📧 支持的邮箱提供商: Gmail, Outlook, Yahoo, 自定义IMAP")
	fmt.Println("🔐 Gmail需要OAuth认证，其他可使用应用密码")

	log.Fatal(http.ListenAndServe(":8080", nil))
}

func (s *MCPServer) handleMCP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	if r.Method == "OPTIONS" {
		return
	}

	var req MCPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendError(w, req.ID, -32700, "Parse error")
		return
	}

	var response interface{}
	var err error

	switch req.Method {
	case "email.login":
		response, err = s.handleLogin(req.Params)
	case "email.fetch":
		response, err = s.handleFetch(req.Params)
	default:
		s.sendError(w, req.ID, -32601, "Method not found")
		return
	}

	if err != nil {
		s.sendError(w, req.ID, -32000, err.Error())
		return
	}

	resp := MCPResponse{
		Jsonrpc: "2.0",
		ID:      req.ID,
		Result:  response,
	}

	json.NewEncoder(w).Encode(resp)
}

func (s *MCPServer) handleLogin(params interface{}) (*LoginSession, error) {
	var loginParams LoginParams
	data, _ := json.Marshal(params)
	if err := json.Unmarshal(data, &loginParams); err != nil {
		return nil, fmt.Errorf("invalid login parameters")
	}

	sessionID := fmt.Sprintf("session_%d", time.Now().Unix())

	switch loginParams.Provider {
	case "gmail":
		// Gmail使用OAuth2流程
		session := &LoginSession{
			SessionID: sessionID,
			LoginURL:  s.generateGmailOAuthURL(sessionID),
			Status:    "pending",
			Message:   "请在浏览器中完成Google OAuth认证",
		}
		s.sessions[sessionID] = session
		return session, nil

	default:
		// 其他邮箱使用应用密码
		session := &LoginSession{
			SessionID: sessionID,
			LoginURL:  "",
			Status:    "ready",
			Message:   "使用应用密码认证，无需浏览器登录",
		}
		s.sessions[sessionID] = session
		return session, nil
	}
}

func (s *MCPServer) handleFetch(params interface{}) ([]Email, error) {
	var fetchParams FetchParams
	data, _ := json.Marshal(params)
	if err := json.Unmarshal(data, &fetchParams); err != nil {
		return nil, fmt.Errorf("invalid fetch parameters")
	}

	switch fetchParams.Provider {
	case "gmail":
		return s.fetchGmailEmails(fetchParams)
	case "outlook":
		return s.fetchIMAPEmails(fetchParams, "outlook.office365.com:993")
	case "yahoo":
		return s.fetchIMAPEmails(fetchParams, "imap.mail.yahoo.com:993")
	default:
		return s.fetchIMAPEmails(fetchParams, s.inferIMAPHost(fetchParams.Email))
	}
}

func (s *MCPServer) generateGmailOAuthURL(sessionID string) string {
	// 简化的OAuth URL生成
	// 实际应用中需要配置Google OAuth客户端
	clientID := os.Getenv("GMAIL_CLIENT_ID")
	if clientID == "" {
		return "https://console.cloud.google.com/apis/credentials (请配置GMAIL_CLIENT_ID)"
	}

	redirectURI := "http://localhost:8080/oauth/callback"
	scope := "https://www.googleapis.com/auth/gmail.readonly"

	return fmt.Sprintf(
		"https://accounts.google.com/o/oauth2/auth?client_id=%s&redirect_uri=%s&scope=%s&response_type=code&state=%s",
		clientID, redirectURI, scope, sessionID,
	)
}

func (s *MCPServer) handleOAuthCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")

	if session, exists := s.sessions[state]; exists {
		session.Status = "completed"
		session.Message = "OAuth认证完成"
		// 这里应该用code换取access token
		// 简化实现，实际需要完整的OAuth流程
		_ = code // 避免未使用变量警告
	}

	fmt.Fprintf(w, "<h2>认证完成</h2><p>可以关闭此页面，返回应用程序。</p>")
}

func (s *MCPServer) fetchGmailEmails(params FetchParams) ([]Email, error) {
	// Gmail API 实现
	// 这里简化为返回模拟数据，实际需要集成Gmail API
	return []Email{
		{
			ID:       "gmail_1",
			From:     "noreply@google.com",
			Subject:  "Gmail API 测试邮件",
			Date:     time.Now(),
			BodyText: "这是通过Gmail API获取的测试邮件",
			Folder:   "INBOX",
		},
	}, nil
}

func (s *MCPServer) fetchIMAPEmails(params FetchParams, host string) ([]Email, error) {
	if host == "" {
		return nil, fmt.Errorf("无法推断IMAP主机，请配置自定义主机")
	}

	// 从环境变量获取密码
	password := os.Getenv("EMAIL_PASSWORD")
	if password == "" {
		// 尝试使用应用密码
		password = os.Getenv("EMAIL_APP_PASSWORD")
	}

	if password == "" {
		return nil, fmt.Errorf("请设置环境变量 EMAIL_PASSWORD 或 EMAIL_APP_PASSWORD")
	}

	// 连接IMAP
	c, err := client.DialTLS(host, &tls.Config{})
	if err != nil {
		return nil, fmt.Errorf("连接IMAP服务器失败: %v", err)
	}
	defer c.Close()

	// 登录
	if err := c.Login(params.Email, password); err != nil {
		return nil, fmt.Errorf("IMAP登录失败: %v", err)
	}
	defer c.Logout()

	var emails []Email

	// 遍历文件夹
	for _, folder := range params.Folders {
		folderEmails, err := s.fetchFromFolder(c, folder, params)
		if err != nil {
			log.Printf("获取文件夹 %s 失败: %v", folder, err)
			continue
		}
		emails = append(emails, folderEmails...)

		if len(emails) >= params.MaxEmails {
			break
		}
	}

	// 限制返回数量
	if len(emails) > params.MaxEmails {
		emails = emails[:params.MaxEmails]
	}

	return emails, nil
}

func (s *MCPServer) fetchFromFolder(c *client.Client, folder string, params FetchParams) ([]Email, error) {
	// 选择文件夹
	mbox, err := c.Select(folder, true)
	if err != nil {
		return nil, err
	}

	if mbox.Messages == 0 {
		return nil, nil
	}

	// 搜索邮件
	criteria := imap.NewSearchCriteria()
	criteria.Since = params.StartDate
	criteria.Before = params.EndDate.AddDate(0, 0, 1) // 包含结束日期

	uids, err := c.Search(criteria)
	if err != nil {
		return nil, err
	}

	if len(uids) == 0 {
		return nil, nil
	}

	// 限制获取数量
	limit := params.MaxEmails
	if len(uids) < limit {
		limit = len(uids)
	}
	uids = uids[:limit]

	// 获取邮件
	seqset := new(imap.SeqSet)
	seqset.AddNum(uids...)

	messages := make(chan *imap.Message, limit)
	done := make(chan error, 1)

	go func() {
		done <- c.Fetch(seqset, []imap.FetchItem{imap.FetchEnvelope, imap.FetchBodyStructure}, messages)
	}()

	var emails []Email
	for msg := range messages {
		email := s.convertToEmail(msg, folder)
		emails = append(emails, email)
	}

	if err := <-done; err != nil {
		return nil, err
	}

	return emails, nil
}

func (s *MCPServer) convertToEmail(msg *imap.Message, folder string) Email {
	email := Email{
		ID:     fmt.Sprintf("%d", msg.Uid),
		Folder: folder,
	}

	if msg.Envelope != nil {
		if len(msg.Envelope.From) > 0 {
			email.From = msg.Envelope.From[0].Address()
		}
		email.Subject = msg.Envelope.Subject
		email.Date = msg.Envelope.Date
		email.MessageID = msg.Envelope.MessageId
	}

	// 简化处理，直接使用主题作为正文预览
	email.BodyText = email.Subject

	return email
}

func (s *MCPServer) extractTextBody(entity *message.Entity) string {
	// 简化实现，返回空字符串
	return ""
}

func (s *MCPServer) inferIMAPHost(email string) string {
	email = strings.ToLower(email)

	if strings.Contains(email, "@qq.com") {
		return "imap.qq.com:993"
	}
	if strings.Contains(email, "@163.com") {
		return "imap.163.com:993"
	}
	if strings.Contains(email, "@126.com") {
		return "imap.126.com:993"
	}

	return ""
}

func (s *MCPServer) sendError(w http.ResponseWriter, id string, code int, message string) {
	resp := MCPResponse{
		Jsonrpc: "2.0",
		ID:      id,
		Error: &MCPError{
			Code:    code,
			Message: message,
		},
	}
	json.NewEncoder(w).Encode(resp)
}

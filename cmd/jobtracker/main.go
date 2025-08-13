package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/YKarmar/JobTracker/internal/analyzer"
	"github.com/YKarmar/JobTracker/internal/client"
	"github.com/YKarmar/JobTracker/internal/config"
	"github.com/YKarmar/JobTracker/internal/exporter"
	"github.com/YKarmar/JobTracker/internal/types"
)

func main() {
	// 命令行参数
	mockMode := flag.Bool("mock", false, "使用模拟数据（用于测试）")
	flag.Parse()

	fmt.Println("=== JobTracker 求职邮件分析工具 ===")
	fmt.Println("正在加载配置...")

	// 1. 加载配置
	cfg, err := config.Load("configs/config.yaml")
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel()

	var emails []types.Email

	if *mockMode {
		fmt.Println("🧪 使用模拟模式进行测试...")
		emails = generateMockEmails()
	} else {
		// 2. 设置MCP邮件客户端
		mcpConfig := client.MCPEmailConfig{
			Provider:    client.ParseEmailProvider(cfg.IMAP.Provider),
			Email:       cfg.IMAP.Email,
			MCPEndpoint: "http://localhost:8080/mcp", // 默认MCP端点
		}

		// 如果有MCP配置，使用配置的值
		if cfg.MCP.Endpoint != "" {
			mcpConfig.MCPEndpoint = cfg.MCP.Endpoint
		}
		if cfg.MCP.APIKey != "" {
			mcpConfig.APIKey = cfg.MCP.APIKey
		}

		emailClient := client.NewMCPEmailClient(mcpConfig)

		// 3. 触发邮箱登录
		fmt.Printf("正在为邮箱 %s 启动登录流程...\n", cfg.IMAP.Email)
		fmt.Printf("检测到邮箱提供商: %s\n", cfg.IMAP.Provider)

		session, err := emailClient.InitiateEmailLogin(ctx)
		if err != nil {
			log.Fatalf("启动邮箱登录失败: %v\n提示: 如需测试，可使用 --mock 参数", err)
		}

		if session.LoginURL != "" {
			fmt.Printf("请在浏览器中完成登录: %s\n", session.LoginURL)

			// 自动打开浏览器
			if err := openBrowser(session.LoginURL); err != nil {
				fmt.Printf("无法自动打开浏览器，请手动访问上述链接\n")
			}

			fmt.Println("等待登录完成...")
			// 这里应该实现等待登录完成的逻辑
			time.Sleep(30 * time.Second) // 简单等待，实际应该轮询状态
		}

		// 4. 获取邮件
		start := config.ParseDateLoose(cfg.Fetch.Start, time.Now().AddDate(0, 0, -7))
		end := config.ParseDateLoose(cfg.Fetch.End, time.Now())

		query := client.EmailQuery{
			StartDate: start,
			EndDate:   end,
			MaxEmails: cfg.Fetch.MaxEmails,
			Folders:   cfg.IMAP.Folders,
			Keywords:  []string{"job", "interview", "offer", "application", "招聘", "面试", "职位", "工作"},
		}

		fmt.Printf("正在获取邮件 (时间范围: %s 到 %s)...\n",
			start.Format("2006-01-02"), end.Format("2006-01-02"))

		emails, err = emailClient.FetchEmails(ctx, query)
		if err != nil {
			log.Fatalf("获取邮件失败: %v", err)
		}
	}

	fmt.Printf("成功获取 %d 封邮件\n", len(emails))

	if len(emails) == 0 {
		fmt.Println("没有找到邮件，程序结束")
		return
	}

	// 5. 使用LLM分析邮件
	llmConfig := analyzer.LLMConfig{
		APIBase:     cfg.LLM.APIBase,
		APIKey:      cfg.LLM.APIKey,
		Model:       cfg.LLM.Model,
		Temperature: cfg.LLM.Temperature,
		MaxTokens:   cfg.LLM.MaxTokens,
	}

	if llmConfig.MaxTokens == 0 {
		llmConfig.MaxTokens = 2000
	}

	jobAnalyzer := analyzer.NewJobAnalyzer(llmConfig)

	fmt.Println("\n正在使用LLM分析邮件内容...")
	jobApplications, err := jobAnalyzer.AnalyzeEmails(ctx, emails)
	if err != nil {
		log.Fatalf("分析邮件失败: %v", err)
	}

	// 6. 显示统计信息
	exporter.PrintJobStatistics(jobApplications)

	// 7. 导出CSV文件
	if len(jobApplications) > 0 {
		exportFile := cfg.Export.File
		if exportFile == "" {
			exportFile = "job_applications_" + time.Now().Format("20060102_150405") + ".csv"
		}

		csvExporter := exporter.NewCSVExporter(exportFile)

		fmt.Printf("\n正在导出结果到 %s...\n", exportFile)
		if err := csvExporter.ExportJobApplications(jobApplications); err != nil {
			log.Printf("导出CSV失败: %v", err)
		} else {
			fmt.Printf("✅ 求职信息已导出到: %s\n", exportFile)
		}

		// 导出统计信息
		if err := csvExporter.ExportStatistics(jobApplications); err != nil {
			log.Printf("导出统计信息失败: %v", err)
		}
	}

	fmt.Println("\n=== 分析完成 ===")

	// 显示详细结果
	if len(jobApplications) > 0 {
		fmt.Println("\n最近的求职活动:")
		count := 0
		for _, app := range jobApplications {
			if count >= 5 { // 只显示最近5条
				break
			}
			fmt.Printf("• %s - %s (%s) [%s]\n",
				app.Company, app.Position, app.Status, app.Email.Date.Format("01-02"))
			count++
		}

		if len(jobApplications) > 5 {
			fmt.Printf("... 还有 %d 条记录，详情请查看CSV文件\n", len(jobApplications)-5)
		}
	}
}

// 生成测试用的模拟邮件数据
func generateMockEmails() []types.Email {
	return []types.Email{
		{
			ID:       "1",
			From:     "noreply@company.com",
			Subject:  "感谢您投递简历 - 软件工程师职位",
			Date:     time.Now().AddDate(0, 0, -1),
			BodyText: "感谢您投递我们公司软件工程师职位的简历。我们已收到您的申请，将在3-5个工作日内回复。",
			BodyHTML: "<p>感谢您投递我们公司软件工程师职位的简历。我们已收到您的申请，将在3-5个工作日内回复。</p>",
			Folder:   "INBOX",
		},
		{
			ID:       "2",
			From:     "hr@techcorp.com",
			Subject:  "邀请您参加在线技术测试",
			Date:     time.Now().AddDate(0, 0, -3),
			BodyText: "恭喜您通过简历筛选！我们邀请您参加在线技术测试，请在48小时内完成。",
			BodyHTML: "<p>恭喜您通过简历筛选！我们邀请您参加在线技术测试，请在48小时内完成。</p>",
			Folder:   "INBOX",
		},
		{
			ID:       "3",
			From:     "recruitment@startup.io",
			Subject:  "Interview Invitation - Frontend Developer Position",
			Date:     time.Now().AddDate(0, 0, -5),
			BodyText: "We would like to invite you for an interview for the Frontend Developer position. Please confirm your availability.",
			BodyHTML: "<p>We would like to invite you for an interview for the Frontend Developer position. Please confirm your availability.</p>",
			Folder:   "INBOX",
		},
	}
}

// 打开浏览器
func openBrowser(url string) error {
	// 这个函数的实现可以移到内部包中，这里简化处理
	fmt.Printf("请手动打开浏览器访问: %s\n", url)
	return nil
}

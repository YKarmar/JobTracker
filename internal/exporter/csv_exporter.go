package exporter

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/YKarmar/JobTracker/internal/types"
)

// CSV导出器
type CSVExporter struct {
	filename string
}

// 创建CSV导出器
func NewCSVExporter(filename string) *CSVExporter {
	return &CSVExporter{
		filename: filename,
	}
}

// 导出求职信息到CSV文件
func (ce *CSVExporter) ExportJobApplications(applications []types.JobApplication) error {
	file, err := os.Create(ce.filename)
	if err != nil {
		return fmt.Errorf("create CSV file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// 写入CSV头部
	headers := []string{
		"公司名称",
		"职位名称",
		"状态",
		"工作地点",
		"状态描述",
		"邮件发件人",
		"邮件主题",
		"邮件日期",
		"邮件文件夹",
		"信息提取时间",
	}

	if err := writer.Write(headers); err != nil {
		return fmt.Errorf("write CSV headers: %w", err)
	}

	// 写入数据行
	for _, app := range applications {
		record := []string{
			app.Company,
			app.Position,
			string(app.Status),
			app.Location,
			app.Description,
			app.Email.From,
			app.Email.Subject,
			app.Email.Date.Format("2006-01-02 15:04:05"),
			app.Email.Folder,
			app.ExtractedAt.Format("2006-01-02 15:04:05"),
		}

		if err := writer.Write(record); err != nil {
			return fmt.Errorf("write CSV record: %w", err)
		}
	}

	return nil
}

// 导出统计信息到CSV文件
func (ce *CSVExporter) ExportStatistics(applications []types.JobApplication) error {
	statsFile := "job_statistics_" + time.Now().Format("20060102_150405") + ".csv"

	file, err := os.Create(statsFile)
	if err != nil {
		return fmt.Errorf("create statistics file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// 统计各种状态的数量
	statusCount := make(map[types.Status]int)
	companyCount := make(map[string]int)

	for _, app := range applications {
		statusCount[app.Status]++
		companyCount[app.Company]++
	}

	// 写入状态统计
	writer.Write([]string{"状态统计"})
	writer.Write([]string{"状态", "数量"})

	for status, count := range statusCount {
		writer.Write([]string{string(status), strconv.Itoa(count)})
	}

	writer.Write([]string{}) // 空行

	// 写入公司统计（前10名）
	writer.Write([]string{"公司投递统计（前10名）"})
	writer.Write([]string{"公司名称", "投递次数"})

	// 简单排序找出前10名
	type companyStats struct {
		name  string
		count int
	}

	var companies []companyStats
	for company, count := range companyCount {
		companies = append(companies, companyStats{company, count})
	}

	// 简单冒泡排序
	for i := 0; i < len(companies)-1; i++ {
		for j := 0; j < len(companies)-i-1; j++ {
			if companies[j].count < companies[j+1].count {
				companies[j], companies[j+1] = companies[j+1], companies[j]
			}
		}
	}

	// 取前10名
	limit := 10
	if len(companies) < limit {
		limit = len(companies)
	}

	for i := 0; i < limit; i++ {
		writer.Write([]string{companies[i].name, strconv.Itoa(companies[i].count)})
	}

	fmt.Printf("统计信息已导出到: %s\n", statsFile)
	return nil
}

// 打印简要统计信息
func PrintJobStatistics(applications []types.JobApplication) {
	if len(applications) == 0 {
		fmt.Println("没有找到求职相关的邮件")
		return
	}

	fmt.Printf("\n=== 求职邮件统计 ===\n")
	fmt.Printf("总共找到 %d 封求职相关邮件\n\n", len(applications))

	// 统计各状态数量
	statusCount := make(map[types.Status]int)
	for _, app := range applications {
		statusCount[app.Status]++
	}

	fmt.Println("状态分布:")
	statusNames := map[types.Status]string{
		types.StatusApplied:   "已申请",
		types.StatusOA:        "在线测试/笔试",
		types.StatusInterview: "面试",
		types.StatusOffer:     "收到Offer",
		types.StatusRejected:  "被拒绝",
		types.StatusWithdrawn: "撤回申请",
		types.StatusOther:     "其他状态",
	}

	for status, count := range statusCount {
		name := statusNames[status]
		if name == "" {
			name = string(status)
		}
		fmt.Printf("  %s: %d 封\n", name, count)
	}

	// 统计公司分布
	companyCount := make(map[string]int)
	for _, app := range applications {
		if app.Company != "" {
			companyCount[app.Company]++
		}
	}

	fmt.Printf("\n涉及公司数量: %d 家\n", len(companyCount))

	if len(companyCount) > 0 {
		fmt.Println("\n投递最多的公司:")
		count := 0
		for company, num := range companyCount {
			if count >= 5 { // 只显示前5名
				break
			}
			fmt.Printf("  %s: %d 次\n", company, num)
			count++
		}
	}
}

package types

import "time"

type Status string

const (
	StatusApplied   Status = "APPLIED"   // 已申请
	StatusOA        Status = "OA"        // 在线测试/笔试
	StatusInterview Status = "INTERVIEW" // 面试中
	StatusOffer     Status = "OFFER"     // 收到offer
	StatusRejected  Status = "REJECTED"  // 被拒绝
	StatusWithdrawn Status = "WITHDRAWN" // 撤回申请
	StatusOther     Status = "OTHER"     // 其他状态
)

type Email struct {
	ID        string
	From      string
	Subject   string
	Date      time.Time
	BodyText  string
	BodyHTML  string
	MessageID string
	Folder    string
}

type JobApplication struct {
	Company     string    `json:"company"`
	Position    string    `json:"position"`
	Status      Status    `json:"status"`
	Location    string    `json:"location,omitempty"`
	Description string    `json:"description,omitempty"`
	Email       Email     `json:"email"`
	ExtractedAt time.Time `json:"extracted_at"`
}

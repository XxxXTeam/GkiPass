package models

import (
	"time"
)

/*
Plan 套餐模型
功能：定义系统的订阅套餐，包括流量限制、速率限制、连接数限制等
*/
type Plan struct {
	BaseModel
	Name            string  `gorm:"type:varchar(64);not null" json:"name"`
	Description     string  `gorm:"type:varchar(512)" json:"description"`
	Price           float64 `gorm:"type:decimal(10,2);not null" json:"price"`
	Duration        int     `gorm:"not null" json:"duration"`
	DurationUnit    string  `gorm:"type:varchar(16);default:'month';not null" json:"duration_unit"`
	TrafficLimit    int64   `gorm:"default:0" json:"traffic_limit"`
	SpeedLimit      int64   `gorm:"default:0" json:"speed_limit"`
	ConnectionLimit int     `gorm:"default:0" json:"connection_limit"`
	RuleLimit       int     `gorm:"default:0" json:"rule_limit"`
	NodeGroupIDs    string  `gorm:"type:text" json:"node_group_ids"`
	Enabled         bool    `gorm:"default:true;not null" json:"enabled"`
	SortOrder       int     `gorm:"default:0" json:"sort_order"`
}

func (Plan) TableName() string {
	return "plans"
}

/*
Order 订单模型
功能：记录用户的充值和购买订单
*/
type Order struct {
	BaseModel
	UserID      string     `gorm:"type:varchar(36);index;not null" json:"user_id"`
	Type        string     `gorm:"type:varchar(32);not null" json:"type"`
	Status      string     `gorm:"type:varchar(16);default:'pending';not null;index" json:"status"`
	Amount      float64    `gorm:"type:decimal(10,2);not null" json:"amount"`
	PayMethod   string     `gorm:"type:varchar(32)" json:"pay_method"`
	PlanID      string     `gorm:"type:varchar(36)" json:"plan_id"`
	Description string     `gorm:"type:varchar(256)" json:"description"`
	PaidAt      *time.Time `gorm:"" json:"paid_at"`
	ExternalID  string     `gorm:"type:varchar(128);index" json:"external_id"`

	User User `gorm:"foreignKey:UserID" json:"-"`
}

func (Order) TableName() string {
	return "orders"
}

/*
Announcement 系统公告
功能：发布和管理面向用户的系统公告
*/
type Announcement struct {
	BaseModel
	Title     string     `gorm:"type:varchar(128);not null" json:"title"`
	Content   string     `gorm:"type:text;not null" json:"content"`
	Type      string     `gorm:"type:varchar(32);default:'info';not null" json:"type"`
	Priority  int        `gorm:"default:0" json:"priority"`
	Enabled   bool       `gorm:"default:true;not null" json:"enabled"`
	StartAt   *time.Time `gorm:"" json:"start_at"`
	EndAt     *time.Time `gorm:"" json:"end_at"`
	CreatedBy string     `gorm:"type:varchar(36)" json:"created_by"`
}

func (Announcement) TableName() string {
	return "announcements"
}

/*
Notification 通知模型
功能：管理发送给用户的个人通知和系统通知
*/
type Notification struct {
	BaseModel
	UserID  string     `gorm:"type:varchar(36);index;not null" json:"user_id"`
	Type    string     `gorm:"type:varchar(32);not null" json:"type"`
	Title   string     `gorm:"type:varchar(128);not null" json:"title"`
	Content string     `gorm:"type:text" json:"content"`
	Level   string     `gorm:"type:varchar(16);default:'info'" json:"level"`
	Read    bool       `gorm:"default:false;index" json:"read"`
	ReadAt  *time.Time `gorm:"" json:"read_at"`

	User User `gorm:"foreignKey:UserID" json:"-"`
}

func (Notification) TableName() string {
	return "notifications"
}

/*
SystemSetting 系统设置
功能：存储系统的各种动态配置项（键值对形式）
*/
type SystemSetting struct {
	BaseModel
	Category string `gorm:"type:varchar(64);index;not null" json:"category"`
	Key      string `gorm:"type:varchar(128);uniqueIndex;not null" json:"key"`
	Value    string `gorm:"type:text" json:"value"`
	Type     string `gorm:"type:varchar(16);default:'string'" json:"type"`
}

func (SystemSetting) TableName() string {
	return "system_settings"
}

/*
PaymentConfig 支付配置
功能：存储 EPAY/USDT 等支付方式的配置信息
*/
type PaymentConfig struct {
	BaseModel
	Name      string `gorm:"type:varchar(64);not null" json:"name"`
	Type      string `gorm:"type:varchar(32);not null;index" json:"type"`
	Config    string `gorm:"type:text" json:"config"`
	Enabled   bool   `gorm:"default:false" json:"enabled"`
	SortOrder int    `gorm:"default:0" json:"sort_order"`
}

func (PaymentConfig) TableName() string {
	return "payment_configs"
}

/*
AuditLog 审计日志
功能：记录系统中的关键操作日志，用于安全审计
*/
type AuditLog struct {
	BaseModel
	UserID   string `gorm:"type:varchar(36);index" json:"user_id"`
	Action   string `gorm:"type:varchar(64);index;not null" json:"action"`
	Resource string `gorm:"type:varchar(64);index" json:"resource"`
	Detail   string `gorm:"type:text" json:"detail"`
	IP       string `gorm:"type:varchar(64)" json:"ip"`
	UA       string `gorm:"type:varchar(512)" json:"ua"`
}

func (AuditLog) TableName() string {
	return "audit_logs"
}

/*
PaymentMonitor 支付监听记录
功能：跟踪待确认的加密货币/第三方支付订单状态
*/
type PaymentMonitor struct {
	BaseModel
	TransactionID  string     `gorm:"type:varchar(36);index;not null" json:"transaction_id"`
	PaymentType    string     `gorm:"type:varchar(32);not null" json:"payment_type"`
	PaymentAddress string     `gorm:"type:varchar(256)" json:"payment_address"`
	ExpectedAmount float64    `gorm:"type:decimal(12,2);not null" json:"expected_amount"`
	Status         string     `gorm:"type:varchar(16);default:'monitoring';not null;index" json:"status"`
	ConfirmCount   int        `gorm:"default:0" json:"confirm_count"`
	LastCheckAt    *time.Time `gorm:"" json:"last_check_at"`
	ExpiresAt      time.Time  `gorm:"not null" json:"expires_at"`
}

func (PaymentMonitor) TableName() string {
	return "payment_monitors"
}

package models

import (
	"time"
)

/*
UserRole 用户角色枚举
功能：定义系统支持的用户角色类型
*/
type UserRole string

const (
	RoleAdmin  UserRole = "admin"
	RoleUser   UserRole = "user"
	RoleGuest  UserRole = "guest"
	RoleSystem UserRole = "system"
)

/*
User 用户模型
功能：存储用户基本信息、认证凭据和账户状态
*/
type User struct {
	BaseModel
	Username    string    `gorm:"type:varchar(64);uniqueIndex;not null" json:"username"`
	Email       string    `gorm:"type:varchar(128);uniqueIndex;not null" json:"email"`
	Password    string    `gorm:"type:varchar(256);not null" json:"-"`
	Role        UserRole  `gorm:"type:varchar(16);default:'user';not null" json:"role"`
	Enabled     bool      `gorm:"default:true;not null" json:"enabled"`
	LastLogin   time.Time `gorm:"" json:"last_login"`
	Avatar      string    `gorm:"type:varchar(512)" json:"avatar"`
	Description string    `gorm:"type:varchar(512)" json:"description"`
	Provider    string    `gorm:"type:varchar(32);index" json:"provider"`     /* OAuth 提供商: github/google */
	ProviderID  string    `gorm:"type:varchar(128);index" json:"provider_id"` /* OAuth 提供商用户ID */

	/* 关联 */
	Permissions   []Permission   `gorm:"many2many:user_permissions;" json:"permissions,omitempty"`
	Subscriptions []Subscription `gorm:"foreignKey:UserID" json:"subscriptions,omitempty"`
	Wallet        *Wallet        `gorm:"foreignKey:UserID" json:"wallet,omitempty"`
	Tunnels       []Tunnel       `gorm:"foreignKey:CreatedBy" json:"tunnels,omitempty"`
}

func (User) TableName() string {
	return "users"
}

/*
Permission 权限模型
功能：定义系统权限项
*/
type Permission struct {
	BaseModel
	Name        string `gorm:"type:varchar(64);uniqueIndex;not null" json:"name"`
	Description string `gorm:"type:varchar(256)" json:"description"`
	Module      string `gorm:"type:varchar(64);index" json:"module"`
}

func (Permission) TableName() string {
	return "permissions"
}

/*
RolePermission 角色权限关联
功能：定义角色对应的默认权限集合
*/
type RolePermission struct {
	Role         UserRole  `gorm:"type:varchar(16);primaryKey" json:"role"`
	PermissionID string    `gorm:"type:varchar(36);primaryKey" json:"permission_id"`
	GrantedAt    time.Time `gorm:"autoCreateTime" json:"granted_at"`
}

func (RolePermission) TableName() string {
	return "role_permissions"
}

/*
Wallet 用户钱包
功能：管理用户余额和充值记录
*/
type Wallet struct {
	BaseModel
	UserID       string  `gorm:"type:varchar(36);uniqueIndex;not null" json:"user_id"`
	Balance      float64 `gorm:"type:decimal(12,2);default:0;not null" json:"balance"`
	FrozenAmount float64 `gorm:"type:decimal(12,2);default:0;not null" json:"frozen_amount"`

	/* 关联 */
	User         User          `gorm:"foreignKey:UserID" json:"-"`
	Transactions []Transaction `gorm:"foreignKey:WalletID" json:"transactions,omitempty"`
}

func (Wallet) TableName() string {
	return "wallets"
}

/*
Transaction 交易记录
功能：记录钱包的所有收支明细
*/
type Transaction struct {
	BaseModel
	WalletID    string  `gorm:"type:varchar(36);index;not null" json:"wallet_id"`
	Type        string  `gorm:"type:varchar(32);not null" json:"type"`
	Status      string  `gorm:"type:varchar(16);default:'pending';not null" json:"status"`
	Amount      float64 `gorm:"type:decimal(12,2);not null" json:"amount"`
	Balance     float64 `gorm:"type:decimal(12,2);not null" json:"balance"`
	Description string  `gorm:"type:varchar(256)" json:"description"`
	OrderID     string  `gorm:"type:varchar(36);index" json:"order_id"`
}

func (Transaction) TableName() string {
	return "transactions"
}

/*
Subscription 用户订阅
功能：管理用户的套餐订阅信息
*/
type Subscription struct {
	BaseModel
	UserID    string    `gorm:"type:varchar(36);index:idx_sub_user_status;not null" json:"user_id"`
	PlanID    string    `gorm:"type:varchar(36);index;not null" json:"plan_id"`
	Status    string    `gorm:"type:varchar(16);default:'active';not null;index:idx_sub_user_status" json:"status"`
	StartAt   time.Time `gorm:"not null" json:"start_at"`
	ExpireAt  time.Time `gorm:"not null;index" json:"expire_at"`
	AutoRenew bool      `gorm:"default:false" json:"auto_renew"`

	/* 关联 */
	User User `gorm:"foreignKey:UserID" json:"-"`
	Plan Plan `gorm:"foreignKey:PlanID" json:"plan,omitempty"`
}

func (Subscription) TableName() string {
	return "subscriptions"
}

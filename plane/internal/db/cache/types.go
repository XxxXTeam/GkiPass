package cache

import "time"

/*
	cache 包本地 DTO 类型定义
	这些类型用于 Redis 缓存序列化/反序列化，
	替代原先对 dbinit 包的依赖。
*/

/* Session 会话信息（存储在Redis） */
type Session struct {
	Token     string    `json:"token"`
	UserID    string    `json:"user_id"`
	Username  string    `json:"username"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

/* NodeStatus 节点实时状态（存储在Redis） */
type NodeStatus struct {
	NodeID        string    `json:"node_id"`
	Online        bool      `json:"online"`
	LastHeartbeat time.Time `json:"last_heartbeat"`
	CurrentLoad   float64   `json:"current_load"`
	Connections   int       `json:"connections"`
}

/* CaptchaSession 验证码会话（存储在Redis） */
type CaptchaSession struct {
	ID        string    `json:"id"`
	Code      string    `json:"code"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

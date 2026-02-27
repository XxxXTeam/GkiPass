package models

import (
	"time"
)

/*
TunnelProtocol 隧道通讯协议枚举
功能：定义入口节点与出口节点之间的传输协议类型。
这些协议决定了节点间流量如何封装和传输，不同协议在性能、
穿透能力和安全性上各有特点。

协议分类：

	基础协议：tcp, udp — 直接转发，无额外封装，性能最高
	WebSocket 协议：ws, wss — HTTP 兼容，可穿越 CDN/反代/防火墙
	TLS 协议：tls, tls-mux — 加密传输，tls-mux 支持单连接多路复用
	高性能协议：kcp, quic — 基于 UDP 的可靠传输，弱网环境表现优异

节点组的 DisabledProtocols 字段可禁用特定协议，
例如某些网络环境不支持 UDP 时可禁用 kcp 和 quic。
*/
type TunnelProtocol string

const (
	ProtocolTCP    TunnelProtocol = "tcp"     /* TCP 直连：最基础的传输协议，透明转发 TCP 流量，无额外开销 */
	ProtocolUDP    TunnelProtocol = "udp"     /* UDP 直连：转发 UDP 数据报，适用于游戏、VoIP 等实时场景 */
	ProtocolWS     TunnelProtocol = "ws"      /* WebSocket：基于 HTTP 升级的全双工协议，可穿越 HTTP 代理和 CDN */
	ProtocolWSS    TunnelProtocol = "wss"     /* WebSocket Secure：WS + TLS 加密，可穿越 HTTPS 代理 */
	ProtocolTLS    TunnelProtocol = "tls"     /* TLS 加密隧道：标准 TLS 加密的 TCP 连接，每个隧道一个 TLS 会话 */
	ProtocolTLSMux TunnelProtocol = "tls-mux" /* TLS 多路复用：单条 TLS 连接承载多个隧道流，减少握手开销 */
	ProtocolKCP    TunnelProtocol = "kcp"     /* KCP 协议：基于 UDP 的可靠传输，以带宽换延迟，适合高丢包网络 */
	ProtocolQUIC   TunnelProtocol = "quic"    /* QUIC 协议：基于 UDP 的加密传输（内置 TLS 1.3），0-RTT 连接，支持多路复用 */
)

/*
Tunnel 隧道模型
功能：定义入口节点到出口节点之间的流量转发通道。

三段协议模型：

	客户端 ←[IngressProtocol]→ 入口节点 ←[Protocol]→ 出口节点 ←[EgressProtocol]→ 目标服务器

	- IngressProtocol：客户端 ↔ 入口节点之间使用的协议（客户端感知的协议）
	- Protocol：入口节点 ↔ 出口节点之间的隧道通讯协议（节点间内部传输）
	- EgressProtocol：出口节点 ↔ 目标服务器之间使用的协议（目标服务感知的协议）

	示例：用户通过 TCP 连接入口节点，节点间用 WSS 加密隧道传输，出口节点用 TCP 连接目标
	  IngressProtocol=tcp, Protocol=wss, EgressProtocol=tcp

节点选择：
  - IngressNodeID / IngressGroupID：指定入口节点或入口组（监听端用户连接的节点）
  - EgressNodeID / EgressGroupID：指定出口节点或出口组（连接目标服务器的节点）
  - 优先使用 Group（组内自动负载均衡），Node 用于精确指定单个节点
*/
type Tunnel struct {
	BaseModel
	Name        string `gorm:"type:varchar(64);not null" json:"name"`             /* 隧道名称 */
	Description string `gorm:"type:varchar(256)" json:"description"`              /* 隧道描述 */
	Enabled     bool   `gorm:"default:true;not null" json:"enabled"`              /* 是否启用 */
	CreatedBy   string `gorm:"type:varchar(36);index;not null" json:"created_by"` /* 创建者用户 ID */

	/*
		节点配置：指定隧道的入口和出口
		NodeID 精确绑定单节点，GroupID 绑定节点组（组内自动调度）
		两者可混用：入口用组，出口用指定节点
	*/
	IngressNodeID  string `gorm:"type:varchar(36);index" json:"ingress_node_id"`  /* 入口节点 ID（精确指定） */
	EgressNodeID   string `gorm:"type:varchar(36);index" json:"egress_node_id"`   /* 出口节点 ID（精确指定） */
	IngressGroupID string `gorm:"type:varchar(36);index" json:"ingress_group_id"` /* 入口节点组 ID */
	EgressGroupID  string `gorm:"type:varchar(36);index" json:"egress_group_id"`  /* 出口节点组 ID */

	/*
		三段协议配置（详见上方三段协议模型说明）
		Protocol 是核心字段，决定节点间隧道如何传输；
		IngressProtocol / EgressProtocol 默认跟随 Protocol，
		可独立设置以实现协议转换（如客户端 TCP → 隧道 WSS → 目标 TCP）
	*/
	Protocol        TunnelProtocol `gorm:"type:varchar(16);default:'tcp';not null" json:"protocol"` /* 节点间隧道通讯协议 */
	IngressProtocol TunnelProtocol `gorm:"type:varchar(16);default:'tcp'" json:"ingress_protocol"`  /* 客户端 → 入口节点协议 */
	EgressProtocol  TunnelProtocol `gorm:"type:varchar(16);default:'tcp'" json:"egress_protocol"`   /* 出口节点 → 目标服务器协议 */

	/* 端口与目标：入口监听 + 出口连接 */
	ListenPort    int    `gorm:"not null" json:"listen_port"`                      /* 入口节点监听端口（客户端连接此端口） */
	TargetAddress string `gorm:"type:varchar(256);not null" json:"target_address"` /* 目标服务器地址（出口节点连接此地址） */
	TargetPort    int    `gorm:"not null" json:"target_port"`                      /* 目标服务器端口 */

	/* 加密配置：对隧道数据进行额外加密（独立于协议层加密） */
	EnableEncryption bool   `gorm:"default:false" json:"enable_encryption"`                          /* 是否启用应用层加密 */
	EncryptionMethod string `gorm:"type:varchar(32);default:'aes-256-gcm'" json:"encryption_method"` /* 加密算法：aes-256-gcm, chacha20-poly1305 */

	/* 流量控制 */
	RateLimitBPS   int64 `gorm:"default:0" json:"rate_limit_bps"`  /* 带宽限制（bit/s），0 表示不限制 */
	MaxConnections int   `gorm:"default:0" json:"max_connections"` /* 最大并发连接数，0 表示不限制 */
	IdleTimeout    int   `gorm:"default:300" json:"idle_timeout"`  /* 空闲连接超时（秒） */

	/* 负载均衡：多目标时的调度策略 */
	LoadBalanceMode string `gorm:"type:varchar(32);default:'round-robin'" json:"load_balance_mode"` /* round-robin, weighted, least-conn, ip-hash */

	/* 运行时统计信息（由节点周期上报） */
	ConnectionCount int64     `gorm:"default:0" json:"connection_count"` /* 累计连接次数 */
	BytesIn         int64     `gorm:"default:0" json:"bytes_in"`         /* 累计入站流量（字节） */
	BytesOut        int64     `gorm:"default:0" json:"bytes_out"`        /* 累计出站流量（字节） */
	LastActive      time.Time `gorm:"" json:"last_active"`               /* 最后活跃时间 */

	/* 关联模型 */
	Rules   []Rule         `gorm:"foreignKey:TunnelID" json:"rules,omitempty"`   /* 转发规则列表 */
	Targets []TunnelTarget `gorm:"foreignKey:TunnelID" json:"targets,omitempty"` /* 目标地址列表（负载均衡） */
	Creator User           `gorm:"foreignKey:CreatedBy" json:"-"`                /* 创建者用户 */
}

func (Tunnel) TableName() string {
	return "tunnels"
}

/*
TunnelTarget 隧道目标地址
功能：支持一个隧道配置多个目标地址，用于负载均衡和故障转移
*/
type TunnelTarget struct {
	BaseModel
	TunnelID string `gorm:"type:varchar(36);index;not null" json:"tunnel_id"`
	Host     string `gorm:"type:varchar(256);not null" json:"host"`
	Port     int    `gorm:"not null" json:"port"`
	Weight   int    `gorm:"default:1" json:"weight"`
	Enabled  bool   `gorm:"default:true" json:"enabled"`
	Healthy  bool   `gorm:"default:true" json:"healthy"`

	/* 关联 */
	Tunnel Tunnel `gorm:"foreignKey:TunnelID" json:"-"`
}

func (TunnelTarget) TableName() string {
	return "tunnel_targets"
}

/*
Rule 转发规则模型
功能：定义具体的流量转发规则，包括协议、端口、ACL 和高级选项
*/
type Rule struct {
	BaseModel
	Name        string `gorm:"type:varchar(64);not null" json:"name"`
	Description string `gorm:"type:varchar(256)" json:"description"`
	Enabled     bool   `gorm:"default:true;not null" json:"enabled"`
	Priority    int    `gorm:"default:0;index" json:"priority"`
	Version     int64  `gorm:"default:1" json:"version"`
	CreatedBy   string `gorm:"type:varchar(36);index" json:"created_by"`
	TunnelID    string `gorm:"type:varchar(36);index" json:"tunnel_id"`
	GroupID     string `gorm:"type:varchar(36);index" json:"group_id"`

	/* 隧道配置 */
	Protocol        TunnelProtocol `gorm:"type:varchar(16);default:'tcp';not null" json:"protocol"`
	ListenPort      int            `gorm:"not null" json:"listen_port"`
	TargetAddress   string         `gorm:"type:varchar(256);not null" json:"target_address"`
	TargetPort      int            `gorm:"not null" json:"target_port"`
	IngressNodeID   string         `gorm:"type:varchar(36);index" json:"ingress_node_id"`
	EgressNodeID    string         `gorm:"type:varchar(36);index" json:"egress_node_id"`
	IngressGroupID  string         `gorm:"type:varchar(36);index" json:"ingress_group_id"`
	EgressGroupID   string         `gorm:"type:varchar(36);index" json:"egress_group_id"`
	IngressProtocol TunnelProtocol `gorm:"type:varchar(16);default:'tcp'" json:"ingress_protocol"`
	EgressProtocol  TunnelProtocol `gorm:"type:varchar(16);default:'tcp'" json:"egress_protocol"`

	/* 高级选项 */
	EnableEncryption bool  `gorm:"default:false" json:"enable_encryption"`
	RateLimitBPS     int64 `gorm:"default:0" json:"rate_limit_bps"`
	MaxConnections   int   `gorm:"default:0" json:"max_connections"`
	IdleTimeout      int   `gorm:"default:300" json:"idle_timeout"`

	/* 统计信息 */
	ConnectionCount int64     `gorm:"default:0" json:"connection_count"`
	BytesIn         int64     `gorm:"default:0" json:"bytes_in"`
	BytesOut        int64     `gorm:"default:0" json:"bytes_out"`
	LastActive      time.Time `gorm:"" json:"last_active"`

	/* 关联 */
	ACLRules []ACLRule `gorm:"foreignKey:RuleID" json:"acl_rules,omitempty"`
	Tunnel   Tunnel    `gorm:"foreignKey:TunnelID" json:"-"`
}

func (Rule) TableName() string {
	return "rules"
}

/*
ACLRule 访问控制规则
功能：定义基于IP、端口、协议的流量过滤策略
*/
type ACLRule struct {
	BaseModel
	RuleID    string `gorm:"type:varchar(36);index;not null" json:"rule_id"`
	Action    string `gorm:"type:varchar(16);not null" json:"action"`
	Priority  int    `gorm:"default:0" json:"priority"`
	SourceIP  string `gorm:"type:varchar(64)" json:"source_ip"`
	DestIP    string `gorm:"type:varchar(64)" json:"dest_ip"`
	Protocol  string `gorm:"type:varchar(16)" json:"protocol"`
	PortRange string `gorm:"type:varchar(64)" json:"port_range"`

	/* 关联 */
	Rule Rule `gorm:"foreignKey:RuleID" json:"-"`
}

func (ACLRule) TableName() string {
	return "rule_acls"
}

/*
TrafficStats 流量统计
功能：按时间维度记录隧道/节点/用户级别的流量统计数据
*/
type TrafficStats struct {
	BaseModel
	NodeID   string `gorm:"type:varchar(36);index:idx_traffic_user_tunnel" json:"node_id"`
	TunnelID string `gorm:"type:varchar(36);index:idx_traffic_user_tunnel" json:"tunnel_id"`
	UserID   string `gorm:"type:varchar(36);index:idx_traffic_user_tunnel" json:"user_id"`
	RuleID   string `gorm:"type:varchar(36);index" json:"rule_id"`

	/* 流量数据 */
	BytesIn     int64 `gorm:"default:0" json:"bytes_in"`
	BytesOut    int64 `gorm:"default:0" json:"bytes_out"`
	Connections int64 `gorm:"default:0" json:"connections"`

	/* 时间维度 */
	Period    string    `gorm:"type:varchar(16);index;not null" json:"period"`
	PeriodKey string    `gorm:"type:varchar(32);index;not null" json:"period_key"`
	StartAt   time.Time `gorm:"index;not null" json:"start_at"`
	EndAt     time.Time `gorm:"not null" json:"end_at"`
}

func (TrafficStats) TableName() string {
	return "traffic_stats"
}

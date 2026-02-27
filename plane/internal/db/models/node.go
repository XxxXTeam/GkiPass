package models

import (
	"time"
)

/*
NodeStatus 节点状态枚举
功能：定义节点在其生命周期中的运行状态，用于健康监控和流量调度决策

状态流转：

	disabled → connecting → online ⇄ error → offline
	管理员可随时将节点置为 disabled
*/
type NodeStatus string

const (
	NodeStatusOnline     NodeStatus = "online"     /* 在线：已通过认证且心跳正常，可接受流量 */
	NodeStatusOffline    NodeStatus = "offline"    /* 离线：超过心跳超时未响应，自动标记 */
	NodeStatusError      NodeStatus = "error"      /* 异常：节点上报了错误或控制面板检测到故障 */
	NodeStatusConnecting NodeStatus = "connecting" /* 连接中：已建立 WebSocket 但尚未完成认证握手 */
	NodeStatusDisabled   NodeStatus = "disabled"   /* 已禁用：管理员手动禁用，不参与流量调度 */
)

/*
NodeRole 节点角色枚举
功能：定义节点在隧道流量路径中承担的职责

流量路径模型：

	客户端 → [入口节点 Ingress] → (隧道协议) → [出口节点 Egress] → 目标服务器

	- Ingress（入口节点）：面向客户端，监听端口接受用户连接，
	  将流量通过隧道协议转发给出口节点。通常部署在靠近用户的边缘位置。
	- Egress（出口节点）：面向目标服务器，接收入口节点转发的流量，
	  解包后连接到实际目标地址。通常部署在靠近目标服务的位置。
	- Both（双向节点）：同时具备入口和出口能力，既可以接受客户端连接，
	  也可以作为出口连接目标服务器。适用于单节点部署或中转节点。
*/
type NodeRole string

const (
	NodeRoleIngress NodeRole = "ingress" /* 入口节点：监听客户端连接，转发到出口节点 */
	NodeRoleEgress  NodeRole = "egress"  /* 出口节点：接收隧道流量，连接目标服务器 */
	NodeRoleBoth    NodeRole = "both"    /* 双向节点：同时承担入口和出口职责 */
)

/*
Node 节点模型
功能：存储节点基本信息、硬件信息、网络配置和运行状态。
每个节点是一个运行 GkiPass 客户端程序的服务器实例，通过 WebSocket 长连接
与控制面板保持通信，接收规则下发和配置更新。
*/
type Node struct {
	BaseModel
	Name        string     `gorm:"type:varchar(64);not null" json:"name"`                           /* 节点显示名称 */
	Description string     `gorm:"type:varchar(256)" json:"description"`                            /* 节点描述信息 */
	Status      NodeStatus `gorm:"type:varchar(16);default:'offline';not null;index" json:"status"` /* 当前运行状态 */
	LastOnline  time.Time  `gorm:"" json:"last_online"`                                             /* 最后一次在线时间 */

	/* 硬件信息：节点首次注册或心跳时上报 */
	HardwareID string `gorm:"type:varchar(128);index" json:"hardware_id"` /* 硬件唯一标识（防止重复注册） */
	SystemInfo string `gorm:"type:text" json:"system_info"`               /* 系统信息 JSON（OS、CPU、内存等） */
	IPAddress  string `gorm:"type:varchar(64)" json:"ip_address"`         /* 节点上报的 IP 地址 */
	Version    string `gorm:"type:varchar(32)" json:"version"`            /* 节点客户端版本号 */

	/* 角色与组：决定节点在隧道中的职责 */
	Role   NodeRole    `gorm:"type:varchar(16);default:'both';not null" json:"role"` /* 节点角色：ingress/egress/both */
	Groups []NodeGroup `gorm:"many2many:node_group_nodes;" json:"groups,omitempty"`  /* 所属节点组列表 */

	/* 网络配置：用于节点间隧道建立 */
	PublicIP   string `gorm:"type:varchar(64)" json:"public_ip"`   /* 公网 IP（用于隧道连接） */
	InternalIP string `gorm:"type:varchar(64)" json:"internal_ip"` /* 内网 IP（用于同机房直连） */
	Port       int    `gorm:"default:0" json:"port"`               /* 隧道监听端口 */

	/* 认证凭证：用于节点 WebSocket 连接认证 */
	Token     string `gorm:"type:varchar(256)" json:"-"` /* 节点认证令牌（不序列化） */
	APIKey    string `gorm:"type:varchar(256)" json:"-"` /* API 调用密钥（不序列化） */
	SecretKey string `gorm:"type:varchar(256)" json:"-"` /* 密钥（不序列化到 JSON） */
}

func (Node) TableName() string {
	return "nodes"
}

/*
NodeGroup 节点组模型
功能：将节点按照用途或地区分组管理。
隧道创建时选择"入口组"和"出口组"，系统自动在组内节点间建立隧道。

典型部署示例：
  - "香港入口组"(role=ingress) → "日本出口组"(role=egress)
  - "全能组"(role=both) 可同时作为入口和出口
*/
type NodeGroup struct {
	BaseModel
	Name        string   `gorm:"type:varchar(64);uniqueIndex;not null" json:"name"`    /* 组名称（唯一） */
	Description string   `gorm:"type:varchar(256)" json:"description"`                 /* 组描述 */
	Role        NodeRole `gorm:"type:varchar(16);default:'both';not null" json:"role"` /* 组角色：ingress/egress/both */

	/*
		入口-出口关联设置
		当 RequiresEgress=true 时，该入口组必须搭配出口组才能创建隧道；
		DefaultEgressID 指定默认的出口组，简化用户操作。
	*/
	RequiresEgress  bool   `gorm:"default:false" json:"requires_egress"`      /* 是否要求配对出口组 */
	DefaultEgressID string `gorm:"type:varchar(36)" json:"default_egress_id"` /* 默认出口组 ID */

	/*
		协议与端口限制
		DisabledProtocols：JSON 数组，该组禁止使用的隧道通讯协议。
		  例如 ["udp","kcp"] 表示该组节点不支持 UDP 和 KCP 协议，
		  用户创建隧道时无法选择这些协议。
		  可选值：tcp, udp, ws, wss, tls, tls-mux, kcp, quic
		AllowedPortRanges：JSON 数组，限制入口监听端口范围。
		  例如 ["10000-20000","30000-40000"]，为空则不限制。
	*/
	DisabledProtocols string `gorm:"type:text" json:"disabled_protocols"`  /* 禁用的隧道通讯协议列表（JSON 数组） */
	AllowedPortRanges string `gorm:"type:text" json:"allowed_port_ranges"` /* 允许的入口监听端口范围（JSON 数组） */

	/* 权限设置 */
	AllowProbeView bool `gorm:"default:false" json:"allow_probe_view"` /* 是否允许普通用户查看该组节点的探测数据 */

	/*
		出口容灾策略配置（节点自主容灾）
		当该组作为出口组时，面板在规则同步中将容灾策略下发给入口节点。
		入口节点自行检测到出口不可达后，根据以下策略自主切换到容灾组，
		无需等待面板下发新规则，实现秒级容灾。

		节点自主容灾流程：
		  1. 入口节点持续连接出口组内的节点
		  2. 所有出口节点连续 FailoverTimeout 秒不可达
		  3. 入口节点自动切换到 FailoverGroupID 对应的容灾出口组
		  4. 原出口组恢复后，若 FailoverAutoRecover=true 则节点自动回切
		  5. 节点将切换/回切事件上报给面板（failover_event 消息）

		面板的职责仅为：
		  - 定义容灾策略并随规则一起下发
		  - 接收和记录节点上报的容灾事件
		  - 在仪表盘展示容灾状态
	*/
	FailoverGroupID     string `gorm:"type:varchar(36)" json:"failover_group_id"` /* 容灾出口组 ID，为空表示不启用容灾 */
	FailoverTimeout     int    `gorm:"default:60" json:"failover_timeout"`        /* 容灾触发超时（秒），出口持续不可达超过此时间触发切换 */
	FailoverAutoRecover bool   `gorm:"default:true" json:"failover_auto_recover"` /* 是否自动回切：原出口组恢复后节点自动切回 */

	/* 关联节点 */
	Nodes []Node `gorm:"many2many:node_group_nodes;" json:"nodes,omitempty"` /* 组内节点列表 */
}

func (NodeGroup) TableName() string {
	return "node_groups"
}

/*
NodeMetrics 节点实时指标
功能：存储节点的运行时性能指标数据
*/
type NodeMetrics struct {
	BaseModel
	NodeID      string  `gorm:"type:varchar(36);index;not null" json:"node_id"`
	CPUUsage    float64 `gorm:"type:decimal(5,2)" json:"cpu_usage"`
	MemoryUsage float64 `gorm:"type:decimal(5,2)" json:"memory_usage"`
	DiskUsage   float64 `gorm:"type:decimal(5,2)" json:"disk_usage"`
	NetworkIn   int64   `gorm:"default:0" json:"network_in"`
	NetworkOut  int64   `gorm:"default:0" json:"network_out"`
	Connections int     `gorm:"default:0" json:"connections"`
	Goroutines  int     `gorm:"default:0" json:"goroutines"`
	GCPause     int64   `gorm:"default:0" json:"gc_pause"`

	/* 关联 */
	Node Node `gorm:"foreignKey:NodeID" json:"-"`
}

func (NodeMetrics) TableName() string {
	return "node_metrics"
}

/*
NodeCertificate 节点证书
功能：管理节点的 TLS 证书信息
*/
type NodeCertificate struct {
	BaseModel
	NodeID      string    `gorm:"type:varchar(36);index;not null" json:"node_id"`
	Type        string    `gorm:"type:varchar(16);not null" json:"type"`
	CommonName  string    `gorm:"type:varchar(128)" json:"common_name"`
	CertPEM     string    `gorm:"type:text" json:"-"`
	KeyPEM      string    `gorm:"type:text" json:"-"`
	CAPem       string    `gorm:"type:text" json:"-"`
	NotBefore   time.Time `gorm:"" json:"not_before"`
	NotAfter    time.Time `gorm:"index" json:"not_after"`
	Fingerprint string    `gorm:"type:varchar(128);uniqueIndex" json:"fingerprint"`
	Revoked     bool      `gorm:"default:false" json:"revoked"`

	/* 关联 */
	Node Node `gorm:"foreignKey:NodeID" json:"-"`
}

func (NodeCertificate) TableName() string {
	return "node_certificates"
}

/*
ConnectionKey 节点连接密钥
功能：用于节点间安全连接认证
*/
type ConnectionKey struct {
	BaseModel
	NodeID    string    `gorm:"type:varchar(36);index;not null" json:"node_id"`
	Key       string    `gorm:"type:varchar(256);uniqueIndex;not null" json:"key"`
	Type      string    `gorm:"type:varchar(16);default:'node';not null" json:"type"` /* node / user */
	Label     string    `gorm:"type:varchar(64)" json:"label"`
	ExpiresAt time.Time `gorm:"index" json:"expires_at"`
	Revoked   bool      `gorm:"default:false" json:"revoked"`
	LastUsed  time.Time `gorm:"" json:"last_used"`

	/* 关联 */
	Node Node `gorm:"foreignKey:NodeID" json:"-"`
}

func (ConnectionKey) TableName() string {
	return "connection_keys"
}

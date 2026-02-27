package models

/*
Policy 策略模型
功能：定义流量策略（协议限制、ACL、路由规则），可绑定到节点
*/
type Policy struct {
	BaseModel
	Name        string `gorm:"type:varchar(64);not null" json:"name"`
	Type        string `gorm:"type:varchar(32);not null;index" json:"type"`    /* protocol / acl / routing */
	Priority    int    `gorm:"default:0" json:"priority"`
	Enabled     bool   `gorm:"default:true" json:"enabled"`
	Config      string `gorm:"type:text" json:"config"`                        /* JSON 格式的策略配置 */
	NodeIDs     string `gorm:"type:text" json:"node_ids"`                      /* JSON 格式的节点ID列表 */
	Description string `gorm:"type:varchar(256)" json:"description"`
}

func (Policy) TableName() string {
	return "policies"
}

/*
NodeGroupConfig 节点组配置
功能：定义节点组的协议限制、端口范围、流量倍率等运营参数
*/
type NodeGroupConfig struct {
	BaseModel
	GroupID           string  `gorm:"type:varchar(36);uniqueIndex;not null" json:"group_id"`
	AllowedProtocols  string  `gorm:"type:text" json:"allowed_protocols"`    /* JSON 数组 */
	PortRange         string  `gorm:"type:varchar(32)" json:"port_range"`    /* 如 "10000-60000" */
	TrafficMultiplier float64 `gorm:"type:decimal(4,2);default:1.0" json:"traffic_multiplier"`

	/* 关联 */
	Group NodeGroup `gorm:"foreignKey:GroupID" json:"-"`
}

func (NodeGroupConfig) TableName() string {
	return "node_group_configs"
}

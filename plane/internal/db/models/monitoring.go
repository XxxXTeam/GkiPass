package models

import "time"

/*
NodeMonitoringConfig 节点监控配置
功能：控制每个节点的监控参数（采集间隔、告警阈值、数据保留天数等）
*/
type NodeMonitoringConfig struct {
	ID                   string    `json:"id" gorm:"primaryKey;size:36"`
	NodeID               string    `json:"node_id" gorm:"size:36;uniqueIndex"`
	MonitoringEnabled    bool      `json:"monitoring_enabled" gorm:"default:true"`
	ReportInterval       int       `json:"report_interval" gorm:"default:60"`
	CollectSystemInfo    bool      `json:"collect_system_info" gorm:"default:true"`
	CollectNetworkStats  bool      `json:"collect_network_stats" gorm:"default:true"`
	CollectTunnelStats   bool      `json:"collect_tunnel_stats" gorm:"default:true"`
	CollectPerformance   bool      `json:"collect_performance" gorm:"default:true"`
	DataRetentionDays    int       `json:"data_retention_days" gorm:"default:30"`
	AlertCPUThreshold    float64   `json:"alert_cpu_threshold" gorm:"default:80"`
	AlertMemoryThreshold float64   `json:"alert_memory_threshold" gorm:"default:80"`
	AlertDiskThreshold   float64   `json:"alert_disk_threshold" gorm:"default:90"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

func (NodeMonitoringConfig) TableName() string { return "node_monitoring_configs" }

/*
DefaultNodeMonitoringConfig 返回指定节点的默认监控配置
功能：统一默认值，避免多处硬编码
*/
func DefaultNodeMonitoringConfig(nodeID string) *NodeMonitoringConfig {
	return &NodeMonitoringConfig{
		NodeID:               nodeID,
		MonitoringEnabled:    true,
		ReportInterval:       60,
		CollectSystemInfo:    true,
		CollectNetworkStats:  true,
		CollectTunnelStats:   true,
		CollectPerformance:   true,
		DataRetentionDays:    30,
		AlertCPUThreshold:    80.0,
		AlertMemoryThreshold: 80.0,
		AlertDiskThreshold:   90.0,
	}
}

/*
NodeMonitoringData 节点实时监控数据
功能：存储节点周期性上报的 CPU / 内存 / 磁盘 / 网络 / 隧道等指标
*/
type NodeMonitoringData struct {
	ID        string    `json:"id" gorm:"primaryKey;size:36"`
	NodeID    string    `json:"node_id" gorm:"size:36;index"`
	Timestamp time.Time `json:"timestamp" gorm:"index"`

	/* 系统 */
	SystemUptime int64 `json:"system_uptime"`

	/* CPU */
	CPUUsage   float64 `json:"cpu_usage"`
	CPULoad1m  float64 `json:"cpu_load_1m"`
	CPULoad5m  float64 `json:"cpu_load_5m"`
	CPULoad15m float64 `json:"cpu_load_15m"`
	CPUCores   int     `json:"cpu_cores"`

	/* 内存 */
	MemoryTotal        int64   `json:"memory_total"`
	MemoryUsed         int64   `json:"memory_used"`
	MemoryAvailable    int64   `json:"memory_available"`
	MemoryUsagePercent float64 `json:"memory_usage_percent"`

	/* 磁盘 */
	DiskTotal        int64   `json:"disk_total"`
	DiskUsed         int64   `json:"disk_used"`
	DiskAvailable    int64   `json:"disk_available"`
	DiskUsagePercent float64 `json:"disk_usage_percent"`

	/* 网络 */
	BandwidthIn  int64 `json:"bandwidth_in"`
	BandwidthOut int64 `json:"bandwidth_out"`

	/* 连接 */
	TCPConnections   int `json:"tcp_connections"`
	UDPConnections   int `json:"udp_connections"`
	ActiveTunnels    int `json:"active_tunnels"`
	TotalConnections int `json:"total_connections"`

	/* 流量 */
	TrafficInBytes  int64 `json:"traffic_in_bytes"`
	TrafficOutBytes int64 `json:"traffic_out_bytes"`
	PacketsIn       int64 `json:"packets_in"`
	PacketsOut      int64 `json:"packets_out"`

	/* 错误 */
	ConnectionErrors int `json:"connection_errors"`
	TunnelErrors     int `json:"tunnel_errors"`

	/* 性能 */
	AvgResponseTime float64 `json:"avg_response_time"`
	MaxResponseTime float64 `json:"max_response_time"`
	MinResponseTime float64 `json:"min_response_time"`
}

func (NodeMonitoringData) TableName() string { return "node_monitoring_data" }

/*
NodePerformanceHistory 节点性能历史（小时/天聚合）
功能：聚合后的性能趋势数据，用于图表展示
*/
type NodePerformanceHistory struct {
	ID              string    `json:"id" gorm:"primaryKey;size:36"`
	NodeID          string    `json:"node_id" gorm:"size:36;index"`
	Date            time.Time `json:"date" gorm:"index"`
	AggregationType string    `json:"aggregation_type" gorm:"size:16"` /* hourly / daily */
	AggregationTime time.Time `json:"aggregation_time"`

	AvgCPUUsage     float64 `json:"avg_cpu_usage"`
	AvgMemoryUsage  float64 `json:"avg_memory_usage"`
	AvgDiskUsage    float64 `json:"avg_disk_usage"`
	AvgBandwidthIn  int64   `json:"avg_bandwidth_in"`
	AvgBandwidthOut int64   `json:"avg_bandwidth_out"`
	AvgConnections  int     `json:"avg_connections"`
	AvgResponseTime float64 `json:"avg_response_time"`

	MaxCPUUsage     float64 `json:"max_cpu_usage"`
	MaxMemoryUsage  float64 `json:"max_memory_usage"`
	MaxConnections  int     `json:"max_connections"`
	MaxResponseTime float64 `json:"max_response_time"`

	TotalTrafficIn  int64 `json:"total_traffic_in"`
	TotalTrafficOut int64 `json:"total_traffic_out"`
	TotalPacketsIn  int64 `json:"total_packets_in"`
	TotalPacketsOut int64 `json:"total_packets_out"`
	TotalErrors     int   `json:"total_errors"`

	UptimeSeconds       int64   `json:"uptime_seconds"`
	DowntimeSeconds     int64   `json:"downtime_seconds"`
	AvailabilityPercent float64 `json:"availability_percent"`

	CreatedAt time.Time `json:"created_at"`
}

func (NodePerformanceHistory) TableName() string { return "node_performance_history" }

/*
NodeAlertRule 节点告警规则
功能：定义针对某个指标的告警阈值和触发条件
*/
type NodeAlertRule struct {
	ID                   string    `json:"id" gorm:"primaryKey;size:36"`
	NodeID               string    `json:"node_id" gorm:"size:36;index"`
	RuleName             string    `json:"rule_name" gorm:"size:128"`
	MetricType           string    `json:"metric_type" gorm:"size:32"` /* cpu / memory / disk / response_time / connections */
	Operator             string    `json:"operator" gorm:"size:4"`     /* > < >= <= = != */
	ThresholdValue       float64   `json:"threshold_value"`
	DurationSeconds      int       `json:"duration_seconds"`
	Severity             string    `json:"severity" gorm:"size:16"` /* info / warning / critical */
	Enabled              bool      `json:"enabled" gorm:"default:true"`
	NotificationChannels string    `json:"notification_channels" gorm:"size:256"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

func (NodeAlertRule) TableName() string { return "node_alert_rules" }

/*
NodeAlertHistory 节点告警历史记录
功能：记录每次触发的告警事件
*/
type NodeAlertHistory struct {
	ID             string     `json:"id" gorm:"primaryKey;size:36"`
	RuleID         string     `json:"rule_id" gorm:"size:36;index"`
	NodeID         string     `json:"node_id" gorm:"size:36;index"`
	AlertType      string     `json:"alert_type" gorm:"size:32"`
	Severity       string     `json:"severity" gorm:"size:16"`
	Message        string     `json:"message" gorm:"size:512"`
	MetricValue    float64    `json:"metric_value"`
	ThresholdValue float64    `json:"threshold_value"`
	Status         string     `json:"status" gorm:"size:16"` /* triggered / acknowledged / resolved */
	TriggeredAt    time.Time  `json:"triggered_at"`
	AcknowledgedAt *time.Time `json:"acknowledged_at,omitempty"`
	ResolvedAt     *time.Time `json:"resolved_at,omitempty"`
	AcknowledgedBy string     `json:"acknowledged_by,omitempty" gorm:"size:36"`
	Details        string     `json:"details,omitempty" gorm:"type:text"`
}

func (NodeAlertHistory) TableName() string { return "node_alert_history" }

/*
MonitoringPermission 监控权限
功能：控制用户对节点监控数据的访问权限
*/
type MonitoringPermission struct {
	ID             string    `json:"id" gorm:"primaryKey;size:36"`
	UserID         string    `json:"user_id" gorm:"size:36;index"`
	NodeID         string    `json:"node_id" gorm:"size:36;index"`
	PermissionType string    `json:"permission_type" gorm:"size:32"` /* view_basic / view_detailed / view_system / view_network / disabled */
	Enabled        bool      `json:"enabled" gorm:"default:true"`
	CreatedBy      string    `json:"created_by" gorm:"size:36"`
	Description    string    `json:"description" gorm:"size:256"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

func (MonitoringPermission) TableName() string { return "monitoring_permissions" }

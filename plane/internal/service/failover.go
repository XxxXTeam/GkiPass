package service

import (
	"sync"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"gkipass/plane/internal/db/models"
)

/*
FailoverEvent 节点上报的容灾事件
功能：记录入口节点自主容灾切换/回切的事件详情
*/
type FailoverEvent struct {
	models.BaseModel
	NodeID          string    `gorm:"type:varchar(36);index;not null" json:"node_id"`   /* 上报事件的入口节点 ID */
	TunnelID        string    `gorm:"type:varchar(36);index;not null" json:"tunnel_id"` /* 涉及的隧道 ID */
	EventType       string    `gorm:"type:varchar(16);not null" json:"event_type"`      /* 事件类型：failover / recovery */
	FromGroupID     string    `gorm:"type:varchar(36);not null" json:"from_group_id"`   /* 原出口组 ID */
	ToGroupID       string    `gorm:"type:varchar(36);not null" json:"to_group_id"`     /* 切换目标出口组 ID */
	Reason          string    `gorm:"type:varchar(256)" json:"reason"`                  /* 触发原因描述 */
	FailureDuration int       `gorm:"default:0" json:"failure_duration"`                /* 故障持续时长（秒） */
	Timestamp       time.Time `gorm:"index;not null" json:"timestamp"`                  /* 事件发生时间（节点本地时间） */
}

func (FailoverEvent) TableName() string {
	return "failover_events"
}

/*
FailoverEventReport 节点上报容灾事件的请求结构
功能：节点通过 WebSocket failover_event 消息上报，面板解析后调用 HandleEvent
*/
type FailoverEventReport struct {
	NodeID          string `json:"node_id"`          /* 上报节点 ID */
	TunnelID        string `json:"tunnel_id"`        /* 隧道 ID */
	EventType       string `json:"event_type"`       /* failover / recovery */
	FromGroupID     string `json:"from_group_id"`    /* 原出口组 */
	ToGroupID       string `json:"to_group_id"`      /* 目标出口组 */
	Reason          string `json:"reason"`           /* 原因：timeout / all_nodes_down / manual */
	FailureDuration int    `json:"failure_duration"` /* 故障持续秒数 */
	Timestamp       int64  `json:"timestamp"`        /* 毫秒时间戳 */
}

/*
FailoverService 出口容灾事件服务（被动接收模式）
功能：接收并记录入口节点自主上报的容灾切换/回切事件，
提供事件查询和统计 API 供仪表盘展示。

架构说明：

	面板（本服务）不主动执行容灾切换，仅负责：
	1. 在规则同步时将容灾策略（FailoverGroupID/Timeout/AutoRecover）下发给入口节点
	2. 接收入口节点通过 WebSocket 上报的 failover_event 消息
	3. 持久化事件到 failover_events 表
	4. 维护实时容灾状态缓存供 API 查询

	实际容灾决策完全由入口节点自主完成：
	- 节点检测出口连接失败 → 计时 → 超时切换 → 上报事件
	- 节点检测原出口恢复 → 自动回切 → 上报事件
*/
type FailoverService struct {
	gormDB *gorm.DB
	logger *zap.Logger

	/*
		实时容灾状态缓存
		key = "nodeID:tunnelID" → 最新事件快照
		由节点上报事件时更新，供 API 快速查询
	*/
	activeFailovers map[string]*FailoverEventReport
	mu              sync.RWMutex
}

/*
NewFailoverService 创建容灾事件服务
*/
func NewFailoverService(gormDB *gorm.DB) *FailoverService {
	return &FailoverService{
		gormDB:          gormDB,
		logger:          zap.L().Named("failover"),
		activeFailovers: make(map[string]*FailoverEventReport),
	}
}

/*
Start 启动容灾事件服务
功能：自动迁移事件表 + 从数据库加载未恢复的容灾事件到缓存
*/
func (s *FailoverService) Start() {
	/* 自动创建 failover_events 表 */
	if err := s.gormDB.AutoMigrate(&FailoverEvent{}); err != nil {
		s.logger.Error("自动迁移 failover_events 表失败", zap.Error(err))
	}

	/* 加载未恢复的容灾事件到缓存 */
	s.loadActiveFailovers()

	s.logger.Info("✓ 出口容灾事件服务已启动（被动接收模式）")
}

/*
Stop 停止容灾事件服务
*/
func (s *FailoverService) Stop() {
	s.logger.Info("出口容灾事件服务已停止")
}

/*
HandleEvent 处理节点上报的容灾事件
功能：验证事件 → 持久化到数据库 → 更新内存缓存 → 记录日志
由 WebSocket 消息处理器在收到 failover_event 类型消息时调用
*/
func (s *FailoverService) HandleEvent(report *FailoverEventReport) error {
	eventTime := time.UnixMilli(report.Timestamp)
	if report.Timestamp == 0 {
		eventTime = time.Now()
	}

	/* 持久化到数据库 */
	event := &FailoverEvent{
		NodeID:          report.NodeID,
		TunnelID:        report.TunnelID,
		EventType:       report.EventType,
		FromGroupID:     report.FromGroupID,
		ToGroupID:       report.ToGroupID,
		Reason:          report.Reason,
		FailureDuration: report.FailureDuration,
		Timestamp:       eventTime,
	}

	if err := s.gormDB.Create(event).Error; err != nil {
		s.logger.Error("保存容灾事件失败", zap.Error(err))
		return err
	}

	/* 更新内存缓存 */
	cacheKey := report.NodeID + ":" + report.TunnelID
	s.mu.Lock()
	if report.EventType == "failover" {
		s.activeFailovers[cacheKey] = report
	} else if report.EventType == "recovery" {
		delete(s.activeFailovers, cacheKey)
	}
	s.mu.Unlock()

	/* 结构化日志 */
	logFunc := s.logger.Info
	if report.EventType == "failover" {
		logFunc = s.logger.Warn
	}
	logFunc("节点容灾事件",
		zap.String("event_type", report.EventType),
		zap.String("node_id", report.NodeID),
		zap.String("tunnel_id", report.TunnelID),
		zap.String("from_group", report.FromGroupID),
		zap.String("to_group", report.ToGroupID),
		zap.String("reason", report.Reason),
		zap.Int("failure_duration_sec", report.FailureDuration))

	return nil
}

/*
GetActiveFailovers 获取当前所有处于容灾状态的隧道
功能：供仪表盘实时展示哪些隧道正在使用容灾出口
*/
func (s *FailoverService) GetActiveFailovers() []*FailoverEventReport {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*FailoverEventReport, 0, len(s.activeFailovers))
	for _, event := range s.activeFailovers {
		copied := *event
		result = append(result, &copied)
	}
	return result
}

/*
GetTunnelFailoverHistory 获取指定隧道的容灾事件历史
功能：供隧道详情页查看历史容灾记录
*/
func (s *FailoverService) GetTunnelFailoverHistory(tunnelID string, limit int) ([]FailoverEvent, error) {
	if limit <= 0 {
		limit = 50
	}
	var events []FailoverEvent
	err := s.gormDB.
		Where("tunnel_id = ?", tunnelID).
		Order("timestamp DESC").
		Limit(limit).
		Find(&events).Error
	return events, err
}

/*
GetGroupFailoverSummary 获取出口组的容灾状态摘要
功能：统计该组相关隧道的容灾情况，供 API 和前端展示
*/
func (s *FailoverService) GetGroupFailoverSummary(groupID string) map[string]interface{} {
	s.mu.RLock()
	activeCount := 0
	for _, event := range s.activeFailovers {
		if event.FromGroupID == groupID {
			activeCount++
		}
	}
	s.mu.RUnlock()

	/* 查询最近 24 小时的事件统计 */
	var totalEvents int64
	since := time.Now().Add(-24 * time.Hour)
	s.gormDB.Model(&FailoverEvent{}).
		Where("(from_group_id = ? OR to_group_id = ?) AND timestamp > ?", groupID, groupID, since).
		Count(&totalEvents)

	return map[string]interface{}{
		"group_id":                groupID,
		"active_failover_tunnels": activeCount,
		"events_last_24h":         totalEvents,
	}
}

/* loadActiveFailovers 从数据库加载未恢复的容灾到缓存 */
func (s *FailoverService) loadActiveFailovers() {
	/*
		查找所有最新事件类型为 failover（而非 recovery）的 node:tunnel 组合。
		使用子查询找到每个 node_id+tunnel_id 的最新事件。
	*/
	var events []FailoverEvent
	s.gormDB.Raw(`
		SELECT fe.* FROM failover_events fe
		INNER JOIN (
			SELECT node_id, tunnel_id, MAX(timestamp) as max_ts
			FROM failover_events
			GROUP BY node_id, tunnel_id
		) latest ON fe.node_id = latest.node_id
			AND fe.tunnel_id = latest.tunnel_id
			AND fe.timestamp = latest.max_ts
		WHERE fe.event_type = 'failover'
	`).Scan(&events)

	s.mu.Lock()
	for _, e := range events {
		cacheKey := e.NodeID + ":" + e.TunnelID
		s.activeFailovers[cacheKey] = &FailoverEventReport{
			NodeID:          e.NodeID,
			TunnelID:        e.TunnelID,
			EventType:       e.EventType,
			FromGroupID:     e.FromGroupID,
			ToGroupID:       e.ToGroupID,
			Reason:          e.Reason,
			FailureDuration: e.FailureDuration,
			Timestamp:       e.Timestamp.UnixMilli(),
		}
	}
	s.mu.Unlock()

	if len(events) > 0 {
		s.logger.Info("从数据库恢复活跃容灾状态", zap.Int("count", len(events)))
	}
}

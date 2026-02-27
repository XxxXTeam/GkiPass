package ws

import (
	"encoding/json"
	"fmt"
	"time"

	"gkipass/plane/internal/db"
	"gkipass/plane/internal/db/dao"
	"gkipass/plane/internal/db/models"
	"gkipass/plane/internal/pkg/logger"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// NodeStats 节点统计数据
type NodeStats struct {
	NodeID         string    `json:"node_id"`
	Timestamp      time.Time `json:"timestamp"`
	Load           float64   `json:"load"`            // CPU 负载 (0-100%)
	CPUUsage       float64   `json:"cpu_usage"`       // CPU 使用率 (0-100%)
	MemoryUsage    float64   `json:"memory_usage"`    // 内存使用率 (0-100%)
	MemoryTotal    int64     `json:"memory_total"`    // 总内存 (字节)
	MemoryUsed     int64     `json:"memory_used"`     // 已用内存 (字节)
	Connections    int       `json:"connections"`     // 当前连接数
	Tunnels        int       `json:"tunnels"`         // 当前隧道数
	TrafficIn      int64     `json:"traffic_in"`      // 入站流量 (字节)
	TrafficOut     int64     `json:"traffic_out"`     // 出站流量 (字节)
	BandwidthIn    float64   `json:"bandwidth_in"`    // 入站带宽 (Mbps)
	BandwidthOut   float64   `json:"bandwidth_out"`   // 出站带宽 (Mbps)
	PacketsIn      int64     `json:"packets_in"`      // 入站数据包数
	PacketsOut     int64     `json:"packets_out"`     // 出站数据包数
	ErrorCount     int       `json:"error_count"`     // 错误计数
	ActiveSessions int       `json:"active_sessions"` // 活跃会话数
}

// StatsHandler 统计数据处理器
type StatsHandler struct {
	db  *db.Manager
	dao *dao.DAO
}

// NewStatsHandler 创建统计处理器
func NewStatsHandler(dbManager *db.Manager, d *dao.DAO) *StatsHandler {
	return &StatsHandler{
		db:  dbManager,
		dao: d,
	}
}

// HandleStatsReport 处理统计数据上报
func (h *StatsHandler) HandleStatsReport(nodeConn *NodeConnection, data json.RawMessage) {
	var stats NodeStats
	if err := json.Unmarshal(data, &stats); err != nil {
		logger.Error("解析统计数据失败", zap.Error(err))
		return
	}

	stats.NodeID = nodeConn.NodeID
	stats.Timestamp = time.Now()

	// 存储到 Redis（实时数据，5分钟过期）
	if err := h.storeStatsToRedis(&stats); err != nil {
		logger.Error("存储统计数据到Redis失败", zap.Error(err))
	}

	// 存储到数据库（历史数据）
	if err := h.storeStatsToDB(&stats); err != nil {
		logger.Error("存储统计数据到DB失败", zap.Error(err))
	}

	logger.Debug("收到节点统计数据",
		zap.String("nodeID", stats.NodeID),
		zap.Float64("load", stats.Load),
		zap.Int("connections", stats.Connections),
		zap.Int64("trafficIn", stats.TrafficIn),
		zap.Int64("trafficOut", stats.TrafficOut))
}

// storeStatsToRedis 存储统计数据到 Redis
func (h *StatsHandler) storeStatsToRedis(stats *NodeStats) error {
	if !h.db.HasCache() {
		return fmt.Errorf("Redis 不可用")
	}

	key := fmt.Sprintf("node:stats:%s", stats.NodeID)

	// 序列化数据
	data, err := json.Marshal(stats)
	if err != nil {
		return err
	}

	// 存储到 Redis，5分钟过期
	return h.db.Cache.Redis.Set(key, string(data), 5*time.Minute)
}

/* hasDAO 检查 DAO 是否可用 */
func (h *StatsHandler) hasDAO() bool {
	return h.dao != nil
}

// storeStatsToDB 存储统计数据到数据库（通过 GORM DAO）
func (h *StatsHandler) storeStatsToDB(stats *NodeStats) error {
	if !h.hasDAO() {
		return fmt.Errorf("DAO 不可用，跳过统计存储")
	}

	metric := &models.NodeMetrics{}
	metric.ID = uuid.New().String()
	metric.NodeID = stats.NodeID
	metric.CPUUsage = stats.CPUUsage
	metric.MemoryUsage = stats.MemoryUsage
	metric.NetworkIn = stats.TrafficIn
	metric.NetworkOut = stats.TrafficOut
	metric.Connections = stats.Connections

	return h.dao.CreateNodeMetrics(metric)
}

// GetNodeStats 从 Redis 获取节点统计数据
func (h *StatsHandler) GetNodeStats(nodeID string) (*NodeStats, error) {
	if !h.db.HasCache() {
		return nil, fmt.Errorf("Redis 不可用")
	}

	key := fmt.Sprintf("node:stats:%s", nodeID)

	var data string
	if err := h.db.Cache.Redis.Get(key, &data); err != nil {
		return nil, err
	}

	var stats NodeStats
	if err := json.Unmarshal([]byte(data), &stats); err != nil {
		return nil, err
	}

	return &stats, nil
}

// GetNodeStatsHistory 获取节点历史统计数据
func (h *StatsHandler) GetNodeStatsHistory(nodeID string, from, to time.Time, limit int) ([]NodeStats, error) {
	if !h.hasDAO() {
		return nil, fmt.Errorf("DAO 不可用")
	}

	metrics, err := h.dao.ListNodeMetrics(nodeID, from, to, limit)
	if err != nil {
		return nil, err
	}

	var statsList []NodeStats
	for _, m := range metrics {
		statsList = append(statsList, NodeStats{
			NodeID:      m.NodeID,
			Timestamp:   m.CreatedAt,
			CPUUsage:    m.CPUUsage,
			MemoryUsage: m.MemoryUsage,
			TrafficIn:   m.NetworkIn,
			TrafficOut:  m.NetworkOut,
			Connections: m.Connections,
		})
	}

	return statsList, nil
}

// AggregateStats 聚合统计数据（按小时）
func (h *StatsHandler) AggregateStats(nodeID string, hours int) error {
	if !h.hasDAO() {
		return fmt.Errorf("DAO 不可用")
	}

	now := time.Now()
	cutoff := now.Add(-time.Duration(hours) * time.Hour)

	metrics, err := h.dao.ListNodeMetrics(nodeID, cutoff, now, 0)
	if err != nil {
		return err
	}

	for _, m := range metrics {
		logger.Debug("统计数据聚合",
			zap.String("nodeID", nodeID),
			zap.Time("time", m.CreatedAt),
			zap.Int64("trafficIn", m.NetworkIn),
			zap.Int64("trafficOut", m.NetworkOut))
	}

	return nil
}

// CleanupOldStats 清理旧的统计数据
func (h *StatsHandler) CleanupOldStats(days int) error {
	if !h.hasDAO() {
		return fmt.Errorf("DAO 不可用")
	}
	cutoff := time.Now().AddDate(0, 0, -days)

	affected, err := h.dao.DeleteOldMetrics(cutoff)
	if err != nil {
		return err
	}

	logger.Info("清理旧统计数据完成",
		zap.Int("days", days),
		zap.Int64("deleted", affected))

	return nil
}

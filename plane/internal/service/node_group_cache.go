package service

import (
	"sync"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"gkipass/plane/internal/db/models"
)

/*
NodeGroupCache 节点组缓存服务
功能：为节点组和在线节点数提供带 TTL 的内存缓存，减少高频数据库查询。
适用于隧道创建/调度、容灾检测等需要频繁查询节点组信息的场景。

缓存策略：
  - 节点组列表：TTL 30 秒，首次查询后缓存
  - 组内在线节点数：TTL 10 秒，由心跳或状态变更主动刷新
  - 组详情按 ID 索引：从列表缓存中 O(1) 查找

并发安全：使用 sync.RWMutex 保护读写
*/
type NodeGroupCache struct {
	gormDB *gorm.DB
	logger *zap.Logger

	/* 节点组列表缓存 */
	groups      []models.NodeGroup
	groupIndex  map[string]*models.NodeGroup /* ID → 指针，O(1) 查找 */
	groupsAt    time.Time                    /* 最后刷新时间 */
	groupsTTL   time.Duration                /* 缓存有效期 */
	groupsMu    sync.RWMutex

	/* 组在线节点数缓存 */
	onlineCounts map[string]onlineEntry
	onlineTTL    time.Duration
	onlineMu     sync.RWMutex
}

/* onlineEntry 在线节点数缓存条目 */
type onlineEntry struct {
	Count     int
	UpdatedAt time.Time
}

/*
NewNodeGroupCache 创建节点组缓存
*/
func NewNodeGroupCache(gormDB *gorm.DB) *NodeGroupCache {
	return &NodeGroupCache{
		gormDB:       gormDB,
		logger:       zap.L().Named("node-group-cache"),
		groupIndex:   make(map[string]*models.NodeGroup),
		groupsTTL:    30 * time.Second,
		onlineCounts: make(map[string]onlineEntry),
		onlineTTL:    10 * time.Second,
	}
}

/*
GetGroup 根据 ID 获取节点组（优先缓存）
*/
func (c *NodeGroupCache) GetGroup(id string) (*models.NodeGroup, bool) {
	c.groupsMu.RLock()
	if time.Since(c.groupsAt) < c.groupsTTL {
		g, ok := c.groupIndex[id]
		c.groupsMu.RUnlock()
		return g, ok
	}
	c.groupsMu.RUnlock()

	/* 缓存过期，刷新 */
	c.refreshGroups()

	c.groupsMu.RLock()
	defer c.groupsMu.RUnlock()
	g, ok := c.groupIndex[id]
	return g, ok
}

/*
ListGroups 获取所有节点组（优先缓存）
*/
func (c *NodeGroupCache) ListGroups() []models.NodeGroup {
	c.groupsMu.RLock()
	if time.Since(c.groupsAt) < c.groupsTTL {
		result := make([]models.NodeGroup, len(c.groups))
		copy(result, c.groups)
		c.groupsMu.RUnlock()
		return result
	}
	c.groupsMu.RUnlock()

	c.refreshGroups()

	c.groupsMu.RLock()
	defer c.groupsMu.RUnlock()
	result := make([]models.NodeGroup, len(c.groups))
	copy(result, c.groups)
	return result
}

/*
ListGroupsByRole 按角色筛选节点组
功能：返回匹配指定角色（含 both）的节点组列表
*/
func (c *NodeGroupCache) ListGroupsByRole(role models.NodeRole) []models.NodeGroup {
	all := c.ListGroups()
	var result []models.NodeGroup
	for i := range all {
		if all[i].Role == role || all[i].Role == models.NodeRoleBoth {
			result = append(result, all[i])
		}
	}
	return result
}

/*
GetOnlineCount 获取组内在线节点数（优先缓存）
*/
func (c *NodeGroupCache) GetOnlineCount(groupID string) int {
	c.onlineMu.RLock()
	entry, ok := c.onlineCounts[groupID]
	c.onlineMu.RUnlock()

	if ok && time.Since(entry.UpdatedAt) < c.onlineTTL {
		return entry.Count
	}

	/* 缓存过期或未命中，查库 */
	var count int64
	c.gormDB.Model(&models.Node{}).
		Joins("JOIN node_group_nodes ON node_group_nodes.node_id = nodes.id").
		Where("node_group_nodes.group_id = ? AND nodes.status = ?", groupID, "online").
		Count(&count)

	result := int(count)
	c.SetOnlineCount(groupID, result)
	return result
}

/*
SetOnlineCount 主动设置组在线节点数
功能：由心跳处理或节点状态变更时调用，避免等待 TTL 过期
*/
func (c *NodeGroupCache) SetOnlineCount(groupID string, count int) {
	c.onlineMu.Lock()
	c.onlineCounts[groupID] = onlineEntry{Count: count, UpdatedAt: time.Now()}
	c.onlineMu.Unlock()
}

/*
InvalidateGroup 使指定组的缓存失效
功能：节点组更新/删除时调用，强制下次查询刷新
*/
func (c *NodeGroupCache) InvalidateGroup(groupID string) {
	c.groupsMu.Lock()
	c.groupsAt = time.Time{} /* 置零，强制下次刷新 */
	c.groupsMu.Unlock()

	c.onlineMu.Lock()
	delete(c.onlineCounts, groupID)
	c.onlineMu.Unlock()
}

/*
InvalidateAll 使全部缓存失效
*/
func (c *NodeGroupCache) InvalidateAll() {
	c.groupsMu.Lock()
	c.groupsAt = time.Time{}
	c.groupsMu.Unlock()

	c.onlineMu.Lock()
	c.onlineCounts = make(map[string]onlineEntry)
	c.onlineMu.Unlock()
}

/* refreshGroups 从数据库刷新节点组列表 */
func (c *NodeGroupCache) refreshGroups() {
	var groups []models.NodeGroup
	if err := c.gormDB.Find(&groups).Error; err != nil {
		c.logger.Error("刷新节点组缓存失败", zap.Error(err))
		return
	}

	index := make(map[string]*models.NodeGroup, len(groups))
	for i := range groups {
		index[groups[i].ID] = &groups[i]
	}

	c.groupsMu.Lock()
	c.groups = groups
	c.groupIndex = index
	c.groupsAt = time.Now()
	c.groupsMu.Unlock()
}

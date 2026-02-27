package service

import (
	"testing"
	"time"

	"gkipass/plane/internal/db/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

/*
setupCacheTestDB 创建 NodeGroupCache 测试专用的内存数据库
*/
func setupCacheTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	if err != nil {
		t.Fatalf("创建测试数据库失败: %v", err)
	}

	err = db.AutoMigrate(&models.Node{}, &models.NodeGroup{})
	if err != nil {
		t.Fatalf("迁移表结构失败: %v", err)
	}

	db.Exec(`CREATE TABLE IF NOT EXISTS node_group_nodes (
		group_id VARCHAR(36) NOT NULL,
		node_id VARCHAR(36) NOT NULL,
		added_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (group_id, node_id)
	)`)

	return db
}

/*
TestNodeGroupCache_GetGroup 测试按 ID 获取节点组
*/
func TestNodeGroupCache_GetGroup(t *testing.T) {
	db := setupCacheTestDB(t)

	/* 写入测试数据 */
	group := models.NodeGroup{
		Name: "测试组",
		Role: models.NodeRoleEgress,
	}
	group.ID = "group-001"
	db.Create(&group)

	cache := NewNodeGroupCache(db)

	/* 首次查询应命中数据库 */
	g, ok := cache.GetGroup("group-001")
	if !ok || g == nil {
		t.Fatal("GetGroup 应返回存在的组")
	}
	if g.Name != "测试组" {
		t.Errorf("组名不匹配: 期望 '测试组', 实际 '%s'", g.Name)
	}

	/* 二次查询应命中缓存（不会再查库） */
	g2, ok := cache.GetGroup("group-001")
	if !ok || g2 == nil {
		t.Fatal("二次 GetGroup 应返回缓存的组")
	}

	/* 不存在的组 */
	_, ok = cache.GetGroup("nonexistent")
	if ok {
		t.Error("不存在的组不应返回 ok=true")
	}
}

/*
TestNodeGroupCache_ListGroups 测试列出所有节点组
*/
func TestNodeGroupCache_ListGroups(t *testing.T) {
	db := setupCacheTestDB(t)

	for i := 0; i < 3; i++ {
		g := models.NodeGroup{
			Name: "group-" + string(rune('A'+i)),
			Role: models.NodeRoleBoth,
		}
		g.ID = "gid-" + string(rune('0'+i))
		db.Create(&g)
	}

	cache := NewNodeGroupCache(db)
	groups := cache.ListGroups()
	if len(groups) != 3 {
		t.Errorf("期望 3 个组, 实际 %d", len(groups))
	}
}

/*
TestNodeGroupCache_ListGroupsByRole 测试按角色筛选
*/
func TestNodeGroupCache_ListGroupsByRole(t *testing.T) {
	db := setupCacheTestDB(t)

	roles := []models.NodeRole{models.NodeRoleIngress, models.NodeRoleEgress, models.NodeRoleBoth}
	for i, role := range roles {
		g := models.NodeGroup{
			Name: "group-" + string(rune('A'+i)),
			Role: role,
		}
		g.ID = "gid-" + string(rune('0'+i))
		db.Create(&g)
	}

	cache := NewNodeGroupCache(db)

	/* egress 筛选应返回 egress + both */
	egressGroups := cache.ListGroupsByRole(models.NodeRoleEgress)
	if len(egressGroups) != 2 {
		t.Errorf("egress 筛选期望 2 个组（egress+both），实际 %d", len(egressGroups))
	}

	/* ingress 筛选应返回 ingress + both */
	ingressGroups := cache.ListGroupsByRole(models.NodeRoleIngress)
	if len(ingressGroups) != 2 {
		t.Errorf("ingress 筛选期望 2 个组（ingress+both），实际 %d", len(ingressGroups))
	}
}

/*
TestNodeGroupCache_OnlineCount 测试在线节点数缓存
*/
func TestNodeGroupCache_OnlineCount(t *testing.T) {
	db := setupCacheTestDB(t)
	cache := NewNodeGroupCache(db)

	/* 手动设置在线数 */
	cache.SetOnlineCount("group-a", 5)
	count := cache.GetOnlineCount("group-a")
	if count != 5 {
		t.Errorf("期望在线数 5, 实际 %d", count)
	}

	/* 未设置的组应查库（返回 0） */
	count = cache.GetOnlineCount("group-nonexistent")
	if count != 0 {
		t.Errorf("不存在的组期望在线数 0, 实际 %d", count)
	}
}

/*
TestNodeGroupCache_Invalidate 测试缓存失效
*/
func TestNodeGroupCache_Invalidate(t *testing.T) {
	db := setupCacheTestDB(t)

	g := models.NodeGroup{Name: "测试组", Role: models.NodeRoleBoth}
	g.ID = "group-001"
	db.Create(&g)

	cache := NewNodeGroupCache(db)

	/* 预热缓存 */
	cache.GetGroup("group-001")
	cache.SetOnlineCount("group-001", 3)

	/* 失效单个组 */
	cache.InvalidateGroup("group-001")
	count := cache.GetOnlineCount("group-001")
	/* 失效后应查库，数据库中没有在线节点所以返回 0 */
	if count != 0 {
		t.Errorf("失效后期望在线数 0, 实际 %d", count)
	}
}

/*
TestNodeGroupCache_TTLExpiry 测试 TTL 过期行为
*/
func TestNodeGroupCache_TTLExpiry(t *testing.T) {
	db := setupCacheTestDB(t)

	g := models.NodeGroup{Name: "测试组", Role: models.NodeRoleBoth}
	g.ID = "group-001"
	db.Create(&g)

	cache := NewNodeGroupCache(db)
	/* 设置极短 TTL 以测试过期 */
	cache.groupsTTL = 1 * time.Millisecond
	cache.onlineTTL = 1 * time.Millisecond

	/* 预热 */
	cache.GetGroup("group-001")
	cache.SetOnlineCount("group-001", 10)

	/* 等待 TTL 过期 */
	time.Sleep(5 * time.Millisecond)

	/* 过期后应重新查库 */
	result, ok := cache.GetGroup("group-001")
	if !ok || result == nil {
		t.Fatal("TTL 过期后应能重新从库中获取")
	}

	/* 在线数应重新查库（库中为 0） */
	count := cache.GetOnlineCount("group-001")
	if count != 0 {
		t.Errorf("TTL 过期后期望查库返回 0, 实际 %d", count)
	}
}

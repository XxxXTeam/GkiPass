package service

import (
	"testing"
	"time"

	"gkipass/plane/internal/db/models"

	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

/*
setupTestDB 创建内存 SQLite 测试数据库
功能：每个测试用例独立的内存数据库，自动迁移表结构
*/
func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	if err != nil {
		t.Fatalf("创建测试数据库失败: %v", err)
	}

	/* 迁移所需表 */
	err = db.AutoMigrate(
		&models.Node{},
		&models.NodeGroup{},
		&FailoverEvent{},
	)
	if err != nil {
		t.Fatalf("迁移表结构失败: %v", err)
	}

	/* 创建多对多关联表 */
	db.Exec(`CREATE TABLE IF NOT EXISTS node_group_nodes (
		group_id VARCHAR(36) NOT NULL,
		node_id VARCHAR(36) NOT NULL,
		added_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (group_id, node_id)
	)`)

	return db
}

/*
TestHandleEvent_Failover 测试容灾事件处理 — 切换
*/
func TestHandleEvent_Failover(t *testing.T) {
	db := setupTestDB(t)
	svc := &FailoverService{
		gormDB:          db,
		activeFailovers: make(map[string]*FailoverEventReport),
	}
	/* 手动初始化 logger 避免空指针 */
	svc.logger = initTestLogger()

	report := &FailoverEventReport{
		NodeID:          "node-001",
		TunnelID:        "tunnel-001",
		EventType:       "failover",
		FromGroupID:     "group-a",
		ToGroupID:       "group-b",
		Reason:          "timeout",
		FailureDuration: 65,
		Timestamp:       time.Now().UnixMilli(),
	}

	err := svc.HandleEvent(report)
	if err != nil {
		t.Fatalf("HandleEvent 失败: %v", err)
	}

	/* 验证数据库写入 */
	var events []FailoverEvent
	db.Find(&events)
	if len(events) != 1 {
		t.Fatalf("期望 1 条事件，实际 %d 条", len(events))
	}
	if events[0].EventType != "failover" {
		t.Errorf("事件类型不匹配: 期望 failover, 实际 %s", events[0].EventType)
	}
	if events[0].FromGroupID != "group-a" {
		t.Errorf("FromGroupID 不匹配: 期望 group-a, 实际 %s", events[0].FromGroupID)
	}

	/* 验证内存缓存更新 */
	actives := svc.GetActiveFailovers()
	if len(actives) != 1 {
		t.Fatalf("期望 1 个活跃容灾, 实际 %d", len(actives))
	}
	if actives[0].TunnelID != "tunnel-001" {
		t.Errorf("活跃容灾 TunnelID 不匹配: %s", actives[0].TunnelID)
	}
}

/*
TestHandleEvent_Recovery 测试容灾事件处理 — 回切
*/
func TestHandleEvent_Recovery(t *testing.T) {
	db := setupTestDB(t)
	svc := &FailoverService{
		gormDB:          db,
		activeFailovers: make(map[string]*FailoverEventReport),
		logger:          initTestLogger(),
	}

	/* 先触发容灾 */
	_ = svc.HandleEvent(&FailoverEventReport{
		NodeID:      "node-001",
		TunnelID:    "tunnel-001",
		EventType:   "failover",
		FromGroupID: "group-a",
		ToGroupID:   "group-b",
		Reason:      "timeout",
		Timestamp:   time.Now().UnixMilli(),
	})

	/* 再触发回切 */
	err := svc.HandleEvent(&FailoverEventReport{
		NodeID:      "node-001",
		TunnelID:    "tunnel-001",
		EventType:   "recovery",
		FromGroupID: "group-b",
		ToGroupID:   "group-a",
		Reason:      "original_recovered",
		Timestamp:   time.Now().UnixMilli(),
	})
	if err != nil {
		t.Fatalf("Recovery HandleEvent 失败: %v", err)
	}

	/* 活跃容灾应清空 */
	actives := svc.GetActiveFailovers()
	if len(actives) != 0 {
		t.Errorf("回切后仍有 %d 个活跃容灾", len(actives))
	}

	/* 数据库应有 2 条事件 */
	var count int64
	db.Model(&FailoverEvent{}).Count(&count)
	if count != 2 {
		t.Errorf("期望 2 条事件, 实际 %d", count)
	}
}

/*
TestGetGroupFailoverSummary 测试出口组容灾摘要
*/
func TestGetGroupFailoverSummary(t *testing.T) {
	db := setupTestDB(t)
	svc := &FailoverService{
		gormDB:          db,
		activeFailovers: make(map[string]*FailoverEventReport),
		logger:          initTestLogger(),
	}

	/* 写入 2 个不同隧道的容灾事件 */
	_ = svc.HandleEvent(&FailoverEventReport{
		NodeID: "node-001", TunnelID: "tunnel-001",
		EventType: "failover", FromGroupID: "group-a", ToGroupID: "group-b",
		Timestamp: time.Now().UnixMilli(),
	})
	_ = svc.HandleEvent(&FailoverEventReport{
		NodeID: "node-002", TunnelID: "tunnel-002",
		EventType: "failover", FromGroupID: "group-a", ToGroupID: "group-b",
		Timestamp: time.Now().UnixMilli(),
	})

	summary := svc.GetGroupFailoverSummary("group-a")
	activeCount, ok := summary["active_failover_tunnels"].(int)
	if !ok || activeCount != 2 {
		t.Errorf("期望 2 个活跃容灾, 实际 %v", summary["active_failover_tunnels"])
	}
}

/*
TestGetTunnelFailoverHistory 测试隧道容灾历史查询
*/
func TestGetTunnelFailoverHistory(t *testing.T) {
	db := setupTestDB(t)
	svc := &FailoverService{
		gormDB:          db,
		activeFailovers: make(map[string]*FailoverEventReport),
		logger:          initTestLogger(),
	}

	/* 为同一隧道写入 3 条事件 */
	for i := 0; i < 3; i++ {
		eventType := "failover"
		if i%2 == 1 {
			eventType = "recovery"
		}
		_ = svc.HandleEvent(&FailoverEventReport{
			NodeID: "node-001", TunnelID: "tunnel-001",
			EventType: eventType, FromGroupID: "group-a", ToGroupID: "group-b",
			Timestamp: time.Now().Add(time.Duration(i) * time.Second).UnixMilli(),
		})
	}

	events, err := svc.GetTunnelFailoverHistory("tunnel-001", 10)
	if err != nil {
		t.Fatalf("查询历史失败: %v", err)
	}
	if len(events) != 3 {
		t.Errorf("期望 3 条历史, 实际 %d", len(events))
	}

	/* 不存在的隧道应返回空 */
	events, err = svc.GetTunnelFailoverHistory("nonexistent", 10)
	if err != nil {
		t.Fatalf("查询不存在隧道失败: %v", err)
	}
	if len(events) != 0 {
		t.Errorf("期望 0 条历史, 实际 %d", len(events))
	}
}

/*
initTestLogger 创建测试用的静默 logger
*/
func initTestLogger() *zap.Logger {
	cfg := zap.NewDevelopmentConfig()
	cfg.Level = zap.NewAtomicLevelAt(zap.WarnLevel)
	l, _ := cfg.Build()
	return l.Named("failover-test")
}

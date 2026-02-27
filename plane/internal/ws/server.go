package ws

import (
	"net/http"

	"gkipass/plane/internal/db/dao"
	"gkipass/plane/internal/service"
	"gkipass/plane/internal/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin: func(r *http.Request) bool {
		/*
			节点 WebSocket 连接不携带 Origin 头，直接放行；
			浏览器连接已由 CORS 中间件验证 Origin，此处统一放行。
			如需收紧，可从配置文件读取允许的 Origin 列表。
		*/
		return true
	},
}

// Server WebSocket 服务器
type Server struct {
	manager *Manager
	handler *Handler
}

/*
NewServer 创建 WebSocket 服务器
功能：初始化连接管理器（含最大连接数限制）和消息处理器。
maxConnections 为可选参数，0 或不传表示不限制。
*/
func NewServer(d *dao.DAO, maxConnections int, failoverSvc ...*service.FailoverService) *Server {
	manager := NewManager(maxConnections)

	var fSvc *service.FailoverService
	if len(failoverSvc) > 0 {
		fSvc = failoverSvc[0]
	}
	handler := NewHandler(manager, d, fSvc)

	return &Server{
		manager: manager,
		handler: handler,
	}
}

// Start 启动服务器
func (s *Server) Start() {
	go s.manager.Run()
	logger.Info("✓ WebSocket 服务器已启动")
}

// HandleWebSocket WebSocket 处理函数
func (s *Server) HandleWebSocket(c *gin.Context) {
	/* 检查连接数限制，防止资源耗尽 */
	if s.manager.IsAtCapacity() {
		logger.Warn("WebSocket 连接数已达上限，拒绝新连接",
			zap.Int("current", s.manager.GetNodeCount()),
			zap.Int("max", s.manager.maxConnections))
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "服务器连接数已满"})
		return
	}

	// 升级为 WebSocket 连接
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		logger.Error("WebSocket 升级失败", zap.Error(err))
		return
	}

	// 处理连接
	s.handler.HandleConnection(conn)
}

// GetManager 获取管理器（用于外部调用）
func (s *Server) GetManager() *Manager {
	return s.manager
}

// GetHandler 获取处理器（用于外部调用）
func (s *Server) GetHandler() *Handler {
	return s.handler
}

// GetStats 获取统计信息
func (s *Server) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"online_nodes": s.manager.GetNodeCount(),
		"node_ids":     s.manager.GetAllNodeIDs(),
	}
}

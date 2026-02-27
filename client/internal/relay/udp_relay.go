package relay

import (
	"context"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

/*
UDPRelay UDP 流量转发器
功能：在入口端口监听 UDP 数据包，将流量转发到目标地址，
维护客户端会话映射实现双向通信，支持会话超时清理
*/
type UDPRelay struct {
	config *TCPRelayConfig
	conn   *net.UDPConn
	ctx    context.Context
	cancel context.CancelFunc
	logger *zap.Logger

	/* UDP 会话管理 */
	sessions sync.Map
	running  atomic.Bool

	/* 流量统计 */
	stats *RelayStats
}

/*
udpSession UDP 会话
功能：维护单个客户端的 UDP 会话状态，包括远端连接和最后活跃时间
*/
type udpSession struct {
	clientAddr *net.UDPAddr
	targetConn *net.UDPConn
	lastActive time.Time
	mu         sync.Mutex
}

/*
NewUDPRelay 创建 UDP 转发器
*/
func NewUDPRelay(config *TCPRelayConfig) *UDPRelay {
	ctx, cancel := context.WithCancel(context.Background())
	return &UDPRelay{
		config: config,
		ctx:    ctx,
		cancel: cancel,
		logger: zap.L().Named("udp-relay"),
		stats:  &RelayStats{StartTime: time.Now()},
	}
}

/*
Start 启动 UDP 转发器
功能：在指定端口监听 UDP 数据包，启动读取和会话清理协程
*/
func (r *UDPRelay) Start() error {
	listenAddr := fmt.Sprintf("%s:%d", r.config.ListenAddr, r.config.ListenPort)

	addr, err := net.ResolveUDPAddr("udp", listenAddr)
	if err != nil {
		return fmt.Errorf("解析 UDP 地址失败 [%s]: %w", listenAddr, err)
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return fmt.Errorf("UDP 监听失败 [%s]: %w", listenAddr, err)
	}

	r.conn = conn
	r.running.Store(true)

	r.logger.Info("UDP 转发器已启动",
		zap.String("listen", listenAddr),
		zap.String("target", fmt.Sprintf("%s:%d", r.config.TargetAddr, r.config.TargetPort)))

	go r.readLoop()
	go r.cleanupSessions()

	return nil
}

/*
readLoop 数据包读取循环
功能：持续读取入站 UDP 数据包，查找或创建对应的客户端会话，
将数据包转发到目标地址
*/
func (r *UDPRelay) readLoop() {
	buf := make([]byte, r.config.BufferSize)
	if r.config.BufferSize == 0 {
		buf = make([]byte, 64*1024)
	}

	for {
		select {
		case <-r.ctx.Done():
			return
		default:
		}

		r.conn.SetReadDeadline(time.Now().Add(1 * time.Second))

		n, clientAddr, err := r.conn.ReadFromUDP(buf)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			if r.running.Load() {
				r.logger.Error("读取 UDP 数据失败", zap.Error(err))
			}
			continue
		}

		if n == 0 {
			continue
		}

		r.stats.BytesIn.Add(int64(n))

		/* 查找或创建会话 */
		session, err := r.getOrCreateSession(clientAddr)
		if err != nil {
			r.logger.Error("创建 UDP 会话失败",
				zap.String("client", clientAddr.String()),
				zap.Error(err))
			r.stats.FailedConns.Add(1)
			continue
		}

		/* 更新会话活跃时间 */
		session.mu.Lock()
		session.lastActive = time.Now()
		session.mu.Unlock()

		/* 转发数据到目标 */
		_, err = session.targetConn.Write(buf[:n])
		if err != nil {
			r.logger.Error("转发 UDP 数据失败",
				zap.String("client", clientAddr.String()),
				zap.Error(err))
			r.stats.FailedConns.Add(1)
		}
	}
}

/*
getOrCreateSession 获取或创建 UDP 会话
功能：根据客户端地址查找已有会话，不存在则创建新的会话连接到目标
*/
func (r *UDPRelay) getOrCreateSession(clientAddr *net.UDPAddr) (*udpSession, error) {
	key := clientAddr.String()

	/* 查找已有会话 */
	if val, ok := r.sessions.Load(key); ok {
		return val.(*udpSession), nil
	}

	/* 检查连接数限制 */
	if r.config.MaxConnections > 0 && r.stats.ActiveConns.Load() >= int64(r.config.MaxConnections) {
		return nil, fmt.Errorf("UDP 会话数已达上限: %d", r.config.MaxConnections)
	}

	/* 创建到目标的 UDP 连接 */
	targetAddr := fmt.Sprintf("%s:%d", r.config.TargetAddr, r.config.TargetPort)
	raddr, err := net.ResolveUDPAddr("udp", targetAddr)
	if err != nil {
		return nil, fmt.Errorf("解析目标地址失败: %w", err)
	}

	targetConn, err := net.DialUDP("udp", nil, raddr)
	if err != nil {
		return nil, fmt.Errorf("连接目标失败: %w", err)
	}

	session := &udpSession{
		clientAddr: clientAddr,
		targetConn: targetConn,
		lastActive: time.Now(),
	}

	r.sessions.Store(key, session)
	r.stats.TotalConns.Add(1)
	r.stats.ActiveConns.Add(1)

	r.logger.Debug("创建 UDP 会话",
		zap.String("client", clientAddr.String()),
		zap.String("target", targetAddr))

	/* 启动反向读取协程（目标 -> 客户端） */
	go r.reverseRead(session)

	return session, nil
}

/*
reverseRead 反向数据读取
功能：从目标端读取响应数据，通过主 UDP 连接回传给客户端
*/
func (r *UDPRelay) reverseRead(session *udpSession) {
	buf := make([]byte, r.config.BufferSize)
	if r.config.BufferSize == 0 {
		buf = make([]byte, 64*1024)
	}

	for {
		select {
		case <-r.ctx.Done():
			return
		default:
		}

		/* 设置读取超时 */
		idleTimeout := r.config.IdleTimeout
		if idleTimeout == 0 {
			idleTimeout = 2 * time.Minute
		}
		session.targetConn.SetReadDeadline(time.Now().Add(idleTimeout))

		n, err := session.targetConn.Read(buf)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				/* 检查会话是否过期 */
				session.mu.Lock()
				elapsed := time.Since(session.lastActive)
				session.mu.Unlock()

				if elapsed > idleTimeout {
					r.removeSession(session)
					return
				}
				continue
			}
			if r.running.Load() {
				r.logger.Debug("目标端读取结束", zap.Error(err))
			}
			r.removeSession(session)
			return
		}

		if n > 0 {
			r.stats.BytesOut.Add(int64(n))

			/* 更新活跃时间 */
			session.mu.Lock()
			session.lastActive = time.Now()
			session.mu.Unlock()

			/* 回传给客户端 */
			_, err = r.conn.WriteToUDP(buf[:n], session.clientAddr)
			if err != nil {
				r.logger.Error("回传 UDP 数据失败",
					zap.String("client", session.clientAddr.String()),
					zap.Error(err))
			}
		}
	}
}

/*
removeSession 移除 UDP 会话
功能：关闭会话的目标连接并从会话映射中删除
*/
func (r *UDPRelay) removeSession(session *udpSession) {
	key := session.clientAddr.String()
	if _, loaded := r.sessions.LoadAndDelete(key); loaded {
		session.targetConn.Close()
		r.stats.ActiveConns.Add(-1)
		r.logger.Debug("移除 UDP 会话", zap.String("client", key))
	}
}

/*
cleanupSessions 清理过期会话
功能：定期扫描所有 UDP 会话，移除超过空闲超时的会话
*/
func (r *UDPRelay) cleanupSessions() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.ctx.Done():
			return
		case <-ticker.C:
			idleTimeout := r.config.IdleTimeout
			if idleTimeout == 0 {
				idleTimeout = 2 * time.Minute
			}

			now := time.Now()
			r.sessions.Range(func(key, value interface{}) bool {
				session := value.(*udpSession)
				session.mu.Lock()
				elapsed := now.Sub(session.lastActive)
				session.mu.Unlock()

				if elapsed > idleTimeout {
					r.removeSession(session)
				}
				return true
			})
		}
	}
}

/*
Stop 停止 UDP 转发器
功能：关闭监听端口并清理所有活跃会话
*/
func (r *UDPRelay) Stop() error {
	r.running.Store(false)
	r.cancel()

	/* 关闭所有会话 */
	r.sessions.Range(func(key, value interface{}) bool {
		session := value.(*udpSession)
		session.targetConn.Close()
		r.sessions.Delete(key)
		return true
	})

	if r.conn != nil {
		r.conn.Close()
	}

	r.logger.Info("UDP 转发器已停止")
	return nil
}

/*
GetStats 获取转发器统计
*/
func (r *UDPRelay) GetStats() map[string]interface{} {
	return r.stats.GetSnapshot()
}

/*
IsRunning 检查转发器是否运行中
*/
func (r *UDPRelay) IsRunning() bool {
	return r.running.Load()
}

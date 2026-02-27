package relay

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

/*
TCPRelay TCP 流量转发器
功能：在入口端口监听 TCP 连接，将流量双向转发到目标地址，
支持速率限制、连接数限制、空闲超时、流量统计等特性
*/
type TCPRelay struct {
	config   *TCPRelayConfig
	listener net.Listener
	ctx      context.Context
	cancel   context.CancelFunc
	logger   *zap.Logger

	/* 运行时状态 */
	running    atomic.Bool
	activeConn sync.WaitGroup
	connCount  atomic.Int64

	/* 流量统计 */
	stats *RelayStats
}

/*
TCPRelayConfig TCP 转发器配置
功能：在通用 RelayConfig 基础上扩展 TCP 转发专用参数
*/
type TCPRelayConfig struct {
	Name           string        `json:"name"`
	ListenAddr     string        `json:"listen_addr"`
	ListenPort     int           `json:"listen_port"`
	TargetAddr     string        `json:"target_addr"`
	TargetPort     int           `json:"target_port"`
	Protocol       string        `json:"protocol"`
	BufferSize     int           `json:"buffer_size"`
	MaxConnections int           `json:"max_connections"`
	IdleTimeout    time.Duration `json:"idle_timeout"`
	ConnTimeout    time.Duration `json:"conn_timeout"`
	RateLimitBPS   int64         `json:"rate_limit_bps"`
	EnableEncrypt  bool          `json:"enable_encrypt"`
	EncryptMethod  string        `json:"encrypt_method"`
}

/*
DefaultTCPRelayConfig 返回默认 TCP 转发器配置
*/
func DefaultTCPRelayConfig() *TCPRelayConfig {
	return &TCPRelayConfig{
		BufferSize:     32 * 1024,
		MaxConnections: 1000,
		IdleTimeout:    5 * time.Minute,
		ConnTimeout:    10 * time.Second,
		RateLimitBPS:   0,
	}
}

/*
RelayStats 转发器统计
功能：记录转发器的实时流量和连接统计数据
*/
type RelayStats struct {
	BytesIn     atomic.Int64 `json:"bytes_in"`
	BytesOut    atomic.Int64 `json:"bytes_out"`
	TotalConns  atomic.Int64 `json:"total_conns"`
	ActiveConns atomic.Int64 `json:"active_conns"`
	FailedConns atomic.Int64 `json:"failed_conns"`
	StartTime   time.Time    `json:"start_time"`
}

/*
GetSnapshot 获取统计快照
功能：返回当前统计数据的只读快照
*/
func (s *RelayStats) GetSnapshot() map[string]interface{} {
	return map[string]interface{}{
		"bytes_in":     s.BytesIn.Load(),
		"bytes_out":    s.BytesOut.Load(),
		"total_conns":  s.TotalConns.Load(),
		"active_conns": s.ActiveConns.Load(),
		"failed_conns": s.FailedConns.Load(),
		"uptime_secs":  int64(time.Since(s.StartTime).Seconds()),
	}
}

/*
NewTCPRelay 创建 TCP 转发器
功能：初始化 TCP 转发器实例
*/
func NewTCPRelay(config *TCPRelayConfig) *TCPRelay {
	ctx, cancel := context.WithCancel(context.Background())
	return &TCPRelay{
		config: config,
		ctx:    ctx,
		cancel: cancel,
		logger: zap.L().Named("tcp-relay"),
		stats:  &RelayStats{StartTime: time.Now()},
	}
}

/*
Start 启动 TCP 转发器
功能：在指定端口开始监听 TCP 连接，每个新连接启动独立的转发协程
*/
func (r *TCPRelay) Start() error {
	listenAddr := fmt.Sprintf("%s:%d", r.config.ListenAddr, r.config.ListenPort)

	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return fmt.Errorf("TCP 监听失败 [%s]: %w", listenAddr, err)
	}

	r.listener = listener
	r.running.Store(true)

	r.logger.Info("TCP 转发器已启动",
		zap.String("listen", listenAddr),
		zap.String("target", fmt.Sprintf("%s:%d", r.config.TargetAddr, r.config.TargetPort)),
		zap.String("name", r.config.Name))

	go r.acceptLoop()
	return nil
}

/*
acceptLoop 接受连接循环
功能：持续接受新的 TCP 连接并分发到处理协程
*/
func (r *TCPRelay) acceptLoop() {
	for {
		select {
		case <-r.ctx.Done():
			return
		default:
		}

		conn, err := r.listener.Accept()
		if err != nil {
			if r.running.Load() {
				r.logger.Error("接受连接失败", zap.Error(err))
			}
			continue
		}

		/* 检查连接数限制 */
		if r.config.MaxConnections > 0 && r.connCount.Load() >= int64(r.config.MaxConnections) {
			r.logger.Warn("连接数已达上限，拒绝新连接",
				zap.Int64("current", r.connCount.Load()),
				zap.Int("max", r.config.MaxConnections))
			conn.Close()
			r.stats.FailedConns.Add(1)
			continue
		}

		r.activeConn.Add(1)
		r.connCount.Add(1)
		r.stats.TotalConns.Add(1)
		r.stats.ActiveConns.Add(1)

		go r.handleConnection(conn)
	}
}

/*
handleConnection 处理单个 TCP 连接
功能：建立到目标地址的连接，启动双向数据流转发，
支持空闲超时检测和流量统计
*/
func (r *TCPRelay) handleConnection(clientConn net.Conn) {
	defer func() {
		clientConn.Close()
		r.activeConn.Done()
		r.connCount.Add(-1)
		r.stats.ActiveConns.Add(-1)
	}()

	clientAddr := clientConn.RemoteAddr().String()
	targetAddr := fmt.Sprintf("%s:%d", r.config.TargetAddr, r.config.TargetPort)

	/* 连接到目标地址 */
	dialer := net.Dialer{Timeout: r.config.ConnTimeout}
	targetConn, err := dialer.DialContext(r.ctx, "tcp", targetAddr)
	if err != nil {
		r.logger.Error("连接目标失败",
			zap.String("target", targetAddr),
			zap.String("client", clientAddr),
			zap.Error(err))
		r.stats.FailedConns.Add(1)
		return
	}
	defer targetConn.Close()

	r.logger.Debug("TCP 连接建立",
		zap.String("client", clientAddr),
		zap.String("target", targetAddr))

	/* 启动双向数据转发 */
	done := make(chan struct{}, 2)

	/* 客户端 -> 目标 */
	go func() {
		defer func() { done <- struct{}{} }()
		n := r.copyWithStats(targetConn, clientConn, &r.stats.BytesIn)
		r.logger.Debug("客户端->目标 流量传输完成",
			zap.String("client", clientAddr),
			zap.Int64("bytes", n))
	}()

	/* 目标 -> 客户端 */
	go func() {
		defer func() { done <- struct{}{} }()
		n := r.copyWithStats(clientConn, targetConn, &r.stats.BytesOut)
		r.logger.Debug("目标->客户端 流量传输完成",
			zap.String("client", clientAddr),
			zap.Int64("bytes", n))
	}()

	/* 等待任一方向结束（半关闭） */
	select {
	case <-done:
	case <-r.ctx.Done():
	}
}

/*
copyWithStats 带统计的数据拷贝
功能：从 src 读取数据写入 dst，同时更新流量统计计数器，
支持速率限制和空闲超时检测
*/
func (r *TCPRelay) copyWithStats(dst net.Conn, src net.Conn, counter *atomic.Int64) int64 {
	buf := make([]byte, r.config.BufferSize)
	var totalBytes int64

	for {
		/* 设置读取超时（空闲检测） */
		if r.config.IdleTimeout > 0 {
			src.SetReadDeadline(time.Now().Add(r.config.IdleTimeout))
		}

		n, readErr := src.Read(buf)
		if n > 0 {
			/* 速率限制 */
			if r.config.RateLimitBPS > 0 {
				r.applyRateLimit(int64(n))
			}

			/* 设置写入超时 */
			dst.SetWriteDeadline(time.Now().Add(30 * time.Second))

			written, writeErr := dst.Write(buf[:n])
			if written > 0 {
				totalBytes += int64(written)
				counter.Add(int64(written))
			}
			if writeErr != nil {
				break
			}
		}
		if readErr != nil {
			if readErr != io.EOF {
				r.logger.Debug("读取数据结束", zap.Error(readErr))
			}
			break
		}
	}

	return totalBytes
}

/*
applyRateLimit 应用速率限制
功能：通过简单的令牌桶算法限制数据传输速率
*/
func (r *TCPRelay) applyRateLimit(bytes int64) {
	if r.config.RateLimitBPS <= 0 {
		return
	}

	/* 计算需要等待的时间（简单令牌桶） */
	waitTime := time.Duration(float64(bytes) / float64(r.config.RateLimitBPS) * float64(time.Second))
	if waitTime > 0 {
		time.Sleep(waitTime)
	}
}

/*
Stop 停止 TCP 转发器
功能：优雅地关闭监听端口并等待所有活跃连接完成
*/
func (r *TCPRelay) Stop() error {
	r.running.Store(false)
	r.cancel()

	if r.listener != nil {
		r.listener.Close()
	}

	/* 等待所有活跃连接关闭（最多等待 10 秒） */
	done := make(chan struct{})
	go func() {
		r.activeConn.Wait()
		close(done)
	}()

	select {
	case <-done:
		r.logger.Info("TCP 转发器已停止，所有连接已关闭")
	case <-time.After(10 * time.Second):
		r.logger.Warn("TCP 转发器停止超时，强制关闭")
	}

	return nil
}

/*
GetStats 获取转发器统计
功能：返回当前转发器的运行时统计数据
*/
func (r *TCPRelay) GetStats() map[string]interface{} {
	return r.stats.GetSnapshot()
}

/*
IsRunning 检查转发器是否运行中
*/
func (r *TCPRelay) IsRunning() bool {
	return r.running.Load()
}

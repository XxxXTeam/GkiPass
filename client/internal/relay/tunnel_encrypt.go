package relay

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

/*
TunnelType 隧道加密类型枚举
功能：定义支持的隧道加密传输协议
*/
type TunnelType string

const (
	TunnelTypeTCP TunnelType = "tcp"
	TunnelTypeTLS TunnelType = "tls"
	TunnelTypeWS  TunnelType = "ws"
	TunnelTypeWSS TunnelType = "wss"
)

/*
TunnelConfig 隧道加密配置
功能：定义隧道加密传输的连接参数
*/
type TunnelConfig struct {
	Type       TunnelType `json:"type"`
	RemoteAddr string     `json:"remote_addr"`
	RemotePort int        `json:"remote_port"`
	Path       string     `json:"path"`

	/* TLS 配置 */
	TLSCertFile           string `json:"tls_cert_file"`
	TLSKeyFile            string `json:"tls_key_file"`
	TLSCAFile             string `json:"tls_ca_file"`
	TLSServerName         string `json:"tls_server_name"`
	TLSInsecureSkipVerify bool   `json:"tls_insecure_skip_verify"`

	/* 连接参数 */
	ConnTimeout    time.Duration `json:"conn_timeout"`
	WriteTimeout   time.Duration `json:"write_timeout"`
	ReadTimeout    time.Duration `json:"read_timeout"`
	PingInterval   time.Duration `json:"ping_interval"`
	MaxMessageSize int64         `json:"max_message_size"`

	/* 重连参数 */
	ReconnectInterval time.Duration `json:"reconnect_interval"`
	MaxReconnects     int           `json:"max_reconnects"`
}

/*
DefaultTunnelConfig 返回默认隧道配置
*/
func DefaultTunnelConfig() *TunnelConfig {
	return &TunnelConfig{
		Type:              TunnelTypeTCP,
		Path:              "/tunnel",
		ConnTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		ReadTimeout:       30 * time.Second,
		PingInterval:      30 * time.Second,
		MaxMessageSize:    64 * 1024,
		ReconnectInterval: 5 * time.Second,
		MaxReconnects:     -1,
	}
}

/*
EncryptedTunnel 加密隧道
功能：通过 TLS/WS/WSS 协议建立加密的数据传输通道，
封装底层网络连接并提供统一的读写接口
*/
type EncryptedTunnel struct {
	config    *TunnelConfig
	conn      net.Conn
	wsConn    *websocket.Conn
	ctx       context.Context
	cancel    context.CancelFunc
	logger    *zap.Logger
	mu        sync.Mutex
	connected atomic.Bool

	/* 统计 */
	bytesIn  atomic.Int64
	bytesOut atomic.Int64
}

/*
NewEncryptedTunnel 创建加密隧道
*/
func NewEncryptedTunnel(config *TunnelConfig) *EncryptedTunnel {
	ctx, cancel := context.WithCancel(context.Background())
	return &EncryptedTunnel{
		config: config,
		ctx:    ctx,
		cancel: cancel,
		logger: zap.L().Named("encrypted-tunnel"),
	}
}

/*
Connect 建立隧道连接
功能：根据配置的隧道类型建立对应的加密连接
*/
func (t *EncryptedTunnel) Connect() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	var err error
	switch t.config.Type {
	case TunnelTypeTCP:
		err = t.connectTCP()
	case TunnelTypeTLS:
		err = t.connectTLS()
	case TunnelTypeWS:
		err = t.connectWS()
	case TunnelTypeWSS:
		err = t.connectWSS()
	default:
		return fmt.Errorf("不支持的隧道类型: %s", t.config.Type)
	}

	if err != nil {
		return err
	}

	t.connected.Store(true)
	t.logger.Info("隧道连接已建立",
		zap.String("type", string(t.config.Type)),
		zap.String("remote", fmt.Sprintf("%s:%d", t.config.RemoteAddr, t.config.RemotePort)))

	return nil
}

/*
connectTCP 建立 TCP 明文连接
*/
func (t *EncryptedTunnel) connectTCP() error {
	addr := net.JoinHostPort(t.config.RemoteAddr, fmt.Sprintf("%d", t.config.RemotePort))
	conn, err := net.DialTimeout("tcp", addr, t.config.ConnTimeout)
	if err != nil {
		return fmt.Errorf("TCP 连接失败 [%s]: %w", addr, err)
	}
	t.conn = conn
	return nil
}

/*
connectTLS 建立 TLS 加密连接
功能：使用 TLS 协议加密 TCP 连接，支持证书验证和服务端名称校验
*/
func (t *EncryptedTunnel) connectTLS() error {
	addr := fmt.Sprintf("%s:%d", t.config.RemoteAddr, t.config.RemotePort)

	tlsConfig := &tls.Config{
		InsecureSkipVerify: t.config.TLSInsecureSkipVerify,
		ServerName:         t.config.TLSServerName,
		MinVersion:         tls.VersionTLS12,
	}

	/* 加载客户端证书（如果配置了） */
	if t.config.TLSCertFile != "" && t.config.TLSKeyFile != "" {
		cert, err := tls.LoadX509KeyPair(t.config.TLSCertFile, t.config.TLSKeyFile)
		if err != nil {
			return fmt.Errorf("加载 TLS 证书失败: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	dialer := &net.Dialer{Timeout: t.config.ConnTimeout}
	conn, err := tls.DialWithDialer(dialer, "tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("TLS 连接失败 [%s]: %w", addr, err)
	}

	t.conn = conn
	return nil
}

/*
connectWS 建立 WebSocket 连接
功能：通过 WebSocket 协议建立隧道连接，适用于需要穿透 HTTP 代理的场景
*/
func (t *EncryptedTunnel) connectWS() error {
	wsURL := url.URL{
		Scheme: "ws",
		Host:   fmt.Sprintf("%s:%d", t.config.RemoteAddr, t.config.RemotePort),
		Path:   t.config.Path,
	}

	dialer := websocket.Dialer{
		HandshakeTimeout: t.config.ConnTimeout,
	}

	conn, _, err := dialer.DialContext(t.ctx, wsURL.String(), http.Header{})
	if err != nil {
		return fmt.Errorf("WebSocket 连接失败 [%s]: %w", wsURL.String(), err)
	}

	conn.SetReadLimit(t.config.MaxMessageSize)
	t.wsConn = conn
	return nil
}

/*
connectWSS 建立 WebSocket Secure 连接
功能：通过 WSS 协议建立加密的 WebSocket 隧道连接
*/
func (t *EncryptedTunnel) connectWSS() error {
	wsURL := url.URL{
		Scheme: "wss",
		Host:   fmt.Sprintf("%s:%d", t.config.RemoteAddr, t.config.RemotePort),
		Path:   t.config.Path,
	}

	tlsConfig := &tls.Config{
		InsecureSkipVerify: t.config.TLSInsecureSkipVerify,
		ServerName:         t.config.TLSServerName,
		MinVersion:         tls.VersionTLS12,
	}

	dialer := websocket.Dialer{
		HandshakeTimeout: t.config.ConnTimeout,
		TLSClientConfig:  tlsConfig,
	}

	conn, _, err := dialer.DialContext(t.ctx, wsURL.String(), http.Header{})
	if err != nil {
		return fmt.Errorf("WSS 连接失败 [%s]: %w", wsURL.String(), err)
	}

	conn.SetReadLimit(t.config.MaxMessageSize)
	t.wsConn = conn
	return nil
}

/*
Read 从隧道读取数据
功能：统一的隧道读取接口，根据底层连接类型选择对应的读取方式
*/
func (t *EncryptedTunnel) Read(p []byte) (int, error) {
	if !t.connected.Load() {
		return 0, fmt.Errorf("隧道未连接")
	}

	var n int
	var err error

	if t.wsConn != nil {
		/* WebSocket 读取 */
		_, message, readErr := t.wsConn.ReadMessage()
		if readErr != nil {
			return 0, readErr
		}
		n = copy(p, message)
	} else if t.conn != nil {
		/* TCP/TLS 读取 */
		n, err = t.conn.Read(p)
	} else {
		return 0, fmt.Errorf("无可用连接")
	}

	if n > 0 {
		t.bytesIn.Add(int64(n))
	}
	return n, err
}

/*
Write 向隧道写入数据
功能：统一的隧道写入接口
*/
func (t *EncryptedTunnel) Write(p []byte) (int, error) {
	if !t.connected.Load() {
		return 0, fmt.Errorf("隧道未连接")
	}

	var n int
	var err error

	if t.wsConn != nil {
		/* WebSocket 写入 */
		t.mu.Lock()
		err = t.wsConn.WriteMessage(websocket.BinaryMessage, p)
		t.mu.Unlock()
		if err == nil {
			n = len(p)
		}
	} else if t.conn != nil {
		/* TCP/TLS 写入 */
		n, err = t.conn.Write(p)
	} else {
		return 0, fmt.Errorf("无可用连接")
	}

	if n > 0 {
		t.bytesOut.Add(int64(n))
	}
	return n, err
}

/*
BridgeToConn 桥接到另一个连接
功能：在隧道和本地连接之间建立双向数据转发
*/
func (t *EncryptedTunnel) BridgeToConn(localConn net.Conn) error {
	done := make(chan struct{}, 2)

	/* 隧道 -> 本地 */
	go func() {
		defer func() { done <- struct{}{} }()
		buf := make([]byte, 32*1024)
		for {
			n, err := t.Read(buf)
			if n > 0 {
				localConn.Write(buf[:n])
			}
			if err != nil {
				return
			}
		}
	}()

	/* 本地 -> 隧道 */
	go func() {
		defer func() { done <- struct{}{} }()
		buf := make([]byte, 32*1024)
		for {
			n, err := localConn.Read(buf)
			if n > 0 {
				t.Write(buf[:n])
			}
			if err != nil {
				if err != io.EOF {
					t.logger.Debug("本地连接读取结束", zap.Error(err))
				}
				return
			}
		}
	}()

	/* 等待任一方向完成 */
	select {
	case <-done:
	case <-t.ctx.Done():
	}

	return nil
}

/*
Close 关闭隧道连接
*/
func (t *EncryptedTunnel) Close() error {
	t.connected.Store(false)
	t.cancel()

	if t.wsConn != nil {
		t.wsConn.WriteMessage(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
		)
		t.wsConn.Close()
	}

	if t.conn != nil {
		t.conn.Close()
	}

	return nil
}

/*
IsConnected 检查隧道是否已连接
*/
func (t *EncryptedTunnel) IsConnected() bool {
	return t.connected.Load()
}

/*
GetStats 获取隧道统计
*/
func (t *EncryptedTunnel) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"type":      string(t.config.Type),
		"connected": t.connected.Load(),
		"bytes_in":  t.bytesIn.Load(),
		"bytes_out": t.bytesOut.Load(),
	}
}

/*
TunnelRelayBridge 隧道转发桥接器
功能：将本地端口监听的连接通过加密隧道转发到远端，
实现 入口节点 -> 加密隧道 -> 出口节点 的完整链路
*/
type TunnelRelayBridge struct {
	localConfig  *TCPRelayConfig
	tunnelConfig *TunnelConfig
	listener     net.Listener
	ctx          context.Context
	cancel       context.CancelFunc
	logger       *zap.Logger
	running      atomic.Bool
	stats        *RelayStats
}

/*
NewTunnelRelayBridge 创建隧道转发桥接器
*/
func NewTunnelRelayBridge(localConfig *TCPRelayConfig, tunnelConfig *TunnelConfig) *TunnelRelayBridge {
	ctx, cancel := context.WithCancel(context.Background())
	return &TunnelRelayBridge{
		localConfig:  localConfig,
		tunnelConfig: tunnelConfig,
		ctx:          ctx,
		cancel:       cancel,
		logger:       zap.L().Named("tunnel-bridge"),
		stats:        &RelayStats{StartTime: time.Now()},
	}
}

/*
Start 启动隧道桥接器
功能：在本地端口监听连接，每个连接建立新的加密隧道转发
*/
func (b *TunnelRelayBridge) Start() error {
	listenAddr := fmt.Sprintf("%s:%d", b.localConfig.ListenAddr, b.localConfig.ListenPort)

	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return fmt.Errorf("监听失败 [%s]: %w", listenAddr, err)
	}

	b.listener = listener
	b.running.Store(true)

	b.logger.Info("隧道桥接器已启动",
		zap.String("listen", listenAddr),
		zap.String("tunnel_type", string(b.tunnelConfig.Type)),
		zap.String("remote", fmt.Sprintf("%s:%d", b.tunnelConfig.RemoteAddr, b.tunnelConfig.RemotePort)))

	go b.acceptLoop()
	return nil
}

/*
acceptLoop 接受连接循环
*/
func (b *TunnelRelayBridge) acceptLoop() {
	for {
		select {
		case <-b.ctx.Done():
			return
		default:
		}

		conn, err := b.listener.Accept()
		if err != nil {
			if b.running.Load() {
				b.logger.Error("接受连接失败", zap.Error(err))
			}
			continue
		}

		b.stats.TotalConns.Add(1)
		b.stats.ActiveConns.Add(1)

		go b.handleConnection(conn)
	}
}

/*
handleConnection 处理连接
功能：为每个本地连接建立加密隧道并进行数据桥接
*/
func (b *TunnelRelayBridge) handleConnection(localConn net.Conn) {
	defer func() {
		localConn.Close()
		b.stats.ActiveConns.Add(-1)
	}()

	/* 建立加密隧道 */
	tunnel := NewEncryptedTunnel(b.tunnelConfig)
	if err := tunnel.Connect(); err != nil {
		b.logger.Error("建立隧道失败",
			zap.String("client", localConn.RemoteAddr().String()),
			zap.Error(err))
		b.stats.FailedConns.Add(1)
		return
	}
	defer tunnel.Close()

	/* 桥接数据 */
	tunnel.BridgeToConn(localConn)
}

/*
Stop 停止隧道桥接器
*/
func (b *TunnelRelayBridge) Stop() error {
	b.running.Store(false)
	b.cancel()

	if b.listener != nil {
		b.listener.Close()
	}

	b.logger.Info("隧道桥接器已停止")
	return nil
}

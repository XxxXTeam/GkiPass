package relay

import (
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

/*
  LoadBalanceMode 负载均衡模式枚举
  功能：定义支持的负载均衡策略
*/
type LoadBalanceMode string

const (
	LBModeRoundRobin LoadBalanceMode = "round-robin"
	LBModeRandom     LoadBalanceMode = "random"
	LBModeWeighted   LoadBalanceMode = "weighted"
	LBModeLeastConn  LoadBalanceMode = "least-conn"
	LBModeIPHash     LoadBalanceMode = "ip-hash"
)

/*
  Backend 后端目标
  功能：代表一个负载均衡的后端目标节点
*/
type Backend struct {
	Host        string `json:"host"`
	Port        int    `json:"port"`
	Weight      int    `json:"weight"`
	Healthy     bool   `json:"healthy"`
	ActiveConns atomic.Int64
	FailCount   atomic.Int64
	LastCheck   time.Time
}

/*
  Address 获取后端完整地址
*/
func (b *Backend) Address() string {
	return fmt.Sprintf("%s:%d", b.Host, b.Port)
}

/*
  LoadBalancer 负载均衡器
  功能：基于多种策略在多个后端目标之间分配连接，
  支持健康检查和自动故障转移
*/
type LoadBalancer struct {
	mode     LoadBalanceMode
	backends []*Backend
	mu       sync.RWMutex
	counter  atomic.Uint64
	logger   *zap.Logger

	/* 健康检查 */
	healthCheckInterval time.Duration
	healthCheckTimeout  time.Duration
	maxFailCount        int64
	stopCh              chan struct{}
}

/*
  NewLoadBalancer 创建负载均衡器
*/
func NewLoadBalancer(mode LoadBalanceMode) *LoadBalancer {
	lb := &LoadBalancer{
		mode:                mode,
		logger:              zap.L().Named("load-balancer"),
		healthCheckInterval: 10 * time.Second,
		healthCheckTimeout:  5 * time.Second,
		maxFailCount:        3,
		stopCh:              make(chan struct{}),
	}
	return lb
}

/*
  AddBackend 添加后端目标
*/
func (lb *LoadBalancer) AddBackend(host string, port int, weight int) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	lb.backends = append(lb.backends, &Backend{
		Host:    host,
		Port:    port,
		Weight:  weight,
		Healthy: true,
	})

	lb.logger.Info("添加后端目标",
		zap.String("host", host),
		zap.Int("port", port),
		zap.Int("weight", weight))
}

/*
  RemoveBackend 移除后端目标
*/
func (lb *LoadBalancer) RemoveBackend(host string, port int) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	addr := fmt.Sprintf("%s:%d", host, port)
	for i, b := range lb.backends {
		if b.Address() == addr {
			lb.backends = append(lb.backends[:i], lb.backends[i+1:]...)
			lb.logger.Info("移除后端目标", zap.String("addr", addr))
			return
		}
	}
}

/*
  Next 获取下一个后端目标
  功能：根据负载均衡策略选择下一个健康的后端目标
*/
func (lb *LoadBalancer) Next(clientIP string) (*Backend, error) {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	healthy := lb.getHealthyBackends()
	if len(healthy) == 0 {
		return nil, fmt.Errorf("没有可用的后端目标")
	}

	switch lb.mode {
	case LBModeRoundRobin:
		return lb.roundRobin(healthy), nil
	case LBModeRandom:
		return lb.random(healthy), nil
	case LBModeWeighted:
		return lb.weighted(healthy), nil
	case LBModeLeastConn:
		return lb.leastConn(healthy), nil
	case LBModeIPHash:
		return lb.ipHash(healthy, clientIP), nil
	default:
		return lb.roundRobin(healthy), nil
	}
}

/*
  getHealthyBackends 获取所有健康的后端
*/
func (lb *LoadBalancer) getHealthyBackends() []*Backend {
	var healthy []*Backend
	for _, b := range lb.backends {
		if b.Healthy {
			healthy = append(healthy, b)
		}
	}
	return healthy
}

/*
  roundRobin 轮询策略
  功能：按顺序依次选择后端目标
*/
func (lb *LoadBalancer) roundRobin(backends []*Backend) *Backend {
	idx := lb.counter.Add(1) - 1
	return backends[idx%uint64(len(backends))]
}

/*
  random 随机策略
*/
func (lb *LoadBalancer) random(backends []*Backend) *Backend {
	return backends[rand.Intn(len(backends))]
}

/*
  weighted 加权轮询策略
  功能：根据权重比例分配请求到不同后端
*/
func (lb *LoadBalancer) weighted(backends []*Backend) *Backend {
	totalWeight := 0
	for _, b := range backends {
		totalWeight += b.Weight
	}

	if totalWeight == 0 {
		return lb.roundRobin(backends)
	}

	r := rand.Intn(totalWeight)
	for _, b := range backends {
		r -= b.Weight
		if r < 0 {
			return b
		}
	}

	return backends[0]
}

/*
  leastConn 最少连接策略
  功能：选择当前活跃连接数最少的后端目标
*/
func (lb *LoadBalancer) leastConn(backends []*Backend) *Backend {
	var selected *Backend
	minConns := int64(^uint64(0) >> 1)

	for _, b := range backends {
		conns := b.ActiveConns.Load()
		if conns < minConns {
			minConns = conns
			selected = b
		}
	}

	return selected
}

/*
  ipHash IP 哈希策略
  功能：相同客户端 IP 始终路由到相同后端，实现会话保持
*/
func (lb *LoadBalancer) ipHash(backends []*Backend, clientIP string) *Backend {
	hash := uint64(0)
	for _, c := range clientIP {
		hash = hash*31 + uint64(c)
	}
	return backends[hash%uint64(len(backends))]
}

/*
  MarkUnhealthy 标记后端为不健康
  功能：记录失败并在达到阈值时标记后端为不可用
*/
func (lb *LoadBalancer) MarkUnhealthy(host string, port int) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	addr := fmt.Sprintf("%s:%d", host, port)
	for _, b := range lb.backends {
		if b.Address() == addr {
			count := b.FailCount.Add(1)
			if count >= lb.maxFailCount {
				b.Healthy = false
				lb.logger.Warn("后端标记为不健康",
					zap.String("addr", addr),
					zap.Int64("fail_count", count))
			}
			return
		}
	}
}

/*
  MarkHealthy 标记后端为健康
*/
func (lb *LoadBalancer) MarkHealthy(host string, port int) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	addr := fmt.Sprintf("%s:%d", host, port)
	for _, b := range lb.backends {
		if b.Address() == addr {
			b.Healthy = true
			b.FailCount.Store(0)
			return
		}
	}
}

/*
  GetStats 获取负载均衡器统计
*/
func (lb *LoadBalancer) GetStats() map[string]interface{} {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	var backendStats []map[string]interface{}
	for _, b := range lb.backends {
		backendStats = append(backendStats, map[string]interface{}{
			"address":      b.Address(),
			"weight":       b.Weight,
			"healthy":      b.Healthy,
			"active_conns": b.ActiveConns.Load(),
			"fail_count":   b.FailCount.Load(),
		})
	}

	return map[string]interface{}{
		"mode":     string(lb.mode),
		"backends": backendStats,
		"total":    len(lb.backends),
		"healthy":  len(lb.getHealthyBackends()),
	}
}

/*
  Stop 停止负载均衡器
*/
func (lb *LoadBalancer) Stop() {
	close(lb.stopCh)
}

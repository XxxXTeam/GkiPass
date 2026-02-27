package service

import (
	"sync"
	"time"

	"gkipass/plane/internal/pkg/logger"

	"go.uber.org/zap"
)

// ConfigHotReloader 配置热更新器
type ConfigHotReloader struct {
	watchers      map[string][]ConfigWatcher
	mu            sync.RWMutex
	stopChan      chan struct{}
	updateChan    chan ConfigUpdate
	checkInterval time.Duration
}

// ConfigWatcher 配置监听器
type ConfigWatcher func(update ConfigUpdate)

// ConfigUpdate 配置更新
type ConfigUpdate struct {
	Type      string // tunnel, node, settings
	Action    string // create, update, delete
	Data      interface{}
	Timestamp time.Time
}

// NewConfigHotReloader 创建配置热更新器
func NewConfigHotReloader() *ConfigHotReloader {
	return &ConfigHotReloader{
		watchers:      make(map[string][]ConfigWatcher),
		stopChan:      make(chan struct{}),
		updateChan:    make(chan ConfigUpdate, 100),
		checkInterval: 5 * time.Second,
	}
}

// Watch 注册配置监听器
func (chr *ConfigHotReloader) Watch(configType string, watcher ConfigWatcher) {
	chr.mu.Lock()
	defer chr.mu.Unlock()
	chr.watchers[configType] = append(chr.watchers[configType], watcher)
	logger.Info("配置监听器已注册", zap.String("type", configType))
}

// NotifyUpdate 通知配置更新
func (chr *ConfigHotReloader) NotifyUpdate(update ConfigUpdate) {
	select {
	case chr.updateChan <- update:
		logger.Debug("配置更新已通知",
			zap.String("type", update.Type),
			zap.String("action", update.Action))
	default:
		logger.Warn("配置更新通道已满，丢弃更新")
	}
}

// Start 启动热更新器
func (chr *ConfigHotReloader) Start() {
	go chr.processUpdates()
	logger.Info("配置热更新器已启动")
}

// Stop 停止热更新器
func (chr *ConfigHotReloader) Stop() {
	close(chr.stopChan)
	logger.Info("配置热更新器已停止")
}

// processUpdates 处理配置更新
func (chr *ConfigHotReloader) processUpdates() {
	for {
		select {
		case update := <-chr.updateChan:
			chr.notifyWatchers(update)
		case <-chr.stopChan:
			return
		}
	}
}

/* notifyWatchers 通知监听器（含 panic 恢复，防止 watcher 崩溃传播） */
func (chr *ConfigHotReloader) notifyWatchers(update ConfigUpdate) {
	chr.mu.RLock()
	watchers := chr.watchers[update.Type]
	chr.mu.RUnlock()

	for _, w := range watchers {
		watcher := w
		go func() {
			defer func() {
				if r := recover(); r != nil {
					logger.Error("配置监听器 panic",
						zap.String("type", update.Type),
						zap.Any("panic", r))
				}
			}()
			watcher(update)
		}()
	}
}

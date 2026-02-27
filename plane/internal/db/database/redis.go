package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

/*
  RedisConfig Redis 连接配置
  功能：管理 Redis 连接参数，支持单机和哨兵模式
*/
type RedisConfig struct {
	Addr     string `yaml:"addr" json:"addr"`
	Password string `yaml:"password" json:"password"`
	DB       int    `yaml:"db" json:"db"`

	/* 连接池配置 */
	PoolSize     int           `yaml:"pool_size" json:"pool_size"`
	MinIdleConns int           `yaml:"min_idle_conns" json:"min_idle_conns"`
	MaxRetries   int           `yaml:"max_retries" json:"max_retries"`
	DialTimeout  time.Duration `yaml:"dial_timeout" json:"dial_timeout"`
	ReadTimeout  time.Duration `yaml:"read_timeout" json:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout" json:"write_timeout"`
}

/*
  DefaultRedisConfig 返回默认 Redis 配置
  功能：提供开箱即用的 Redis 连接参数
*/
func DefaultRedisConfig() *RedisConfig {
	return &RedisConfig{
		Addr:         "localhost:6379",
		Password:     "",
		DB:           0,
		PoolSize:     10,
		MinIdleConns: 3,
		MaxRetries:   3,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	}
}

/*
  RedisClient Redis 客户端封装
  功能：提供 Redis 连接管理和常用操作的封装，
  支持缓存、会话、验证码存储等多种用途
*/
type RedisClient struct {
	client *redis.Client
	ctx    context.Context
}

/*
  NewRedisClient 创建 Redis 客户端
  功能：根据配置初始化 Redis 连接，连接失败时返回 nil（Redis 为可选组件）
*/
func NewRedisClient(cfg *RedisConfig) (*RedisClient, error) {
	if cfg == nil || cfg.Addr == "" {
		return nil, nil
	}

	client := redis.NewClient(&redis.Options{
		Addr:         cfg.Addr,
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
		MaxRetries:   cfg.MaxRetries,
		DialTimeout:  cfg.DialTimeout,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	})

	ctx := context.Background()

	/* 测试连接 */
	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		return nil, fmt.Errorf("Redis 连接失败 [%s]: %w", cfg.Addr, err)
	}

	log.Printf("✓ Redis 连接成功 [%s]", cfg.Addr)

	return &RedisClient{
		client: client,
		ctx:    ctx,
	}, nil
}

/*
  Client 获取底层 Redis 客户端
  功能：直接访问 go-redis 原始客户端进行高级操作
*/
func (r *RedisClient) Client() *redis.Client {
	return r.client
}

/* ===== 缓存操作 ===== */

/*
  Set 设置缓存
  功能：将键值对写入 Redis，支持设置过期时间
*/
func (r *RedisClient) Set(key string, value interface{}, expiration time.Duration) error {
	return r.client.Set(r.ctx, key, value, expiration).Err()
}

/*
  Get 获取缓存
  功能：根据键获取 Redis 中存储的值
*/
func (r *RedisClient) Get(key string) (string, error) {
	return r.client.Get(r.ctx, key).Result()
}

/*
  Del 删除缓存
  功能：从 Redis 中删除指定的键
*/
func (r *RedisClient) Del(keys ...string) error {
	return r.client.Del(r.ctx, keys...).Err()
}

/*
  Exists 检查键是否存在
  功能：判断 Redis 中是否存在指定的键
*/
func (r *RedisClient) Exists(key string) (bool, error) {
	n, err := r.client.Exists(r.ctx, key).Result()
	return n > 0, err
}

/*
  SetNX 设置键值对（仅在键不存在时）
  功能：原子性地设置键值对，常用于分布式锁
*/
func (r *RedisClient) SetNX(key string, value interface{}, expiration time.Duration) (bool, error) {
	return r.client.SetNX(r.ctx, key, value, expiration).Result()
}

/*
  Incr 自增
  功能：对存储在键中的数字值加 1
*/
func (r *RedisClient) Incr(key string) (int64, error) {
	return r.client.Incr(r.ctx, key).Result()
}

/*
  Expire 设置过期时间
  功能：为已存在的键设置过期时间
*/
func (r *RedisClient) Expire(key string, expiration time.Duration) error {
	return r.client.Expire(r.ctx, key, expiration).Err()
}

/* ===== 哈希操作 ===== */

/*
  HSet 设置哈希字段
  功能：设置哈希表中指定字段的值
*/
func (r *RedisClient) HSet(key string, values ...interface{}) error {
	return r.client.HSet(r.ctx, key, values...).Err()
}

/*
  HGet 获取哈希字段值
  功能：获取哈希表中指定字段的值
*/
func (r *RedisClient) HGet(key, field string) (string, error) {
	return r.client.HGet(r.ctx, key, field).Result()
}

/*
  HGetAll 获取哈希表所有字段和值
  功能：返回哈希表中所有的字段和值
*/
func (r *RedisClient) HGetAll(key string) (map[string]string, error) {
	return r.client.HGetAll(r.ctx, key).Result()
}

/*
  HDel 删除哈希表字段
  功能：从哈希表中删除指定的字段
*/
func (r *RedisClient) HDel(key string, fields ...string) error {
	return r.client.HDel(r.ctx, key, fields...).Err()
}

/* ===== 列表操作 ===== */

/*
  LPush 从列表左侧插入
  功能：将一个或多个值插入到列表头部
*/
func (r *RedisClient) LPush(key string, values ...interface{}) error {
	return r.client.LPush(r.ctx, key, values...).Err()
}

/*
  LRange 获取列表指定范围的元素
  功能：返回列表中指定区间内的元素
*/
func (r *RedisClient) LRange(key string, start, stop int64) ([]string, error) {
	return r.client.LRange(r.ctx, key, start, stop).Result()
}

/* ===== 发布/订阅 ===== */

/*
  Publish 发布消息
  功能：向指定频道发送消息
*/
func (r *RedisClient) Publish(channel string, message interface{}) error {
	return r.client.Publish(r.ctx, channel, message).Err()
}

/*
  Subscribe 订阅频道
  功能：订阅一个或多个频道，返回消息管道
*/
func (r *RedisClient) Subscribe(channels ...string) *redis.PubSub {
	return r.client.Subscribe(r.ctx, channels...)
}

/*
  Close 关闭 Redis 连接
  功能：优雅地关闭 Redis 客户端连接
*/
func (r *RedisClient) Close() error {
	if r.client != nil {
		return r.client.Close()
	}
	return nil
}

/*
  IsAvailable 检查 Redis 是否可用
  功能：通过 Ping 命令检测 Redis 连接状态
*/
func (r *RedisClient) IsAvailable() bool {
	if r == nil || r.client == nil {
		return false
	}
	return r.client.Ping(r.ctx).Err() == nil
}

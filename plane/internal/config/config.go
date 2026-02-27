package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config 全局配置
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	Redis    RedisConfig    `yaml:"redis"`
	Auth     AuthConfig     `yaml:"auth"`
	TLS      TLSConfig      `yaml:"tls"`
	Log      LogConfig      `yaml:"log"`
	Captcha  CaptchaConfig  `yaml:"captcha"`
	Payment  PaymentConfig  `yaml:"payment"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Host         string `yaml:"host"`
	Port         int    `yaml:"port"`
	HTTP3Port    int    `yaml:"http3_port"`   // HTTP/3 (QUIC) 端口
	Mode         string `yaml:"mode"`         // debug, release
	EnableHTTP3  bool   `yaml:"enable_http3"` // 启用 HTTP/3
	ReadTimeout  int    `yaml:"read_timeout"`
	WriteTimeout int    `yaml:"write_timeout"`

	/* CORS 跨域配置 */
	CORSAllowedOrigins []string `yaml:"cors_allowed_origins"` /* 允许的来源列表，["*"] 表示允许所有（仅开发环境） */

	/* WebSocket 配置 */
	WSMaxConnections int `yaml:"ws_max_connections"` /* WebSocket 最大连接数，0 表示不限制 */
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Type     string `yaml:"type"`     // 数据库类型: sqlite, mysql, postgres
	Host     string `yaml:"host"`     // 数据库主机
	Port     int    `yaml:"port"`     // 数据库端口
	User     string `yaml:"user"`     // 数据库用户名
	Password string `yaml:"password"` // 数据库密码
	DBName   string `yaml:"db_name"`  // 数据库名称
	SSLMode  string `yaml:"ssl_mode"` // SSL模式 (postgres)
	Charset  string `yaml:"charset"`  // 字符集 (mysql)

	/* SQLite 专用 */
	SQLitePath string `yaml:"sqlite_path"`

	/* 连接池 */
	MaxOpenConns int `yaml:"max_open_conns"` // 最大打开连接数
	MaxIdleConns int `yaml:"max_idle_conns"` // 最大空闲连接数

	/* 日志 */
	LogLevel string `yaml:"log_level"` // silent, error, warn, info
}

// RedisConfig Redis配置
type RedisConfig struct {
	Addr         string `yaml:"addr"`
	Password     string `yaml:"password"`
	DB           int    `yaml:"db"`
	PoolSize     int    `yaml:"pool_size"`
	MinIdleConns int    `yaml:"min_idle_conns"`
	MaxRetries   int    `yaml:"max_retries"`
}

// AuthConfig 认证配置
type AuthConfig struct {
	JWTSecret     string      `yaml:"jwt_secret"`
	JWTExpiration int         `yaml:"jwt_expiration"` // 单位：小时
	AdminPassword string      `yaml:"admin_password"`
	GitHub        GitHubOAuth `yaml:"github"`
}

// GitHubOAuth GitHub OAuth2 配置
type GitHubOAuth struct {
	Enabled      bool   `yaml:"enabled"`
	ClientID     string `yaml:"client_id"`
	ClientSecret string `yaml:"client_secret"`
	RedirectURL  string `yaml:"redirect_url"`
}

// TLSConfig TLS配置
type TLSConfig struct {
	Enabled    bool   `yaml:"enabled"`
	CertFile   string `yaml:"cert_file"`
	KeyFile    string `yaml:"key_file"`
	CAFile     string `yaml:"ca_file"`
	MinVersion string `yaml:"min_version"` // TLS 1.2, TLS 1.3
	EnableALPN bool   `yaml:"enable_alpn"` // ALPN for HTTP/2, HTTP/3
}

// LogConfig 日志配置
type LogConfig struct {
	Level      string `yaml:"level"`       // debug, info, warn, error
	Format     string `yaml:"format"`      // json, console
	OutputPath string `yaml:"output_path"` // 日志文件路径
	MaxSize    int    `yaml:"max_size"`    // 单个日志文件大小(MB)
	MaxBackups int    `yaml:"max_backups"` // 保留的旧日志文件数量
	MaxAge     int    `yaml:"max_age"`     // 保留天数
	Compress   bool   `yaml:"compress"`    // 是否压缩
}

// CaptchaConfig 验证码配置
type CaptchaConfig struct {
	Enabled        bool   `yaml:"enabled"`         // 是否启用验证码
	Type           string `yaml:"type"`            // 类型: image, turnstile, gocaptcha
	EnableLogin    bool   `yaml:"enable_login"`    // 登录页面启用
	EnableRegister bool   `yaml:"enable_register"` // 注册页面启用
	Expiration     int    `yaml:"expiration"`      // 过期时间（秒）

	/* 传统图片验证码配置 */
	ImageWidth  int `yaml:"image_width"`
	ImageHeight int `yaml:"image_height"`
	CodeLength  int `yaml:"code_length"`

	/* GoCaptcha 行为验证码配置 */
	GoCaptchaMode        string `yaml:"gocaptcha_mode"`         // click, slide, drag, rotate
	GoCaptchaWidth       int    `yaml:"gocaptcha_width"`        // 验证码图片宽度
	GoCaptchaHeight      int    `yaml:"gocaptcha_height"`       // 验证码图片高度
	GoCaptchaThumbWidth  int    `yaml:"gocaptcha_thumb_width"`  // 缩略图宽度
	GoCaptchaThumbHeight int    `yaml:"gocaptcha_thumb_height"` // 缩略图高度

	/* Cloudflare Turnstile 配置 */
	TurnstileSiteKey   string `yaml:"turnstile_site_key"`
	TurnstileSecretKey string `yaml:"turnstile_secret_key"`
}

// PaymentConfig 支付配置
type PaymentConfig struct {
	CallbackSecret string `yaml:"callback_secret"` /* 支付回调签名密钥 */
	CryptoSalt     string `yaml:"crypto_salt"`     /* 加密货币地址生成盐值 */
}

// LoadConfig 从文件加载配置
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	config := &Config{}
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	config.warnInsecureDefaults()
	return config, nil
}

/*
warnInsecureDefaults 检查生产环境下是否使用了不安全的默认值
功能：在 release 模式下对 JWT 默认密钥、默认管理员密码等输出警告日志，
提醒运维人员及时修改，避免上线后被利用。
*/
func (c *Config) warnInsecureDefaults() {
	if c.Server.Mode != "release" {
		return
	}

	if c.Auth.JWTSecret == "change-this-secret-in-production" || len(c.Auth.JWTSecret) < 16 {
		fmt.Println("[SECURITY WARNING] 生产环境使用了默认或过短的 JWT 密钥，请立即修改 auth.jwt_secret")
	}
	if c.Auth.AdminPassword == "admin123" {
		fmt.Println("[SECURITY WARNING] 生产环境使用了默认管理员密码 'admin123'，请立即修改 auth.admin_password")
	}
	if c.Payment.CallbackSecret == "" {
		fmt.Println("[SECURITY WARNING] 支付回调签名密钥为空，请配置 payment.callback_secret")
	}
	if len(c.Server.CORSAllowedOrigins) == 0 {
		return
	}
	for _, o := range c.Server.CORSAllowedOrigins {
		if o == "*" {
			fmt.Println("[SECURITY WARNING] 生产环境 CORS 允许所有来源（*），请配置具体域名白名单 server.cors_allowed_origins")
			break
		}
	}
}

// LoadConfigOrDefault 加载配置或使用默认值
func LoadConfigOrDefault(path string) *Config {
	if path == "" {
		return DefaultConfig()
	}

	config, err := LoadConfig(path)
	if err != nil {
		fmt.Printf("Failed to load config: %v, using defaults\n", err)
		return DefaultConfig()
	}

	return config
}

// DefaultConfig 默认配置
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Host:               "0.0.0.0",
			Port:               8080,
			HTTP3Port:          8443,
			Mode:               "debug",
			EnableHTTP3:        false,
			ReadTimeout:        30,
			WriteTimeout:       30,
			CORSAllowedOrigins: []string{"*"}, /* 开发模式默认允许所有，生产环境应改为具体域名 */
			WSMaxConnections:   1000,          /* 默认最大 1000 个 WebSocket 连接 */
		},
		Database: DatabaseConfig{
			Type:         "sqlite",
			SQLitePath:   "./data/gkipass.db",
			Host:         "localhost",
			Port:         3306,
			User:         "root",
			Password:     "",
			DBName:       "gkipass",
			SSLMode:      "disable",
			Charset:      "utf8mb4",
			MaxOpenConns: 25,
			MaxIdleConns: 5,
			LogLevel:     "warn",
		},
		Redis: RedisConfig{
			Addr:         "localhost:6379",
			Password:     "",
			DB:           0,
			PoolSize:     10,
			MinIdleConns: 3,
			MaxRetries:   3,
		},
		Auth: AuthConfig{
			JWTSecret:     "change-this-secret-in-production",
			JWTExpiration: 24,
			AdminPassword: "admin123",
			GitHub: GitHubOAuth{
				Enabled:      false,
				ClientID:     "",
				ClientSecret: "",
				RedirectURL:  "http://localhost:3000/auth/callback/github",
			},
		},
		TLS: TLSConfig{
			Enabled:    false,
			CertFile:   "",
			KeyFile:    "",
			CAFile:     "",
			MinVersion: "TLS 1.3",
			EnableALPN: true,
		},
		Log: LogConfig{
			Level:      "info",
			Format:     "console",
			OutputPath: "./logs/gkipass.log",
			MaxSize:    100,
			MaxBackups: 10,
			MaxAge:     30,
			Compress:   true,
		},
		Payment: PaymentConfig{
			CallbackSecret: "",
			CryptoSalt:     "",
		},
		Captcha: CaptchaConfig{
			Enabled:              false,
			Type:                 "gocaptcha",
			EnableLogin:          false,
			EnableRegister:       true,
			Expiration:           300,
			ImageWidth:           240,
			ImageHeight:          80,
			CodeLength:           6,
			GoCaptchaMode:        "click",
			GoCaptchaWidth:       300,
			GoCaptchaHeight:      220,
			GoCaptchaThumbWidth:  150,
			GoCaptchaThumbHeight: 40,
			TurnstileSiteKey:     "",
			TurnstileSecretKey:   "",
		},
	}
}

// SaveConfig 保存配置到文件
func SaveConfig(cfg *Config, path string) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	/* 0600：仅所有者可读写，配置文件含敏感信息（密钥/密码） */
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

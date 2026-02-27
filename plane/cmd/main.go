package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"gkipass/plane/internal/api"
	"gkipass/plane/internal/config"
	"gkipass/plane/internal/db"
	"gkipass/plane/internal/db/dao"
	"gkipass/plane/internal/pkg/initializer"
	"gkipass/plane/internal/pkg/logger"
	"gkipass/plane/internal/server"
	"gkipass/plane/internal/service"
	"gkipass/plane/internal/ws"

	"go.uber.org/zap"
)

var (
	configPath = flag.String("config", "./config.yaml", "配置文件路径")
	port       = flag.Int("port", 0, "覆盖服务器端口")
)

/*
main 程序入口
启动流程：
 1. 初始化引导日志 → 检测首次运行 → 创建目录/配置/证书
 2. 加载配置文件 → 用配置重新初始化日志
 3. 初始化数据库（SQLite/MySQL/Postgres + 可选 Redis）
 4. 并行启动独立服务：JWT 管理器、端口管理器、清理服务、WebSocket 服务器
 5. 组装路由 → 启动 HTTP/2（+ 可选 HTTP/3）服务器
 6. 等待 SIGINT/SIGTERM → 优雅关闭
*/
func main() {
	startupBegin := time.Now()
	flag.Parse()

	/* 阶段 1：引导日志（配置加载前使用临时 console 日志） */
	if err := logger.Init(&logger.Config{
		Level:  "info",
		Format: "console",
	}); err != nil {
		log.Fatalf("初始化日志系统失败: %v", err)
	}
	defer logger.Sync()

	/* 阶段 2：首次运行检测与初始化 */
	isFirstRun := initializer.IsFirstRun(*configPath)
	if err := initializer.InitDirectories(); err != nil {
		logger.Fatal("初始化目录失败", zap.Error(err))
	}
	if isFirstRun {
		initializer.PrintWelcome()
		if err := initializer.InitConfig(*configPath); err != nil {
			logger.Fatal("初始化配置失败", zap.Error(err))
		}
		if err := initializer.InitCertificates("./certs"); err != nil {
			logger.Fatal("初始化证书失败", zap.Error(err))
		}
	} else {
		printBanner()
	}

	/* 阶段 3：加载配置 → 用配置重新初始化日志系统 */
	cfg := config.LoadConfigOrDefault(*configPath)
	if *port > 0 {
		cfg.Server.Port = *port
	}
	if err := logger.Init(&logger.Config{
		Level:      cfg.Log.Level,
		Format:     cfg.Log.Format,
		OutputPath: cfg.Log.OutputPath,
		MaxSize:    cfg.Log.MaxSize,
		MaxBackups: cfg.Log.MaxBackups,
		MaxAge:     cfg.Log.MaxAge,
		Compress:   cfg.Log.Compress,
	}); err != nil {
		logger.Fatal("重新初始化日志系统失败", zap.Error(err))
	}

	/* 阶段 4：初始化数据库（必须串行，后续服务依赖它） */
	dbStart := time.Now()
	dbManager, err := db.NewManager(&db.Config{
		DBType:        cfg.Database.Type,
		SQLitePath:    cfg.Database.SQLitePath,
		DBHost:        cfg.Database.Host,
		DBPort:        cfg.Database.Port,
		DBUser:        cfg.Database.User,
		DBPassword:    cfg.Database.Password,
		DBName:        cfg.Database.DBName,
		DBSSLMode:     cfg.Database.SSLMode,
		DBCharset:     cfg.Database.Charset,
		MaxOpenConns:  cfg.Database.MaxOpenConns,
		MaxIdleConns:  cfg.Database.MaxIdleConns,
		DBLogLevel:    cfg.Database.LogLevel,
		RedisAddr:     cfg.Redis.Addr,
		RedisPassword: cfg.Redis.Password,
		RedisDB:       cfg.Redis.DB,
	})
	if err != nil {
		logger.Fatal("初始化数据库失败", zap.Error(err))
	}
	defer dbManager.Close()
	logger.Info("✓ 数据库初始化完成", zap.Duration("耗时", time.Since(dbStart)))

	/* 首次启动：创建默认管理员（空数据库时自动执行） */
	if err := initializer.InitAdmin(dbManager.GormDB); err != nil {
		logger.Fatal("初始化管理员失败", zap.Error(err))
	}

	/* 初始化 GORM DAO 层 */
	gormDAO := dao.New(dbManager.GormDB)

	/*
		阶段 5：并行启动独立服务
		JWT 管理器、端口管理器、清理服务、WebSocket 服务器互不依赖，
		使用 sync.WaitGroup 并行初始化以缩短启动时间
	*/
	servicesStart := time.Now()
	var (
		jwtManager      *service.JWTManager
		cleanupService  *service.CleanupService
		failoverService *service.FailoverService
		wsServer        *ws.Server
		wg              sync.WaitGroup
	)

	/*
		串行初始化有依赖关系的服务（轻量级，耗时可忽略）：
		- FailoverService 必须先创建，WebSocket Handler 依赖它
		- PortManager 也在主线程初始化
	*/
	failoverService = service.NewFailoverService(dbManager.GormDB)
	failoverService.Start()
	defer failoverService.Stop()
	service.GetPortManager(gormDAO)

	wg.Add(3)

	/* JWT 密钥管理器：生成/加载签名密钥 */
	go func() {
		defer wg.Done()
		jwtManager = service.NewJWTManager(dbManager)
		if err := jwtManager.Start(); err != nil {
			logger.Fatal("初始化 JWT 管理器失败", zap.Error(err))
		}
		logger.Debug("✓ JWT 管理器就绪")
	}()

	/* 定时清理服务：过期 session、临时数据等 */
	go func() {
		defer wg.Done()
		cleanupService = service.NewCleanupService(gormDAO)
		logger.Debug("✓ 清理服务就绪")
	}()

	/* WebSocket 服务器：管理节点长连接（依赖 failoverService） */
	go func() {
		defer wg.Done()
		wsServer = ws.NewServer(gormDAO, cfg.Server.WSMaxConnections, failoverService)
		wsServer.Start()
		logger.Debug("✓ WebSocket 服务器就绪")
	}()

	wg.Wait()

	/* 服务间依赖串行处理 */
	defer jwtManager.Stop()
	cfg.Auth.JWTSecret = jwtManager.GetSecret()
	go cleanupService.Start()
	defer cleanupService.Stop()

	logger.Info("✓ 后台服务并行初始化完成", zap.Duration("耗时", time.Since(servicesStart)))

	/* 阶段 6：组装路由 + 启动 HTTP 服务器 */
	app := api.NewApp(cfg, dbManager)
	router := api.SetupRouter(app, wsServer)
	http2Addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)

	var tlsConfig *tls.Config
	if cfg.TLS.Enabled && cfg.TLS.CertFile != "" && cfg.TLS.KeyFile != "" {
		tlsConfig = createTLSConfig(cfg)
	}

	http2Server := server.NewHTTP2Server(
		http2Addr, router, tlsConfig,
		time.Duration(cfg.Server.ReadTimeout)*time.Second,
		time.Duration(cfg.Server.WriteTimeout)*time.Second,
	)
	go func() {
		if cfg.TLS.Enabled {
			logger.Info("✓ HTTPS 服务器启动", zap.String("addr", http2Addr))
		} else {
			logger.Info("✓ HTTP 服务器启动", zap.String("addr", http2Addr))
		}
		var err error
		if cfg.TLS.Enabled {
			err = http2Server.Start(cfg.TLS.CertFile, cfg.TLS.KeyFile)
		} else {
			err = http2Server.StartInsecure()
		}
		if err != nil && err != http.ErrServerClosed {
			logger.Fatal("HTTP 服务器异常退出", zap.Error(err))
		}
	}()

	var http3Server *server.HTTP3Server
	if cfg.Server.EnableHTTP3 && cfg.TLS.Enabled {
		http3Addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.HTTP3Port)
		http3Server = server.NewHTTP3Server(http3Addr, router, tlsConfig)
		go func() {
			logger.Info("✓ HTTP/3 (QUIC) 服务器启动", zap.String("addr", http3Addr))
			if err := http3Server.Start(); err != nil {
				logger.Error("HTTP/3 服务器错误", zap.Error(err))
			}
		}()
	} else if cfg.Server.EnableHTTP3 {
		logger.Warn("HTTP/3 已启用但 TLS 未配置，跳过 HTTP/3 服务器")
	}

	logger.Info("✓ GkiPass 控制面板启动完成",
		zap.Duration("总耗时", time.Since(startupBegin)),
		zap.String("监听地址", http2Addr))

	/* 阶段 7：等待退出信号 → 优雅关闭 */
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("收到退出信号，正在优雅关闭...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := http2Server.Shutdown(ctx); err != nil {
		logger.Error("关闭 HTTP/2 服务器失败", zap.Error(err))
	}
	if http3Server != nil {
		if err := http3Server.Shutdown(ctx); err != nil {
			logger.Error("关闭 HTTP/3 服务器失败", zap.Error(err))
		}
	}

	logger.Info("✓ 所有服务器已停止")
}

func printBanner() {
	banner := `
╔═══════════════════════════════════════════════════════╗
║                                                       ║
║   ██████╗ ██╗  ██╗██╗    ██████╗  █████╗ ███████╗███╗
║  ██╔════╝ ██║ ██╔╝██║    ██╔══██╗██╔══██╗██╔════╝████║
║  ██║  ███╗█████╔╝ ██║    ██████╔╝███████║███████╗╚═██║
║  ██║   ██║██╔═██╗ ██║    ██╔═══╝ ██╔══██║╚════██║  ██║
║  ╚██████╔╝██║  ██╗██║    ██║     ██║  ██║███████║  ██║
║   ╚═════╝ ╚═╝  ╚═╝╚═╝    ╚═╝     ╚═╝  ╚═╝╚══════╝  ╚═╝
║                                                       ║
║           Bidirectional Tunnel Control Plane         ║
║                      v2.0.0                           ║
║                                                       ║
╚═══════════════════════════════════════════════════════╝
`
	fmt.Println(banner)
}

package app

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"proman/server/internal/config"
	"proman/server/internal/handler"
	"proman/server/internal/middleware"
	"proman/server/internal/pkg/markdownpreview"
	"proman/server/internal/pkg/migrate"
	"proman/server/internal/pkg/ratelimit"
	"proman/server/internal/repository"
	"proman/server/internal/service"
)

type App struct {
	Config config.Config
	DB     *gorm.DB
	Redis  *redis.Client
	Router *gin.Engine
	Server *http.Server
	Logger *log.Logger
}

func New(loggerInstance *log.Logger) (*App, error) {
	loggerInstance = normalizeLogger(loggerInstance)

	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}

	if cfg.AppEnv != "local" {
		gin.SetMode(gin.ReleaseMode)
	}

	db, err := newDB(cfg)
	if err != nil {
		return nil, err
	}

	loggerInstance.Println("mysql connected")

	if err := migrate.Run(db, filepath.Join("migrations")); err != nil {
		return nil, err
	}

	loggerInstance.Println("database migrations applied")

	redisClient, err := newRedis(cfg)
	if err != nil {
		return nil, err
	}

	loggerInstance.Println("redis connected")

	userRepo := repository.NewUserRepository(db)
	projectRepo := repository.NewProjectRepository(db)
	announcementRepo := repository.NewAnnouncementRepository(db)
	versionRepo := repository.NewVersionRepository(db)
	changelogRepo := repository.NewChangelogRepository(db)
	authService := service.NewAuthService(userRepo, cfg.JWTSecret, cfg.JWTExpire)
	projectService := service.NewProjectService(projectRepo)
	announcementService := service.NewAnnouncementService(projectRepo, announcementRepo)
	versionService := service.NewVersionServiceWithCompare(projectRepo, versionRepo, changelogRepo)
	changelogService := service.NewChangelogService(changelogRepo, versionRepo)
	changelogExportService := service.NewChangelogExportService(projectRepo, versionRepo, changelogRepo)
	publicService := service.NewPublicService(projectRepo, versionRepo, changelogRepo, announcementRepo)
	if err := authService.EnsureAdmin(context.Background(), cfg.AdminUsername, cfg.AdminPassword); err != nil {
		return nil, err
	}

	limiter := ratelimit.NewRedisLimiter(redisClient)

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(middleware.ErrorHandler())
	router.Use(corsMiddleware(cfg.CORSAllowOrigins))

	healthHandler := handler.NewHealthHandler()
	authHandler := handler.NewAuthHandler(authService)
	markdownHandler := handler.NewMarkdownHandler(markdownpreview.NewRenderer())
	projectHandler := handler.NewProjectHandler(projectService)
	changelogExportHandler := handler.NewChangelogExportHandler(changelogExportService)
	announcementHandler := handler.NewAnnouncementHandler(announcementService)
	versionHandler := handler.NewVersionHandler(versionService)
	changelogHandler := handler.NewChangelogHandler(changelogService)
	publicHandler := handler.NewPublicHandler(publicService)
	router.GET("/healthz", healthHandler.Get)
	router.POST("/api/auth/login", middleware.LoginRateLimit(limiter), authHandler.Login)

	api := router.Group("/api")
	api.Use(middleware.JWTAuth(cfg.JWTSecret))
	api.POST("/markdown/preview", markdownHandler.Preview)
	api.GET("/projects", projectHandler.List)
	api.POST("/projects", projectHandler.Create)
	api.GET("/projects/:id", projectHandler.Get)
	api.PUT("/projects/:id", projectHandler.Update)
	api.POST("/projects/:id/token/refresh", projectHandler.RefreshToken)
	api.DELETE("/projects/:id", projectHandler.Delete)
	api.GET("/projects/:id/changelogs/export", changelogExportHandler.Export)
	api.GET("/projects/:id/announcements", announcementHandler.List)
	api.POST("/projects/:id/announcements", announcementHandler.Create)
	api.GET("/announcements/:id", announcementHandler.Get)
	api.PUT("/announcements/:id", announcementHandler.Update)
	api.PUT("/announcements/:id/publish", announcementHandler.Publish)
	api.PUT("/announcements/:id/revoke", announcementHandler.Revoke)
	api.DELETE("/announcements/:id", announcementHandler.Delete)
	api.GET("/projects/:id/versions", versionHandler.List)
	api.GET("/projects/:id/versions/compare", versionHandler.Compare)
	api.POST("/projects/:id/versions", versionHandler.Create)
	api.GET("/versions/:id", versionHandler.Get)
	api.PUT("/versions/:id", versionHandler.Update)
	api.DELETE("/versions/:id", versionHandler.Delete)
	api.PUT("/versions/:id/publish", versionHandler.Publish)
	api.GET("/versions/:id/changelogs", changelogHandler.List)
	api.POST("/versions/:id/changelogs", changelogHandler.Create)
	api.PUT("/changelogs/:id", changelogHandler.Update)
	api.DELETE("/changelogs/:id", changelogHandler.Delete)
	api.PUT("/versions/:id/changelogs/reorder", changelogHandler.Reorder)

	v1 := router.Group("/v1")
	v1.Use(middleware.ProjectTokenAuth(projectRepo))
	v1.Use(middleware.PublicRateLimit(limiter))
	v1.GET("/project", publicHandler.GetProject)
	v1.GET("/versions", publicHandler.ListVersions)
	v1.GET("/versions/:version/changelogs", publicHandler.GetVersionChangelogs)
	v1.GET("/announcements", publicHandler.ListAnnouncements)

	registerFrontendRoutes(router, loggerInstance)

	server := &http.Server{
		Addr:              ":" + cfg.HTTPPort,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	return &App{
		Config: cfg,
		DB:     db,
		Redis:  redisClient,
		Router: router,
		Server: server,
		Logger: loggerInstance,
	}, nil
}

func (a *App) Run() error {
	return a.Server.ListenAndServe()
}

func (a *App) Shutdown(ctx context.Context) error {
	var errs []error

	if a.Server != nil {
		if err := a.Server.Shutdown(ctx); err != nil {
			errs = append(errs, err)
		}
	}

	if a.Redis != nil {
		if err := a.Redis.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if a.DB != nil {
		sqlDB, err := a.DB.DB()
		if err != nil {
			errs = append(errs, err)
		} else if err := sqlDB.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

func newDB(cfg config.Config) (*gorm.DB, error) {
	logLevel := logger.Warn
	if cfg.AppEnv == "local" {
		logLevel = logger.Info
	}

	db, err := gorm.Open(mysql.Open(cfg.MySQLDSN), &gorm.Config{
		Logger:  logger.Default.LogMode(logLevel),
		NowFunc: func() time.Time { return time.Now().UTC() },
	})
	if err != nil {
		return nil, fmt.Errorf("connect mysql: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get sql db: %w", err)
	}

	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetMaxOpenConns(20)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)

	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("ping mysql: %w", err)
	}

	return db, nil
}

func newRedis(cfg config.Config) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
	})

	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("ping redis: %w", err)
	}

	return client, nil
}

func corsMiddleware(allowOrigins []string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(allowOrigins))
	for _, origin := range allowOrigins {
		allowed[strings.TrimSpace(origin)] = struct{}{}
	}

	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		if origin != "" {
			if _, ok := allowed[origin]; ok {
				c.Header("Access-Control-Allow-Origin", origin)
				c.Header("Vary", "Origin")
				c.Header("Access-Control-Allow-Credentials", "true")
				c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type")
				c.Header("Access-Control-Expose-Headers", "Content-Disposition")
				c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			}
		}

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

func normalizeLogger(loggerInstance *log.Logger) *log.Logger {
	if loggerInstance != nil {
		return loggerInstance
	}

	return log.New(os.Stdout, "[proman] ", log.Ldate|log.Ltime|log.Lmicroseconds|log.LUTC)
}

package integration

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"proman/server/internal/handler"
	"proman/server/internal/middleware"
	"proman/server/internal/pkg/markdownpreview"
	"proman/server/internal/pkg/ratelimit"
	"proman/server/internal/repository"
	"proman/server/internal/service"
)

func newTestRouter(db *gorm.DB, redisClient *redis.Client, jwtSecret string) *gin.Engine {
	gin.SetMode(gin.TestMode)

	userRepo := repository.NewUserRepository(db)
	projectRepo := repository.NewProjectRepository(db)
	announcementRepo := repository.NewAnnouncementRepository(db)
	versionRepo := repository.NewVersionRepository(db)
	changelogRepo := repository.NewChangelogRepository(db)

	authService := service.NewAuthService(userRepo, jwtSecret, 12*time.Hour)
	projectService := service.NewProjectService(projectRepo)
	announcementService := service.NewAnnouncementService(projectRepo, announcementRepo)
	versionService := service.NewVersionServiceWithCompare(projectRepo, versionRepo, changelogRepo)
	changelogService := service.NewChangelogService(changelogRepo, versionRepo)
	changelogExportService := service.NewChangelogExportService(projectRepo, versionRepo, changelogRepo)
	publicService := service.NewPublicService(projectRepo, versionRepo, changelogRepo, announcementRepo)

	limiter := ratelimit.NewRedisLimiter(redisClient)

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(middleware.ErrorHandler())

	authHandler := handler.NewAuthHandler(authService)
	markdownHandler := handler.NewMarkdownHandler(markdownpreview.NewRenderer())
	projectHandler := handler.NewProjectHandler(projectService)
	announcementHandler := handler.NewAnnouncementHandler(announcementService)
	versionHandler := handler.NewVersionHandler(versionService)
	changelogHandler := handler.NewChangelogHandler(changelogService)
	publicHandler := handler.NewPublicHandler(publicService)
	changelogExportHandler := handler.NewChangelogExportHandler(changelogExportService)

	router.POST("/api/auth/login", middleware.LoginRateLimit(limiter), authHandler.Login)

	api := router.Group("/api")
	api.Use(middleware.JWTAuth(jwtSecret))
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

	return router
}

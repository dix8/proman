package app

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

var reservedFrontendPrefixes = []string{
	"/api",
	"/v1",
	"/healthz",
}

func registerFrontendRoutes(router *gin.Engine, loggerInstance *log.Logger) {
	distDir, ok := findFrontendDistDir()
	if !ok {
		loggerInstance.Println("frontend dist not found, skip static file serving")
		return
	}

	indexFile := filepath.Join(distDir, "index.html")
	if _, err := os.Stat(indexFile); err != nil {
		loggerInstance.Printf("frontend index not found at %s, skip static file serving", indexFile)
		return
	}

	loggerInstance.Printf("frontend static files enabled from %s", distDir)

	router.GET("/", func(c *gin.Context) {
		c.File(indexFile)
	})

	router.NoRoute(func(c *gin.Context) {
		if c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead {
			c.Status(http.StatusNotFound)
			return
		}

		requestPath := c.Request.URL.Path
		for _, prefix := range reservedFrontendPrefixes {
			if requestPath == prefix || strings.HasPrefix(requestPath, prefix+"/") {
				c.Status(http.StatusNotFound)
				return
			}
		}

		if filePath, ok := resolveFrontendFilePath(distDir, requestPath); ok {
			if filePath == "" {
				c.Status(http.StatusNotFound)
				return
			}
			c.File(filePath)
			return
		}

		c.File(indexFile)
	})
}

func findFrontendDistDir() (string, bool) {
	candidates := []string{
		filepath.Join("web", "dist"),
		filepath.Join("..", "web", "dist"),
	}

	for _, candidate := range candidates {
		info, err := os.Stat(candidate)
		if err == nil && info.IsDir() {
			return candidate, true
		}
	}

	return "", false
}

func resolveFrontendFilePath(distDir, requestPath string) (string, bool) {
	cleaned := filepath.Clean("/" + requestPath)
	relativePath := strings.TrimPrefix(cleaned, "/")
	if relativePath == "" || relativePath == "." {
		return "", false
	}

	fullPath := filepath.Join(distDir, relativePath)
	rel, err := filepath.Rel(distDir, fullPath)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", false
	}

	info, err := os.Stat(fullPath)
	if err == nil && !info.IsDir() {
		return fullPath, true
	}

	if filepath.Ext(relativePath) != "" {
		return "", true
	}

	return "", false
}

package testutil

import (
	"bufio"
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	mysqlDriver "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"proman/server/internal/model"
	"proman/server/internal/pkg/migrate"
	"proman/server/internal/pkg/password"
)

const defaultMySQLDSN = "root:root@tcp(127.0.0.1:3306)/proman?charset=utf8mb4&parseTime=True&loc=UTC"

var (
	bootstrapOnce sync.Once
	bootstrapErr  error
	testDSN       string
	fixtureSeq    uint64
)

func OpenMySQL(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := resolveTestDSN(t)
	bootstrapOnce.Do(func() {
		testDSN = dsn
		bootstrapErr = bootstrapTestDatabase(dsn)
	})
	if bootstrapErr != nil {
		t.Fatalf("bootstrap test database: %v", bootstrapErr)
	}
	if testDSN != dsn {
		t.Fatalf("test DSN mismatch: bootstrap=%s current=%s", testDSN, dsn)
	}

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger:  logger.Default.LogMode(logger.Silent),
		NowFunc: func() time.Time { return time.Now().UTC() },
	})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql db: %v", err)
	}
	t.Cleanup(func() {
		_ = sqlDB.Close()
	})

	ResetTables(t, db)
	return db
}

func ResetTables(t *testing.T, db *gorm.DB) {
	t.Helper()

	statements := []string{
		"SET FOREIGN_KEY_CHECKS = 0",
		"TRUNCATE TABLE changelogs",
		"TRUNCATE TABLE versions",
		"TRUNCATE TABLE announcements",
		"TRUNCATE TABLE projects",
		"TRUNCATE TABLE users",
		"SET FOREIGN_KEY_CHECKS = 1",
	}
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatalf("reset table with %q: %v", statement, err)
		}
	}
}

func CreateUser(t *testing.T, db *gorm.DB, username, rawPassword string) *model.User {
	t.Helper()

	hashed, err := password.Hash(rawPassword)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	user := &model.User{
		Username:     username,
		PasswordHash: hashed,
	}
	if err := db.WithContext(context.Background()).Create(user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	return user
}

func CreateProject(t *testing.T, db *gorm.DB, userID uint64, name string) *model.Project {
	t.Helper()

	project := &model.Project{
		UserID:         userID,
		Name:           name,
		Description:    "test project",
		APITokenHash:   fmt.Sprintf("token-hash-%d", nextFixtureSeq()),
		TokenUpdatedAt: time.Now().UTC(),
	}
	if err := db.WithContext(context.Background()).Create(project).Error; err != nil {
		t.Fatalf("create project: %v", err)
	}
	return project
}

func CreateVersion(t *testing.T, db *gorm.DB, projectID uint64, major, minor, patch uint, status string) *model.Version {
	t.Helper()

	version := &model.Version{
		ProjectID: projectID,
		Major:     major,
		Minor:     minor,
		Patch:     patch,
		Status:    status,
	}
	if status == model.VersionStatusPublished {
		publishedAt := time.Now().UTC()
		version.PublishedAt = &publishedAt
	}

	if err := db.WithContext(context.Background()).Create(version).Error; err != nil {
		t.Fatalf("create version: %v", err)
	}
	return version
}

func CreateChangelog(t *testing.T, db *gorm.DB, versionID uint64, changelogType, content string, sortOrder uint) *model.Changelog {
	t.Helper()

	changelog := &model.Changelog{
		VersionID: versionID,
		Type:      changelogType,
		Content:   content,
		SortOrder: sortOrder,
	}
	if err := db.WithContext(context.Background()).Create(changelog).Error; err != nil {
		t.Fatalf("create changelog: %v", err)
	}
	return changelog
}

func CreateAnnouncement(t *testing.T, db *gorm.DB, projectID uint64, title, content, status string, isPinned bool) *model.Announcement {
	t.Helper()

	announcement := &model.Announcement{
		ProjectID: projectID,
		Title:     title,
		Content:   content,
		Status:    status,
		IsPinned:  isPinned,
	}
	if status == model.AnnouncementStatusPublished {
		publishedAt := time.Now().UTC()
		announcement.PublishedAt = &publishedAt
	}

	if err := db.WithContext(context.Background()).Create(announcement).Error; err != nil {
		t.Fatalf("create announcement: %v", err)
	}
	return announcement
}

func resolveTestDSN(t *testing.T) string {
	t.Helper()

	raw := strings.TrimSpace(os.Getenv("PROMAN_TEST_MYSQL_DSN"))
	if raw == "" {
		raw = strings.TrimSpace(os.Getenv("MYSQL_DSN"))
	}
	if raw == "" {
		raw = readMySQLDSNFromEnvFile()
	}
	if raw == "" {
		raw = defaultMySQLDSN
	}

	cfg, err := mysqlDriver.ParseDSN(raw)
	if err != nil {
		t.Fatalf("parse mysql dsn: %v", err)
	}
	if cfg.DBName == "" {
		cfg.DBName = "proman"
	}
	if !strings.HasSuffix(cfg.DBName, "_test") {
		cfg.DBName = cfg.DBName + "_test"
	}
	cfg.DBName = fmt.Sprintf("%s_p%d", cfg.DBName, os.Getpid())
	if !cfg.ParseTime {
		cfg.ParseTime = true
	}
	if cfg.Params == nil {
		cfg.Params = map[string]string{}
	}
	if _, ok := cfg.Params["charset"]; !ok {
		cfg.Params["charset"] = "utf8mb4"
	}
	if _, ok := cfg.Params["loc"]; !ok {
		cfg.Params["loc"] = "UTC"
	}

	return cfg.FormatDSN()
}

func bootstrapTestDatabase(dsn string) error {
	cfg, err := mysqlDriver.ParseDSN(dsn)
	if err != nil {
		return err
	}

	adminCfg := *cfg
	testDBName := cfg.DBName
	adminCfg.DBName = ""

	sqlDB, err := sql.Open("mysql", adminCfg.FormatDSN())
	if err != nil {
		return err
	}
	defer sqlDB.Close()

	if err := sqlDB.Ping(); err != nil {
		return err
	}

	createDatabaseSQL := fmt.Sprintf(
		"CREATE DATABASE IF NOT EXISTS `%s` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci",
		testDBName,
	)
	if _, err := sqlDB.Exec(createDatabaseSQL); err != nil {
		return err
	}

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger:  logger.Default.LogMode(logger.Silent),
		NowFunc: func() time.Time { return time.Now().UTC() },
	})
	if err != nil {
		return err
	}

	return migrate.Run(db, migrationsDir())
}

func migrationsDir() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "..", "..", "migrations")
}

func serverDir() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "..", "..")
}

func readMySQLDSNFromEnvFile() string {
	candidates := []string{
		filepath.Join(serverDir(), ".env.local"),
		filepath.Join(serverDir(), ".env.example"),
	}

	for _, candidate := range candidates {
		file, err := os.Open(candidate)
		if err != nil {
			continue
		}

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			parts := strings.SplitN(line, "=", 2)
			if len(parts) != 2 {
				continue
			}
			if strings.TrimSpace(parts[0]) == "MYSQL_DSN" {
				_ = file.Close()
				return strings.TrimSpace(parts[1])
			}
		}
		_ = file.Close()
	}

	return ""
}

func nextFixtureSeq() uint64 {
	return atomic.AddUint64(&fixtureSeq, 1)
}

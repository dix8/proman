package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"gorm.io/gorm"

	"proman/server/internal/model"
	"proman/server/internal/pkg/apperror"
	"proman/server/internal/repository"
)

type ChangelogExportService struct {
	projectRepo   *repository.ProjectRepository
	versionRepo   *repository.VersionRepository
	changelogRepo *repository.ChangelogRepository
}

type ExportFile struct {
	Filename           string
	ContentType        string
	ContentDisposition string
	Content            []byte
}

type exportVersionBundle struct {
	Version    model.Version
	Changelogs []model.Changelog
}

type exportJSONFile struct {
	Project    exportJSONProject   `json:"project"`
	ExportedAt string              `json:"exported_at"`
	Versions   []exportJSONVersion `json:"versions"`
}

type exportJSONProject struct {
	ID   uint64 `json:"id"`
	Name string `json:"name"`
}

type exportJSONVersion struct {
	ID          uint64                `json:"id"`
	Version     string                `json:"version"`
	Status      string                `json:"status"`
	PublishedAt *string               `json:"published_at"`
	Changelogs  []exportJSONChangelog `json:"changelogs"`
}

type exportJSONChangelog struct {
	ID        uint64 `json:"id"`
	VersionID uint64 `json:"version_id"`
	Type      string `json:"type"`
	Content   string `json:"content"`
	SortOrder uint   `json:"sort_order"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

func NewChangelogExportService(projectRepo *repository.ProjectRepository, versionRepo *repository.VersionRepository, changelogRepo *repository.ChangelogRepository) *ChangelogExportService {
	return &ChangelogExportService{
		projectRepo:   projectRepo,
		versionRepo:   versionRepo,
		changelogRepo: changelogRepo,
	}
}

func (s *ChangelogExportService) Export(ctx context.Context, userID, projectID uint64, format string, versionID *uint64) (*ExportFile, error) {
	if userID == 0 || projectID == 0 {
		return nil, apperror.New(http.StatusBadRequest, 40001, "参数错误")
	}

	format = strings.TrimSpace(format)
	if format != "markdown" && format != "json" {
		return nil, apperror.New(http.StatusBadRequest, 40001, "参数错误")
	}

	project, err := s.projectRepo.FindByIDAndUserID(ctx, projectID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.New(http.StatusNotFound, 40401, "项目不存在")
		}
		return nil, apperror.Internal(err)
	}

	var bundles []exportVersionBundle
	var exportName string

	if versionID != nil {
		version, err := s.versionRepo.FindByIDAndProjectIDAndUserID(ctx, *versionID, projectID, userID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, apperror.New(http.StatusNotFound, 40402, "版本不存在")
			}
			return nil, apperror.Internal(err)
		}

		changelogs, err := s.changelogRepo.ListByVersionID(ctx, version.ID)
		if err != nil {
			return nil, apperror.Internal(err)
		}

		bundles = append(bundles, exportVersionBundle{
			Version:    *version,
			Changelogs: changelogs,
		})
		exportName = fmt.Sprintf("%s-v%s-changelogs", sanitizeFilename(project.Name), version.VersionString())
	} else {
		versions, err := s.versionRepo.ListByProjectIDAndUserIDForExport(ctx, projectID, userID)
		if err != nil {
			return nil, apperror.Internal(err)
		}

		for _, version := range versions {
			changelogs, err := s.changelogRepo.ListByVersionID(ctx, version.ID)
			if err != nil {
				return nil, apperror.Internal(err)
			}
			bundles = append(bundles, exportVersionBundle{
				Version:    version,
				Changelogs: changelogs,
			})
		}
		exportName = fmt.Sprintf("%s-all-changelogs", sanitizeFilename(project.Name))
	}

	if format == "json" {
		content, err := s.buildJSON(project.ID, project.Name, bundles)
		if err != nil {
			return nil, apperror.Internal(err)
		}
		filename := exportName + ".json"
		return &ExportFile{
			Filename:           filename,
			ContentType:        "application/json; charset=utf-8",
			ContentDisposition: buildContentDisposition(filename),
			Content:            content,
		}, nil
	}

	content, err := s.buildMarkdown(project.Name, bundles)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	filename := exportName + ".md"
	return &ExportFile{
		Filename:           filename,
		ContentType:        "text/markdown; charset=utf-8",
		ContentDisposition: buildContentDisposition(filename),
		Content:            content,
	}, nil
}

func (s *ChangelogExportService) buildJSON(projectID uint64, projectName string, bundles []exportVersionBundle) ([]byte, error) {
	payload := exportJSONFile{
		Project: exportJSONProject{
			ID:   projectID,
			Name: projectName,
		},
		ExportedAt: time.Now().UTC().Format("2006-01-02T15:04:05Z"),
		Versions:   make([]exportJSONVersion, 0, len(bundles)),
	}

	for _, bundle := range bundles {
		var publishedAt *string
		if bundle.Version.PublishedAt != nil {
			value := bundle.Version.PublishedAt.UTC().Format("2006-01-02T15:04:05Z")
			publishedAt = &value
		}

		item := exportJSONVersion{
			ID:          bundle.Version.ID,
			Version:     bundle.Version.VersionString(),
			Status:      bundle.Version.Status,
			PublishedAt: publishedAt,
			Changelogs:  make([]exportJSONChangelog, 0, len(bundle.Changelogs)),
		}

		for _, changelog := range bundle.Changelogs {
			item.Changelogs = append(item.Changelogs, exportJSONChangelog{
				ID:        changelog.ID,
				VersionID: changelog.VersionID,
				Type:      changelog.Type,
				Content:   changelog.Content,
				SortOrder: changelog.SortOrder,
				CreatedAt: changelog.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
				UpdatedAt: changelog.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z"),
			})
		}

		payload.Versions = append(payload.Versions, item)
	}

	return json.MarshalIndent(payload, "", "  ")
}

func (s *ChangelogExportService) buildMarkdown(projectName string, bundles []exportVersionBundle) ([]byte, error) {
	var builder strings.Builder
	builder.WriteString("# ")
	builder.WriteString(projectName)
	builder.WriteString(" 更新日志\n")

	for _, bundle := range bundles {
		builder.WriteString("\n## ")
		builder.WriteString(bundle.Version.VersionString())
		builder.WriteString("\n\n状态：")
		builder.WriteString(bundle.Version.Status)
		builder.WriteString("\n发布时间：")
		if bundle.Version.PublishedAt != nil {
			builder.WriteString(bundle.Version.PublishedAt.UTC().Format("2006-01-02T15:04:05Z"))
		} else {
			builder.WriteString("-")
		}
		builder.WriteString("\n")

		for _, group := range exportMarkdownGroups(bundle.Changelogs) {
			if len(group.items) == 0 {
				continue
			}
			builder.WriteString("\n### ")
			builder.WriteString(group.title)
			builder.WriteString("\n\n")
			for _, item := range group.items {
				builder.WriteString(formatMarkdownBullet(item.Content))
				builder.WriteString("\n")
			}
		}
	}

	return []byte(builder.String()), nil
}

type markdownGroup struct {
	title string
	items []model.Changelog
}

func exportMarkdownGroups(changelogs []model.Changelog) []markdownGroup {
	groups := []markdownGroup{
		{title: "Added"},
		{title: "Changed"},
		{title: "Fixed"},
		{title: "Improved"},
		{title: "Deprecated"},
		{title: "Removed"},
	}

	for _, changelog := range changelogs {
		switch changelog.Type {
		case model.ChangelogTypeAdded:
			groups[0].items = append(groups[0].items, changelog)
		case model.ChangelogTypeChanged:
			groups[1].items = append(groups[1].items, changelog)
		case model.ChangelogTypeFixed:
			groups[2].items = append(groups[2].items, changelog)
		case model.ChangelogTypeImproved:
			groups[3].items = append(groups[3].items, changelog)
		case model.ChangelogTypeDeprecated:
			groups[4].items = append(groups[4].items, changelog)
		case model.ChangelogTypeRemoved:
			groups[5].items = append(groups[5].items, changelog)
		}
	}

	return groups
}

func formatMarkdownBullet(content string) string {
	lines := strings.Split(content, "\n")
	if len(lines) == 0 {
		return "- "
	}
	var builder strings.Builder
	builder.WriteString("- ")
	builder.WriteString(lines[0])
	for _, line := range lines[1:] {
		builder.WriteString("\n  ")
		builder.WriteString(line)
	}
	return builder.String()
}

func sanitizeFilename(name string) string {
	replacer := strings.NewReplacer(
		" ", "-",
		"<", "-",
		">", "-",
		":", "-",
		"\"", "-",
		"/", "-",
		"\\", "-",
		"|", "-",
		"?", "-",
		"*", "-",
	)
	return replacer.Replace(name)
}

func buildContentDisposition(filename string) string {
	escaped := url.PathEscape(filename)
	fallback := asciiFallbackFilename(filename)
	return fmt.Sprintf("attachment; filename=\"%s\"; filename*=UTF-8''%s", fallback, escaped)
}

func asciiFallbackFilename(filename string) string {
	lower := strings.ToLower(filename)
	if strings.HasSuffix(lower, ".md") {
		return "project-export.md"
	}
	if strings.HasSuffix(lower, ".json") {
		return "project-export.json"
	}
	return "project-export"
}

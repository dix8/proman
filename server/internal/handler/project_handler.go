package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"proman/server/internal/middleware"
	"proman/server/internal/model"
	"proman/server/internal/pkg/apperror"
	"proman/server/internal/pkg/response"
	"proman/server/internal/service"
)

type ProjectHandler struct {
	projectService *service.ProjectService
}

type createProjectRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type updateProjectRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type projectData struct {
	ID             uint64 `json:"id"`
	Name           string `json:"name"`
	Description    string `json:"description"`
	TokenUpdatedAt string `json:"token_updated_at"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
}

func NewProjectHandler(projectService *service.ProjectService) *ProjectHandler {
	return &ProjectHandler{projectService: projectService}
}

func (h *ProjectHandler) Create(c *gin.Context) {
	var req createProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(apperror.New(http.StatusBadRequest, 40001, "参数错误"))
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	req.Description = strings.TrimSpace(req.Description)

	result, err := h.projectService.Create(c.Request.Context(), service.CreateProjectInput{
		UserID:      middleware.CurrentUserID(c),
		Name:        req.Name,
		Description: req.Description,
	})
	if err != nil {
		_ = c.Error(err)
		return
	}

	response.Success(c, gin.H{
		"project":       newProjectData(result.Project),
		"project_token": result.ProjectToken,
	})
}

func (h *ProjectHandler) List(c *gin.Context) {
	page, err := parseIntQueryWithDefault(c.Query("page"), 1)
	if err != nil {
		_ = c.Error(apperror.New(http.StatusBadRequest, 40001, "参数错误"))
		return
	}

	pageSize, err := parseIntQueryWithDefault(c.Query("page_size"), 20)
	if err != nil {
		_ = c.Error(apperror.New(http.StatusBadRequest, 40001, "参数错误"))
		return
	}

	result, err := h.projectService.List(c.Request.Context(), service.ListProjectsInput{
		UserID:   middleware.CurrentUserID(c),
		Page:     page,
		PageSize: pageSize,
		Keyword:  strings.TrimSpace(c.Query("keyword")),
	})
	if err != nil {
		_ = c.Error(err)
		return
	}

	list := make([]projectData, 0, len(result.List))
	for _, project := range result.List {
		projectCopy := project
		list = append(list, newProjectData(&projectCopy))
	}

	response.Success(c, gin.H{
		"list":      list,
		"total":     result.Total,
		"page":      result.Page,
		"page_size": result.PageSize,
	})
}

func (h *ProjectHandler) Get(c *gin.Context) {
	projectID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || projectID == 0 {
		_ = c.Error(apperror.New(http.StatusBadRequest, 40001, "参数错误"))
		return
	}

	project, err := h.projectService.GetByID(c.Request.Context(), middleware.CurrentUserID(c), projectID)
	if err != nil {
		_ = c.Error(err)
		return
	}

	response.Success(c, newProjectData(project))
}

func (h *ProjectHandler) Update(c *gin.Context) {
	projectID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || projectID == 0 {
		_ = c.Error(apperror.New(http.StatusBadRequest, 40001, "参数错误"))
		return
	}

	var req updateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(apperror.New(http.StatusBadRequest, 40001, "参数错误"))
		return
	}

	project, err := h.projectService.Update(c.Request.Context(), service.UpdateProjectInput{
		UserID:      middleware.CurrentUserID(c),
		ProjectID:   projectID,
		Name:        strings.TrimSpace(req.Name),
		Description: strings.TrimSpace(req.Description),
	})
	if err != nil {
		_ = c.Error(err)
		return
	}

	response.Success(c, newProjectData(project))
}

func (h *ProjectHandler) RefreshToken(c *gin.Context) {
	projectID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || projectID == 0 {
		_ = c.Error(apperror.New(http.StatusBadRequest, 40001, "参数错误"))
		return
	}

	result, err := h.projectService.RefreshToken(c.Request.Context(), service.RefreshProjectTokenInput{
		UserID:    middleware.CurrentUserID(c),
		ProjectID: projectID,
	})
	if err != nil {
		_ = c.Error(err)
		return
	}

	response.Success(c, gin.H{
		"project_token":    result.ProjectToken,
		"token_updated_at": result.Project.TokenUpdatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	})
}

func (h *ProjectHandler) Delete(c *gin.Context) {
	projectID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || projectID == 0 {
		_ = c.Error(apperror.New(http.StatusBadRequest, 40001, "参数错误"))
		return
	}

	if err := h.projectService.Delete(c.Request.Context(), middleware.CurrentUserID(c), projectID); err != nil {
		_ = c.Error(err)
		return
	}

	response.Success(c, gin.H{})
}

func newProjectData(project *model.Project) projectData {
	return projectData{
		ID:             project.ID,
		Name:           project.Name,
		Description:    project.Description,
		TokenUpdatedAt: project.TokenUpdatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		CreatedAt:      project.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		UpdatedAt:      project.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
}

func parseIntQueryWithDefault(raw string, defaultValue int) (int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return defaultValue, nil
	}
	return strconv.Atoi(raw)
}

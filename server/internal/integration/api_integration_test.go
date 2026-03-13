package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"proman/server/internal/pkg/jwtutil"
	"proman/server/internal/testutil"
)

const integrationJWTSecret = "integration-secret"

type testEnv struct {
	router *gin.Engine
	db     *gorm.DB
	redis  *redis.Client
}

type envelope struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

type loginData struct {
	Token     string `json:"token"`
	ExpiresAt string `json:"expires_at"`
}

type projectData struct {
	ID          uint64 `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type createProjectData struct {
	Project      projectData `json:"project"`
	ProjectToken string      `json:"project_token"`
}

type listProjectsData struct {
	List []projectData `json:"list"`
}

type refreshProjectTokenData struct {
	ProjectToken string `json:"project_token"`
}

type versionData struct {
	ID          uint64  `json:"id"`
	Version     string  `json:"version"`
	Status      string  `json:"status"`
	PublishedAt *string `json:"published_at"`
}

type changelogData struct {
	ID        uint64 `json:"id"`
	VersionID uint64 `json:"version_id"`
	Type      string `json:"type"`
	Content   string `json:"content"`
	SortOrder uint   `json:"sort_order"`
}

type listChangelogsData struct {
	List []changelogData `json:"list"`
}

type announcementData struct {
	ID          uint64  `json:"id"`
	ProjectID   uint64  `json:"project_id"`
	Title       string  `json:"title"`
	Content     string  `json:"content"`
	IsPinned    bool    `json:"is_pinned"`
	Status      string  `json:"status"`
	PublishedAt *string `json:"published_at"`
}

type listAnnouncementsData struct {
	List []announcementData `json:"list"`
}

type compareVersionItem struct {
	ID      uint64 `json:"id"`
	Version string `json:"version"`
	Status  string `json:"status"`
}

type compareChangelogItem struct {
	ID      uint64 `json:"id"`
	Version string `json:"version"`
	Type    string `json:"type"`
	Content string `json:"content"`
}

type compareChangelogGroups struct {
	Added      []compareChangelogItem `json:"added"`
	Changed    []compareChangelogItem `json:"changed"`
	Fixed      []compareChangelogItem `json:"fixed"`
	Improved   []compareChangelogItem `json:"improved"`
	Deprecated []compareChangelogItem `json:"deprecated"`
	Removed    []compareChangelogItem `json:"removed"`
}

type compareVersionsData struct {
	FromVersion compareVersionItem     `json:"from_version"`
	ToVersion   compareVersionItem     `json:"to_version"`
	Versions    []compareVersionItem   `json:"versions"`
	Changelogs  compareChangelogGroups `json:"changelogs"`
}

type publicProjectData struct {
	Name string `json:"name"`
}

type publicVersionData struct {
	Version string `json:"version"`
	Status  string `json:"status"`
}

type publicVersionsData struct {
	List []publicVersionData `json:"list"`
}

type publicVersionChangelogsData struct {
	Version struct {
		Version string `json:"version"`
		Status  string `json:"status"`
	} `json:"version"`
	Changelogs []struct {
		Type    string `json:"type"`
		Content string `json:"content"`
	} `json:"changelogs"`
}

type publicAnnouncementsData struct {
	List []struct {
		Title  string `json:"title"`
		Status string `json:"status"`
	} `json:"list"`
}

func TestAuthAndJWTFlow(t *testing.T) {
	env := newTestEnv(t)
	testutil.CreateUser(t, env.db, "admin", "admin123456")

	loginResponse := jsonRequest(t, env.router, http.MethodPost, "/api/auth/login", map[string]any{
		"username": "admin",
		"password": "admin123456",
	}, "")
	assertStatus(t, loginResponse, http.StatusOK)
	loginData := decodeEnvelopeData[loginData](t, loginResponse)
	if loginData.Token == "" {
		t.Fatalf("expected login token to be non-empty")
	}
	if _, err := jwtutil.ParseToken(loginData.Token, integrationJWTSecret); err != nil {
		t.Fatalf("parse issued login token: %v", err)
	}

	wrongPasswordResponse := jsonRequest(t, env.router, http.MethodPost, "/api/auth/login", map[string]any{
		"username": "admin",
		"password": "wrong-password",
	}, "")
	assertAppErrorResponse(t, wrongPasswordResponse, http.StatusUnauthorized, 40101)

	noJWTResponse := jsonRequest(t, env.router, http.MethodGet, "/api/projects?page=1&page_size=20", nil, "")
	assertAppErrorResponse(t, noJWTResponse, http.StatusUnauthorized, 40102)

	invalidJWTResponse := jsonRequest(t, env.router, http.MethodGet, "/api/projects?page=1&page_size=20", nil, "invalid-jwt")
	assertAppErrorResponse(t, invalidJWTResponse, http.StatusUnauthorized, 40102)
}

func TestProjectAndProjectTokenFlow(t *testing.T) {
	env := newTestEnv(t)
	testutil.CreateUser(t, env.db, "admin", "admin123456")
	jwtToken := loginAndGetJWT(t, env.router)

	createProjectResult := createProjectViaAPI(t, env.router, jwtToken, "HTTP Project", "integration flow")
	projectID := createProjectResult.Project.ID
	projectToken := createProjectResult.ProjectToken

	listResponse := jsonRequest(t, env.router, http.MethodGet, "/api/projects?page=1&page_size=20", nil, jwtToken)
	assertStatus(t, listResponse, http.StatusOK)
	listData := decodeEnvelopeData[listProjectsData](t, listResponse)
	if len(listData.List) != 1 || listData.List[0].ID != projectID {
		t.Fatalf("expected project list to contain created project, got %+v", listData.List)
	}

	detailResponse := jsonRequest(t, env.router, http.MethodGet, fmt.Sprintf("/api/projects/%d", projectID), nil, jwtToken)
	assertStatus(t, detailResponse, http.StatusOK)
	detailData := decodeEnvelopeData[projectData](t, detailResponse)
	if detailData.Name != "HTTP Project" {
		t.Fatalf("expected project detail name to be updated, got %+v", detailData)
	}

	updateResponse := jsonRequest(t, env.router, http.MethodPut, fmt.Sprintf("/api/projects/%d", projectID), map[string]any{
		"name":        "HTTP Project Updated",
		"description": "updated description",
	}, jwtToken)
	assertStatus(t, updateResponse, http.StatusOK)
	updatedProject := decodeEnvelopeData[projectData](t, updateResponse)
	if updatedProject.Name != "HTTP Project Updated" {
		t.Fatalf("expected updated project name, got %+v", updatedProject)
	}

	publicProjectResponse := bearerRequest(t, env.router, http.MethodGet, "/v1/project", nil, projectToken)
	assertStatus(t, publicProjectResponse, http.StatusOK)
	publicProject := decodeEnvelopeData[publicProjectData](t, publicProjectResponse)
	if publicProject.Name != "HTTP Project Updated" {
		t.Fatalf("expected /v1/project to expose updated project name, got %+v", publicProject)
	}

	refreshTokenResponse := jsonRequest(t, env.router, http.MethodPost, fmt.Sprintf("/api/projects/%d/token/refresh", projectID), map[string]any{}, jwtToken)
	assertStatus(t, refreshTokenResponse, http.StatusOK)
	refreshTokenData := decodeEnvelopeData[refreshProjectTokenData](t, refreshTokenResponse)
	if refreshTokenData.ProjectToken == "" || refreshTokenData.ProjectToken == projectToken {
		t.Fatalf("expected refresh to issue a new project token, got %+v", refreshTokenData)
	}

	oldTokenPublicResponse := bearerRequest(t, env.router, http.MethodGet, "/v1/project", nil, projectToken)
	assertAppErrorResponse(t, oldTokenPublicResponse, http.StatusUnauthorized, 40103)

	newTokenPublicResponse := bearerRequest(t, env.router, http.MethodGet, "/v1/project", nil, refreshTokenData.ProjectToken)
	assertStatus(t, newTokenPublicResponse, http.StatusOK)

	deleteResponse := jsonRequest(t, env.router, http.MethodDelete, fmt.Sprintf("/api/projects/%d", projectID), nil, jwtToken)
	assertStatus(t, deleteResponse, http.StatusOK)

	deletedDetailResponse := jsonRequest(t, env.router, http.MethodGet, fmt.Sprintf("/api/projects/%d", projectID), nil, jwtToken)
	assertAppErrorResponse(t, deletedDetailResponse, http.StatusNotFound, 40401)

	deletedTokenPublicResponse := bearerRequest(t, env.router, http.MethodGet, "/v1/project", nil, refreshTokenData.ProjectToken)
	assertAppErrorResponse(t, deletedTokenPublicResponse, http.StatusUnauthorized, 40103)
}

func TestVersionAndChangelogFlow(t *testing.T) {
	env := newTestEnv(t)
	testutil.CreateUser(t, env.db, "admin", "admin123456")
	jwtToken := loginAndGetJWT(t, env.router)
	project := createProjectViaAPI(t, env.router, jwtToken, "Version Flow Project", "version integration flow")

	version := createVersionViaAPI(t, env.router, jwtToken, project.Project.ID, 1, 0, 0)
	if version.Status != "draft" {
		t.Fatalf("expected created version to be draft, got %+v", version)
	}

	firstChangelog := createChangelogViaAPI(t, env.router, jwtToken, version.ID, "added", "first changelog")
	secondChangelog := createChangelogViaAPI(t, env.router, jwtToken, version.ID, "fixed", "second changelog")

	updateChangelogResponse := jsonRequest(t, env.router, http.MethodPut, fmt.Sprintf("/api/changelogs/%d", secondChangelog.ID), map[string]any{
		"type":    "fixed",
		"content": "second changelog updated",
	}, jwtToken)
	assertStatus(t, updateChangelogResponse, http.StatusOK)

	reorderResponse := jsonRequest(t, env.router, http.MethodPut, fmt.Sprintf("/api/versions/%d/changelogs/reorder", version.ID), map[string]any{
		"items": []map[string]any{
			{"id": secondChangelog.ID, "sort_order": 1},
			{"id": firstChangelog.ID, "sort_order": 2},
		},
	}, jwtToken)
	assertStatus(t, reorderResponse, http.StatusOK)

	listChangelogsResponse := jsonRequest(t, env.router, http.MethodGet, fmt.Sprintf("/api/versions/%d/changelogs?page=1&page_size=20", version.ID), nil, jwtToken)
	assertStatus(t, listChangelogsResponse, http.StatusOK)
	changelogList := decodeEnvelopeData[listChangelogsData](t, listChangelogsResponse)
	if len(changelogList.List) != 2 || changelogList.List[0].ID != secondChangelog.ID {
		t.Fatalf("expected reordered changelog list to start with second changelog, got %+v", changelogList.List)
	}

	deleteChangelogResponse := jsonRequest(t, env.router, http.MethodDelete, fmt.Sprintf("/api/changelogs/%d", firstChangelog.ID), nil, jwtToken)
	assertStatus(t, deleteChangelogResponse, http.StatusOK)

	publishResponse := jsonRequest(t, env.router, http.MethodPut, fmt.Sprintf("/api/versions/%d/publish", version.ID), map[string]any{}, jwtToken)
	assertStatus(t, publishResponse, http.StatusOK)
	publishedVersion := decodeEnvelopeData[versionData](t, publishResponse)
	if publishedVersion.Status != "published" {
		t.Fatalf("expected published version status, got %+v", publishedVersion)
	}

	updatePublishedVersionResponse := jsonRequest(t, env.router, http.MethodPut, fmt.Sprintf("/api/versions/%d", version.ID), map[string]any{
		"major": 1,
		"minor": 0,
		"patch": 1,
	}, jwtToken)
	assertAppErrorResponse(t, updatePublishedVersionResponse, http.StatusConflict, 40902)

	deletePublishedVersionResponse := jsonRequest(t, env.router, http.MethodDelete, fmt.Sprintf("/api/versions/%d", version.ID), nil, jwtToken)
	assertAppErrorResponse(t, deletePublishedVersionResponse, http.StatusConflict, 40903)

	createPublishedChangelogResponse := jsonRequest(t, env.router, http.MethodPost, fmt.Sprintf("/api/versions/%d/changelogs", version.ID), map[string]any{
		"type":    "added",
		"content": "should fail on published version",
	}, jwtToken)
	assertAppErrorResponse(t, createPublishedChangelogResponse, http.StatusConflict, 40902)

	reorderPublishedChangelogResponse := jsonRequest(t, env.router, http.MethodPut, fmt.Sprintf("/api/versions/%d/changelogs/reorder", version.ID), map[string]any{
		"items": []map[string]any{
			{"id": secondChangelog.ID, "sort_order": 1},
		},
	}, jwtToken)
	assertAppErrorResponse(t, reorderPublishedChangelogResponse, http.StatusConflict, 40902)
}

func TestAnnouncementFlow(t *testing.T) {
	env := newTestEnv(t)
	testutil.CreateUser(t, env.db, "admin", "admin123456")
	jwtToken := loginAndGetJWT(t, env.router)
	project := createProjectViaAPI(t, env.router, jwtToken, "Announcement Flow Project", "announcement integration flow")

	createAnnouncementResponse := jsonRequest(t, env.router, http.MethodPost, fmt.Sprintf("/api/projects/%d/announcements", project.Project.ID), map[string]any{
		"title":     "draft announcement",
		"content":   "draft announcement content",
		"is_pinned": true,
	}, jwtToken)
	assertStatus(t, createAnnouncementResponse, http.StatusOK)
	announcement := decodeEnvelopeData[announcementData](t, createAnnouncementResponse)
	if announcement.Status != "draft" {
		t.Fatalf("expected created announcement to be draft, got %+v", announcement)
	}

	publishResponse := jsonRequest(t, env.router, http.MethodPut, fmt.Sprintf("/api/announcements/%d/publish", announcement.ID), map[string]any{}, jwtToken)
	assertStatus(t, publishResponse, http.StatusOK)
	publishedAnnouncement := decodeEnvelopeData[announcementData](t, publishResponse)
	if publishedAnnouncement.Status != "published" || publishedAnnouncement.PublishedAt == nil {
		t.Fatalf("expected announcement to be published with published_at, got %+v", publishedAnnouncement)
	}
	initialPublishedAt := *publishedAnnouncement.PublishedAt

	updatePublishedResponse := jsonRequest(t, env.router, http.MethodPut, fmt.Sprintf("/api/announcements/%d", announcement.ID), map[string]any{
		"title":     "published announcement updated",
		"content":   "published content updated",
		"is_pinned": false,
	}, jwtToken)
	assertStatus(t, updatePublishedResponse, http.StatusOK)
	updatedAnnouncement := decodeEnvelopeData[announcementData](t, updatePublishedResponse)
	if updatedAnnouncement.Status != "published" {
		t.Fatalf("expected published announcement to stay published after update, got %+v", updatedAnnouncement)
	}
	if updatedAnnouncement.PublishedAt == nil || *updatedAnnouncement.PublishedAt != initialPublishedAt {
		t.Fatalf("expected published_at to remain unchanged after published update, got %+v want %s", updatedAnnouncement, initialPublishedAt)
	}

	revokeResponse := jsonRequest(t, env.router, http.MethodPut, fmt.Sprintf("/api/announcements/%d/revoke", announcement.ID), map[string]any{}, jwtToken)
	assertStatus(t, revokeResponse, http.StatusOK)
	revokedAnnouncement := decodeEnvelopeData[announcementData](t, revokeResponse)
	if revokedAnnouncement.Status != "draft" || revokedAnnouncement.PublishedAt != nil {
		t.Fatalf("expected revoked announcement to be draft with nil published_at, got %+v", revokedAnnouncement)
	}
}

func TestComparePublicExportAndRateLimitFlow(t *testing.T) {
	env := newTestEnv(t)
	testutil.CreateUser(t, env.db, "admin", "admin123456")
	jwtToken := loginAndGetJWT(t, env.router)

	project := createProjectViaAPI(t, env.router, jwtToken, "Public Compare Export Project", "public integration flow")
	projectID := project.Project.ID
	projectToken := project.ProjectToken

	version100 := createVersionViaAPI(t, env.router, jwtToken, projectID, 1, 0, 0)
	createChangelogViaAPI(t, env.router, jwtToken, version100.ID, "added", "base feature")
	publishVersionViaAPI(t, env.router, jwtToken, version100.ID)

	version110 := createVersionViaAPI(t, env.router, jwtToken, projectID, 1, 1, 0)
	createChangelogViaAPI(t, env.router, jwtToken, version110.ID, "fixed", "reverse order bug")
	publishVersionViaAPI(t, env.router, jwtToken, version110.ID)

	draftVersion := createVersionViaAPI(t, env.router, jwtToken, projectID, 2, 0, 0)
	createChangelogViaAPI(t, env.router, jwtToken, draftVersion.ID, "changed", "draft change")

	createDraftAnnouncementResponse := jsonRequest(t, env.router, http.MethodPost, fmt.Sprintf("/api/projects/%d/announcements", projectID), map[string]any{
		"title":     "draft notice",
		"content":   "draft notice content",
		"is_pinned": false,
	}, jwtToken)
	assertStatus(t, createDraftAnnouncementResponse, http.StatusOK)
	draftAnnouncement := decodeEnvelopeData[announcementData](t, createDraftAnnouncementResponse)

	createPublishedAnnouncementResponse := jsonRequest(t, env.router, http.MethodPost, fmt.Sprintf("/api/projects/%d/announcements", projectID), map[string]any{
		"title":     "published notice",
		"content":   "published notice content",
		"is_pinned": true,
	}, jwtToken)
	assertStatus(t, createPublishedAnnouncementResponse, http.StatusOK)
	publishedAnnouncement := decodeEnvelopeData[announcementData](t, createPublishedAnnouncementResponse)
	publishAnnouncementResponse := jsonRequest(t, env.router, http.MethodPut, fmt.Sprintf("/api/announcements/%d/publish", publishedAnnouncement.ID), map[string]any{}, jwtToken)
	assertStatus(t, publishAnnouncementResponse, http.StatusOK)

	normalCompareResponse := jsonRequest(
		t,
		env.router,
		http.MethodGet,
		fmt.Sprintf("/api/projects/%d/versions/compare?from_version_id=%d&to_version_id=%d", projectID, version100.ID, version110.ID),
		nil,
		jwtToken,
	)
	assertStatus(t, normalCompareResponse, http.StatusOK)
	normalCompareData := decodeEnvelopeData[compareVersionsData](t, normalCompareResponse)
	if normalCompareData.FromVersion.Version != "1.0.0" || normalCompareData.ToVersion.Version != "1.1.0" {
		t.Fatalf("expected compare range 1.0.0 -> 1.1.0, got %+v", normalCompareData)
	}
	if len(normalCompareData.Changelogs.Added) != 1 || normalCompareData.Changelogs.Added[0].Content != "base feature" {
		t.Fatalf("expected compare added group to contain base feature, got %+v", normalCompareData.Changelogs.Added)
	}
	if len(normalCompareData.Changelogs.Fixed) != 1 || normalCompareData.Changelogs.Fixed[0].Content != "reverse order bug" {
		t.Fatalf("expected compare fixed group to contain reverse order bug, got %+v", normalCompareData.Changelogs.Fixed)
	}

	reversedCompareResponse := jsonRequest(
		t,
		env.router,
		http.MethodGet,
		fmt.Sprintf("/api/projects/%d/versions/compare?from_version_id=%d&to_version_id=%d", projectID, version110.ID, version100.ID),
		nil,
		jwtToken,
	)
	assertStatus(t, reversedCompareResponse, http.StatusOK)
	reversedCompareData := decodeEnvelopeData[compareVersionsData](t, reversedCompareResponse)
	if reversedCompareData.FromVersion.Version != "1.0.0" || reversedCompareData.ToVersion.Version != "1.1.0" {
		t.Fatalf("expected reversed compare to normalize range, got %+v", reversedCompareData)
	}

	draftCompareResponse := jsonRequest(
		t,
		env.router,
		http.MethodGet,
		fmt.Sprintf("/api/projects/%d/versions/compare?from_version_id=%d&to_version_id=%d", projectID, draftVersion.ID, version110.ID),
		nil,
		jwtToken,
	)
	assertAppErrorResponse(t, draftCompareResponse, http.StatusConflict, 40906)

	publicProjectResponse := bearerRequest(t, env.router, http.MethodGet, "/v1/project", nil, projectToken)
	assertStatus(t, publicProjectResponse, http.StatusOK)
	publicProject := decodeEnvelopeData[publicProjectData](t, publicProjectResponse)
	if publicProject.Name != project.Project.Name {
		t.Fatalf("expected /v1/project name %q, got %+v", project.Project.Name, publicProject)
	}

	publicVersionsResponse := bearerRequest(t, env.router, http.MethodGet, "/v1/versions?page=1&page_size=20", nil, projectToken)
	assertStatus(t, publicVersionsResponse, http.StatusOK)
	publicVersions := decodeEnvelopeData[publicVersionsData](t, publicVersionsResponse)
	if len(publicVersions.List) != 2 {
		t.Fatalf("expected /v1/versions to return 2 published versions, got %+v", publicVersions.List)
	}
	for _, version := range publicVersions.List {
		if version.Status != "published" {
			t.Fatalf("expected /v1/versions to expose only published versions, got %+v", publicVersions.List)
		}
	}

	invalidVersionFormatResponse := bearerRequest(t, env.router, http.MethodGet, "/v1/versions/not-a-version/changelogs", nil, projectToken)
	assertAppErrorResponse(t, invalidVersionFormatResponse, http.StatusBadRequest, 40001)

	unpublishedVersionResponse := bearerRequest(t, env.router, http.MethodGet, "/v1/versions/2.0.0/changelogs", nil, projectToken)
	assertAppErrorResponse(t, unpublishedVersionResponse, http.StatusNotFound, 40402)

	publicChangelogResponse := bearerRequest(t, env.router, http.MethodGet, "/v1/versions/1.0.0/changelogs", nil, projectToken)
	assertStatus(t, publicChangelogResponse, http.StatusOK)
	publicChangelogData := decodeEnvelopeData[publicVersionChangelogsData](t, publicChangelogResponse)
	if publicChangelogData.Version.Version != "1.0.0" || len(publicChangelogData.Changelogs) != 1 {
		t.Fatalf("expected /v1 changelog response for 1.0.0, got %+v", publicChangelogData)
	}

	publicAnnouncementsResponse := bearerRequest(t, env.router, http.MethodGet, "/v1/announcements?page=1&page_size=20", nil, projectToken)
	assertStatus(t, publicAnnouncementsResponse, http.StatusOK)
	publicAnnouncements := decodeEnvelopeData[publicAnnouncementsData](t, publicAnnouncementsResponse)
	if len(publicAnnouncements.List) != 1 || publicAnnouncements.List[0].Title != "published notice" {
		t.Fatalf("expected /v1/announcements to expose only published notice, got %+v (draft id=%d)", publicAnnouncements.List, draftAnnouncement.ID)
	}

	exportProjectMarkdownResponse := rawRequest(t, env.router, http.MethodGet, fmt.Sprintf("/api/projects/%d/changelogs/export?format=markdown", projectID), nil, map[string]string{
		"Authorization": "Bearer " + jwtToken,
	})
	assertStatus(t, exportProjectMarkdownResponse, http.StatusOK)
	assertHeaderContains(t, exportProjectMarkdownResponse, "Content-Type", "text/markdown")
	assertHeaderContains(t, exportProjectMarkdownResponse, "Content-Disposition", "all-changelogs.md")
	projectMarkdown := exportProjectMarkdownResponse.Body.String()
	if !strings.Contains(projectMarkdown, "## 1.0.0") || !strings.Contains(projectMarkdown, "## 2.0.0") {
		t.Fatalf("expected whole-project markdown export to include published and draft versions, got %s", projectMarkdown)
	}

	exportProjectJSONResponse := rawRequest(t, env.router, http.MethodGet, fmt.Sprintf("/api/projects/%d/changelogs/export?format=json", projectID), nil, map[string]string{
		"Authorization": "Bearer " + jwtToken,
	})
	assertStatus(t, exportProjectJSONResponse, http.StatusOK)
	assertHeaderContains(t, exportProjectJSONResponse, "Content-Type", "application/json")
	assertHeaderContains(t, exportProjectJSONResponse, "Content-Disposition", "all-changelogs.json")
	projectJSON := exportProjectJSONResponse.Body.String()
	if !strings.Contains(projectJSON, `"version": "1.0.0"`) || !strings.Contains(projectJSON, `"version": "2.0.0"`) {
		t.Fatalf("expected whole-project json export to include published and draft versions, got %s", projectJSON)
	}

	exportSingleMarkdownResponse := rawRequest(t, env.router, http.MethodGet, fmt.Sprintf("/api/projects/%d/changelogs/export?format=markdown&version_id=%d", projectID, version100.ID), nil, map[string]string{
		"Authorization": "Bearer " + jwtToken,
	})
	assertStatus(t, exportSingleMarkdownResponse, http.StatusOK)
	assertHeaderContains(t, exportSingleMarkdownResponse, "Content-Disposition", "v1.0.0-changelogs.md")
	singleMarkdown := exportSingleMarkdownResponse.Body.String()
	if !strings.Contains(singleMarkdown, "## 1.0.0") || strings.Contains(singleMarkdown, "## 1.1.0") {
		t.Fatalf("expected single markdown export to contain only version 1.0.0, got %s", singleMarkdown)
	}

	exportSingleJSONResponse := rawRequest(t, env.router, http.MethodGet, fmt.Sprintf("/api/projects/%d/changelogs/export?format=json&version_id=%d", projectID, version100.ID), nil, map[string]string{
		"Authorization": "Bearer " + jwtToken,
	})
	assertStatus(t, exportSingleJSONResponse, http.StatusOK)
	assertHeaderContains(t, exportSingleJSONResponse, "Content-Disposition", "v1.0.0-changelogs.json")
	singleJSON := exportSingleJSONResponse.Body.String()
	if !strings.Contains(singleJSON, `"version": "1.0.0"`) || strings.Contains(singleJSON, `"version": "1.1.0"`) {
		t.Fatalf("expected single json export to contain only version 1.0.0, got %s", singleJSON)
	}

	if err := env.redis.FlushDB(context.Background()).Err(); err != nil {
		t.Fatalf("flush redis before /v1 rate limit test: %v", err)
	}

	lastStatus := 0
	lastCode := 0
	for index := 1; index <= 61; index += 1 {
		response := bearerRequest(t, env.router, http.MethodGet, "/v1/project", nil, projectToken)
		lastStatus = response.Code
		if index == 61 {
			lastCode = decodeEnvelope(t, response).Code
		}
	}
	if lastStatus != http.StatusTooManyRequests || lastCode != 42902 {
		t.Fatalf("expected 61st /v1/project request to hit 42902, got status=%d code=%d", lastStatus, lastCode)
	}
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()

	db := testutil.OpenMySQL(t)
	redisClient := testutil.OpenRedis(t)
	router := newTestRouter(db, redisClient, integrationJWTSecret)

	return &testEnv{
		router: router,
		db:     db,
		redis:  redisClient,
	}
}

func loginAndGetJWT(t *testing.T, router *gin.Engine) string {
	t.Helper()

	response := jsonRequest(t, router, http.MethodPost, "/api/auth/login", map[string]any{
		"username": "admin",
		"password": "admin123456",
	}, "")
	assertStatus(t, response, http.StatusOK)
	data := decodeEnvelopeData[loginData](t, response)
	if data.Token == "" {
		t.Fatalf("expected login to return a non-empty token")
	}
	return data.Token
}

func createProjectViaAPI(t *testing.T, router *gin.Engine, jwtToken, name, description string) createProjectData {
	t.Helper()

	response := jsonRequest(t, router, http.MethodPost, "/api/projects", map[string]any{
		"name":        name,
		"description": description,
	}, jwtToken)
	assertStatus(t, response, http.StatusOK)
	return decodeEnvelopeData[createProjectData](t, response)
}

func createVersionViaAPI(t *testing.T, router *gin.Engine, jwtToken string, projectID uint64, major, minor, patch int) versionData {
	t.Helper()

	response := jsonRequest(t, router, http.MethodPost, fmt.Sprintf("/api/projects/%d/versions", projectID), map[string]any{
		"major": major,
		"minor": minor,
		"patch": patch,
	}, jwtToken)
	assertStatus(t, response, http.StatusOK)
	return decodeEnvelopeData[versionData](t, response)
}

func createChangelogViaAPI(t *testing.T, router *gin.Engine, jwtToken string, versionID uint64, changelogType, content string) changelogData {
	t.Helper()

	response := jsonRequest(t, router, http.MethodPost, fmt.Sprintf("/api/versions/%d/changelogs", versionID), map[string]any{
		"type":    changelogType,
		"content": content,
	}, jwtToken)
	assertStatus(t, response, http.StatusOK)
	return decodeEnvelopeData[changelogData](t, response)
}

func publishVersionViaAPI(t *testing.T, router *gin.Engine, jwtToken string, versionID uint64) {
	t.Helper()

	response := jsonRequest(t, router, http.MethodPut, fmt.Sprintf("/api/versions/%d/publish", versionID), map[string]any{}, jwtToken)
	assertStatus(t, response, http.StatusOK)
}

func jsonRequest(t *testing.T, router *gin.Engine, method, path string, body any, jwtToken string) *httptest.ResponseRecorder {
	t.Helper()

	headers := map[string]string{}
	if jwtToken != "" {
		headers["Authorization"] = "Bearer " + jwtToken
	}
	return rawRequest(t, router, method, path, body, headers)
}

func bearerRequest(t *testing.T, router *gin.Engine, method, path string, body any, bearerToken string) *httptest.ResponseRecorder {
	t.Helper()

	return rawRequest(t, router, method, path, body, map[string]string{
		"Authorization": "Bearer " + bearerToken,
	})
}

func rawRequest(t *testing.T, router *gin.Engine, method, path string, body any, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()

	var requestBody []byte
	if body != nil {
		var err error
		requestBody, err = json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal request body for %s %s: %v", method, path, err)
		}
	}

	request := httptest.NewRequest(method, path, bytes.NewReader(requestBody))
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	for key, value := range headers {
		request.Header.Set(key, value)
	}

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
}

func decodeEnvelope(t *testing.T, recorder *httptest.ResponseRecorder) envelope {
	t.Helper()

	var payload envelope
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode envelope: %v, raw=%s", err, recorder.Body.String())
	}
	return payload
}

func decodeEnvelopeData[T any](t *testing.T, recorder *httptest.ResponseRecorder) T {
	t.Helper()

	payload := decodeEnvelope(t, recorder)
	var data T
	if len(payload.Data) == 0 {
		t.Fatalf("expected envelope data, got empty payload: %+v", payload)
	}
	if err := json.Unmarshal(payload.Data, &data); err != nil {
		t.Fatalf("decode envelope data: %v, raw=%s", err, string(payload.Data))
	}
	return data
}

func assertStatus(t *testing.T, recorder *httptest.ResponseRecorder, want int) {
	t.Helper()

	if recorder.Code != want {
		t.Fatalf("expected status %d, got %d body=%s", want, recorder.Code, recorder.Body.String())
	}
}

func assertAppErrorResponse(t *testing.T, recorder *httptest.ResponseRecorder, wantStatus, wantCode int) {
	t.Helper()

	assertStatus(t, recorder, wantStatus)
	payload := decodeEnvelope(t, recorder)
	if payload.Code != wantCode {
		t.Fatalf("expected app error code %d, got %d body=%s", wantCode, payload.Code, recorder.Body.String())
	}
}

func assertHeaderContains(t *testing.T, recorder *httptest.ResponseRecorder, key, wantSubstring string) {
	t.Helper()

	actual := recorder.Header().Get(key)
	if !strings.Contains(actual, wantSubstring) {
		t.Fatalf("expected header %s to contain %q, got %q", key, wantSubstring, actual)
	}
}

package service

import (
	"context"
	"testing"

	"proman/server/internal/model"
	"proman/server/internal/repository"
	"proman/server/internal/testutil"
)

func TestAnnouncementServiceStateFlowAndPublishedUpdate(t *testing.T) {
	db := testutil.OpenMySQL(t)
	user := testutil.CreateUser(t, db, "admin", "admin123456")
	project := testutil.CreateProject(t, db, user.ID, "announcement-project")

	projectRepo := repository.NewProjectRepository(db)
	announcementRepo := repository.NewAnnouncementRepository(db)
	service := NewAnnouncementService(projectRepo, announcementRepo)

	announcement, err := service.Create(context.Background(), CreateAnnouncementInput{
		UserID:    user.ID,
		ProjectID: project.ID,
		Title:     "draft notice",
		Content:   "draft content",
		IsPinned:  false,
	})
	if err != nil {
		t.Fatalf("create announcement failed: %v", err)
	}
	if announcement.Status != model.AnnouncementStatusDraft {
		t.Fatalf("expected new announcement to be draft, got %q", announcement.Status)
	}

	published, err := service.Publish(context.Background(), user.ID, announcement.ID)
	if err != nil {
		t.Fatalf("publish announcement failed: %v", err)
	}
	if published.Status != model.AnnouncementStatusPublished {
		t.Fatalf("expected published status, got %q", published.Status)
	}
	if published.PublishedAt == nil {
		t.Fatalf("expected published_at to be set after publish")
	}
	initialPublishedAt := *published.PublishedAt

	updated, err := service.Update(context.Background(), UpdateAnnouncementInput{
		UserID:         user.ID,
		AnnouncementID: announcement.ID,
		Title:          "published notice updated",
		Content:        "published content updated",
		IsPinned:       true,
	})
	if err != nil {
		t.Fatalf("update published announcement failed: %v", err)
	}
	if updated.Status != model.AnnouncementStatusPublished {
		t.Fatalf("expected published announcement to stay published after update, got %q", updated.Status)
	}
	if updated.PublishedAt == nil || !updated.PublishedAt.Equal(initialPublishedAt) {
		t.Fatalf("expected published_at to remain unchanged after published update, got %v want %v", updated.PublishedAt, initialPublishedAt)
	}
	if !updated.IsPinned {
		t.Fatalf("expected published announcement update to persist pinned flag")
	}

	_, err = service.Publish(context.Background(), user.ID, announcement.ID)
	assertAppErrorCode(t, err, 40905)

	revoked, err := service.Revoke(context.Background(), user.ID, announcement.ID)
	if err != nil {
		t.Fatalf("revoke announcement failed: %v", err)
	}
	if revoked.Status != model.AnnouncementStatusDraft {
		t.Fatalf("expected revoked announcement to return to draft, got %q", revoked.Status)
	}
	if revoked.PublishedAt != nil {
		t.Fatalf("expected published_at to be cleared on revoke, got %v", revoked.PublishedAt)
	}

	_, err = service.Revoke(context.Background(), user.ID, announcement.ID)
	assertAppErrorCode(t, err, 40905)
}

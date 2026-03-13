package repository

import (
	"context"
	"errors"
	"testing"

	"gorm.io/gorm"

	"proman/server/internal/model"
	"proman/server/internal/testutil"
)

func TestProjectRepositorySoftDeleteCascadeByIDAndUserID(t *testing.T) {
	db := testutil.OpenMySQL(t)
	user := testutil.CreateUser(t, db, "admin", "admin123456")
	project := testutil.CreateProject(t, db, user.ID, "cascade-project")
	version := testutil.CreateVersion(t, db, project.ID, 1, 0, 0, model.VersionStatusDraft)
	changelog := testutil.CreateChangelog(t, db, version.ID, model.ChangelogTypeAdded, "cascade changelog", 1)
	announcement := testutil.CreateAnnouncement(t, db, project.ID, "cascade announcement", "cascade content", model.AnnouncementStatusDraft, false)

	repo := NewProjectRepository(db)
	if err := repo.SoftDeleteCascadeByIDAndUserID(context.Background(), project.ID, user.ID); err != nil {
		t.Fatalf("SoftDeleteCascadeByIDAndUserID failed: %v", err)
	}

	if _, err := repo.FindByIDAndUserID(context.Background(), project.ID, user.ID); err == nil {
		t.Fatalf("expected deleted project to disappear from scoped query")
	}

	var deletedProject model.Project
	if err := db.Unscoped().First(&deletedProject, project.ID).Error; err != nil {
		t.Fatalf("find deleted project unscoped: %v", err)
	}
	if !deletedProject.DeletedAt.Valid {
		t.Fatalf("expected project deleted_at to be set")
	}

	var deletedVersion model.Version
	if err := db.Unscoped().First(&deletedVersion, version.ID).Error; err != nil {
		t.Fatalf("find deleted version unscoped: %v", err)
	}
	if !deletedVersion.DeletedAt.Valid {
		t.Fatalf("expected version deleted_at to be set")
	}

	var deletedChangelog model.Changelog
	if err := db.Unscoped().First(&deletedChangelog, changelog.ID).Error; err != nil {
		t.Fatalf("find deleted changelog unscoped: %v", err)
	}
	if !deletedChangelog.DeletedAt.Valid {
		t.Fatalf("expected changelog deleted_at to be set")
	}

	var deletedAnnouncement model.Announcement
	if err := db.Unscoped().First(&deletedAnnouncement, announcement.ID).Error; err != nil {
		t.Fatalf("find deleted announcement unscoped: %v", err)
	}
	if !deletedAnnouncement.DeletedAt.Valid {
		t.Fatalf("expected announcement deleted_at to be set")
	}

	var scopedVersion model.Version
	if err := db.First(&scopedVersion, version.ID).Error; err == nil || !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected scoped version lookup to fail with record not found, got %v", err)
	}
}

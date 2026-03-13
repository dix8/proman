package service

import (
	"context"
	"testing"

	"proman/server/internal/model"
	"proman/server/internal/repository"
	"proman/server/internal/testutil"
)

func TestChangelogServiceReorderRules(t *testing.T) {
	db := testutil.OpenMySQL(t)
	user := testutil.CreateUser(t, db, "admin", "admin123456")
	project := testutil.CreateProject(t, db, user.ID, "changelog-project")
	version := testutil.CreateVersion(t, db, project.ID, 1, 0, 0, model.VersionStatusDraft)

	first := testutil.CreateChangelog(t, db, version.ID, model.ChangelogTypeAdded, "first", 1)
	second := testutil.CreateChangelog(t, db, version.ID, model.ChangelogTypeFixed, "second", 2)
	third := testutil.CreateChangelog(t, db, version.ID, model.ChangelogTypeChanged, "third", 3)

	changelogRepo := repository.NewChangelogRepository(db)
	versionRepo := repository.NewVersionRepository(db)
	service := NewChangelogService(changelogRepo, versionRepo)

	err := service.Reorder(context.Background(), ReorderChangelogsInput{
		UserID:    user.ID,
		VersionID: version.ID,
		Items: []repository.ChangelogReorderItem{
			{ID: second.ID, SortOrder: 1},
			{ID: first.ID, SortOrder: 2},
		},
	})
	assertAppErrorCode(t, err, 40001)

	if err := service.Reorder(context.Background(), ReorderChangelogsInput{
		UserID:    user.ID,
		VersionID: version.ID,
		Items: []repository.ChangelogReorderItem{
			{ID: second.ID, SortOrder: 1},
			{ID: first.ID, SortOrder: 2},
			{ID: third.ID, SortOrder: 3},
		},
	}); err != nil {
		t.Fatalf("reorder valid changelogs failed: %v", err)
	}

	reordered, err := changelogRepo.ListAllByVersionIDAndUserID(context.Background(), version.ID, user.ID)
	if err != nil {
		t.Fatalf("list reordered changelogs: %v", err)
	}
	if len(reordered) != 3 {
		t.Fatalf("expected 3 changelogs after reorder, got %d", len(reordered))
	}
	if reordered[0].ID != second.ID || reordered[1].ID != first.ID || reordered[2].ID != third.ID {
		t.Fatalf("unexpected reorder result: got ids [%d %d %d]", reordered[0].ID, reordered[1].ID, reordered[2].ID)
	}

	publishedVersion := testutil.CreateVersion(t, db, project.ID, 1, 1, 0, model.VersionStatusPublished)
	publishedFirst := testutil.CreateChangelog(t, db, publishedVersion.ID, model.ChangelogTypeAdded, "published-first", 1)
	publishedSecond := testutil.CreateChangelog(t, db, publishedVersion.ID, model.ChangelogTypeFixed, "published-second", 2)

	err = service.Reorder(context.Background(), ReorderChangelogsInput{
		UserID:    user.ID,
		VersionID: publishedVersion.ID,
		Items: []repository.ChangelogReorderItem{
			{ID: publishedSecond.ID, SortOrder: 1},
			{ID: publishedFirst.ID, SortOrder: 2},
		},
	})
	assertAppErrorCode(t, err, 40902)
}

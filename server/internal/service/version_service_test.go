package service

import (
	"context"
	"testing"

	"proman/server/internal/model"
	"proman/server/internal/repository"
	"proman/server/internal/testutil"
)

func TestValidateSemverParts(t *testing.T) {
	validZero := 0
	validOne := 1
	negative := -1

	testCases := []struct {
		name      string
		major     *int
		minor     *int
		patch     *int
		wantCode  int
		wantMajor uint
		wantMinor uint
		wantPatch uint
	}{
		{
			name:     "nil parts return bad request",
			major:    &validOne,
			minor:    nil,
			patch:    &validZero,
			wantCode: 40001,
		},
		{
			name:     "negative part returns bad request",
			major:    &negative,
			minor:    &validZero,
			patch:    &validZero,
			wantCode: 40001,
		},
		{
			name:      "valid parts pass through",
			major:     &validOne,
			minor:     &validZero,
			patch:     &validOne,
			wantMajor: 1,
			wantMinor: 0,
			wantPatch: 1,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			major, minor, patch, err := validateSemverParts(testCase.major, testCase.minor, testCase.patch)
			if testCase.wantCode != 0 {
				assertAppErrorCode(t, err, testCase.wantCode)
				return
			}

			if err != nil {
				t.Fatalf("validateSemverParts returned unexpected error: %v", err)
			}
			if major != testCase.wantMajor || minor != testCase.wantMinor || patch != testCase.wantPatch {
				t.Fatalf(
					"unexpected semver parts: got %d.%d.%d want %d.%d.%d",
					major, minor, patch,
					testCase.wantMajor, testCase.wantMinor, testCase.wantPatch,
				)
			}
		})
	}
}

func TestVersionServiceCreateRejectsDuplicateVersion(t *testing.T) {
	db := testutil.OpenMySQL(t)
	user := testutil.CreateUser(t, db, "admin", "admin123456")
	project := testutil.CreateProject(t, db, user.ID, "duplicate-version-project")

	projectRepo := repository.NewProjectRepository(db)
	versionRepo := repository.NewVersionRepository(db)
	service := NewVersionService(projectRepo, versionRepo)

	firstMajor := 1
	firstMinor := 2
	firstPatch := 3

	if _, err := service.Create(context.Background(), CreateVersionInput{
		UserID:    user.ID,
		ProjectID: project.ID,
		Major:     &firstMajor,
		Minor:     &firstMinor,
		Patch:     &firstPatch,
	}); err != nil {
		t.Fatalf("first version create failed: %v", err)
	}

	_, err := service.Create(context.Background(), CreateVersionInput{
		UserID:    user.ID,
		ProjectID: project.ID,
		Major:     &firstMajor,
		Minor:     &firstMinor,
		Patch:     &firstPatch,
	})
	assertAppErrorCode(t, err, 40901)
}

func TestVersionServicePublishRules(t *testing.T) {
	db := testutil.OpenMySQL(t)
	user := testutil.CreateUser(t, db, "admin", "admin123456")
	project := testutil.CreateProject(t, db, user.ID, "publish-rules-project")
	version := testutil.CreateVersion(t, db, project.ID, 1, 0, 0, model.VersionStatusDraft)

	projectRepo := repository.NewProjectRepository(db)
	versionRepo := repository.NewVersionRepository(db)
	service := NewVersionService(projectRepo, versionRepo)

	_, err := service.Publish(context.Background(), user.ID, version.ID)
	assertAppErrorCode(t, err, 40904)

	testutil.CreateChangelog(t, db, version.ID, model.ChangelogTypeAdded, "first changelog", 1)

	published, err := service.Publish(context.Background(), user.ID, version.ID)
	if err != nil {
		t.Fatalf("publish draft version failed: %v", err)
	}
	if published.Status != model.VersionStatusPublished {
		t.Fatalf("expected version status to become published, got %q", published.Status)
	}
	if published.PublishedAt == nil {
		t.Fatalf("expected published_at to be set on successful publish")
	}

	_, err = service.Publish(context.Background(), user.ID, version.ID)
	assertAppErrorCode(t, err, 40902)
}

func TestVersionServiceCompareNormalizesRangeAndAllowsSameVersion(t *testing.T) {
	db := testutil.OpenMySQL(t)
	user := testutil.CreateUser(t, db, "admin", "admin123456")
	project := testutil.CreateProject(t, db, user.ID, "compare-project")

	version100 := testutil.CreateVersion(t, db, project.ID, 1, 0, 0, model.VersionStatusPublished)
	version110 := testutil.CreateVersion(t, db, project.ID, 1, 1, 0, model.VersionStatusPublished)
	testutil.CreateChangelog(t, db, version100.ID, model.ChangelogTypeAdded, "base feature", 1)
	testutil.CreateChangelog(t, db, version110.ID, model.ChangelogTypeFixed, "reverse order bug", 1)

	projectRepo := repository.NewProjectRepository(db)
	versionRepo := repository.NewVersionRepository(db)
	changelogRepo := repository.NewChangelogRepository(db)
	service := NewVersionServiceWithCompare(projectRepo, versionRepo, changelogRepo)

	reversedResult, err := service.Compare(context.Background(), user.ID, project.ID, version110.ID, version100.ID)
	if err != nil {
		t.Fatalf("compare reverse order failed: %v", err)
	}
	if reversedResult.FromVersion.Version != "1.0.0" {
		t.Fatalf("expected normalized from version 1.0.0, got %q", reversedResult.FromVersion.Version)
	}
	if reversedResult.ToVersion.Version != "1.1.0" {
		t.Fatalf("expected normalized to version 1.1.0, got %q", reversedResult.ToVersion.Version)
	}
	if len(reversedResult.Versions) != 2 {
		t.Fatalf("expected 2 versions in compare range, got %d", len(reversedResult.Versions))
	}
	if reversedResult.Versions[0].Version != "1.0.0" || reversedResult.Versions[1].Version != "1.1.0" {
		t.Fatalf("expected ascending range versions [1.0.0 1.1.0], got %+v", reversedResult.Versions)
	}
	if len(reversedResult.Changelogs.Added) != 1 || reversedResult.Changelogs.Added[0].Version != "1.0.0" {
		t.Fatalf("expected added changelog from 1.0.0, got %+v", reversedResult.Changelogs.Added)
	}
	if len(reversedResult.Changelogs.Fixed) != 1 || reversedResult.Changelogs.Fixed[0].Version != "1.1.0" {
		t.Fatalf("expected fixed changelog from 1.1.0, got %+v", reversedResult.Changelogs.Fixed)
	}

	sameVersionResult, err := service.Compare(context.Background(), user.ID, project.ID, version110.ID, version110.ID)
	if err != nil {
		t.Fatalf("compare same version failed: %v", err)
	}
	if sameVersionResult.FromVersion.Version != "1.1.0" || sameVersionResult.ToVersion.Version != "1.1.0" {
		t.Fatalf("expected same version compare to stay at 1.1.0, got from=%q to=%q", sameVersionResult.FromVersion.Version, sameVersionResult.ToVersion.Version)
	}
	if len(sameVersionResult.Versions) != 1 || sameVersionResult.Versions[0].Version != "1.1.0" {
		t.Fatalf("expected same version compare to contain only 1.1.0, got %+v", sameVersionResult.Versions)
	}
}

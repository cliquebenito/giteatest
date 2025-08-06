//go:build !correct

// Copyright 2017 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/unit"
	"code.gitea.io/gitea/models/unittest"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/markup"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/util"
)

var (
	countRepospts        = CountRepositoryOptions{OwnerID: 10}
	countReposptsPublic  = CountRepositoryOptions{OwnerID: 10, Private: util.OptionalBoolFalse}
	countReposptsPrivate = CountRepositoryOptions{OwnerID: 10, Private: util.OptionalBoolTrue}
)

func TestGetRepositoryCount(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	ctx := db.DefaultContext
	count, err1 := CountRepositories(ctx, countRepospts)
	privateCount, err2 := CountRepositories(ctx, countReposptsPrivate)
	publicCount, err3 := CountRepositories(ctx, countReposptsPublic)
	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.NoError(t, err3)
	assert.Equal(t, int64(3), count)
	assert.Equal(t, privateCount+publicCount, count)
}

func TestGetPublicRepositoryCount(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	count, err := CountRepositories(db.DefaultContext, countReposptsPublic)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), count)
}

func TestGetPrivateRepositoryCount(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	count, err := CountRepositories(db.DefaultContext, countReposptsPrivate)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), count)
}

func TestRepoAPIURL(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	repo := unittest.AssertExistsAndLoadBean(t, &Repository{ID: 10})

	assert.Equal(t, "https://try.gitea.io/api/v1/repos/user12/repo10", repo.APIURL())
}

func TestWatchRepo(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	const repoID = 3
	const userID = 2

	assert.NoError(t, WatchRepo(db.DefaultContext, userID, repoID, true))
	unittest.AssertExistsAndLoadBean(t, &Watch{RepoID: repoID, UserID: userID})
	unittest.CheckConsistencyFor(t, &Repository{ID: repoID})

	assert.NoError(t, WatchRepo(db.DefaultContext, userID, repoID, false))
	unittest.AssertNotExistsBean(t, &Watch{RepoID: repoID, UserID: userID})
	unittest.CheckConsistencyFor(t, &Repository{ID: repoID})
}

func TestMetas(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	repo := &Repository{Name: "testRepo"}
	repo.Owner = &user_model.User{Name: "testOwner"}
	repo.OwnerName = repo.Owner.Name

	repo.Units = nil

	metas := repo.ComposeMetas()
	assert.Equal(t, "testRepo", metas["repo"])
	assert.Equal(t, "testOwner", metas["user"])

	externalTracker := RepoUnit{
		Type: unit.TypeExternalTracker,
		Config: &ExternalTrackerConfig{
			ExternalTrackerFormat: "https://someurl.com/{user}/{repo}/{issue}",
		},
	}

	testSuccess := func(expectedStyle string) {
		repo.Units = []*RepoUnit{&externalTracker}
		repo.RenderingMetas = nil
		metas := repo.ComposeMetas()
		assert.Equal(t, expectedStyle, metas["style"])
		assert.Equal(t, "testRepo", metas["repo"])
		assert.Equal(t, "testOwner", metas["user"])
		assert.Equal(t, "https://someurl.com/{user}/{repo}/{issue}", metas["format"])
	}

	testSuccess(markup.IssueNameStyleNumeric)

	externalTracker.ExternalTrackerConfig().ExternalTrackerStyle = markup.IssueNameStyleAlphanumeric
	testSuccess(markup.IssueNameStyleAlphanumeric)

	externalTracker.ExternalTrackerConfig().ExternalTrackerStyle = markup.IssueNameStyleNumeric
	testSuccess(markup.IssueNameStyleNumeric)

	externalTracker.ExternalTrackerConfig().ExternalTrackerStyle = markup.IssueNameStyleRegexp
	testSuccess(markup.IssueNameStyleRegexp)

	repo, err := GetRepositoryByID(db.DefaultContext, 3)
	assert.NoError(t, err)

	metas = repo.ComposeMetas()
	assert.Contains(t, metas, "org")
	assert.Contains(t, metas, "teams")
	assert.Equal(t, "user3", metas["org"])
	assert.Equal(t, ",owners,team1,", metas["teams"])
}

func TestGetRepositoryByURL(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	t.Run("InvalidPath", func(t *testing.T) {
		repo, err := GetRepositoryByURL(db.DefaultContext, "something")

		assert.Nil(t, repo)
		assert.Error(t, err)
	})

	t.Run("ValidHttpURL", func(t *testing.T) {
		test := func(t *testing.T, url string) {
			repo, err := GetRepositoryByURL(db.DefaultContext, url)

			assert.NotNil(t, repo)
			assert.NoError(t, err)

			assert.Equal(t, repo.ID, int64(2))
			assert.Equal(t, repo.OwnerID, int64(2))
		}

		test(t, "https://try.gitea.io/user2/repo2")
		test(t, "https://try.gitea.io/user2/repo2.git")
	})

	t.Run("ValidGitSshURL", func(t *testing.T) {
		test := func(t *testing.T, url string) {
			repo, err := GetRepositoryByURL(db.DefaultContext, url)

			assert.NotNil(t, repo)
			assert.NoError(t, err)

			assert.Equal(t, repo.ID, int64(2))
			assert.Equal(t, repo.OwnerID, int64(2))
		}

		test(t, "git+ssh://sshuser@try.gitea.io/user2/repo2")
		test(t, "git+ssh://sshuser@try.gitea.io/user2/repo2.git")

		test(t, "git+ssh://try.gitea.io/user2/repo2")
		test(t, "git+ssh://try.gitea.io/user2/repo2.git")
	})

	t.Run("ValidImplicitSshURL", func(t *testing.T) {
		test := func(t *testing.T, url string) {
			repo, err := GetRepositoryByURL(db.DefaultContext, url)

			assert.NotNil(t, repo)
			assert.NoError(t, err)

			assert.Equal(t, repo.ID, int64(2))
			assert.Equal(t, repo.OwnerID, int64(2))
		}

		test(t, "sshuser@try.gitea.io:user2/repo2")
		test(t, "sshuser@try.gitea.io:user2/repo2.git")

		test(t, "try.gitea.io:user2/repo2")
		test(t, "try.gitea.io:user2/repo2.git")
	})
}

func TestComposeHTTPSCloneURL_ShouldBeRoutingLink(t *testing.T) {
	setting.IAM.Enabled = true
	setting.OneWork.Enabled = true
	setting.AppURL = "https://vm-gitt-ws-syngx-50078.vdc01.ws.dev.sbt/ssd/tools/sc-ift2/"
	setting.IAM.BaseURL = "https://ift.pd10.pvw.sbt/ssd/tools/sc-ift2/"
	got := ComposeHTTPSCloneURL("owner", "repo")
	assert.Equal(t, "https://ift.pd10.pvw.sbt/ssd/tools/sc-ift2/owner/repo.git", got)
}

func TestComposeHTTPSCloneURL_ShouldBeSCLink(t *testing.T) {
	setting.AppURL = "https://vm-gitt-ws-syngx-50078.vdc01.ws.dev.sbt/ssd/tools/sc-ift2/"
	setting.IAM.BaseURL = "https://ift.pd10.pvw.sbt/ssd/tools/sc-ift2/"
	got := ComposeHTTPSCloneURL("owner", "repo")
	assert.Equal(t, "https://vm-gitt-ws-syngx-50078.vdc01.ws.dev.sbt/ssd/tools/sc-ift2/owner/repo.git", got)
}

func TestComposeHTTPSCloneURL_IAMIsEnabled_ShouldBeSCLink(t *testing.T) {
	setting.IAM.Enabled = true
	setting.AppURL = "https://vm-gitt-ws-syngx-50078.vdc01.ws.dev.sbt/ssd/tools/sc-ift2/"
	setting.IAM.BaseURL = "https://ift.pd10.pvw.sbt/ssd/tools/sc-ift2/"
	got := ComposeHTTPSCloneURL("owner", "repo")
	assert.Equal(t, "https://vm-gitt-ws-syngx-50078.vdc01.ws.dev.sbt/ssd/tools/sc-ift2/owner/repo.git", got)
}

func TestComposeHTTPSCloneURL_OWIsEnabled_ShouldBeSCLink(t *testing.T) {
	setting.OneWork.Enabled = true
	setting.AppURL = "https://vm-gitt-ws-syngx-50078.vdc01.ws.dev.sbt/ssd/tools/sc-ift2/"
	setting.IAM.BaseURL = "https://ift.pd10.pvw.sbt/ssd/tools/sc-ift2/"
	got := ComposeHTTPSCloneURL("owner", "repo")
	assert.Equal(t, "https://vm-gitt-ws-syngx-50078.vdc01.ws.dev.sbt/ssd/tools/sc-ift2/owner/repo.git", got)
}

func TestComposeSSHCloneURL_ShouldBeRoutingDomain(t *testing.T) {
	setting.SSH.User = "user"
	setting.SSH.Domain = "vm-gitt-ws-syngx-50078.vdc01.ws.dev.sbt"
	setting.IAM.Enabled = true
	setting.OneWork.Enabled = true
	setting.IAM.SSHDomain = "ift.pd10.pvw.sbt"
	setting.SSH.Port = 2222
	got := ComposeSSHCloneURL("owner", "repo")
	assert.Equal(t, "ssh://user@ift.pd10.pvw.sbt:2222/owner/repo.git", got)
}

func TestComposeSSHCloneURL_ShouldBeSCDomain(t *testing.T) {
	setting.SSH.User = "user"
	setting.SSH.Domain = "vm-gitt-ws-syngx-50078.vdc01.ws.dev.sbt"
	setting.IAM.SSHDomain = "ift.pd10.pvw.sbt"
	setting.SSH.Port = 2222
	got := ComposeSSHCloneURL("owner", "repo")
	assert.Equal(t, "ssh://user@vm-gitt-ws-syngx-50078.vdc01.ws.dev.sbt:2222/owner/repo.git", got)
}

func TestComposeSSHCloneURL_IAMIsEnabled_ShouldBeSCDomain(t *testing.T) {
	setting.IAM.Enabled = true
	setting.SSH.User = "user"
	setting.SSH.Domain = "vm-gitt-ws-syngx-50078.vdc01.ws.dev.sbt"
	setting.IAM.SSHDomain = "ift.pd10.pvw.sbt"
	setting.SSH.Port = 2222
	got := ComposeSSHCloneURL("owner", "repo")
	assert.Equal(t, "ssh://user@vm-gitt-ws-syngx-50078.vdc01.ws.dev.sbt:2222/owner/repo.git", got)
}

func TestComposeSSHCloneURL_OWIsEnabled_ShouldBeSCDomain(t *testing.T) {
	setting.OneWork.Enabled = true
	setting.SSH.User = "user"
	setting.SSH.Domain = "vm-gitt-ws-syngx-50078.vdc01.ws.dev.sbt"
	setting.IAM.SSHDomain = "ift.pd10.pvw.sbt"
	setting.SSH.Port = 2222
	got := ComposeSSHCloneURL("owner", "repo")
	assert.Equal(t, "ssh://user@vm-gitt-ws-syngx-50078.vdc01.ws.dev.sbt:2222/owner/repo.git", got)
}

func Test_isOneWork(t *testing.T) {
	tests := []struct {
		name                       string
		IAMEnabled, OneWorkEnabled bool
		want                       bool
	}{
		{
			name:           "All conditions are true",
			IAMEnabled:     true,
			OneWorkEnabled: true,
			want:           true,
		},
		{
			name:           "IAM not enabled",
			IAMEnabled:     false,
			OneWorkEnabled: true,
			want:           false,
		},
		{
			name:           "OneWork not enabled",
			IAMEnabled:     true,
			OneWorkEnabled: false,
			want:           false,
		},
		{
			name:           "OneWork type equals standalone",
			IAMEnabled:     true,
			OneWorkEnabled: true,
			want:           false,
		},
		{
			name:           "IAM and OneWork not enabled",
			IAMEnabled:     false,
			OneWorkEnabled: false,
			want:           false,
		},
		{
			name:           "IAM not enabled, OneWork type equals standalone",
			IAMEnabled:     false,
			OneWorkEnabled: true,
			want:           false,
		},
		{
			name:           "OneWork not enabled, OneWork type equals standalone",
			IAMEnabled:     true,
			OneWorkEnabled: false,
			want:           false,
		},
		{
			name:           "All conditions are false",
			IAMEnabled:     false,
			OneWorkEnabled: false,
			want:           false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setting.IAM.Enabled = tt.IAMEnabled
			setting.OneWork.Enabled = tt.OneWorkEnabled

			got := isOneWork()
			assert.Equal(t, tt.want, got)
		})
	}
}

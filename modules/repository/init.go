// Copyright 2019 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repository

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"gitlab.com/gitlab-org/gitaly/v16/proto/go/gitalypb"

	"code.gitea.io/gitea/integration/gitaly"
	"code.gitea.io/gitea/modules/git"
	asymkey_service "code.gitea.io/gitea/services/asymkey"

	issues_model "code.gitea.io/gitea/models/issues"
	repo_model "code.gitea.io/gitea/models/repo"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/label"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/options"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/templates/vars"
	"code.gitea.io/gitea/modules/util"
)

type OptionFile struct {
	DisplayName string
	Description string
}

var (
	// Gitignores contains the gitiginore files
	Gitignores []string

	// Licenses contains the license files
	Licenses []string

	// Readmes contains the readme files
	Readmes []string

	// LabelTemplateFiles contains the label template files, each item has its DisplayName and Description
	LabelTemplateFiles   []OptionFile
	labelTemplateFileMap = map[string]string{} // DisplayName => FileName mapping
)

type optionFileList struct {
	all    []string // all files provided by bindata & custom-path. Sorted.
	custom []string // custom files provided by custom-path. Non-sorted, internal use only.
}

// mergeCustomLabelFiles merges the custom label files. Always use the file's main name (DisplayName) as the key to de-duplicate.
func mergeCustomLabelFiles(fl optionFileList) []string {
	exts := map[string]int{"": 0, ".yml": 1, ".yaml": 2} // "yaml" file has the highest priority to be used.

	m := map[string]string{}
	merge := func(list []string) {
		sort.Slice(list, func(i, j int) bool { return exts[filepath.Ext(list[i])] < exts[filepath.Ext(list[j])] })
		for _, f := range list {
			m[strings.TrimSuffix(f, filepath.Ext(f))] = f
		}
	}
	merge(fl.all)
	merge(fl.custom)

	files := make([]string, 0, len(m))
	for _, f := range m {
		files = append(files, f)
	}
	sort.Strings(files)
	return files
}

// LoadRepoConfig loads the repository config
func LoadRepoConfig() error {
	types := []string{"gitignore", "license", "readme", "label"} // option file directories
	typeFiles := make([]optionFileList, len(types))
	for i, t := range types {
		var err error
		if typeFiles[i].all, err = options.AssetFS().ListFiles(t, true); err != nil {
			return fmt.Errorf("failed to list %s files: %w", t, err)
		}
		sort.Strings(typeFiles[i].all)
		customPath := filepath.Join(setting.CustomPath, "options", t)
		if isDir, err := util.IsDir(customPath); err != nil {
			return fmt.Errorf("failed to check custom %s dir: %w", t, err)
		} else if isDir {
			if typeFiles[i].custom, err = util.StatDir(customPath); err != nil {
				return fmt.Errorf("failed to list custom %s files: %w", t, err)
			}
		}
	}

	Gitignores = typeFiles[0].all
	Licenses = typeFiles[1].all
	Readmes = typeFiles[2].all

	// Load label templates
	LabelTemplateFiles = nil
	labelTemplateFileMap = map[string]string{}
	for _, file := range mergeCustomLabelFiles(typeFiles[3]) {
		description, err := label.LoadTemplateDescription(file)
		if err != nil {
			return fmt.Errorf("failed to load labels: %w", err)
		}
		displayName := strings.TrimSuffix(file, filepath.Ext(file))
		labelTemplateFileMap[displayName] = file
		LabelTemplateFiles = append(LabelTemplateFiles, OptionFile{DisplayName: displayName, Description: description})
	}

	// Filter out invalid names and promote preferred licenses.
	sortedLicenses := make([]string, 0, len(Licenses))
	for _, name := range setting.Repository.PreferredLicenses {
		if util.SliceContainsString(Licenses, name, true) {
			sortedLicenses = append(sortedLicenses, name)
		}
	}
	for _, name := range Licenses {
		if !util.SliceContainsString(setting.Repository.PreferredLicenses, name, true) {
			sortedLicenses = append(sortedLicenses, name)
		}
	}
	Licenses = sortedLicenses
	return nil
}

func prepareRepoCommit(ctx context.Context, doer *user_model.User, repository *repo_model.Repository, tmpDir string, opts CreateRepoOptions) error {
	//commitTimeStr := time.Now().Format(time.RFC3339)
	//authorSig := repository.Owner.NewGitSig()

	// Because this may call hooks we should pass in the environment
	//env := append(os.Environ(),
	//	"GIT_AUTHOR_NAME="+authorSig.Name,
	//	"GIT_AUTHOR_EMAIL="+authorSig.Email,
	//	"GIT_AUTHOR_DATE="+commitTimeStr,
	//	"GIT_COMMITTER_NAME="+authorSig.Name,
	//	"GIT_COMMITTER_EMAIL="+authorSig.Email,
	//	"GIT_COMMITTER_DATE="+commitTimeStr,
	//)
	//// todo change to gitaly
	//// Clone to temporary path and do the init commit.
	//if stdout, _, err := git.NewCommand(ctx, "clone").AddDynamicArguments(repoPath, tmpDir).
	//	SetDescription(fmt.Sprintf("prepareRepoCommit (git clone): %s to %s", repoPath, tmpDir)).
	//	RunStdString(&git.RunOpts{Dir: "", Env: env}); err != nil {
	//	log.Error("Failed to clone from %v into %s: stdout: %s\nError: %v", repository, tmpDir, stdout, err)
	//	return fmt.Errorf("git clone: %w", err)
	//}

	createdFiles := make(map[string][]byte)

	// README
	data, err := options.Readme(opts.Readme)
	if err != nil {
		return fmt.Errorf("GetRepoInitFile[%s]: %w", opts.Readme, err)
	}

	cloneLink := repository.CloneLink()
	match := map[string]string{
		"Name":           repository.Name,
		"Description":    repository.Description,
		"CloneURL.SSH":   cloneLink.SSH,
		"CloneURL.HTTPS": cloneLink.HTTPS,
		"OwnerName":      repository.OwnerName,
	}
	res, err := vars.Expand(string(data), match)
	if err != nil {
		// here we could just log the error and continue the rendering
		log.Error("unable to expand template vars for repository README: %s, err: %v", opts.Readme, err)
	}
	if err = os.WriteFile(filepath.Join(tmpDir, "README.md"),
		[]byte(res), 0o644); err != nil {
		return fmt.Errorf("write README.md: %w", err)
	}

	createdFiles["README.md"] = []byte(res)

	// .gitignore
	if len(opts.Gitignores) > 0 {
		var buf bytes.Buffer
		names := strings.Split(opts.Gitignores, ",")
		for _, name := range names {
			data, err = options.Gitignore(name)
			if err != nil {
				return fmt.Errorf("GetRepoInitFile[%s]: %w", name, err)
			}
			buf.WriteString("# ---> " + name + "\n")
			buf.Write(data)
			buf.WriteString("\n")
		}

		if buf.Len() > 0 {
			if err = os.WriteFile(filepath.Join(tmpDir, ".gitignore"), buf.Bytes(), 0o644); err != nil {
				return fmt.Errorf("write .gitignore: %w", err)
			}
		}
		createdFiles[".gitignore"] = buf.Bytes()
	}

	// LICENSE
	if len(opts.License) > 0 {
		data, err = getLicense(opts.License, &licenseValues{
			Owner: repository.OwnerName,
			Email: repository.Owner.Email,
			Repo:  repository.Name,
			Year:  time.Now().Format("2006"),
		})
		if err != nil {
			return fmt.Errorf("getLicense[%s]: %w", opts.License, err)
		}

		if err = os.WriteFile(filepath.Join(tmpDir, "LICENSE"), data, 0o644); err != nil {
			return fmt.Errorf("write LICENSE: %w", err)
		}
		createdFiles["LICENSE"] = data
	}

	//files.CreateOrUpdateRepoFile(ctx, repository, repository.Owner, opts)

	err = filesCommit(ctx, doer, repository, createdFiles)
	if err != nil {
		return fmt.Errorf("filesCommit: %w", err)
	}
	return nil
}

func filesCommit(ctx context.Context, doer *user_model.User, repo *repo_model.Repository, content map[string][]byte) error {
	gitRepo, closer, err := git.RepositoryFromContextOrOpen(ctx, repo.OwnerName, repo.Name, repo.RepoPath())
	if err != nil {
		return err
	}
	defer closer.Close()
	//commit, err := gitRepo.GetBranchCommit(repo.DefaultBranch)
	//if err != nil {
	//	return err // Couldn't get a commit for the branch
	//}
	//
	//LastCommitID := strings.TrimSpace(commit.ID.String())
	//
	//// Assigned LastCommitID in opts if it hasn't been set
	//if LastCommitID == "" {
	//	LastCommitID = commit.ID.String()
	//} else {
	//	lastCommitID, err := gitRepo.ConvertToSHA1(repo.DefaultBranch)
	//	if err != nil {
	//		return fmt.Errorf("ConvertToSHA1: Invalid last commit ID: %w", err)
	//	}
	//	LastCommitID = lastCommitID.String()
	//}

	var requestMessages []*gitalypb.UserCommitFilesRequest
	requestMessages = append(requestMessages, &gitalypb.UserCommitFilesRequest{
		UserCommitFilesRequestPayload: &gitalypb.UserCommitFilesRequest_Header{
			Header: &gitalypb.UserCommitFilesRequestHeader{
				Repository: MatchRepository(repo.OwnerName, repo.Name),
				User: &gitalypb.User{
					GlId:       strconv.FormatInt(doer.ID, 10),
					Name:       []byte(repo.OwnerName),
					Email:      []byte(doer.GetDefaultEmail()),
					GlUsername: doer.Name,
				},
				BranchName:        []byte(repo.DefaultBranch),
				Force:             true,
				CommitMessage:     []byte("Initial commit"),
				CommitAuthorName:  []byte(doer.Name),
				CommitAuthorEmail: []byte(doer.GetDefaultEmail()),
				StartBranchName:   []byte(repo.DefaultBranch),
				StartRepository:   MatchRepository(repo.OwnerName, repo.Name),
				//StartSha:          LastCommitID,
			},
		},
	})

	for fileName, fileContent := range content {
		requestMessages = append(requestMessages, &gitalypb.UserCommitFilesRequest{
			UserCommitFilesRequestPayload: &gitalypb.UserCommitFilesRequest_Action{
				Action: &gitalypb.UserCommitFilesAction{
					UserCommitFilesActionPayload: &gitalypb.UserCommitFilesAction_Header{
						Header: &gitalypb.UserCommitFilesActionHeader{
							Action:   gitalypb.UserCommitFilesActionHeader_CREATE,
							FilePath: []byte(fileName),
							//PreviousPath: []byte(repo.RepoPath()),
						},
					},
				},
			}})

		requestMessages = append(requestMessages, &gitalypb.UserCommitFilesRequest{
			UserCommitFilesRequestPayload: &gitalypb.UserCommitFilesRequest_Action{
				Action: &gitalypb.UserCommitFilesAction{
					UserCommitFilesActionPayload: &gitalypb.UserCommitFilesAction_Content{
						Content: fileContent,
					},
				},
			},
		})
	}

	ctxWithCancel, cancel := context.WithCancel(ctx)
	defer cancel()
	userCommitFilesClient, err := gitRepo.OperationClient.UserCommitFiles(ctxWithCancel)
	if err != nil {
		return err
	}

	for _, requestToGitaly := range requestMessages {
		err = userCommitFilesClient.Send(requestToGitaly)
		if err != nil {
			return fmt.Errorf("opClientUserFiles: opsContent: %w", err)
		}
	}

	recv, err := userCommitFilesClient.CloseAndRecv()
	if err != nil || recv.IndexError != "" || recv.PreReceiveError != "" {
		return err
	}
	return nil
}

// initRepoCommit temporarily changes with work directory.
func initRepoCommit(ctx context.Context, tmpPath string, repo *repo_model.Repository, u *user_model.User, defaultBranch string) (err error) {
	commitTimeStr := time.Now().Format(time.RFC3339)

	sig := u.NewGitSig()
	// Because this may call hooks we should pass in the environment
	env := append(os.Environ(),
		"GIT_AUTHOR_NAME="+sig.Name,
		"GIT_AUTHOR_EMAIL="+sig.Email,
		"GIT_AUTHOR_DATE="+commitTimeStr,
		"GIT_COMMITTER_DATE="+commitTimeStr,
	)
	committerName := sig.Name
	committerEmail := sig.Email

	if stdout, _, err := git.NewCommand(ctx, "add", "--all").
		SetDescription(fmt.Sprintf("initRepoCommit (git add): %s", tmpPath)).
		RunStdString(&git.RunOpts{Dir: tmpPath}); err != nil {
		log.Error("git add --all failed: Stdout: %s\nError: %v", stdout, err)
		return fmt.Errorf("git add --all: %w", err)
	}

	cmd := git.NewCommand(ctx, "commit", "--message=Initial commit").
		AddOptionFormat("--author='%s <%s>'", sig.Name, sig.Email)

	sign, keyID, signer, _ := asymkey_service.SignInitialCommit(ctx, tmpPath, u)
	if sign {
		cmd.AddOptionFormat("-S%s", keyID)

		if repo.GetTrustModel() == repo_model.CommitterTrustModel || repo.GetTrustModel() == repo_model.CollaboratorCommitterTrustModel {
			// need to set the committer to the KeyID owner
			committerName = signer.Name
			committerEmail = signer.Email
		}
	} else {
		cmd.AddArguments("--no-gpg-sign")
	}

	env = append(env,
		"GIT_COMMITTER_NAME="+committerName,
		"GIT_COMMITTER_EMAIL="+committerEmail,
	)

	if stdout, _, err := cmd.
		SetDescription(fmt.Sprintf("initRepoCommit (git commit): %s", tmpPath)).
		RunStdString(&git.RunOpts{Dir: tmpPath, Env: env}); err != nil {
		log.Error("Failed to commit: %v: Stdout: %s\nError: %v", cmd.String(), stdout, err)
		return fmt.Errorf("git commit: %w", err)
	}

	if len(defaultBranch) == 0 {
		defaultBranch = setting.Repository.DefaultBranch
	}

	// todo change to gitaly
	if stdout, _, err := git.NewCommand(ctx, "push", "origin").AddDynamicArguments("HEAD:" + defaultBranch).
		SetDescription(fmt.Sprintf("initRepoCommit (git push): %s", tmpPath)).
		RunStdString(&git.RunOpts{Dir: tmpPath, Env: InternalPushingEnvironment(u, repo)}); err != nil {
		log.Error("Failed to push back to HEAD: Stdout: %s\nError: %v", stdout, err)
		return fmt.Errorf("git push: %w", err)
	}

	return nil
}

// CheckInitRepository проверка на существование репозитория и его последующее создане.
func CheckInitRepository(ctx context.Context, repo *gitalypb.Repository) (err error) {

	// создаем клиент для репозитория в гитали.
	ctx2, rc, err := gitaly.NewRepositoryClient(ctx)
	if err != nil {
		return err
	}

	// проверяем, есть ли репозиторий. если есть -- ошибка.
	existsResp, err := rc.RepositoryExists(ctx2, &gitalypb.RepositoryExistsRequest{Repository: repo})
	if err != nil {
		log.Error("Unable to check if %s exists. Error: %v", repo.RelativePath, err)
		return err
	}
	if existsResp.Exists {
		return repo_model.ErrRepoFilesAlreadyExist{
			Uname: repo.GlProjectPath,
			Name:  repo.GlRepository,
		}
	}

	return err
}

func MatchRepository(owner, name string) *gitalypb.Repository {
	path := repo_model.RepoPath(owner, name)
	return &gitalypb.Repository{
		GlRepository:  name,
		GlProjectPath: owner,
		RelativePath:  path,
		StorageName:   setting.Gitaly.MainServerName,
	}
}

func CreateRepositoryGitaly(ctx context.Context, defaultBranch string, gitalyRepo *gitalypb.Repository) (err error) {
	// создаем клиент для репозитория в гитали.
	ctx2, rc, err := gitaly.NewRepositoryClient(ctx)
	if err != nil {
		return err
	}

	_, err = rc.CreateRepository(ctx2, &gitalypb.CreateRepositoryRequest{
		Repository:    gitalyRepo,
		DefaultBranch: []byte(defaultBranch),
	})
	if err != nil {
		return fmt.Errorf("git.CreateRepositoryGitaly: %w", err)
	}
	return err
}

// InitRepository initializes LICENSE and README and .gitignore if needed.
func initRepository(ctx context.Context, doer *user_model.User, repo *repo_model.Repository, opts CreateRepoOptions) (err error) {
	if repo.DefaultBranch == "" {
		repo.DefaultBranch = setting.Repository.DefaultBranch
	}

	err = CreateRepositoryGitaly(ctx, repo.DefaultBranch, MatchRepository(repo.OwnerName, repo.Name))
	if err != nil {
		return err
	}

	// Initialize repository according to user's choice.
	if opts.AutoInit {
		tmpDir, err := os.MkdirTemp(os.TempDir(), "gitea-"+repo.Name)
		if err != nil {
			return fmt.Errorf("Failed to create temp dir for repository %s: %w", repo.RepoPath(), err)
		}
		defer func() {
			if err := util.RemoveAll(tmpDir); err != nil {
				log.Warn("Unable to remove temporary directory: %s: Error: %v", tmpDir, err)
			}
		}()

		if err = prepareRepoCommit(ctx, doer, repo, tmpDir, opts); err != nil {
			return fmt.Errorf("prepareRepoCommit: %w", err)
		}

		//// Apply changes and commit.
		//if err = initRepoCommit(ctx, tmpDir, repo, u, opts.DefaultBranch); err != nil {
		//	return fmt.Errorf("initRepoCommit: %w", err)
		//}
	}

	// Re-fetch the repository from database before updating it (else it would
	// override changes that were done earlier with sql)
	if repo, err = repo_model.GetRepositoryByID(ctx, repo.ID); err != nil {
		return fmt.Errorf("getRepositoryByID: %w", err)
	}

	if !opts.AutoInit {
		repo.IsEmpty = true
	}

	if err = UpdateRepository(ctx, repo, false); err != nil {
		return fmt.Errorf("updateRepository: %w", err)
	}

	return nil
}

// InitializeLabels adds a label set to a repository using a template
func InitializeLabels(ctx context.Context, id int64, labelTemplate string, isOrg bool) error {
	list, err := LoadTemplateLabelsByDisplayName(labelTemplate)
	if err != nil {
		return err
	}

	labels := make([]*issues_model.Label, len(list))
	for i := 0; i < len(list); i++ {
		labels[i] = &issues_model.Label{
			Name:        list[i].Name,
			Exclusive:   list[i].Exclusive,
			Description: list[i].Description,
			Color:       list[i].Color,
		}
		if isOrg {
			labels[i].OrgID = id
		} else {
			labels[i].RepoID = id
		}
	}
	for _, label := range labels {
		if err = issues_model.NewLabel(ctx, label); err != nil {
			return err
		}
	}
	return nil
}

// LoadTemplateLabelsByDisplayName loads a label template by its display name
func LoadTemplateLabelsByDisplayName(displayName string) ([]*label.Label, error) {
	if fileName, ok := labelTemplateFileMap[displayName]; ok {
		return label.LoadTemplateFile(fileName)
	}
	return nil, label.ErrTemplateLoad{TemplateFile: displayName, OriginalError: fmt.Errorf("label template %q not found", displayName)}
}

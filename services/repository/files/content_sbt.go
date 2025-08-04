package files

import (
	"code.gitea.io/gitea/models"
	repoModel "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/modules/analyze"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/git"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/routers/sbt/response"
	"io"
	"path"
	"time"
)

// GetContentsOrListSbt Получение содержимого репозитория (дерево файлов и последний коммит для каждого файла)
func GetContentsOrListSbt(ctx *context.Context, repo *repoModel.Repository, treePath, ref string) (interface{}, error) {
	if repo.IsEmpty {
		return make([]interface{}, 0), nil
	}
	if ref == "" {
		ref = repo.DefaultBranch
	}

	// Check that the path given in opts.treePath is valid (not a git path)
	cleanTreePath := CleanUploadFileName(treePath)
	if cleanTreePath == "" && treePath != "" {
		return nil, models.ErrFilenameInvalid{
			Path: treePath,
		}
	}
	treePath = cleanTreePath

	gitRepo, closer, err := git.RepositoryFromContextOrOpen(ctx, repo.OwnerName, repo.Name, repo.RepoPath())
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	// Get the commit object for the ref
	commit, err := gitRepo.GetCommit(ref)
	if err != nil {
		return nil, err
	}

	entry, err := commit.GetTreeEntryByPath(treePath)
	if err != nil {
		return nil, err
	}

	if entry.Type() != "tree" { // для типов blob и commit - отображение только одного файла
		lastCommit, err := commit.GetCommitByPath(treePath)
		if err != nil {
			return nil, err
		}

		return convertEntryAndCommitToContentResponse(commit, gitRepo, entry, treePath, lastCommit, false)
	}

	gitTree, err := commit.SubTree(treePath)
	if err != nil {
		return nil, err
	}

	entries, err := gitTree.ListEntries()
	if err != nil {
		return nil, err
	}

	commitInfos, _, err := entries.GetCommitsInfo(ctx, commit, treePath)
	if err != nil {
		return nil, err
	}

	var fileList []*response.ContentsResponse

	// Для каждого файла и папки создаем респонс
	for _, e := range entries {
		subTreePath := path.Join(treePath, e.Name())

		var lastCommit *git.Commit

		for _, lastCommitInfo := range commitInfos {
			if lastCommitInfo.Entry.ID == e.ID {
				lastCommit = lastCommitInfo.Commit

				break
			}
		}

		fileContentResponse, err := convertEntryAndCommitToContentResponse(commit, gitRepo, e, subTreePath, lastCommit, true)

		if err != nil {
			return nil, err
		}
		fileList = append(fileList, fileContentResponse)
	}

	return fileList, nil
}

// convertEntryAndCommitToContentResponse из git.TreeEntry и (последнего) git.Commit создаем response.ContentsResponse
func convertEntryAndCommitToContentResponse(commit *git.Commit, gitRepo *git.Repository, entry *git.TreeEntry, treePath string, lastCommit *git.Commit, forList bool) (*response.ContentsResponse, error) {
	// All content types have these fields in populated
	contentsResponse := &response.ContentsResponse{
		Name:              entry.Name(),
		Path:              treePath,
		SHA:               entry.ID.String(),
		LastCommitSHA:     lastCommit.ID.String(),
		LastCommitDate:    lastCommit.Committer.When.Format(time.RFC3339),
		LastCommitMessage: lastCommit.CommitMessage,
		Size:              entry.Size(),
	}

	// Now populate the rest of the ContentsResponse based on entry type
	if entry.IsRegular() || entry.IsExecutable() {
		contentsResponse.Type = string(ContentTypeRegular)
		if blobResponse, err := getBlobByShaSbt(gitRepo, entry.ID.String(), entry.Name()); err != nil {
			return nil, err
		} else if !forList {
			// We don't show the content if we are getting a list of FileContentResponses
			contentsResponse.Encoding = &blobResponse.Encoding
			contentsResponse.Content = &blobResponse.Content
			contentsResponse.Language = &blobResponse.Language
		}
	} else if entry.IsDir() {
		contentsResponse.Type = string(ContentTypeDir)
	} else if entry.IsLink() {
		contentsResponse.Type = string(ContentTypeLink)
		// The target of a symlink file is the content of the file
		targetFromContent, err := entry.Blob().GetBlobContent()
		if err != nil {
			return nil, err
		}
		contentsResponse.Target = &targetFromContent
	} else if entry.IsSubModule() {
		contentsResponse.Type = string(ContentTypeSubmodule)
		submodule, err := commit.GetSubModule(treePath)
		if err != nil {
			return nil, err
		}
		if submodule != nil && submodule.URL != "" {
			contentsResponse.SubmoduleGitURL = &submodule.URL
		}
	}

	return contentsResponse, nil
}

type GitBlobResponse struct {
	Content  string `json:"content"`
	Encoding string `json:"encoding"`
	SHA      string `json:"sha"`
	Size     int64  `json:"size"`
	Language string `json:"language"`
}

// getBlobByShaSbt получаем GitBlobResponse объекта репозитория по хешу
func getBlobByShaSbt(gitRepo *git.Repository, sha string, name string) (*GitBlobResponse, error) {
	gitBlob, err := gitRepo.GetBlob(sha)
	if err != nil {
		return nil, err
	}
	content := ""
	if gitBlob.Size() <= setting.API.DefaultMaxBlobSize {
		content, err = gitBlob.GetBlobContentBase64()
		if err != nil {
			return nil, err
		}
	}
	bType, _ := gitBlob.GuessContentType()
	var language string

	if bType.IsText() {
		language, _ = getLanguage(gitBlob, name)
	}

	return &GitBlobResponse{
		SHA:      gitBlob.ID.String(),
		Size:     gitBlob.Size(),
		Encoding: "base64",
		Content:  content,
		Language: language,
	}, nil
}

// getLanguage получаем ЯП из файла
func getLanguage(b *git.Blob, name string) (string, error) {
	dataRc, err := b.DataAsync()
	if err != nil {
		return "", err
	}

	defer dataRc.Close()

	out, err := io.ReadAll(dataRc)

	if err != nil {
		return "", err
	}
	language := analyze.GetCodeLanguage(name, out)
	return language, nil
}

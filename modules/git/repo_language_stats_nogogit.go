// Copyright 2020 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

//go:build !gogit

package git

import (
	"strings"

	"code.gitea.io/gitea/modules/analyze"
	"github.com/go-enry/go-enry/v2"
	"gitlab.com/gitlab-org/gitaly/v16/proto/go/gitalypb"
)

// GetLanguageStats calculates language stats for git repository at specified commit
func (repo *Repository) GetLanguageStats(commitID string) (map[string]int64, error) {
	commit, err := repo.GetCommit(commitID)
	if err != nil {
		return nil, err
	}

	tree := commit.Tree

	entries, err := tree.ListEntriesRecursiveWithSize()
	if err != nil {
		return nil, err
	}

	checker, deferable := repo.CheckAttributeReader(commitID)
	defer deferable()

	// sizes contains the current calculated size of all files by language
	sizes := make(map[string]int64)
	// by default we will only count the sizes of programming languages or markup languages
	// unless they are explicitly set using linguist-language
	includedLanguage := map[string]bool{}
	// or if there's only one language in the repository
	firstExcludedLanguage := ""
	firstExcludedLanguageSize := int64(0)

	for _, f := range entries {
		select {
		case <-repo.Ctx.Done():
			return sizes, repo.Ctx.Err()
		default:
		}

		blobClient, err := repo.BlobClient.GetBlob(repo.Ctx, &gitalypb.GetBlobRequest{Repository: repo.GitalyRepo, Oid: f.ID.String(), Limit: -1})
		if err != nil {
			return nil, err
		}

		content := make([]byte, 0, fileSizeLimit)
		canRead := true
		for canRead {
			blobResponse, _ := blobClient.Recv()
			if blobResponse == nil {
				canRead = false
			} else {
				f.size += blobResponse.GetSize()
				f.sized = true
				content = append(content, blobResponse.Data...)
			}
		}

		notVendored := false
		notGenerated := false

		if checker != nil {
			attrs, err := checker.CheckPath(f.Name())
			if err == nil {
				if vendored, has := attrs["linguist-vendored"]; has {
					if vendored == "set" || vendored == "true" {
						continue
					}
					notVendored = vendored == "false"
				}
				if generated, has := attrs["linguist-generated"]; has {
					if generated == "set" || generated == "true" {
						continue
					}
					notGenerated = generated == "false"
				}
				if language, has := attrs["linguist-language"]; has && language != "unspecified" && language != "" {
					// group languages, such as Pug -> HTML; SCSS -> CSS
					group := enry.GetLanguageGroup(language)
					if len(group) != 0 {
						language = group
					}

					// this language will always be added to the size
					sizes[language] += f.Size()
					continue
				} else if language, has := attrs["gitlab-language"]; has && language != "unspecified" && language != "" {
					// strip off a ? if present
					if idx := strings.IndexByte(language, '?'); idx >= 0 {
						language = language[:idx]
					}
					if len(language) != 0 {
						// group languages, such as Pug -> HTML; SCSS -> CSS
						group := enry.GetLanguageGroup(language)
						if len(group) != 0 {
							language = group
						}

						// this language will always be added to the size
						sizes[language] += f.Size()
						continue
					}
				}

			}
		}

		if (!notVendored && analyze.IsVendor(f.Name())) || enry.IsDotFile(f.Name()) ||
			enry.IsDocumentation(f.Name()) || enry.IsConfiguration(f.Name()) {
			continue
		}

		if !notGenerated && enry.IsGenerated(f.Name(), content) {
			continue
		}

		// FIXME: Why can't we split this and the IsGenerated tests to avoid reading the blob unless absolutely necessary?
		// - eg. do the all the detection tests using filename first before reading content.
		language := analyze.GetCodeLanguage(f.Name(), content)
		if language == enry.OtherLanguage || language == "" {
			continue
		}

		// group languages, such as Pug -> HTML; SCSS -> CSS
		group := enry.GetLanguageGroup(language)
		if group != "" {
			language = group
		}

		included, checked := includedLanguage[language]
		if !checked {
			langtype := enry.GetLanguageType(language)
			included = langtype == enry.Programming || langtype == enry.Markup
			includedLanguage[language] = included
		}
		if included {
			sizes[language] += f.Size()
		} else if len(sizes) == 0 && (firstExcludedLanguage == "" || firstExcludedLanguage == language) {
			firstExcludedLanguage = language
			firstExcludedLanguageSize += f.Size()
		}
		continue
	}

	return sizes, nil
}

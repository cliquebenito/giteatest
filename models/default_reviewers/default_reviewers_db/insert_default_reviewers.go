package default_reviewers_db

import (
	"context"
	"fmt"

	"code.gitea.io/gitea/models/default_reviewers"
)

// InsertDefaultReviewers добавление default reviewers
func (r defaultReviewersDB) InsertDefaultReviewers(ctx context.Context, defaultReviewers []*default_reviewers.DefaultReviewers) error {
	_, err := r.engine.Insert(defaultReviewers)
	if err != nil {
		return fmt.Errorf("insert default reviewers: %w", err)
	}
	return nil
}

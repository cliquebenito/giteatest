package convert

import (
	issuesModel "code.gitea.io/gitea/models/issues"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/routers/sbt/response"
)

// ToComment конвертирует issuesModel.Comment в response.Comment
func ToComment(ctx *context.Context, comment *issuesModel.Comment) *response.Comment {
	reactions := make([]*response.Reaction, len(comment.Reactions))

	for i, issueReaction := range comment.Reactions {
		reactions[i] = ToReaction(issueReaction)
	}

	return &response.Comment{
		ID:             comment.ID,
		Type:           comment.Type.String(),
		Poster:         ToUser(ctx, comment.Poster, nil),
		Body:           comment.Content,
		Attachments:    ToAttachments(comment.Attachments),
		Patch:          comment.Patch,
		Reactions:      reactions,
		TreePath:       comment.TreePath,
		Created:        comment.CreatedUnix.AsTime(),
		Updated:        comment.UpdatedUnix.AsTime(),
		IsOwner:        comment.ShowRole.HasRole("Owner"),
		IsAuthor:       comment.ShowRole.HasRole("Poster"),
		IsCollaborator: comment.ShowRole.HasRole("Writer"),
	}
}

func IssueToComments(ctx *context.Context, issue *issuesModel.Issue) []*response.Comment {
	comments := issue.Comments

	responseComments := make([]*response.Comment, len(comments))

	editedHistoryCountMap, _ := issuesModel.QueryIssueContentHistoryEditedCountMap(ctx, issue.ID)

	var comment *response.Comment

	for i := range comments {
		comment = ToComment(ctx, comments[i])
		comment.EditCounts = editedHistoryCountMap[comment.ID]
		responseComments[i] = comment
	}

	return responseComments
}

func ToReaction(reaction *issuesModel.Reaction) *response.Reaction {
	return &response.Reaction{
		Content:      reaction.Type,
		UserId:       reaction.User.ID,
		UserName:     reaction.User.Name,
		UserFullName: reaction.User.FullName,
	}
}

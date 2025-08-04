package webhook

import (
	"net/http"
	"strings"

	webhook2 "code.gitea.io/gitea/models/webhook"
	webhook_module "code.gitea.io/gitea/modules/webhook"
	"code.gitea.io/gitea/routers/api/v2/models"
)

// ToWebHookConvertor converts models.CreateHookOption to webhook.Webhook
func ToWebHookConvertor(request *models.CreateHookOption) *webhook2.Webhook {
	return &webhook2.Webhook{
		OwnerID:     request.OwnerID,
		RepoID:      request.RepoID,
		URL:         request.Config.Url,
		ContentType: webhook2.ToHookContentType(request.Config.ContentType),
		Secret:      request.Config.Secret,
		HTTPMethod:  http.MethodPost,
		HookEvent: &webhook_module.HookEvent{
			ChooseEvents: true,
			HookEvents:   mapEvents(request.Events),
			BranchFilter: request.BranchFilter,
		},
		IsActive: *request.Active,
		Type:     request.Type,
	}
}

// mapEvents - отвечает за проставление bool флагов в структуре HookEvents
func mapEvents(events []string) webhook_module.HookEvents {
	eventMap := make(map[string]bool)
	for _, e := range events {
		eventMap[strings.ToLower(e)] = true
	}

	return webhook_module.HookEvents{
		Create:                    eventMap["create"],
		Delete:                    eventMap["delete"],
		Fork:                      eventMap["fork"],
		Issues:                    eventMap["issues"],
		IssueAssign:               eventMap["issue_assign"],
		IssueLabel:                eventMap["issue_label"],
		IssueMilestone:            eventMap["issue_milestone"],
		IssueComment:              eventMap["issue_comment"],
		Push:                      eventMap["push"],
		PullRequest:               eventMap["pull_request"],
		PullRequestAssign:         eventMap["pull_request_assign"],
		PullRequestLabel:          eventMap["pull_request_label"],
		PullRequestMilestone:      eventMap["pull_request_milestone"],
		PullRequestComment:        eventMap["pull_request_comment"],
		PullRequestReview:         eventMap["pull_request_review"],
		PullRequestSync:           eventMap["pull_request_sync"],
		PullRequestReviewApproved: eventMap["pull_request_review_approved"],
		PullRequestReviewRejected: eventMap["pull_request_review_rejected"],
		PullRequestReviewComment:  eventMap["pull_request_review_comment"],
		Wiki:                      eventMap["wiki"],
		Repository:                eventMap["repository"],
		Release:                   eventMap["release"],
		Package:                   eventMap["package"],
	}
}

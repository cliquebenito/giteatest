package convert

import (
	"code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/routers/sbt/response"
)

// ToCollaboration конвертирует в ДТО соавтора с правами доступа
func ToCollaboration(ctx *context.Context, collaborator *repo.Collaborator) response.Collaboration {
	return response.Collaboration{
		User: ToUser(ctx, collaborator.User, nil),
		AccessMode: &response.AccessMode{
			Created: collaborator.Collaboration.CreatedUnix.AsTime(),
			Mode:    collaborator.Collaboration.Mode.String(),
			Updated: collaborator.Collaboration.UpdatedUnix.AsTime(),
		},
	}
}

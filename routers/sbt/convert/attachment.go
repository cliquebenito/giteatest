package convert

import (
	repoModel "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/routers/sbt/response"
)

// ToAttachment конвертирует models.Attachment в response.Attachment
func ToAttachment(a *repoModel.Attachment) *response.Attachment {
	return &response.Attachment{
		Name:          a.Name,
		Created:       a.CreatedUnix.AsTime(),
		DownloadCount: a.DownloadCount,
		Size:          a.Size,
		UploaderId:    a.UploaderID,
		UUID:          a.UUID,
	}
}

func ToAttachments(attachments []*repoModel.Attachment) []*response.Attachment {
	converted := make([]*response.Attachment, 0, len(attachments))
	for _, attachment := range attachments {
		converted = append(converted, ToAttachment(attachment))
	}
	return converted
}

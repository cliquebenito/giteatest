package response

import (
	"time"
)

type Attachment struct {
	Name          string    `json:"name"`
	Size          int64     `json:"size"`
	DownloadCount int64     `json:"downloadCount"`
	Created       time.Time `json:"createdAt"`
	UploaderId    int64     `json:"uploaderId"`
	UUID          string    `json:"uuid"`
}

type AttachmentUuid struct {
	UUID string `json:"uuid"`
}

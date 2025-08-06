package audit

import (
	"strconv"

	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/sbt/audit"
)

type AuditRequiredParams struct {
	DoerName      string
	DoerID        string
	RemoteAddress string
}

// NewRequiredAuditParams will check the required audit params
func NewRequiredAuditParams(ctx *context.Context) AuditRequiredParams {
	result := AuditRequiredParams{}

	if ctx.Doer != nil {
		result.DoerName = ctx.Doer.Name
		result.DoerID = strconv.FormatInt(ctx.Doer.ID, 10)
	} else {
		result.DoerName = audit.EmptyRequiredField
		result.DoerID = audit.EmptyRequiredField
	}

	if ctx.Req != nil {
		result.RemoteAddress = ctx.RealIP()
	} else {
		result.RemoteAddress = audit.EmptyRequiredField
	}
	return result
}

// NewRequiredAuditParamsFromApiContext will check the required audit params
func NewRequiredAuditParamsFromApiContext(ctx *context.APIContext) AuditRequiredParams {
	result := AuditRequiredParams{}
	result.DoerName = audit.EmptyRequiredField
	result.DoerID = audit.EmptyRequiredField
	result.RemoteAddress = audit.EmptyRequiredField

	if ctx.Doer != nil {
		result.DoerName = ctx.Doer.Name
		result.DoerID = strconv.FormatInt(ctx.Doer.ID, 10)
	}

	if ctx.Req != nil {
		result.RemoteAddress = ctx.RealIP()
	}
	return result
}

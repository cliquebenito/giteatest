package auth

import (
	"errors"
	"net/http"
	"strconv"

	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/sbt/audit"
	"code.gitea.io/gitea/services/auth/iamprivileger"
)

func handleIAMCommonErrors(ctx *context.Context, err error) {
	auditParams := map[string]string{}
	var doerName, doerID, remoteAddress string
	var orgID int64

	if ctx.Doer != nil {
		doerName = ctx.Doer.Name
		doerID = strconv.FormatInt(ctx.Doer.ID, 10)
	} else {
		doerName = audit.EmptyRequiredField
		doerID = audit.EmptyRequiredField
	}

	if ctx.Req != nil {
		remoteAddress = ctx.Req.RemoteAddr
	} else {
		remoteAddress = audit.EmptyRequiredField
	}

	if ctx.Org != nil && ctx.Org.Organization != nil {
		orgID = ctx.Org.Organization.ID
	}

	if ctx.Req != nil && ctx.Req.URL != nil {
		auditParams["request_url"] = ctx.Req.URL.RequestURI()
	}

	if iamErr := new(ErrorParseIAMJWT); errors.As(err, &iamErr) {
		ctx.HTML(http.StatusForbidden, "status/403")
		log.Error(iamErr.Error())
		auditParams["error"] = "Error has occurred while parsing the JWT"
		audit.CreateAndSendEvent(audit.UnauthorizedRequestEvent, doerName, doerID, audit.StatusFailure, remoteAddress, auditParams)

		return
	}

	if iamErr := new(ErrorParsePrivileges); errors.As(err, &iamErr) {
		ctx.HTML(http.StatusForbidden, "status/403")
		log.Error(iamErr.Error())
		auditParams["error"] = "Error has occurred while parsing the privileges"
		audit.CreateAndSendEvent(audit.UnauthorizedRequestEvent, doerName, doerID, audit.StatusFailure, remoteAddress, auditParams)

		return
	}

	if iamErr := new(iamprivileger.ErrorTenantNotFound); errors.As(err, &iamErr) {
		ctx.HTML(http.StatusNotFound, "status/404")
		log.Error(iamErr.Error())
		auditParams["error"] = "Error has occurred while searching tenant"
		audit.CreateAndSendEvent(audit.UnauthorizedRequestEvent, doerName, doerID, audit.StatusFailure, remoteAddress, auditParams)

		return
	}

	if iamErr := new(iamprivileger.ErrorOrganizationNotFound); errors.As(err, &iamErr) {
		ctx.HTML(http.StatusNotFound, "status/404")
		log.Error(iamErr.Error())
		auditParams["error"] = "Error has occurred while searching organization"
		if orgID != 0 {
			auditParams["org_id"] = strconv.FormatInt(orgID, 10)
		}
		audit.CreateAndSendEvent(audit.UnauthorizedRequestEvent, doerName, doerID, audit.StatusFailure, remoteAddress, auditParams)

		return
	}

	if iamErr := new(ErrorApplyPrivileges); errors.As(err, &iamErr) {
		ctx.HTML(http.StatusInternalServerError, "status/500")
		log.Error(iamErr.Error())
		auditParams["error"] = "Error has occurred while applying privileges"
		if orgID != 0 {
			auditParams["org_id"] = strconv.FormatInt(orgID, 10)
		}
		audit.CreateAndSendEvent(audit.UnauthorizedRequestEvent, doerName, doerID, audit.StatusFailure, remoteAddress, auditParams)

		return
	}

	return
}

func iamProxyAuth(authMethod Method) func(ctx *context.Context) {
	return func(ctx *context.Context) {
		ar, err := authShared(ctx.Base, ctx.Session, authMethod)
		if err != nil {
			handleIAMCommonErrors(ctx, err)
			return
		}

		ctx.Doer = ar.Doer
		ctx.IsSigned = ar.Doer != nil
		ctx.IsBasicAuth = ar.IsBasicAuth
		if ctx.Doer == nil {
			// ensure the session uid is deleted
			_ = ctx.Session.Delete("uid")
		}
	}
}

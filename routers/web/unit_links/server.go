package unit_links

import (
	goCtx "context"
	"fmt"
	"net/http"
	"strconv"

	"code.gitea.io/gitea/models/gitnames"
	"code.gitea.io/gitea/models/unit_links"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/sbt/audit"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/structs"
	"code.gitea.io/gitea/modules/web"
	"code.gitea.io/gitea/routers/private/task_tracker_client"
)

type unitLinksDB interface {
	GetUnitLinks(ctx goCtx.Context, pullRequestID int64) (unit_links.AllUnitLinks, error)
}

type taskTrackerClient interface {
	GetDescriptions(
		ctx goCtx.Context,
		codes []gitnames.UnitCode,
	) (task_tracker_client.GetDescriptionsResponse, error)
}

type Server struct {
	unitLinksDB
	taskTrackerClient

	unitBaseURL    string
	iamUnitBaseURL string
}

func NewServer(unitLinksDB unitLinksDB, client taskTrackerClient, unitBaseURL, iamUnitBaseURL string) Server {
	return Server{unitLinksDB: unitLinksDB, taskTrackerClient: client, unitBaseURL: unitBaseURL, iamUnitBaseURL: iamUnitBaseURL}
}

func (s Server) GetUnitLinksWithDescription(ctx *context.Context) {
	var doerName, doerID, remoteAddress string

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

	auditParams := map[string]string{}

	if ctx.Repo != nil && ctx.Repo.Repository != nil {
		auditParams["repository"] = ctx.Repo.Repository.Name
		auditParams["repository_id"] = strconv.FormatInt(ctx.Repo.Repository.ID, 10)
		auditParams["owner"] = ctx.Repo.Repository.OwnerName
	}

	request := web.GetForm(ctx).(*structs.GetUnitLinksWithDescriptionRequest)
	if err := request.Validate(); err != nil {
		auditParams["error"] = "Error occurred while trying to validation"
		audit.CreateAndSendEvent(audit.UnitLinksRequestCreateEvent, doerName, doerID, audit.StatusFailure, remoteAddress, auditParams)
		errDescription := fmt.Sprintf("validation error: %s", err.Error())
		log.Error("validation error: %s", errDescription)
		ctx.JSON(http.StatusBadRequest, errDescription)

		return
	}

	var (
		response      structs.GetUnitLinksWithDescriptionResponse
		pullRequestID = request.PullRequestID
	)

	unitLinks, err := s.unitLinksDB.GetUnitLinks(ctx, pullRequestID)
	if err != nil {
		errDescription := fmt.Sprintf("get unit links error: %s", err.Error())
		log.Error("get unit links error: %s", errDescription)
		ctx.JSON(http.StatusInternalServerError, errDescription)
		auditParams["error"] = "Error has occurred while getting unit links"
		audit.CreateAndSendEvent(audit.UnitLinksRequestCreateEvent, doerName, doerID, audit.StatusFailure, remoteAddress, auditParams)

		return
	}

	if unitLinks.IsEmpty() {
		log.Debug("pull request '%d' has no unit links", pullRequestID)
		ctx.JSON(http.StatusOK, response)
		auditParams["error"] = "Empty unit links"
		audit.CreateAndSendEvent(audit.UnitLinksRequestCreateEvent, doerName, doerID, audit.StatusFailure, remoteAddress, auditParams)

		return
	}

	var codes []gitnames.UnitCode
	for _, unitLink := range unitLinks {
		codes = append(codes, gitnames.UnitCode{Code: unitLink.ToUnitID})
	}

	getDescriptionResponse, err := s.taskTrackerClient.GetDescriptions(ctx, codes)
	if err != nil {
		errDescription := fmt.Sprintf("get description http request: %d, error: %s", pullRequestID, err.Error())
		log.Error("http call: %s", errDescription)
		ctx.JSON(http.StatusInternalServerError, errDescription)
		auditParams["error"] = "Error has occurred while description http request"
		audit.CreateAndSendEvent(audit.UnitLinksRequestCreateEvent, doerName, doerID, audit.StatusFailure, remoteAddress, auditParams)

		return
	}
	response, err = s.enrichUnitLinksWithDescription(codes, getDescriptionResponse)
	if err != nil {
		errDescription := fmt.Sprintf("enrich unit links: %s", err.Error())
		log.Error(errDescription)
		ctx.JSON(http.StatusInternalServerError, errDescription)
		auditParams["error"] = "Error has occurred while enrich unit links"
		audit.CreateAndSendEvent(audit.UnitLinksRequestCreateEvent, doerName, doerID, audit.StatusFailure, remoteAddress, auditParams)

		return
	}
	audit.CreateAndSendEvent(audit.UnitLinksRequestCreateEvent, doerName, doerID, audit.StatusSuccess, remoteAddress, auditParams)

	ctx.JSON(http.StatusOK, response)
}

const (
	notFoundStatus                = "not found"
	descriptionInconsistentStatus = "description inconsistent"
)

func (s Server) enrichUnitLinksWithDescription(
	codes []gitnames.UnitCode,
	model task_tracker_client.GetDescriptionsResponse,
) (structs.GetUnitLinksWithDescriptionResponse, error) {
	unitCodeToResp := map[gitnames.UnitCode]task_tracker_client.GetDescriptionsContent{}

	for _, unit := range model.Content {
		unitCodeToResp[gitnames.UnitCode{Code: unit.Unit.Code}] = unit
	}

	var response structs.GetUnitLinksWithDescriptionResponse
	for _, code := range codes {
		resp, exists := unitCodeToResp[code]
		if !exists {
			unitErr := structs.GetDescriptionError{Code: code.Code, Description: notFoundStatus}
			response.Errors = append(response.Errors, unitErr)

			continue
		}

		converted, err := s.convertTaskTrackerModel(resp)
		if err != nil {
			errDescription := fmt.Sprintf("%s: %s", descriptionInconsistentStatus, err.Error())
			unitErr := structs.GetDescriptionError{Code: code.Code, Description: errDescription}
			response.Errors = append(response.Errors, unitErr)

			continue
		}

		response.Units = append(response.Units, converted)
	}

	return response, nil
}

func (s Server) convertTaskTrackerModel(
	unitContent task_tracker_client.GetDescriptionsContent,
) (structs.GetDescriptionUnit, error) {
	url := s.unitBaseURL
	if setting.IAM.Enabled && setting.OneWork.Enabled {
		url = s.iamUnitBaseURL
	}

	unit := structs.GetDescriptionUnit{
		Code: unitContent.Unit.Code,
		Name: unitContent.Unit.Summary,
		URL:  fmt.Sprintf("%s/%s", url, unitContent.Unit.Code),
	}

	for _, attribute := range unitContent.AttributesAndValues {
		switch attribute.Attribute.Code {
		case task_tracker_client.PriorityAttributeName:
			unit.Priority = attribute.Value.Name
		case task_tracker_client.StatusAttributeName:
			unit.Status = attribute.Value.Name
		}
	}

	return unit, nil
}

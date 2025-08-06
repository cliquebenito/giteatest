package task_tracker_client

import (
	"bytes"
	"code.gitea.io/gitea/models/unit_links"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"code.gitea.io/gitea/modules/log"
)

// SendAddPullRequestLinks отправить событие о привязке юнитов в TaskTracker
func (c TaskTrackerClient) SendAddPullRequestLinks(
	ctx context.Context,
	unitLinks unit_links.AllPayloadToAddOrDeletePr,
	userName string,
	pullRequestID int64,
	pullRequestURL string,
) error {
	var unitCodes []string

	for _, unitLink := range unitLinks {
		unitCodes = append(unitCodes, unitLink.ToUnitID)
	}

	request := AddPullRequestLinkRequest{
		PrID:     pullRequestID,
		PrURL:    pullRequestURL,
		PrStatus: unitLinks[0].FromUnitStatus,

		UnitCodes: unitCodes,
		UserLogin: userName,
	}

	requestBody, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("marshal add request: %w", err)
	}

	methodPath := fmt.Sprintf("%s/pull_request/add", c.baseURL)

	if err = c.doNewRequest(ctx, requestBody, http.MethodPatch, methodPath); err != nil {
		log.Error("Error has occurred while doing add pull request")
		return fmt.Errorf("run add request: %w", err)
	}

	return nil
}

// SendDeletePullRequestLinks отправить событие об отвязке юнитов в TaskTracker
func (c TaskTrackerClient) SendDeletePullRequestLinks(
	ctx context.Context,
	unitLinks unit_links.AllPayloadToAddOrDeletePr,
	userName string,
	pullRequestID int64,
) error {
	var unitCodes []string

	for _, unitLink := range unitLinks {
		unitCodes = append(unitCodes, unitLink.ToUnitID)
	}

	request := DeletePullRequestLinkRequest{
		PrID:      pullRequestID,
		UnitCodes: unitCodes,
		UserLogin: userName,
	}

	requestBody, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("marshal delete request: %w", err)
	}

	methodPath := fmt.Sprintf("%s/pull_request/delete", c.baseURL)

	log.Debug("unit_linker: delete unit_links request: %s", string(requestBody))

	if err = c.doNewRequest(ctx, requestBody, http.MethodPost, methodPath); err != nil {
		log.Error("Error has occurred while doing delete request")
		return fmt.Errorf("run delete request: %w", err)
	}

	log.Debug("unit_linker: delete unit_links request: success")

	return nil
}

// SendUpdatePullRequestStatus отправляем запрос в task tracker об обновлении статуса pr
func (c TaskTrackerClient) SendUpdatePullRequestStatus(
	ctx context.Context,
	payloads unit_links.AllPayloadToAddOrDeletePr,
	userName string,
	pullRequestID int64,
) error {
	if len(payloads) == 0 {
		return nil
	}

	request := UpdatePullRequestStatusRequest{
		UserLogin: userName,
		PrStatus:  payloads[0].FromUnitStatus,
	}
	requestBody, err := json.Marshal(request)
	if err != nil {
		log.Error("Error has occurred marshal payload: %v", err)
		return fmt.Errorf("marshal update request: %w", err)
	}

	methodPath := fmt.Sprintf("%s/pull_request/%d/update", c.baseURL, pullRequestID)

	if err = c.doNewRequest(ctx, requestBody, http.MethodPatch, methodPath); err != nil {
		log.Error("Error has occurred while doing update request: %v", err)
		return fmt.Errorf("run update request: %w", err)
	}

	return nil
}

func (c TaskTrackerClient) doNewRequest(ctx context.Context, requestBody []byte, method, methodPath string) error {
	request, err := http.NewRequestWithContext(ctx, method, methodPath, bytes.NewReader(requestBody))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	request.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.token))
	request.Header.Add("Content-Type", "application/json")

	response, err := c.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", response.StatusCode)
	}

	return nil
}

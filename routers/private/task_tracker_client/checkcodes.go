package task_tracker_client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"code.gitea.io/gitea/models/gitnames"
	"code.gitea.io/gitea/modules/log"
)

// CheckCodes проверяет наличие кодов в TaskTracker
func (c TaskTrackerClient) CheckCodes(ctx context.Context, codes []gitnames.UnitCode) (CheckCodesResponse, error) {
	var requestModel CheckCodesRequest
	for _, code := range codes {
		requestModel.Codes = append(requestModel.Codes, code.Code)
	}

	requestBody, err := json.Marshal(requestModel)
	if err != nil {
		return CheckCodesResponse{}, fmt.Errorf("marshal request: %w", err)
	}

	log.Debug("unit_linker: check codes request body: %s", string(requestBody))

	methodPath := fmt.Sprintf("%s/unit/search", c.baseURL)

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, methodPath, bytes.NewReader(requestBody))
	if err != nil {
		return CheckCodesResponse{}, fmt.Errorf("create request: %w", err)
	}

	request.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.token))
	request.Header.Add("Content-Type", "application/json")

	response, err := c.httpClient.Do(request)
	if err != nil {
		return CheckCodesResponse{}, fmt.Errorf("do request: %w", err)
	}

	if response.StatusCode != http.StatusOK {
		return CheckCodesResponse{}, fmt.Errorf("unexpected status code: %d", response.StatusCode)
	}

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return CheckCodesResponse{}, fmt.Errorf("read response: %w", err)
	}

	defer func() {
		if err = response.Body.Close(); err != nil {
			log.Error("close response body: %v", err)
		}
	}()

	var responseModel CheckCodesResponse
	if err = json.Unmarshal(responseBody, &responseModel); err != nil {
		return CheckCodesResponse{}, fmt.Errorf("unmarshal response: %w", err)
	}

	log.Debug("unit_linker: check codes response body: %s", string(responseBody))

	return responseModel, nil
}

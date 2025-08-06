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

// GetDescriptions получает расширенное описание юнитов TaskTracker
func (c TaskTrackerClient) GetDescriptions(ctx context.Context, codes []gitnames.UnitCode) (GetDescriptionsResponse, error) {
	request := GetDescriptionsRequest{
		Page:       getDefaultPage(),
		Attributes: getDefaultAttributes(),
	}

	for _, code := range codes {
		request.Filters.Units = append(request.Filters.Units, code.Code)
	}

	methodPath := fmt.Sprintf("%s/unit/find", c.baseURL)

	response, err := c.doRequest(ctx, request, methodPath)
	if err != nil {
		return GetDescriptionsResponse{}, fmt.Errorf("do request: %w", err)
	}

	return response, nil
}

func (c TaskTrackerClient) doRequest(ctx context.Context, requestModel GetDescriptionsRequest, methodPath string) (GetDescriptionsResponse, error) {
	requestBody, err := json.Marshal(requestModel)
	if err != nil {
		return GetDescriptionsResponse{}, fmt.Errorf("marshal request: %w", err)
	}

	log.Debug("unit_linker: get description request body: %s", string(requestBody))

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, methodPath, bytes.NewReader(requestBody))
	if err != nil {
		return GetDescriptionsResponse{}, fmt.Errorf("create request: %w", err)
	}

	request.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.token))
	request.Header.Add("Content-Type", "application/json")

	response, err := c.httpClient.Do(request)
	if err != nil {
		return GetDescriptionsResponse{}, fmt.Errorf("do request: %w", err)
	}

	if response.StatusCode != http.StatusOK {
		return GetDescriptionsResponse{}, fmt.Errorf("unexpected status code: %d", response.StatusCode)
	}

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return GetDescriptionsResponse{}, fmt.Errorf("read response: %w", err)
	}

	defer func() {
		if err = response.Body.Close(); err != nil {
			log.Error("close response body: %v", err)
		}
	}()

	var responseModel GetDescriptionsResponse
	if err = json.Unmarshal(responseBody, &responseModel); err != nil {
		return GetDescriptionsResponse{}, fmt.Errorf("unmarshal response: %w", err)
	}

	log.Debug("unit_linker: get description response body: %s", string(responseBody))

	return responseModel, nil
}

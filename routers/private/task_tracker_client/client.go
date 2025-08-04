package task_tracker_client

import (
	"net/http"
)

// TaskTrackerClientName имя client для получения конфига Sec Man
const TaskTrackerClientName = "task_tracker"

// TaskTrackerClient клиент для работы с TaskTracker
type TaskTrackerClient struct {
	httpClient *http.Client
	baseURL    string
	token      string
}

func New(baseURL, token string, httpclient *http.Client) TaskTrackerClient {
	return TaskTrackerClient{
		httpClient: httpclient,
		baseURL:    baseURL,
		token:      token,
	}
}

package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	api "renovate-operator/api/v1alpha1"
	crdmanager "renovate-operator/internal/crdManager"
	"renovate-operator/internal/types"
	"testing"

	"github.com/go-logr/logr"
)

func TestAzureDevOpsWebhook_Integration(t *testing.T) {
	tests := []struct {
		name             string
		payload          AzureDevOpsEvent
		namespace        string
		job              string
		expectedStatus   int
		expectedMessage  string
		shouldCallUpdate bool
		expectedProject  string
	}{
		{
			name: "valid push event triggers schedule",
			payload: AzureDevOpsEvent{
				EventType: "git.push",
				Resource: AzureDevOpsEventResource{
					Repository: AzureDevOpsRepository{
						Name:    "hetzner-k8s-platform",
						Project: AzureDevOpsProject{Name: "MIA-Guild-DevOps"},
					},
				},
			},
			namespace:        "renovate",
			job:              "platform",
			expectedStatus:   http.StatusAccepted,
			expectedMessage:  "renovate job scheduled",
			shouldCallUpdate: true,
			expectedProject:  "MIA-Guild-DevOps/hetzner-k8s-platform",
		},
		{
			name: "non-push event is ignored",
			payload: AzureDevOpsEvent{
				EventType: "git.pullrequest.created",
				Resource: AzureDevOpsEventResource{
					Repository: AzureDevOpsRepository{
						Name:    "some-repo",
						Project: AzureDevOpsProject{Name: "MyProject"},
					},
				},
			},
			namespace:        "renovate",
			job:              "platform",
			expectedStatus:   http.StatusOK,
			expectedMessage:  "event ignored",
			shouldCallUpdate: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updateCalled := false
			mockManager := &mockWebhookManager{
				updateProjectStatusFunc: func(ctx context.Context, project string, jobId crdmanager.RenovateJobIdentifier, status *types.RenovateStatusUpdate) error {
					updateCalled = true
					if project != tt.expectedProject {
						t.Errorf("expected project %s, got %s", tt.expectedProject, project)
					}
					if jobId.Name != tt.job {
						t.Errorf("expected job name %s, got %s", tt.job, jobId.Name)
					}
					if jobId.Namespace != tt.namespace {
						t.Errorf("expected namespace %s, got %s", tt.namespace, jobId.Namespace)
					}
					if status.Status != api.JobStatusScheduled {
						t.Errorf("expected status %s, got %s", api.JobStatusScheduled, status.Status)
					}
					if status.Priority != 1 {
						t.Errorf("expected priority 1, got %d", status.Priority)
					}
					return nil
				},
			}

			server := &Server{
				manager: mockManager,
				logger:  logr.Discard(),
			}

			body, err := json.Marshal(tt.payload)
			if err != nil {
				t.Fatalf("failed to marshal payload: %v", err)
			}

			url := "/webhook/v1/azure?namespace=" + tt.namespace + "&job=" + tt.job
			req := httptest.NewRequest(http.MethodPost, url, bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			server.azureDevOpsWebhook(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			var response map[string]string
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("failed to unmarshal response: %v", err)
			}
			if message, ok := response["message"]; ok {
				if message != tt.expectedMessage {
					t.Errorf("expected message %q, got %q", tt.expectedMessage, message)
				}
			}
			if updateCalled != tt.shouldCallUpdate {
				t.Errorf("expected updateCalled=%v, got %v", tt.shouldCallUpdate, updateCalled)
			}
		})
	}
}

func TestAzureDevOpsWebhook_MissingQueryParams(t *testing.T) {
	tests := []struct {
		name           string
		namespace      string
		job            string
		expectedStatus int
	}{
		{name: "missing namespace", namespace: "", job: "test-job", expectedStatus: http.StatusBadRequest},
		{name: "missing job", namespace: "default", job: "", expectedStatus: http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := &Server{manager: &mockWebhookManager{}, logger: logr.Discard()}

			payload := AzureDevOpsEvent{
				EventType: "git.push",
				Resource: AzureDevOpsEventResource{
					Repository: AzureDevOpsRepository{
						Name:    "repo",
						Project: AzureDevOpsProject{Name: "project"},
					},
				},
			}
			body, _ := json.Marshal(payload)

			url := "/webhook/v1/azure?"
			if tt.namespace != "" {
				url += "namespace=" + tt.namespace + "&"
			}
			if tt.job != "" {
				url += "job=" + tt.job
			}
			req := httptest.NewRequest(http.MethodPost, url, bytes.NewReader(body))
			w := httptest.NewRecorder()
			server.azureDevOpsWebhook(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestAzureDevOpsWebhook_InvalidJSON(t *testing.T) {
	server := &Server{manager: &mockWebhookManager{}, logger: logr.Discard()}

	req := httptest.NewRequest(http.MethodPost, "/webhook/v1/azure?namespace=default&job=test", bytes.NewReader([]byte("not json")))
	w := httptest.NewRecorder()
	server.azureDevOpsWebhook(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

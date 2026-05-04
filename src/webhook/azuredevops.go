package webhook

import (
	"encoding/json"
	"net/http"

	api "renovate-operator/api/v1alpha1"
	crdmanager "renovate-operator/internal/crdManager"
	"renovate-operator/internal/types"
)

type AzureDevOpsEvent struct {
	EventType string                       `json:"eventType"`
	Resource  AzureDevOpsEventResource     `json:"resource"`
}

type AzureDevOpsEventResource struct {
	Repository AzureDevOpsRepository `json:"repository"`
}

type AzureDevOpsRepository struct {
	Name    string             `json:"name"`
	Project AzureDevOpsProject `json:"project"`
}

type AzureDevOpsProject struct {
	Name string `json:"name"`
}

func (s *Server) azureDevOpsWebhook(w http.ResponseWriter, r *http.Request) {
	var payload AzureDevOpsEvent
	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		s.logger.Error(err, "failed to decode Azure DevOps webhook payload")
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "failed to decode payload"})
		return
	}

	valid, reason := isValidAzureDevOpsEvent(&payload)
	if !valid {
		s.logger.Info("ignoring Azure DevOps webhook event", "reason", reason)
		s.writeJSON(w, http.StatusOK, map[string]string{"message": "event ignored", "reason": reason})
		return
	}

	namespace := r.URL.Query().Get("namespace")
	job := r.URL.Query().Get("job")
	if namespace == "" || job == "" {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing namespace or job query parameter"})
		return
	}

	project := payload.Resource.Repository.Project.Name + "/" + payload.Resource.Repository.Name

	s.logger.Info("received Azure DevOps push event", "repository", project, "priority", 1)
	err = s.manager.UpdateProjectStatus(
		r.Context(),
		project,
		crdmanager.RenovateJobIdentifier{
			Name:      job,
			Namespace: namespace,
		},
		&types.RenovateStatusUpdate{
			Status:   api.JobStatusScheduled,
			Priority: 1,
		},
	)
	if err != nil {
		s.logger.Error(err, "Failed to process Azure DevOps webhook for repo", "repo", project)
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "failed to process webhook"})
		return
	}

	s.writeJSON(w, http.StatusAccepted, map[string]string{"message": "renovate job scheduled", "repository": project})
}

func isValidAzureDevOpsEvent(payload *AzureDevOpsEvent) (bool, string) {
	if payload.EventType != "git.push" {
		return false, "event type is not git.push"
	}
	if payload.Resource.Repository.Name == "" {
		return false, "missing repository name in event"
	}
	if payload.Resource.Repository.Project.Name == "" {
		return false, "missing project name in event"
	}
	return true, ""
}

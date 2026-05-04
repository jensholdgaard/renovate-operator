package webhook

// Contract tests validate that Azure DevOps webhook event payloads
// (captured in testdata/) deserialize correctly into our Go types.
//
// These fixtures are based on the Azure DevOps Service Hooks event documentation:
// https://learn.microsoft.com/en-us/azure/devops/service-hooks/events?view=azure-devops
//
// Testing approach inspired by:
// - FluxCD notification-controller (thin interface + hand-written fakes)
//   https://github.com/fluxcd/notification-controller
//   Licensed under Apache-2.0
// - mcdafydd/go-azuredevops (testdata fixtures for webhook events)
//   https://github.com/mcdafydd/go-azuredevops
//   Licensed under BSD-3-Clause

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// fullGitPushEvent represents the complete Azure DevOps git.push webhook payload.
// Decoded with DisallowUnknownFields so the test fails if the fixture contains
// fields not covered by this struct — forcing us to keep the struct in sync with
// the real API response shape.
//
// Schema: https://learn.microsoft.com/en-us/azure/devops/service-hooks/events?view=azure-devops#git.push
type fullGitPushEvent struct {
	SubscriptionID string `json:"subscriptionId"`
	NotificationID int    `json:"notificationId"`
	ID             string `json:"id"`
	EventType      string `json:"eventType"`
	PublisherID    string `json:"publisherId"`
	Message        struct {
		Text string `json:"text"`
	} `json:"message"`
	DetailedMessage struct {
		Text string `json:"text"`
	} `json:"detailedMessage"`
	Resource struct {
		Commits []struct {
			CommitID  string `json:"commitId"`
			Author    struct {
				Name  string `json:"name"`
				Email string `json:"email"`
				Date  string `json:"date"`
			} `json:"author"`
			Committer struct {
				Name  string `json:"name"`
				Email string `json:"email"`
				Date  string `json:"date"`
			} `json:"committer"`
			Comment string `json:"comment"`
			URL     string `json:"url"`
		} `json:"commits"`
		RefUpdates []struct {
			Name        string `json:"name"`
			OldObjectID string `json:"oldObjectId"`
			NewObjectID string `json:"newObjectId"`
		} `json:"refUpdates"`
		Repository struct {
			ID      string `json:"id"`
			Name    string `json:"name"`
			URL     string `json:"url"`
			Project struct {
				ID         string `json:"id"`
				Name       string `json:"name"`
				State      string `json:"state"`
				Visibility string `json:"visibility"`
			} `json:"project"`
			DefaultBranch string `json:"defaultBranch"`
			RemoteURL     string `json:"remoteUrl"`
		} `json:"repository"`
		PushedBy struct {
			DisplayName string `json:"displayName"`
			UniqueName  string `json:"uniqueName"`
			ID          string `json:"id"`
		} `json:"pushedBy"`
		PushID int    `json:"pushId"`
		Date   string `json:"date"`
		URL    string `json:"url"`
	} `json:"resource"`
	ResourceVersion    string `json:"resourceVersion"`
	ResourceContainers struct {
		Collection struct {
			ID string `json:"id"`
		} `json:"collection"`
		Account struct {
			ID string `json:"id"`
		} `json:"account"`
		Project struct {
			ID string `json:"id"`
		} `json:"project"`
	} `json:"resourceContainers"`
	CreatedDate string `json:"createdDate"`
}

func loadWebhookTestdata(t *testing.T, filename string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", filename))
	if err != nil {
		t.Fatalf("failed to read testdata/%s: %v", filename, err)
	}
	return data
}

// unmarshalWebhookStrict decodes JSON with DisallowUnknownFields so that any
// field present in the fixture but missing from the target struct causes a failure.
func unmarshalWebhookStrict(t *testing.T, data []byte, v interface{}) {
	t.Helper()
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		t.Fatalf("strict unmarshal failed (unknown field in fixture?): %v", err)
	}
}

func TestContract_GitPushWebhook_FullPayload(t *testing.T) {
	data := loadWebhookTestdata(t, "azure_devops_git_push.json")

	var event fullGitPushEvent
	unmarshalWebhookStrict(t, data, &event)

	// Verify required fields present in the full payload
	if event.EventType == "" {
		t.Error("contract violation: eventType must not be empty")
	}
	if event.EventType != "git.push" {
		t.Errorf("contract violation: expected eventType 'git.push', got %q", event.EventType)
	}
	if event.Resource.Repository.Name == "" {
		t.Error("contract violation: resource.repository.name must not be empty")
	}
	if event.Resource.Repository.Project.Name == "" {
		t.Error("contract violation: resource.repository.project.name must not be empty")
	}
	if len(event.Resource.Commits) == 0 {
		t.Error("contract violation: resource.commits must not be empty for a push event")
	}
	if len(event.Resource.RefUpdates) == 0 {
		t.Error("contract violation: resource.refUpdates must not be empty for a push event")
	}
	if event.Resource.PushedBy.DisplayName == "" {
		t.Error("contract violation: resource.pushedBy.displayName must not be empty")
	}
}

// TestContract_GitPushWebhook_OurStruct verifies that the real webhook payload
// deserializes correctly into our actual AzureDevOpsEvent struct — the one used
// in production code. This uses standard Unmarshal (not strict) because our
// production struct intentionally only captures the subset of fields we route on.
func TestContract_GitPushWebhook_OurStruct(t *testing.T) {
	data := loadWebhookTestdata(t, "azure_devops_git_push.json")

	var event AzureDevOpsEvent
	if err := json.Unmarshal(data, &event); err != nil {
		t.Fatalf("failed to unmarshal into AzureDevOpsEvent: %v", err)
	}

	// Validate our struct captured the fields we route on
	if event.EventType != "git.push" {
		t.Errorf("expected eventType 'git.push', got %q", event.EventType)
	}
	if event.Resource.Repository.Name != "my-application" {
		t.Errorf("expected repository name 'my-application', got %q", event.Resource.Repository.Name)
	}
	if event.Resource.Repository.Project.Name != "MyProject" {
		t.Errorf("expected project name 'MyProject', got %q", event.Resource.Repository.Project.Name)
	}

	// Validate through the same validation function used in production
	valid, reason := isValidAzureDevOpsEvent(&event)
	if !valid {
		t.Errorf("expected valid event, got invalid: %s", reason)
	}
}

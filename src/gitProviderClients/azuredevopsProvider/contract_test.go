package azuredevopsProvider

// Contract tests validate that the Azure DevOps REST API response shapes
// (captured in testdata/) deserialize correctly into our Go types.
//
// These fixtures are based on the Azure DevOps REST API 7.2 documentation:
// https://learn.microsoft.com/en-us/rest/api/azure/devops/?view=azure-devops-rest-7.2
//
// Re-record the testdata fixtures periodically (or when bumping API versions)
// by capturing real responses from the ADO API. The tests will fail if the
// response structure drifts from what our code expects.
//
// Testing approach inspired by:
// - FluxCD notification-controller (thin interface + hand-written fakes)
//   https://github.com/fluxcd/notification-controller
//   Licensed under Apache-2.0
// - mcdafydd/go-azuredevops (httptest server pattern from google/go-github)
//   https://github.com/mcdafydd/go-azuredevops
//   Licensed under BSD-3-Clause

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// repositoryResponse represents the full Azure DevOps Git Repository response.
// Uses DisallowUnknownFields decoding so that if the testdata fixture contains
// fields not covered by this struct, the test fails — forcing us to update the
// struct when the API response shape changes.
//
// Schema: https://learn.microsoft.com/en-us/rest/api/azure/devops/git/repositories/get?view=azure-devops-rest-7.2
type repositoryResponse struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	URL     string `json:"url"`
	Project struct {
		ID         string `json:"id"`
		Name       string `json:"name"`
		State      string `json:"state"`
		Visibility string `json:"visibility"`
	} `json:"project"`
	DefaultBranch    string              `json:"defaultBranch"`
	Size             int                 `json:"size"`
	RemoteURL        string              `json:"remoteUrl"`
	SshURL           string              `json:"sshUrl"`
	IsFork           bool                `json:"isFork"`
	ParentRepository *parentRepositoryOf `json:"parentRepository"`
}

type parentRepositoryOf struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Project struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"project"`
}

func loadTestdata(t *testing.T, filename string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", filename))
	if err != nil {
		t.Fatalf("failed to read testdata/%s: %v", filename, err)
	}
	return data
}

// unmarshalStrict decodes JSON with DisallowUnknownFields so that any field
// present in the fixture but missing from the target struct causes a test failure.
func unmarshalStrict(t *testing.T, data []byte, v interface{}) {
	t.Helper()
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		t.Fatalf("strict unmarshal failed (unknown field in fixture?): %v", err)
	}
}

func TestContract_GetRepository(t *testing.T) {
	data := loadTestdata(t, "get_repository.json")

	var repo repositoryResponse
	unmarshalStrict(t, data, &repo)

	// Verify required fields are populated (contract assertions)
	if repo.ID == "" {
		t.Error("contract violation: repository ID must not be empty")
	}
	if repo.Name == "" {
		t.Error("contract violation: repository name must not be empty")
	}
	if repo.Project.ID == "" {
		t.Error("contract violation: project ID must not be empty")
	}
	if repo.Project.Name == "" {
		t.Error("contract violation: project name must not be empty")
	}
	if repo.DefaultBranch == "" {
		t.Error("contract violation: defaultBranch must not be empty")
	}

	// Verify the field our client actually uses
	if repo.IsFork != false {
		t.Errorf("expected isFork=false in non-fork fixture, got %v", repo.IsFork)
	}
	if repo.ParentRepository != nil {
		t.Error("expected parentRepository=nil for non-fork repository")
	}
}

func TestContract_GetRepository_Fork(t *testing.T) {
	data := loadTestdata(t, "get_repository_fork.json")

	var repo repositoryResponse
	unmarshalStrict(t, data, &repo)

	if !repo.IsFork {
		t.Error("contract violation: expected isFork=true for fork fixture")
	}
	if repo.ParentRepository == nil {
		t.Fatal("contract violation: fork must have parentRepository")
	}
	if repo.ParentRepository.ID == "" {
		t.Error("contract violation: parentRepository.id must not be empty")
	}
	if repo.ParentRepository.Name == "" {
		t.Error("contract violation: parentRepository.name must not be empty")
	}
}

// TestContract_GetRepository_MinimalDecode verifies that the minimal struct
// used in our actual client (only isFork field) correctly deserializes from
// a full API response. This catches if the field path or JSON tag breaks.
func TestContract_GetRepository_MinimalDecode(t *testing.T) {
	data := loadTestdata(t, "get_repository_fork.json")

	// This is the exact struct used in AzureDevOpsClient.IsFork()
	var minimal struct {
		IsFork bool `json:"isFork"`
	}
	if err := json.Unmarshal(data, &minimal); err != nil {
		t.Fatalf("failed to unmarshal into minimal struct: %v", err)
	}
	if !minimal.IsFork {
		t.Error("minimal decode failed: expected isFork=true")
	}
}

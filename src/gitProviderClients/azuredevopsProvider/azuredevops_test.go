package azuredevopsProvider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestIsFork_True(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/MyProject/_apis/git/repositories/my-repo" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"isFork": true})
	}))
	defer server.Close()

	client := &AzureDevOpsClient{Endpoint: server.URL, Token: "test-pat", HTTPClient: server.Client()}
	isFork, err := client.IsFork(context.Background(), "MyProject/my-repo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !isFork {
		t.Error("expected isFork=true, got false")
	}
}

func TestIsFork_False(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"isFork": false})
	}))
	defer server.Close()

	client := &AzureDevOpsClient{Endpoint: server.URL, Token: "test-pat", HTTPClient: server.Client()}
	isFork, err := client.IsFork(context.Background(), "MyProject/my-repo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if isFork {
		t.Error("expected isFork=false, got true")
	}
}

func TestIsFork_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"message":"unauthorized"}`))
	}))
	defer server.Close()

	client := &AzureDevOpsClient{Endpoint: server.URL, Token: "bad-token", HTTPClient: server.Client()}
	_, err := client.IsFork(context.Background(), "MyProject/my-repo")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "azure devops API returned status 401") {
		t.Errorf("expected error to contain status 401, got %q", err.Error())
	}
}

func TestIsFork_BasicAuth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if !ok {
			t.Error("expected Basic auth, got none")
		}
		if username != "" {
			t.Errorf("expected empty username, got %q", username)
		}
		if password != "my-pat-token" {
			t.Errorf("expected password 'my-pat-token', got %q", password)
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"isFork": false})
	}))
	defer server.Close()

	client := &AzureDevOpsClient{Endpoint: server.URL, Token: "my-pat-token", HTTPClient: server.Client()}
	_, err := client.IsFork(context.Background(), "Project/Repo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestIsFork_ProjectParsing(t *testing.T) {
	tests := []struct {
		name         string
		project      string
		expectedPath string
		expectErr    bool
	}{
		{
			name:         "standard format",
			project:      "MyProject/my-repo",
			expectedPath: "/MyProject/_apis/git/repositories/my-repo",
		},
		{
			name:         "project with spaces gets encoded",
			project:      "My Project/my-repo",
			expectedPath: "/My Project/_apis/git/repositories/my-repo",
		},
		{
			name:      "missing slash",
			project:   "just-a-name",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedPath string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				capturedPath = r.URL.Path
				_ = json.NewEncoder(w).Encode(map[string]interface{}{"isFork": false})
			}))
			defer server.Close()

			client := &AzureDevOpsClient{Endpoint: server.URL, Token: "pat", HTTPClient: server.Client()}
			_, err := client.IsFork(context.Background(), tt.project)

			if tt.expectErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if capturedPath != tt.expectedPath {
				t.Errorf("expected path %q, got %q", tt.expectedPath, capturedPath)
			}
		})
	}
}

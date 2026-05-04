package azuredevopsProvider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"renovate-operator/gitProviderClients"
)

// AzureDevOpsClient implements GitProviderClient for the Azure DevOps REST API.
type AzureDevOpsClient struct {
	Endpoint   string
	Token      string
	HTTPClient *http.Client
}

func (c *AzureDevOpsClient) IsFork(ctx context.Context, project string) (bool, error) {
	// project format: "ProjectName/RepoName"
	parts := strings.SplitN(project, "/", 2)
	if len(parts) != 2 {
		return false, fmt.Errorf("invalid Azure DevOps project format %q, expected \"Project/Repo\"", project)
	}
	adoProject := url.PathEscape(parts[0])
	repoName := url.PathEscape(parts[1])

	apiURL := fmt.Sprintf("%s/%s/_apis/git/repositories/%s?api-version=7.1", c.Endpoint, adoProject, repoName)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return false, err
	}
	req.SetBasicAuth("", c.Token)
	req.Header.Set("Accept", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return false, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return false, fmt.Errorf("azure devops API returned status %d for %s: %s", resp.StatusCode, project, string(body))
	}

	var repo struct {
		IsFork bool `json:"isFork"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&repo); err != nil {
		return false, fmt.Errorf("failed to decode Azure DevOps API response for %s: %w", project, err)
	}
	return repo.IsFork, nil
}

func (c *AzureDevOpsClient) SearchReposByTopic(_ context.Context, _ string) ([]gitProviderClients.Repository, error) {
	return nil, fmt.Errorf("searching repositories by topic is not supported by Azure DevOps provider")
}

func (c *AzureDevOpsClient) ListRepoWebhooks(_ context.Context, _, _ string) ([]gitProviderClients.Webhook, error) {
	return nil, fmt.Errorf("listing webhooks is not supported by Azure DevOps provider")
}

func (c *AzureDevOpsClient) CreateRepoWebhook(_ context.Context, _, _ string, _ gitProviderClients.CreateWebhookOptions) (*gitProviderClients.Webhook, error) {
	return nil, fmt.Errorf("creating webhooks is not supported by Azure DevOps provider")
}

func (c *AzureDevOpsClient) DeleteRepoWebhook(_ context.Context, _, _ string, _ int64) error {
	return fmt.Errorf("deleting webhooks is not supported by Azure DevOps provider")
}

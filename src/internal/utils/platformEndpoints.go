package utils

import (
	api "renovate-operator/api/v1alpha1"
)

func GetPlatformAndEndpoint(provider *api.RenovateProvider) (string, string) {
	if provider == nil {
		return "", ""
	}
	endpoint := provider.Endpoint
	if endpoint == "" {
		switch provider.Name {
		case "github":
			endpoint = "https://api.github.com"
		case "gitlab":
			endpoint = "https://gitlab.com/api/v4"
		case "azure":
			// No default: Azure DevOps URLs require an organization segment
			// (e.g., https://dev.azure.com/{org}). The user must configure
			// provider.endpoint explicitly with their organization.
			endpoint = ""
		}
	}
	return provider.Name, endpoint
}

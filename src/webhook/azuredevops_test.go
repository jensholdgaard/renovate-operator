package webhook

import "testing"

func TestAzureDevOpsEventValidation(t *testing.T) {
	tests := []struct {
		name    string
		payload AzureDevOpsEvent
		valid   bool
		reason  string
	}{
		{
			name: "valid git.push event",
			payload: AzureDevOpsEvent{
				EventType: "git.push",
				Resource: AzureDevOpsEventResource{
					Repository: AzureDevOpsRepository{
						Name:    "hetzner-k8s-platform",
						Project: AzureDevOpsProject{Name: "MIA-Guild-DevOps"},
					},
				},
			},
			valid:  true,
			reason: "",
		},
		{
			name: "invalid event type",
			payload: AzureDevOpsEvent{
				EventType: "git.pullrequest.created",
				Resource: AzureDevOpsEventResource{
					Repository: AzureDevOpsRepository{
						Name:    "some-repo",
						Project: AzureDevOpsProject{Name: "MyProject"},
					},
				},
			},
			valid:  false,
			reason: "event type is not git.push",
		},
		{
			name: "missing repository name",
			payload: AzureDevOpsEvent{
				EventType: "git.push",
				Resource: AzureDevOpsEventResource{
					Repository: AzureDevOpsRepository{
						Name:    "",
						Project: AzureDevOpsProject{Name: "MyProject"},
					},
				},
			},
			valid:  false,
			reason: "missing repository name in event",
		},
		{
			name: "missing project name",
			payload: AzureDevOpsEvent{
				EventType: "git.push",
				Resource: AzureDevOpsEventResource{
					Repository: AzureDevOpsRepository{
						Name:    "my-repo",
						Project: AzureDevOpsProject{Name: ""},
					},
				},
			},
			valid:  false,
			reason: "missing project name in event",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid, reason := isValidAzureDevOpsEvent(&tt.payload)
			if valid != tt.valid || reason != tt.reason {
				t.Errorf("expected valid=%v, reason=%q; got valid=%v, reason=%q", tt.valid, tt.reason, valid, reason)
			}
		})
	}
}

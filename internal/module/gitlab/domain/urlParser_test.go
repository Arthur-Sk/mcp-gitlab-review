package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseMRURL(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		wantProject string
		wantIID     int
		wantErr     bool
	}{
		{
			name:        "standard GitLab URL",
			url:         "https://gitlab.com/mygroup/myproject/-/merge_requests/42",
			wantProject: "mygroup/myproject",
			wantIID:     42,
		},
		{
			name:        "nested subgroup URL",
			url:         "https://gitlab.com/org/team/subteam/project/-/merge_requests/123",
			wantProject: "org/team/subteam/project",
			wantIID:     123,
		},
		{
			name:        "self-hosted GitLab URL",
			url:         "https://git.example.com/company/backend/-/merge_requests/7",
			wantProject: "company/backend",
			wantIID:     7,
		},
		{
			name:    "missing merge_requests segment",
			url:     "https://gitlab.com/mygroup/myproject/-/issues/42",
			wantErr: true,
		},
		{
			name:    "missing MR IID",
			url:     "https://gitlab.com/mygroup/myproject/-/merge_requests",
			wantErr: true,
		},
		{
			name:    "non-numeric MR IID",
			url:     "https://gitlab.com/mygroup/myproject/-/merge_requests/abc",
			wantErr: true,
		},
		{
			name:    "too short path",
			url:     "https://gitlab.com/merge_requests/42",
			wantErr: true,
		},
		{
			name:    "empty URL",
			url:     "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ref, err := ParseMRURL(tt.url)

			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, ref)

				return
			}

			require.NoError(t, err)
			require.NotNil(t, ref)
			assert.Equal(t, tt.wantProject, ref.ProjectPath)
			assert.Equal(t, tt.wantIID, ref.MRIID)
		})
	}
}

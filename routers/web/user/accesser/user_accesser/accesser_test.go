package user_accesser

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"code.gitea.io/gitea/modules/structs"
	"code.gitea.io/gitea/routers/web/user/accesser"
)

var ctx = context.Background()

func Test_requestAccesser_IsReadAccessGranted(t *testing.T) {
	tests := []struct {
		name    string
		request accesser.UserAccessRequest
		want    bool
	}{
		// private
		{
			request: accesser.UserAccessRequest{
				DoerID: 1, TargetUserID: 1,
				DoerTenantIDs: []string{"default"}, TargetTenantIDs: []string{"default"},
				Visibility: structs.VisibleTypePrivate,
			},
			want: true,
		},
		{
			request: accesser.UserAccessRequest{
				DoerID: 1, TargetUserID: 2,
				DoerTenantIDs: []string{"default"}, TargetTenantIDs: []string{"default"},
				Visibility: structs.VisibleTypePrivate,
			},
			want: false,
		},
		{
			request: accesser.UserAccessRequest{
				DoerID: 1, TargetUserID: 2,
				DoerTenantIDs: []string{"default"}, TargetTenantIDs: []string{"default-1"},
				Visibility: structs.VisibleTypePrivate,
			},
			want: false,
		},

		// limited
		{
			request: accesser.UserAccessRequest{
				DoerID: 1, TargetUserID: 1,
				DoerTenantIDs: []string{"default"}, TargetTenantIDs: []string{"default"},
				Visibility: structs.VisibleTypeLimited,
			},
			want: true,
		},
		{
			request: accesser.UserAccessRequest{
				DoerID: 1, TargetUserID: 2,
				DoerTenantIDs: []string{"default"}, TargetTenantIDs: []string{"default"},
				Visibility: structs.VisibleTypeLimited,
			},
			want: true,
		},
		{
			request: accesser.UserAccessRequest{
				DoerID: 1, TargetUserID: 2,
				DoerTenantIDs: []string{"default"}, TargetTenantIDs: []string{"default-1"},
				Visibility: structs.VisibleTypeLimited,
			},
			want: false,
		},

		// public
		{
			request: accesser.UserAccessRequest{
				DoerID: 1, TargetUserID: 1,
				DoerTenantIDs: []string{"default"}, TargetTenantIDs: []string{"default"},
				Visibility: structs.VisibleTypePublic,
			},
			want: true,
		},
		{
			request: accesser.UserAccessRequest{
				DoerID: 1, TargetUserID: 2,
				DoerTenantIDs: []string{"default"}, TargetTenantIDs: []string{"default"},
				Visibility: structs.VisibleTypePublic,
			},
			want: true,
		},
		{
			request: accesser.UserAccessRequest{
				DoerID: 1, TargetUserID: 2,
				DoerTenantIDs: []string{"default"}, TargetTenantIDs: []string{"default-1"},
				Visibility: structs.VisibleTypePublic,
			},
			want: true,
		},
	}

	a := requestAccesser{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := a.IsReadAccessGranted(ctx, tt.request)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

//go:build !correct

package custom

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"code.gitea.io/gitea/models/organization/custom/mocks"

	"github.com/stretchr/testify/assert"
	"xorm.io/builder"
)

func TestInsertCustomPrivilegesForTeam(t *testing.T) {
	mockDB := mocks.NewDbEngine(t)
	dbCustomPrivilege := customPrivilegeDB{
		engine: mockDB,
	}
	tests := []struct {
		name        string
		input       []ScTeamCustomPrivilege
		mockReturn  int64
		mockError   error
		expectError bool
	}{
		{
			name: "Successful insert",
			input: []ScTeamCustomPrivilege{
				{TeamName: "Team1", RepositoryID: 1, AllRepositories: false, CustomPrivileges: "changeBranch"},
			},
			mockReturn:  1,
			mockError:   nil,
			expectError: false,
		},
		{
			name: "Database error",
			input: []ScTeamCustomPrivilege{
				{TeamName: "Team2", RepositoryID: 2, AllRepositories: true, CustomPrivileges: "viewBranch"},
			},
			mockReturn:  0,
			mockError:   fmt.Errorf("DB error"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB.On("Insert", tt.input).Return(tt.mockReturn, tt.mockError)
			err := dbCustomPrivilege.InsertCustomPrivilegesForTeam(tt.input)

			assert.Equal(t, err == nil, tt.expectError)
		})
	}
}

func TestGetCustomPrivilegesByTeam(t *testing.T) {
	mockDB := mocks.NewDbEngine(t)
	dbCustomPrivilege := customPrivilegeDB{
		engine: mockDB,
	}
	tests := []struct {
		name         string
		teamName     string
		mockResponse []*ScTeamCustomPrivilege
		mockError    error
		expectError  bool
	}{
		{
			name:         "Success - found privileges",
			teamName:     "teamA",
			mockResponse: []*ScTeamCustomPrivilege{{ID: 1, TeamName: "teamA"}},
			mockError:    nil,
			expectError:  false,
		},
		{
			name:         "Success - no privileges found",
			teamName:     "teamB",
			mockResponse: []*ScTeamCustomPrivilege{},
			mockError:    nil,
			expectError:  false,
		},
		{
			name:         "Error - database failure",
			teamName:     "teamC",
			mockResponse: nil,
			mockError:    errors.New("DB error"),
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB.On("Where", builder.Eq{"team_name": tt.teamName}).Return(dbCustomPrivilege)
			mockDB.On("Find", tt.mockResponse).Return(tt.mockError)

			result, err := dbCustomPrivilege.GetCustomPrivilegesByTeam(tt.teamName)

			assert.Equal(t, tt.mockResponse, result)
			assert.Equal(t, err == nil, tt.expectError)
		})
	}
}

func TestDeleteCustomPrivilegesByParams(t *testing.T) {
	mockDB := mocks.NewDbEngine(t)
	ctx := context.Background()
	dbCustomPrivilege := customPrivilegeDB{
		engine: mockDB,
	}

	tests := []struct {
		name          string
		privilege     ScTeamCustomPrivilege
		mockError     error
		mockReturn    int64
		expectedError bool
	}{
		{
			name: "Successful deletion",
			privilege: ScTeamCustomPrivilege{
				TeamName:         "teamA",
				RepositoryID:     1,
				AllRepositories:  false,
				CustomPrivileges: "changeBranch",
			},
			mockError:     nil,
			mockReturn:    1,
			expectedError: true,
		},
		{
			name: "Error during deletion",
			privilege: ScTeamCustomPrivilege{
				TeamName:         "teamB",
				RepositoryID:     2,
				AllRepositories:  true,
				CustomPrivileges: "approvePR",
			},
			mockError:     errors.New("deleting custom privileges"),
			mockReturn:    1,
			expectedError: false,
		},
		{
			name: "Empty privilege data",
			privilege: ScTeamCustomPrivilege{
				TeamName:         "",
				RepositoryID:     0,
				AllRepositories:  false,
				CustomPrivileges: "",
			},
			mockReturn:    1,
			mockError:     nil,
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB.On("Delete", &tt.privilege).Return(tt.mockReturn, tt.mockError)

			err := dbCustomPrivilege.DeleteCustomPrivilegesByParams(ctx, tt.privilege)

			assert.Equal(t, err == nil, tt.expectedError)
		})
	}
}

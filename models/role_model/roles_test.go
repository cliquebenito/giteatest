package role_model

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetRoleByString(t *testing.T) {
	cases := []struct {
		name           string
		roleString     string
		expectedRole   Role
		expectedExists bool
	}{
		{
			name:           "role is not in allRoles",
			roleString:     "role",
			expectedRole:   0,
			expectedExists: false,
		},
		{
			name:           "owner is in allRoles",
			roleString:     "owner",
			expectedRole:   OWNER,
			expectedExists: true,
		},
		{
			name:           "manager is in allRoles",
			roleString:     "manager",
			expectedRole:   MANAGER,
			expectedExists: true,
		},
		{
			name:           "writer is in allRoles",
			roleString:     "writer",
			expectedRole:   WRITER,
			expectedExists: true,
		},
		{
			name:           "reader is in allRoles",
			roleString:     "reader",
			expectedRole:   READER,
			expectedExists: true,
		},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			role, exists := GetRoleByString(test.roleString)
			assert.Equal(t, test.expectedRole, role)
			assert.Equal(t, test.expectedExists, exists)
		})
	}
}
func TestConcurrentMapAccess(t *testing.T) {
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 1000; j++ {
				_ = GetAllRoles()
			}
		}()
	}

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 1000; j++ {
				DeleteRole(Role(j%5 + 1))
			}
		}()
	}

	wg.Wait()
}

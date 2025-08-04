//go:build !correct

package auth

import (
	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/organization"
	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/tenant"
	"code.gitea.io/gitea/models/unittest"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/structs"
	"code.gitea.io/gitea/services/org"
	"code.gitea.io/gitea/services/user"
	"github.com/stretchr/testify/assert"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

const Zero = 0 // нулевой	элемент

var (
	UserCount               = 6                                   // общее количество пользователей
	OrganizationCountOfUser = 2                                   // количество оранизаций необходимых для создания
	PrivilegesGroupCount    = 2                                   // количество групп привилегий необходимых для создания
	RepositoryCount         = 2                                   // количество оранизаций необходимых для создания
	AdminCount              = UserCount / OrganizationCountOfUser // количество пользователей - администраторов
	TenantCount             = AdminCount                          // количество тенантов необходимых для создания
)

func TestMain(m *testing.M) {
	unittest.MainTest(m, &unittest.TestOptions{
		GiteaRootPath: filepath.Join("..", ".."),
	})
}

// userInfo возвращает объект пользователя.
func userInfo(t *testing.T, isAdmin bool, id int64) *user_model.User {
	return &user_model.User{
		ID:                      id,
		Name:                    t.Name() + strconv.Itoa(int(id)),
		Email:                   t.Name() + "@" + strconv.Itoa(int(id)) + ".com",
		Passwd:                  ";p['////..-++']",
		IsAdmin:                 isAdmin,
		Theme:                   setting.UI.DefaultTheme,
		MustChangePassword:      false,
		AllowCreateOrganization: true,
	}
}

// CreateUsers позволяет создать пользователей.
func GetUsers(t *testing.T) (users []*user_model.User) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	us, err := user_model.GetAllUsers()
	if err != nil {
		assert.Error(t, err)
	}
	for i := len(us); i < UserCount+len(us); i++ {
		var isAdmin bool
		if i < AdminCount+len(us) {
			isAdmin = true
		}
		createdUser := userInfo(t, isAdmin, int64(i))
		assert.NoError(t, user_model.CreateUser(createdUser))
		unittest.AssertExistsIf(t, true, createdUser)
		users = append(users, createdUser)
	}
	return
}

// DeleteUsers удаляет пользователя
func DeleteUsers(t *testing.T) {
	us, err := user_model.GetAllUsers()
	if err != nil {
		assert.Error(t, err)
	}
	for _, userr := range us {
		assert.NoError(t, user.DeleteUser(db.DefaultContext, &user_model.User{ID: userr.ID}, false))
	}
}

func GetOrgs(t *testing.T, users []*user_model.User) (orgs []*organization.Organization) {
	for _, targetUser := range users {
		for i := range OrganizationCountOfUser {
			rule := structs.VisibleTypeLimited
			postfix := "limit"

			if i%2 == Zero {
				rule = structs.VisibleTypePrivate
				postfix = "private"
			}

			orgName := strings.Join([]string{targetUser.Name, postfix, "org"}, "_")
			createdOrg := CreateOrganization(t, orgName, rule, targetUser)
			unittest.AssertExistsAndLoadBean(t, createdOrg)
			orgs = append(orgs, createdOrg)
		}
	}
	return
}

func CreateOrganization(t *testing.T, orgName string, rule structs.VisibleType, owner *user_model.User) (org *organization.Organization) {
	if owner.CanCreateOrganization() {
		org = &organization.Organization{Name: orgName, Visibility: rule}
		assert.NoError(t, organization.CreateOrganization(org, owner))
		unittest.AssertExistsAndLoadBean(t, &user_model.User{Name: orgName, Type: user_model.UserTypeOrganization})
	}
	return
}

func DeleteOrganization(t *testing.T, orgs []*organization.Organization) {
	for _, delOrg := range orgs {
		if delOrg == nil {
			continue
		}
		orgOrg := organization.OrgFromUser(delOrg.AsUser())

		if err := org.DeleteOrganization(orgOrg); err != nil {

			if err != nil {
				assert.Error(t, err)
			}
		}
	}

}

type MockTenant struct {
	User         []*user_model.User
	Organization []*organization.Organization
	Repository   []*repo_model.Repository
	Tenant       []*tenant.ScTenant
}

// TestCheckPermissionUserMultiTenant тест направленный на проверку работы режима мультитенантности.
func TestCheckPermissionUserMultiTenant(t *testing.T) {
	var newTenant MockTenant
	newTenant.User = GetUsers(t)
	newTenant.Organization = GetOrgs(t, newTenant.User)
	DeleteOrganization(t, newTenant.Organization)
	DeleteUsers(t)
}

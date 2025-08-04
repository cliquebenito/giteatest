package admin

import (
	"net/http"

	"code.gitea.io/gitea/models/role_model"
	"code.gitea.io/gitea/models/tenant"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/log"
	audit "code.gitea.io/gitea/modules/sbt/audit/utils"
	"code.gitea.io/gitea/modules/web"
	"code.gitea.io/gitea/services/forms"
	"code.gitea.io/gitea/services/privileges"
)

type Server struct {
	processor privileges.PrivilegesProcessor
}

func NewPrivilegesServer(privileger privileges.PrivilegesProcessor) *Server {
	return &Server{
		processor: privileger,
	}
}

// ApplyPrvileges метод для назначения и удаления привилегий
func (s *Server) ApplyPrivileges(ctx *context.APIContext) {
	// swagger:operation POST /admin/privileges admin ApplyPrivileges
	// ---
	// summary: Apply privileges to users by granting or revoking specific roles
	// description: This endpoint applies privileges to users by either granting or revoking specific privilege groups within a tenant and project.
	// produces:
	// - application/json
	// parameters:
	// - name: body
	//   in: body
	//   description: Privileges to be applied (grant or revoke)
	//   required: true
	//   schema:
	//     type: object
	//     required:
	//       - apply_privilege_groups
	//     properties:
	//       apply_privilege_groups:
	//         type: object
	//         required:
	//           - grant
	//           - revoke
	//         properties:
	//           grant:
	//             type: array
	//             items:
	//               type: object
	//               properties:
	//                 user_key:
	//                   type: string
	//                   description: Key of the user to grant privileges to
	//                 privilege_groups:
	//                   type: array
	//                   items:
	//                     type: object
	//                     properties:
	//                       tenant_key:
	//                         type: string
	//                         description: Key of the tenant
	//                       project_key:
	//                         type: string
	//                         description: Key of the project
	//                       privilege_group:
	//                         type: string
	//                         description: Name of the privilege group to be granted
	//           revoke:
	//             type: array
	//             items:
	//               type: object
	//               properties:
	//                 user_key:
	//                   type: string
	//                   description: Key of the user to revoke privileges from
	//                 privilege_groups:
	//                   type: array
	//                   items:
	//                     type: object
	//                     properties:
	//                       tenant_key:
	//                         type: string
	//                         description: Key of the tenant
	//                       project_key:
	//                         type: string
	//                         description: Key of the project
	//                       privilege_group:
	//                         type: string
	//                         description: Name of the privilege group to be revoked
	// responses:
	//   "200":
	//     description: Privileges successfully applied
	//     schema:
	//       type: object
	//       properties:
	//         applied_status:
	//           type: object
	//           properties:
	//             granted:
	//               type: array
	//               items:
	//                 type: object
	//                 properties:
	//                   user_key:
	//                     type: string
	//                     format: string
	//                     description: ID of the user
	//                   privilege_groups:
	//                     type: array
	//                     items:
	//                       type: object
	//                       properties:
	//                         tenant_id:
	//                           type: string
	//                           description: ID of the tenant
	//                         project_id:
	//                           type: integer
	//                           format: int64
	//                           description: ID of the project
	//                         privilege_group:
	//                           type: string
	//                           description: Granted role
	//             revoked:
	//               type: array
	//               items:
	//                 type: object
	//                 properties:
	//                   user_key:
	//                     type: string
	//                     format: string
	//                     description: ID of the user
	//                   privilege_groups:
	//                     type: array
	//                     items:
	//                       type: object
	//                       properties:
	//                         tenant_id:
	//                           type: string
	//                           description: ID of the tenant
	//                         project_id:
	//                           type: integer
	//                           format: int64
	//                           description: ID of the project
	//                         privilege_group:
	//                           type: string
	//                           description: Revoked role
	//   "400":
	//     description: Bad request
	//     schema:
	//       type: object
	//       properties:
	//         errors:
	//           type: array
	//           items:
	//             type: string
	//           description: List of error messages
	//         message:
	//           type: string
	//           description: Method returning the error
	//         url:
	//           type: string
	//           description: Swagger documentation URL
	//   "500":
	//     description: Privileges applying error
	//     schema:
	//       type: object
	//       properties:
	//         applied_status:
	//           type: object
	//           properties:
	//             granted:
	//               type: array
	//               items:
	//                 type: object
	//                 properties:
	//                   user_key:
	//                     type: string
	//                     format: string
	//                     description: ID of the user
	//                   privilege_groups:
	//                     type: array
	//                     items:
	//                       type: object
	//                       properties:
	//                         tenant_id:
	//                           type: string
	//                           description: ID of the tenant
	//                         project_id:
	//                           type: integer
	//                           format: int64
	//                           description: ID of the project
	//                         privilege_group:
	//                           type: string
	//                           description: Granted role
	//             revoked:
	//               type: array
	//               items:
	//                 type: object
	//                 properties:
	//                   user_key:
	//                     type: string
	//                     format: string
	//                     description: ID of the user
	//                   privilege_groups:
	//                     type: array
	//                     items:
	//                       type: object
	//                       properties:
	//                         tenant_id:
	//                           type: string
	//                           description: ID of the tenant
	//                         project_id:
	//                           type: integer
	//                           format: int64
	//                           description: ID of the project
	//                         privilege_group:
	//                           type: string
	//                           description: Revoked role
	//         errors:
	//           type: object
	//           properties:
	//             grant:
	//               type: array
	//               items:
	//                 type: object
	//                 properties:
	//                   user_key:
	//                     type: string
	//                     format: string
	//                     description: ID of the user
	//                   privilege_groups:
	//                     type: array
	//                     items:
	//                       type: object
	//                       properties:
	//                         tenant_id:
	//                           type: string
	//                           description: ID of the tenant
	//                         project_id:
	//                           type: integer
	//                           format: int64
	//                           description: ID of the project
	//                         privilege_group:
	//                           type: string
	//                           description: Role that caused an error
	//                         error:
	//                           type: string
	//                           description: Error message
	//             revoke:
	//               type: array
	//               items:
	//                 type: object
	//                 properties:
	//                   user_key:
	//                     type: string
	//                     format: string
	//                     description: ID of the user
	//                   privilege_groups:
	//                     type: array
	//                     items:
	//                       type: object
	//                       properties:
	//                         tenant_id:
	//                           type: string
	//                           description: ID of the tenant
	//                         project_id:
	//                           type: integer
	//                           format: int64
	//                           description: ID of the project
	//                         privilege_group:
	//                           type: string
	//                           description: Role that caused an error
	//                         error:
	//                           type: string
	//                           description: Error message
	form := web.GetForm(ctx).(*forms.ApplyPrivilegeRequest)
	if form == nil {
		log.Error("Error has occurred while getting form")
		ctx.Error(http.StatusBadRequest, "", "empty request")
		return
	}
	if err := form.Validate(); err != nil {
		log.Debug("invalid form")
		ctx.Error(http.StatusBadRequest, "validation error", err)
		return
	}
	auditValues := audit.NewRequiredAuditParamsFromApiContext(ctx)
	result, err := s.processor.ApplyPrivilegesRequest(ctx, *form, auditValues)
	if err != nil {
		log.Error("Error has occurred while applying privileges: %v", err)
		ctx.JSON(http.StatusInternalServerError, result)
		return
	}
	ctx.JSON(http.StatusCreated, result)
}

// GetPrivileges метод для получения привилегий
func (s *Server) GetPrivileges(ctx *context.APIContext) {
	// swagger:operation GET /admin/privileges admin getPrivileges
	// ---
	// summary: Query privileges for users within tenants and projects
	// description: This endpoint retrieves privileges for specific users, tenants, and projects based on the provided filters.
	// produces:
	// - application/json
	// parameters:
	// - name: body
	//   in: body
	//   description: Array of filters for querying privileges
	//   required: true
	//   schema:
	//     type: array
	//     items:
	//       type: object
	//       properties:
	//         tenant_key:
	//           type: string
	//           description: Key of the tenant
	//         project_key:
	//           type: string
	//           description: Key of the project
	//         user_key:
	//           type: string
	//           description: Key of the user
	// responses:
	//   200:
	//     description: Privileges successfully retrieved
	//     schema:
	//       type: object
	//       properties:
	//         granted:
	//           type: array
	//           description: List of privileges granted to users
	//           items:
	//             type: object
	//             properties:
	//               user_key:
	//                 type: string
	//                 description: Key of the user
	//               privilege_groups:
	//                 type: array
	//                 description: Privilege groups assigned to the user
	//                 items:
	//                   type: object
	//                   properties:
	//                     tenant_key:
	//                       type: string
	//                       description: Key of the tenant
	//                     project_key:
	//                       type: string
	//                       description: Key of the project
	//                     privilege_group:
	//                       type: string
	//                       description: Assigned privilege group
	//   "400":
	//     description: Bad request
	//     schema:
	//       type: object
	//       properties:
	//         errors:
	//           type: array
	//           items:
	//             type: string
	//           description: List of error messages
	//         message:
	//           type: string
	//           description: Method returning the error
	//         url:
	//           type: string
	//           description: Swagger documentation URL
	//   "500":
	//     description: Internal server error
	//     schema:
	//       type: object
	//       properties:
	//         errors:
	//           type: array
	//           items:
	//             type: string
	//           description: List of error messages
	//         message:
	//           type: string
	//           description: Method returning the error
	//         url:
	//           type: string
	//           description: Swagger documentation URL
	form := web.GetForm(ctx).(*forms.GetPrivilegesRequest)
	if form == nil {
		log.Error("Error has occurred while getting form")
		ctx.Error(http.StatusBadRequest, "", "empty request")
		return
	}

	if err := form.Validate(); err != nil {
		log.Debug("invalid form")
		ctx.Error(http.StatusBadRequest, "validation error", err)
		return
	}
	result, err := s.processor.GetPrivilegesRequest(ctx, *form)
	if err != nil {
		if tenant.IsTenantOrganizationsNotExists(err) {
			ctx.Error(http.StatusBadRequest, "Get privileges", err)
			log.Error("Error has occurred while getting privileges: %v", err)
			return
		}
		if role_model.IsErrNonExistentRole(err) {
			ctx.Error(http.StatusBadRequest, "Get privileges", err)
			log.Error("Error has occurred while getting role: %v", err)
		}
		log.Error("Error has occurred while getting privileges: %v", err)
		ctx.Error(http.StatusInternalServerError, "Get privileges", err)
		return
	}
	ctx.JSON(http.StatusOK, result)
}

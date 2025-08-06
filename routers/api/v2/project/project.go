package project

import (
	"net/http"

	"code.gitea.io/gitea/models/tenant"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/web"
	"code.gitea.io/gitea/services/forms"
	"code.gitea.io/gitea/services/project"
)

func CreateProjectRequest(ctx *context.APIContext) {
	// swagger:operation POST /projects/create createProject
	// ---
	// summary: Create a new project
	// description: This endpoint creates a new project within the specified tenant. It requires details such as tenant ID, organization name, organization key, description, and visibility level.
	// produces:
	// - application/json
	// parameters:
	// - name: body
	//   in: body
	//   description: Details of the organization to be created
	//   required: true
	//   schema:
	//     type: object
	//     required:
	//       - tenant_id
	//       - name
	//       - project_key
	//       - description
	//       - visibility
	//     properties:
	//       tenant_id:
	//         type: string
	//         description: ID of the tenant to which the organization belongs
	//       name:
	//         type: string
	//         description: Name of the organization
	//       project_key:
	//         type: string
	//         description: Unique key for the organization
	//       description:
	//         type: string
	//         description: Detailed description of the organization
	//       visibility:
	//         type: integer
	//         description: Visibility level of the organization
	// responses:
	//   200:
	//     description: Project details successfully retrieved
	//     schema:
	//       type: object
	//       properties:
	//         id:
	//           type: integer
	//           format: int64
	//           description: Unique identifier of the project
	//         name:
	//           type: string
	//           description: Name of the project
	//         project_key:
	//           type: string
	//           description: Unique key of the project
	//         visibility:
	//           type: integer
	//           description: Visibility level of the project
	//         uri:
	//           type: string
	//           description: Relative URI of the project
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

	form := web.GetForm(ctx).(*forms.CreateProjectRequest)
	if form == nil {
		log.Error("Error has occurred while getting form")
		ctx.Error(http.StatusBadRequest, "", "empty request")
		return
	}
	if err := form.Validate(); err != nil {
		log.Error("Error has occurred while validating form")
		ctx.Error(http.StatusBadRequest, "validation error", err)
		return
	}
	response, err := project.CreateProject(*ctx, *form)
	if err != nil {
		if tenant.IsProjectKeyAlreadyUsed(err) ||
			project.IsProjectNameAlreadyUsed(err) {
			log.Error("Error has occurred while creating project: %v", err)
			ctx.Error(http.StatusConflict, "", err)
			return
		}
		if tenant.IsTenantNotActive(err) ||
			tenant.IsTenantKeyNotExists(err) ||
			project.IsVisibilityIncorrect(err) {
			log.Error("Error has occurred while creating project: %v", err)
			ctx.Error(http.StatusBadRequest, "", err)
			return
		}
		if tenant.IsTenantKeyNotExists(err) {
			log.Error("Error has occurred while creating project: %v", err)
			ctx.Error(http.StatusNotFound, "", err)
			return
		}
		log.Error("Error has occurred while creating project: %v", err)
		ctx.JSON(http.StatusInternalServerError, err)
		return
	}
	ctx.JSON(http.StatusCreated, response)
}

func GetProjectRequest(ctx *context.APIContext) {
	// swagger:operation GET /projects getProject
	// ---
	// summary: Get project info
	// description: This endpoint retrieves information about a specific project based on the provided tenant key and project key.
	// produces:
	// - application/json
	// parameters:
	// - name: tenant_key
	//   in: query
	//   description: Unique key of the tenant
	//   required: true
	//   type: string
	// - name: project_key
	//   in: query
	//   description: Unique key of the project
	//   required: true
	//   type: string
	//
	// responses:
	//   "200":
	//     description: Project details
	//     schema:
	//       type: object
	//       properties:
	//         id:
	//           type: integer
	//           format: int64
	//           description: Unique identifier of the project
	//         name:
	//           type: string
	//           description: Name of the project
	//         project_key:
	//           type: string
	//           description: Unique key of the project
	//         visibility:
	//           type: integer
	//           description: Visibility level of the project
	//         uri:
	//           type: string
	//           description: Relative URI of the project
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

	var form forms.ProjectInfoRequest
	err := form.BindFromContext(ctx)
	if err != nil {
		log.Error("Error has occurred while binding form: %v", err)
		ctx.Error(http.StatusBadRequest, "", err)
		return
	}
	if err = form.Validate(); err != nil {
		log.Error("Error has occurred while validating form: %v", err)
		ctx.Error(http.StatusBadRequest, "validation error", err)
		return
	}

	response, err := project.GetProject(*ctx, form.TenantKey, form.ProjectKey)
	if err != nil {
		if tenant.IsTenantOrganizationsNotExists(err) {
			ctx.Error(http.StatusBadRequest, "Get Project", err)
			log.Error("Error has occurred while getting project: %v,project not exists", err)
			return
		}
		log.Error("Error has occurred while getting project: %v", err)
		ctx.JSON(http.StatusInternalServerError, err)
		return
	}
	ctx.JSON(http.StatusOK, response)
}

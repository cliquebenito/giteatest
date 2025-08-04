package user

import (
	"net/http"

	user2 "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/web"
	"code.gitea.io/gitea/routers/api/v2/models"
	"code.gitea.io/gitea/services/user"
)

// CreateUserRequest handler для создания юзера
func CreateUserRequest(ctx *context.APIContext) {
	// swagger:operation POST /admin/users createUser
	// ---
	// summary: Returns the new user
	// description: This endpoint is responsible for creating a user
	// tags:
	// - user
	// produces:
	// - application/json
	// parameters:
	// - name: body
	//   in: body
	//   description: The user to create
	//   required: true
	//   schema:
	//     type: object
	//     required:
	//       - user_key
	//       - name
	//       - email
	//       - full_name
	//     properties:
	//       user_key:
	//         type: string
	//         description: User login name
	//       name:
	//         type: string
	//         description: Name of the user
	//       email:
	//         type: string
	//         description: Email of the user
	//       full_name:
	//         type: string
	//         description: Full name of the user
	// responses:
	//   200:
	//     description: User created successfully
	//     schema:
	//       type: object
	//       properties:
	//         id:
	//           type: integer
	//           format: int64
	//           description: ID of the user
	//         name:
	//           type: string
	//           description: Name of the user
	//         full_name:
	//           type: string
	//           description: Full name of the user
	//         email:
	//           type: string
	//           description: Email of the user
	//         user_key:
	//           type: string
	//           description: User login name
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
	form := web.GetForm(ctx).(*models.CreateUserRequest)
	if form == nil {
		log.Error("Error has occurred while getting form")
		ctx.Error(http.StatusBadRequest, "", "empty request")
		return
	}
	if !form.Validate() {
		log.Error("Error has occurred while validating form")
		ctx.Error(http.StatusBadRequest, "", "validation error")
		return
	}
	response, err := user.CreateUserRequest(ctx, *form)
	if err != nil {
		if user2.IsLoginNameAlreadyUsed(err) ||
			user2.IsErrEmailAlreadyUsed(err) ||
			user2.IsErrUserAlreadyExist(err) {
			ctx.Error(http.StatusConflict, "", err)
			log.Error("Error has occurred while creating user: %v", err)
			return
		}
		log.Error("Error has occurred while creating user: %v", err)
		ctx.Error(http.StatusInternalServerError, "", err)
		return
	}
	ctx.JSON(http.StatusCreated, response)
}

// GetUserRequest handler для получения юзера
func GetUserRequest(ctx *context.APIContext) {
	// swagger:operation GET /admin/users getUser
	// ---
	// summary: Returns the user
	// description: This endpoint is responsible for getting user details
	// tags:
	// - user
	// produces:
	// - application/json
	// parameters:
	// - name: user_key
	//   in: query
	//   description: User login name
	//   required: true
	//   type: string
	//
	// responses:
	//   "200":
	//     description: User details
	//     schema:
	//       type: object
	//       properties:
	//         id:
	//           type: integer
	//           format: int64
	//           description: ID of the user
	//         name:
	//           type: string
	//           description: Name of the user
	//         user_key:
	//           type: string
	//           description: User login name
	//         full_name:
	//           type: string
	//           description: Full name of the user
	//         email:
	//           type: string
	//           description: Email of the user
	//         visibility:
	//           type: integer
	//           description: Visibility of the user
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
	var form models.UserInfoRequest
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
	response, err := user.GetUserRequest(ctx, form)
	if err != nil {
		if user2.IsErrLoginNameNotExist(err) {
			log.Error("Error has occurred while searching user: %v", err)
			ctx.Error(http.StatusBadRequest, "", err)
			return
		}
		log.Error("Error has occurred while getting user: %v", err)
		ctx.Error(http.StatusInternalServerError, "", err)
		return
	}
	ctx.JSON(http.StatusOK, response)
}

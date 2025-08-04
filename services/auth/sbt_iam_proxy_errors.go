package auth

import (
	"fmt"

	iampriveleges "code.gitea.io/gitea/modules/auth/iam/iamprivileges"
	"code.gitea.io/gitea/modules/auth/iam/iamtoken"
)

type ErrorIAMUserNotFound struct {
	token      iamtoken.IAMJWT
	privileges iampriveleges.SourceControlPrivilegesByTenant

	error
}

func NewErrorUserNotFound(
	token iamtoken.IAMJWT,
	privileges iampriveleges.SourceControlPrivilegesByTenant,
	err error,
) *ErrorIAMUserNotFound {
	return &ErrorIAMUserNotFound{token: token, privileges: privileges, error: err}
}

func (e *ErrorIAMUserNotFound) Error() string {
	return fmt.Sprintf("user from token not found: %s", e.error.Error())
}

type ErrorParseIAMJWT struct {
	error
}

func NewErrorParseIAMJWT(err error) *ErrorParseIAMJWT {
	return &ErrorParseIAMJWT{error: err}
}

func (e *ErrorParseIAMJWT) Error() string {
	return fmt.Sprintf("parse iam token: %s", e.error.Error())
}

func (e *ErrorParseIAMJWT) Unwrap() error {
	return e.error
}

type ErrorParsePrivileges struct {
	error
}

func NewErrorParsePrivileges(err error) *ErrorParsePrivileges {
	return &ErrorParsePrivileges{error: err}
}

func (e *ErrorParsePrivileges) Error() string {
	return fmt.Sprintf("parse iam privileges: %s", e.error.Error())
}

func (e *ErrorParsePrivileges) Unwrap() error {
	return e.error
}

type ErrorApplyPrivileges struct {
	error
}

func NewErrorApplyPrivileges(err error) *ErrorApplyPrivileges {
	return &ErrorApplyPrivileges{error: err}
}

func (e *ErrorApplyPrivileges) Error() string {
	return fmt.Sprintf("apply iam privileges: %s", e.error.Error())
}

func (e *ErrorApplyPrivileges) Unwrap() error {
	return e.error
}

type ErrorIncorrectTokenType struct{}

func NewErrorIncorrectTokenType() *ErrorIncorrectTokenType {
	return &ErrorIncorrectTokenType{}
}

func (e *ErrorIncorrectTokenType) Error() string {
	return fmt.Errorf("incorrect token type").Error()
}

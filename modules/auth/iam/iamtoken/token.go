package iamtoken

import (
	"github.com/golang-jwt/jwt/v4"
)

type SourceControlGlobalRole string

const (
	AdminRole SourceControlGlobalRole = "sc_admin"
	UserRole  SourceControlGlobalRole = "sc_user"
)

type IAMJWT struct {
	JWTToken *jwt.Token

	GlobalID string
	Name     string
	FullName string

	Email string

	Role SourceControlGlobalRole

	TenantName string
}

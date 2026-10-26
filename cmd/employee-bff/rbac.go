package main

import "github.com/orange-cat-investments/oci/internal/app/employeebff"

var (
	SessionAuthMiddleware         = employeebff.SessionAuthMiddleware
	RequireRoleMiddleware         = employeebff.RequireRoleMiddleware
	CSRFProtectMiddleware         = employeebff.CSRFProtectMiddleware
	InternalServiceAuthMiddleware = employeebff.InternalServiceAuthMiddleware
)

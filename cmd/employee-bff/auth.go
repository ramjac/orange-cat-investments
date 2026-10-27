package main

import "github.com/orange-cat-investments/oci/internal/app/employeebff"

type (
	EmployeeSession = employeebff.EmployeeSession
	SessionStore    = employeebff.SessionStore
	AuthHandler     = employeebff.AuthHandler
)

var (
	NewSessionStore     = employeebff.NewSessionStore
	NewAuthHandler      = employeebff.NewAuthHandler
	EmployeeFromContext = employeebff.EmployeeFromContext
	WithEmployeeSession = employeebff.WithEmployeeSession
	generateRandomToken = employeebff.GenerateRandomToken
	getCSRFCookie       = employeebff.GetCSRFCookie
	setCSRFCookie       = employeebff.SetCSRFCookie
)

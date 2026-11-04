import re

with open("internal/app/employeebff/production_wiring_test.go", "r") as f:
    code = f.read()

# We need to add session cookies to the employee and leave-requests routes in the test.
old_emp = """	// 1. Workforce Employees route
	reqEmp := httptest.NewRequest(http.MethodGet, "/api/v1/workforce/employees", nil)
	recEmp := httptest.NewRecorder()
	router.ServeHTTP(recEmp, reqEmp)
	assert.Equal(t, http.StatusOK, recEmp.Code)
	assert.True(t, mockBLL.listEmployeesCalled, "expected ListEmployees RPC to be invoked")

	// 2. Workforce Leave Requests route
	reqLeave := httptest.NewRequest(http.MethodGet, "/api/v1/workforce/leave-requests", nil)
	recLeave := httptest.NewRecorder()
	router.ServeHTTP(recLeave, reqLeave)
	assert.Equal(t, http.StatusOK, recLeave.Code)
	assert.True(t, mockBLL.listLeaveRequestsCalled, "expected ListLeaveRequests RPC to be invoked")"""

new_emp = """	// 1. Workforce Employees route (Requires authenticated employee session)
	sessStore := bff.SessionStore()
	sess, err := sessStore.Create("emp-test-1", "testuser@example.com", "Test User", "engineering", "Engineering")
	require.NoError(t, err)

	// Ensure missing session yields 401
	reqMissingAuth := httptest.NewRequest(http.MethodGet, "/api/v1/workforce/employees", nil)
	recMissingAuth := httptest.NewRecorder()
	router.ServeHTTP(recMissingAuth, reqMissingAuth)
	assert.Equal(t, http.StatusUnauthorized, recMissingAuth.Code)

	reqEmp := httptest.NewRequest(http.MethodGet, "/api/v1/workforce/employees", nil)
	reqEmp.AddCookie(&http.Cookie{Name: "__Host-employee-session", Value: sess.SessionID})
	recEmp := httptest.NewRecorder()
	router.ServeHTTP(recEmp, reqEmp)
	assert.Equal(t, http.StatusOK, recEmp.Code)
	assert.True(t, mockBLL.listEmployeesCalled, "expected ListEmployees RPC to be invoked")

	// 2. Workforce Leave Requests route
	reqLeave := httptest.NewRequest(http.MethodGet, "/api/v1/workforce/leave-requests", nil)
	reqLeave.AddCookie(&http.Cookie{Name: "__Host-employee-session", Value: sess.SessionID})
	recLeave := httptest.NewRecorder()
	router.ServeHTTP(recLeave, reqLeave)
	assert.Equal(t, http.StatusOK, recLeave.Code)
	assert.True(t, mockBLL.listLeaveRequestsCalled, "expected ListLeaveRequests RPC to be invoked")"""

code = code.replace(old_emp, new_emp)
code = code.replace("""	// 4. Operations Tickets route (requires authenticated employee session)
	sessStore := bff.SessionStore()
	sess, err := sessStore.Create("emp-test-1", "testuser@example.com", "Test User", "engineering", "Engineering")
	require.NoError(t, err)""", """	// 4. Operations Tickets route (requires authenticated employee session)""")

with open("internal/app/employeebff/production_wiring_test.go", "w") as f:
    f.write(code)

import re

with open("internal/app/employeebff/router.go", "r") as f:
    code = f.read()

old_wf = """	// 3. Workforce domain routes
	if b.wfHandler != nil {
		b.wfHandler.RegisterRoutes(mux)
	}"""

new_wf = """	// 3. Workforce domain routes
	if b.wfHandler != nil {
		// Secure workforce routes with session authentication
		sessionAuth := SessionAuthMiddleware(b.sessionStore)
		wfMux := http.NewServeMux()
		b.wfHandler.RegisterRoutes(wfMux)

		// Mount workforce routes inside a handler that applies session auth
		mux.Handle("/api/v1/workforce/", sessionAuth(wfMux))
		mux.Handle("/workforce/", sessionAuth(wfMux))
		mux.Handle("/api/v1/employee/", sessionAuth(wfMux))
		mux.Handle("/employee/", sessionAuth(wfMux))
		mux.Handle("/api/v1/nextcloud/", sessionAuth(wfMux))
		mux.Handle("/nextcloud/", sessionAuth(wfMux))
		mux.Handle("/webhooks/", wfMux) // Webhooks generally use signature verification, not session auth
	}"""
code = code.replace(old_wf, new_wf)

with open("internal/app/employeebff/router.go", "w") as f:
    f.write(code)

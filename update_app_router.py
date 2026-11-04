import re

with open("internal/app/employeebff/router.go", "r") as f:
    code = f.read()

old_wf = """	// 3. Workforce domain routes
	if b.wfHandler != nil {
		b.wfHandler.RegisterRoutes(mux)
	}"""

new_wf = """	// 3. Workforce domain routes
	if b.wfHandler != nil {
		sessionAuth := SessionAuthMiddleware(b.sessionStore)
		wfMux := http.NewServeMux()
		b.wfHandler.RegisterRoutes(wfMux)
		mux.Handle("/", sessionAuth(wfMux))
	}"""

# Wait, `mux.Handle("/", sessionAuth(wfMux))` would intercept EVERYTHING including healthz and auth if it falls through.
# It's better to just mount `/api/v1/workforce/`, `/workforce/`, `/api/v1/employee/`, `/employee/`, and `/nextcloud/` via sessionAuth, but `RegisterRoutes` registers a lot of prefixes.

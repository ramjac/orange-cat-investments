import re

with open("internal/app/employeebff/router.go", "r") as f:
    code = f.read()

old_webhooks = """		mux.Handle("/api/v1/nextcloud/", sessionAuth(wfMux))
		mux.Handle("/nextcloud/", sessionAuth(wfMux))
		mux.Handle("/webhooks/", wfMux) // Webhooks generally use signature verification, not session auth
	}"""

new_webhooks = """		mux.Handle("/api/v1/nextcloud/", sessionAuth(wfMux))
		mux.Handle("/nextcloud/", sessionAuth(wfMux))
		mux.Handle("/api/v1/webhooks/", wfMux) // Webhooks generally use signature verification, not session auth
		mux.Handle("/webhooks/", wfMux)
	}"""
code = code.replace(old_webhooks, new_webhooks)

with open("internal/app/employeebff/router.go", "w") as f:
    f.write(code)

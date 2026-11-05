import re

with open("internal/handler/timesheet_payroll_handlers.go", "r") as f:
    code = f.read()

# PR comment: "Inconsistent Role / Authorization Architecture: internal/handler/timesheet_payroll_handlers.go introduces an ad-hoc isElevatedRole helper hardcoding []string{"admin", "hr", "ops", "engineering"}. This duplicates and diverges from the RBAC patterns in internal/app/employeebff/rbac.go. Furthermore, role-based checks for workforce actions (e.g., manager approvals) should align with established persona roles rather than an uncoordinated string slice in the HTTP handler."

# The reviewer suggests using the existing RBAC patterns. But how do we use RBAC patterns inside `internal/handler/timesheet_payroll_handlers.go` without duplicating the slice?
# The RBAC pattern in `internal/app/employeebff/rbac.go` is `RequireRoleMiddleware`. It checks the role.
# We could check the role explicitly against established roles or just use the same pattern.
# But wait, `isElevatedRole` is defined inside `timesheet_payroll_handlers.go`.
# Is there a global helper?
# Let's see if there's a helper in `pkg/session/valkey` or `pkg/session`.

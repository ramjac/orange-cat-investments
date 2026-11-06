import re

with open("internal/handler/timesheet_payroll_handlers.go", "r") as f:
    code = f.read()

helper_func = """func isPayrollAdmin(role string) bool {
	role = strings.ToLower(strings.TrimSpace(role))
	return role == "admin" || role == "hr"
}

"""

code = code.replace(helper_func, "")
code = code.replace(")\n\nfunc (h *WorkforceHandler)", ")\n\n" + helper_func + "func (h *WorkforceHandler)")

with open("internal/handler/timesheet_payroll_handlers.go", "w") as f:
    f.write(code)

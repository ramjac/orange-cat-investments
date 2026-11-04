import re

with open("internal/app/employeebff/rbac.go", "r") as f:
    code = f.read()

code = code.replace("ctx := context.WithValue(r.Context(), employeeContextKey, sess)", "ctx := session.WithEmployeeSession(r.Context(), sess)")

with open("internal/app/employeebff/rbac.go", "w") as f:
    f.write(code)

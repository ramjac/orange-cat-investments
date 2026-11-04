import re

with open("internal/app/employeebff/rbac.go", "r") as f:
    code = f.read()

old_rbac_ctx = """type contextKey string

const employeeContextKey contextKey = "employee_session"

func EmployeeFromContext(ctx context.Context) *EmployeeSession {
	if s, ok := ctx.Value(employeeContextKey).(*EmployeeSession); ok {
		return s
	}
	return nil
}

func WithEmployeeSession(ctx context.Context, session *EmployeeSession) context.Context {
	return context.WithValue(ctx, employeeContextKey, session)
}"""

new_rbac_ctx = """func EmployeeFromContext(ctx context.Context) *EmployeeSession {
	return session.EmployeeFromContext(ctx)
}

func WithEmployeeSession(ctx context.Context, sess *EmployeeSession) context.Context {
	return session.WithEmployeeSession(ctx, sess)
}"""

code = code.replace(old_rbac_ctx, new_rbac_ctx)

old_import = """	sessionvalkey "github.com/orange-cat-investments/oci/pkg/session/valkey"
)"""
new_import = """	"github.com/orange-cat-investments/oci/pkg/session"
	sessionvalkey "github.com/orange-cat-investments/oci/pkg/session/valkey"
)"""

code = code.replace(old_import, new_import)

with open("internal/app/employeebff/rbac.go", "w") as f:
    f.write(code)

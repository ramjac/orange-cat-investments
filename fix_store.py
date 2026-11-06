import re

with open("pkg/session/valkey/store.go", "r") as f:
    code = f.read()

bad = """// IsElevated returns true if the employee holds an elevated administrative role.
func (s *EmployeeSession) IsElevated() bool {
	role := strings.ToLower(strings.TrimSpace(s.Role))
	for _, r := range []string{"admin", "hr", "ops", "engineering"} {
		if role == r {
			return true
		}
	}
	return false
}


"""
code = code.replace(bad, "")

with open("pkg/session/valkey/store.go", "w") as f:
    f.write(code)

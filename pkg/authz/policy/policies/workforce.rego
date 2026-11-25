package oci.authz.workforce

import rego.v1

default allow := false

elevated_vet_roles := {"veterinary", "vet", "admin"}
elevated_hr_roles := {"hr", "admin"}
manager_roles := {"hr", "admin", "manager", "ops", "facilities", "engineering", "investment", "veterinary"}

self_target if {
	input.resource.employee_id != ""
	input.actor.id != ""
	input.actor.id == input.resource.employee_id
}

allow if {
	input.action == "toggle_medical_hold"
	input.actor.role in elevated_vet_roles
}

allow if {
	input.action == "offboard_employee"
	input.actor.role in elevated_hr_roles
}

allow if {
	input.action == "approve_leave"
	input.actor.role in elevated_hr_roles
	not self_target
}

allow if {
	input.action == "approve_leave"
	input.actor.role == "manager"
	not self_target
}

allow if {
	input.action == "approve_leave"
	input.actor.role == "admin"
}

allow if {
	input.action == "submit_vet_audit"
	input.actor.role in elevated_vet_roles
}

allow if {
	input.action == "submit_manager_assessment"
	input.actor.role in manager_roles
	not self_target
}

allow if {
	input.action == "manage_review_cycle"
	input.actor.role in {"hr", "admin", "manager", "veterinary"}
}

deny_reasons contains msg if {
	input.actor.role == ""
	msg := "authenticated actor role is required"
}

deny_reasons contains msg if {
	input.action == "toggle_medical_hold"
	not input.actor.role in elevated_vet_roles
	msg := sprintf("role %q is not authorized to modify feline emergency medical holds (veterinary or admin role required)", [input.actor.role])
}

deny_reasons contains msg if {
	input.action == "offboard_employee"
	not input.actor.role in elevated_hr_roles
	msg := sprintf("role %q is not authorized to offboard employees (hr or admin role required)", [input.actor.role])
}

deny_reasons contains msg if {
	input.action == "approve_leave"
	self_target
	input.actor.role != "admin"
	msg := "employees cannot self-approve their own leave requests"
}

deny_reasons contains msg if {
	input.action == "approve_leave"
	not input.actor.role in {"hr", "admin", "manager"}
	msg := sprintf("role %q is not authorized to approve or reject leave requests", [input.actor.role])
}

deny_reasons contains msg if {
	input.action == "submit_vet_audit"
	not input.actor.role in elevated_vet_roles
	msg := sprintf("role %q is not authorized to submit veterinary wellness audits", [input.actor.role])
}

deny_reasons contains msg if {
	input.action == "submit_manager_assessment"
	self_target
	msg := "employees cannot submit a manager assessment for their own review cycle"
}

deny_reasons contains msg if {
	input.action == "submit_manager_assessment"
	not input.actor.role in manager_roles
	msg := sprintf("role %q is not authorized to submit manager assessments", [input.actor.role])
}

deny_reasons contains msg if {
	input.action == "manage_review_cycle"
	not input.actor.role in {"hr", "admin", "manager", "veterinary"}
	msg := sprintf("role %q is not authorized to create or modify review cycles", [input.actor.role])
}

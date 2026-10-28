package domain

// Permission is a fine-grained capability checked on the server.
type Permission = string

// Permissions. Built-in roles map onto these; tenants can also define
// custom roles with any subset (least privilege).
const (
	PermOrgManage        Permission = "org.manage"        // departments, teams, job profiles
	PermUsersManage      Permission = "users.manage"      // employees, invitations, roles
	PermQueuesManage     Permission = "queues.manage"     // queues, members, schedules
	PermFlowsManage      Permission = "flows.manage"      // IVR flows and phone numbers
	PermAIManage         Permission = "ai.manage"         // AI agent profiles
	PermRoutingManage    Permission = "routing.manage"    // topics, draft routing rules
	PermRoutingApprove   Permission = "routing.approve"   // activate routing rules
	PermKBManage         Permission = "kb.manage"         // knowledge sources (email connectors need settings.manage)
	PermKBRestricted     Permission = "kb.restricted"     // read sources marked restricted
	PermTicketsAll       Permission = "tickets.all"       // see and manage every ticket
	PermCallsAll         Permission = "calls.all"         // see every call and live state
	PermCallsControl     Permission = "calls.control"     // hang up / transfer any call
	PermCallsMonitor     Permission = "calls.monitor"     // spy, whisper, barge
	PermRecordingsPlay   Permission = "recordings.play"   // play any recording
	PermRecordingsExport Permission = "recordings.export" // download recordings
	PermQAEvaluate       Permission = "qa.evaluate"       // quality evaluations
	PermReportsView      Permission = "reports.view"
	PermCampaignsManage  Permission = "campaigns.manage"
	PermSettingsManage   Permission = "settings.manage" // settings, email connectors, retention, inbound key, webhooks
	PermAuditView        Permission = "audit.view"
)

// AllPermissions lists every permission (for role editors).
var AllPermissions = []Permission{
	PermOrgManage, PermUsersManage, PermQueuesManage, PermFlowsManage, PermAIManage,
	PermRoutingManage, PermRoutingApprove, PermKBManage, PermKBRestricted, PermTicketsAll,
	PermCallsAll, PermCallsControl, PermCallsMonitor, PermRecordingsPlay, PermRecordingsExport,
	PermQAEvaluate, PermReportsView, PermCampaignsManage, PermSettingsManage, PermAuditView,
}

var supervisorPerms = []Permission{
	PermTicketsAll, PermCallsAll, PermCallsControl, PermCallsMonitor, PermRecordingsPlay,
	PermQAEvaluate, PermReportsView, PermCampaignsManage, PermRoutingManage,
}

// PermissionSet is a lookup set of permissions.
type PermissionSet map[Permission]bool

// Has reports whether p is granted.
func (s PermissionSet) Has(p Permission) bool { return s[p] }

// List returns the granted permissions in canonical order.
func (s PermissionSet) List() []Permission {
	out := []Permission{}
	for _, p := range AllPermissions {
		if s[p] {
			out = append(out, p)
		}
	}
	return out
}

// PermissionsFor resolves the effective permissions of a user. A custom
// role, when assigned, replaces the built-in role's permissions; admins
// always keep every permission.
func PermissionsFor(role Role, custom *CustomRole) PermissionSet {
	set := PermissionSet{}
	if role == RoleAdmin {
		for _, p := range AllPermissions {
			set[p] = true
		}
		return set
	}
	if custom != nil {
		for _, p := range custom.Permissions {
			set[p] = true
		}
		return set
	}
	if role == RoleSupervisor {
		for _, p := range supervisorPerms {
			set[p] = true
		}
	}
	return set
}

// ValidPermission reports whether p is a known permission.
func ValidPermission(p string) bool {
	for _, known := range AllPermissions {
		if known == p {
			return true
		}
	}
	return false
}

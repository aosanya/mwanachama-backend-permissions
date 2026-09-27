package models

type Grant struct {
	Subject   string `json:"subject"`
	Scope     string `json:"scope"`
	RoleCode  string `json:"role_code"`
	GrantedBy string `json:"granted_by,omitempty"`
	GrantedAt string `json:"granted_at"`
	Deleted   bool   `json:"deleted"`
}

// ScopeEverywhere is the one scope this module reads rather than compares:
// a grant made here applies in every scope.
const ScopeEverywhere = "*"

package models

type Grant struct {
	Subject   string `json:"subject"`
	Scope     string `json:"scope"`
	RoleCode  string `json:"role_code"`
	GrantedBy string `json:"granted_by,omitempty"`
	GrantedAt string `json:"granted_at"`
	Deleted   bool   `json:"deleted"`
}

const ScopeEverywhere = "*"

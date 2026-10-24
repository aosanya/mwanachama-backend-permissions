package models

type PowerLayer string

const (
	PowerLayerDefinition PowerLayer = "definition"
	PowerLayerOverride   PowerLayer = "override"
)

type Power struct {
	Partition string     `json:"partition"`
	Power     string     `json:"power"`
	RoleKind  string     `json:"role_kind"`
	Layer     PowerLayer `json:"layer"`
	Held      bool       `json:"held"`
	SetBy     string     `json:"set_by,omitempty"`
	SetAt     string     `json:"set_at"`
}

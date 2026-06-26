package rbac

type Role struct {
	ID              int64    `json:"id"`
	Name            string   `json:"name"`
	Description     string   `json:"description"`
	PermissionCount int      `json:"permission_count,omitempty"`
	Permissions     []string `json:"permissions,omitempty"`
}

type Permission struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

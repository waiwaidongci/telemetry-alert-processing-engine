package tenant

import "time"

// Tenant is the top-level ownership boundary. Every resource in the system is
// scoped to one tenant.
type Tenant struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

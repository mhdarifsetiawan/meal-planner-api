package model

import "time"

type User struct {
	ID        string    `json:"id"` // UUID from Supabase auth.users.id
	Email     string    `json:"email"`
	Name      *string   `json:"name,omitempty"`
	CityID    *int      `json:"city_id,omitempty"`
	Role      string    `json:"role"` // "user" | "admin"
	CreatedAt time.Time `json:"created_at"`
}

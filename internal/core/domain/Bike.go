package domain

import (
	"time"

	"github.com/google/uuid"
)

// swagger:model domain.Bike
type Bike struct {
	UserID     uuid.UUID    `json:"user_id"`
	BikeID     uuid.UUID    `json:"bike_id"`
	BikeName   string       `json:"bike_name"`
	Type       BikeType     `json:"type"`
	Model      string       `json:"model"` // stels
	Components []*Component `json:"components"`
	Year       int          `json:"year"`
	Mileage    int          `json:"mileage"`
	CreatedAt  time.Time    `json:"created_at"`
	UpdatedAt  time.Time    `json:"updated_at"`
}

type BikeType string

const (
	BMX  BikeType = "bmx"
	MTB  BikeType = "mtb"
	Road BikeType = "road"
)

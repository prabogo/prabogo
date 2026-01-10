package model

import (
	"time"
)

type Variant struct {
	ID int `json:"id" db:"id"`
	ClientInput
}

type VariantInput struct {
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

type VariantFilter struct {
	IDs []int `json:"ids"`
}

func VariantPrepare(v *ClientInput) {
	v.CreatedAt = time.Now()
	v.UpdatedAt = time.Now()
}

func (c VariantFilter) IsEmpty() bool {
	return len(c.IDs) == 0
}

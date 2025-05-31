package model

import (
	"go-boilerplate/utils"
	"time"
)

const (
	UpsertClientMessage = "client.upsert"
)

type Client struct {
	ID int `json:"id" db:"id"`
	ClientInput
}

type ClientInput struct {
	Name      string    `json:"name" db:"name"`
	BearerKey string    `json:"bearer_key" db:"bearer_key"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

type ClientFilter struct {
	IDs        []int    `json:"ids"`
	Names      []string `json:"names"`
	BearerKeys []string `json:"bearer_keys"`
}

func ClientPrepare(v *ClientInput) {
	v.CreatedAt = time.Now()
	v.UpdatedAt = time.Now()
	if v.BearerKey == "" {
		v.BearerKey = utils.GenerateSecureToken(25)
	}
}

func (c ClientFilter) IsEmpty() bool {
	if len(c.IDs) == 0 && len(c.Names) == 0 && len(c.BearerKeys) == 0 {
		return true
	}
	return false
}

package redis_outbound_adapter

import (
	"context"
	"encoding/json"

	_ "github.com/doug-martin/goqu/v9/dialect/postgres"

	"prabogo/internal/model"
	outbound_port "prabogo/internal/port/outbound"
	"prabogo/utils/redis"
)

type clientAdapter struct{}

func NewClientAdapter() outbound_port.ClientCachePort {
	return &clientAdapter{}
}

func (adapter *clientAdapter) Set(data model.Client) error {
	bytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return redis.Set(context.Background(), data.BearerKey, string(bytes))
}

func (adapter *clientAdapter) Get(bearerKey string) (model.Client, error) {
	var client model.Client
	result, err := redis.Get(context.Background(), bearerKey)
	if err != nil {
		return model.Client{}, err
	}

	err = json.Unmarshal([]byte(result), &client)
	if err != nil {
		return model.Client{}, err
	}

	return client, nil
}

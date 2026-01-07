package fixtures

import (
	"time"

	"prabogo/internal/model"
)

type ClientTestData struct{}

func NewClientTestData() *ClientTestData {
	return &ClientTestData{}
}

func (c *ClientTestData) ValidClientInput() model.ClientInput {
	now := time.Now()
	return model.ClientInput{
		Name:      "Test Client",
		BearerKey: "test-bearer-key-" + now.Format("20060102150405"),
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func (c *ClientTestData) ValidClient() model.Client {
	return model.Client{
		ID:          1,
		ClientInput: c.ValidClientInput(),
	}
}

func (c *ClientTestData) ValidClientFilter() model.ClientFilter {
	return model.ClientFilter{
		IDs:        []int{1},
		Names:      []string{"Test Client"},
		BearerKeys: []string{"test-bearer-key"},
	}
}

func (c *ClientTestData) MultipleClients(count int) []model.Client {
	clients := make([]model.Client, count)
	now := time.Now()
	for i := 0; i < count; i++ {
		clients[i] = model.Client{
			ID: i + 1,
			ClientInput: model.ClientInput{
				Name:      "Client " + string(rune('A'+i)),
				BearerKey: "key-" + string(rune('a'+i)),
				CreatedAt: now,
				UpdatedAt: now,
			},
		}
	}
	return clients
}

func (c *ClientTestData) MultipleClientInputs(count int) []model.ClientInput {
	inputs := make([]model.ClientInput, count)
	now := time.Now()
	for i := 0; i < count; i++ {
		inputs[i] = model.ClientInput{
			Name:      "Client " + string(rune('A'+i)),
			BearerKey: "key-" + string(rune('a'+i)) + "-" + now.Format("150405"),
			CreatedAt: now,
			UpdatedAt: now,
		}
	}
	return inputs
}

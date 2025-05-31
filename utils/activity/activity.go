package activity

import (
	"context"

	"github.com/google/uuid"
)

type key int

const (
	TransactionID key = iota
	Action
	ClientID
	Payload
	Result
)

func NewContext(action string) context.Context {
	trxID := uuid.New().String()
	ctx := context.WithValue(context.Background(), TransactionID, trxID)
	return context.WithValue(ctx, Action, action)
}

func GetTransactionID(ctx context.Context) (string, bool) {
	trxID, ok := ctx.Value(TransactionID).(string)
	return trxID, ok
}

func WithAction(ctx context.Context, action string) context.Context {
	return context.WithValue(ctx, Action, action)
}

func GetAction(ctx context.Context) (string, bool) {
	action, ok := ctx.Value(Action).(string)
	return action, ok
}

func WithClientID(ctx context.Context, clientID string) context.Context {
	return context.WithValue(ctx, ClientID, clientID)
}

func GetClientID(ctx context.Context) (string, bool) {
	clientID, ok := ctx.Value(ClientID).(string)
	return clientID, ok
}

func WithPayload(ctx context.Context, payload interface{}) context.Context {
	return context.WithValue(ctx, Payload, payload)
}

func GetPayload(ctx context.Context) interface{} {
	return ctx.Value(Payload)
}

func WithResult(ctx context.Context, payload interface{}) context.Context {
	return context.WithValue(ctx, Result, payload)
}

func GetResult(ctx context.Context) interface{} {
	return ctx.Value(Result)
}

func GetFields(ctx context.Context) map[string]interface{} {
	fields := make(map[string]interface{})

	if id, ok := GetTransactionID(ctx); ok {
		fields["transaction_id"] = id
	}

	if action, ok := GetAction(ctx); ok {
		fields["action"] = action
	}

	if clientID, ok := GetClientID(ctx); ok {
		fields["client_id"] = clientID
	}

	fields["payload"] = GetPayload(ctx)
	fields["result"] = GetResult(ctx)

	return fields
}

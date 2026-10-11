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
	Error
	Message
)

func NewContext(action string, parent ...context.Context) context.Context {
	trxID := uuid.New().String()

	// Preserve an optional parent context (e.g. an OpenTelemetry span context)
	// so the activity logging fields stay linked to the parent trace.
	base := context.Background()
	if len(parent) > 0 && parent[0] != nil {
		base = parent[0]
	}

	ctx := context.WithValue(base, TransactionID, trxID)
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

func WithError(ctx context.Context, err error) context.Context {
	var errMsg interface{}
	if err != nil {
		errMsg = err.Error()
	}
	return context.WithValue(ctx, Error, errMsg)
}

func GetError(ctx context.Context) interface{} {
	return ctx.Value(Error)
}

func WithMessage(ctx context.Context, message string) context.Context {
	return context.WithValue(ctx, Message, message)
}

func GetMessage(ctx context.Context) (string, bool) {
	message, ok := ctx.Value(Message).(string)
	return message, ok
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

	if errMsg := GetError(ctx); errMsg != nil {
		fields["error"] = errMsg
	}

	if message, ok := GetMessage(ctx); ok {
		fields["message"] = message
	}

	return fields
}

package model

type Request struct {
	TransactionID string `json:"transaction_id"`
	Data          any    `json:"data"`
}

package demo

import "time"

type pubSubMessage struct {
	Message      pubSubMessageBody `json:"message"`
	Subscription string            `json:"subscription"`
}

type pubSubMessageBody struct {
	Data        string `json:"data"`
	MessageID   string `json:"messageId"`
	PublishTime string `json:"publishTime"`
}

type demoRegisteredPayload struct {
	UserID    string    `json:"userId"`
	ExpiresAt time.Time `json:"expiresAt"`
	TraceID   string    `json:"traceId"`
}

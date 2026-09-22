package tc

import "encoding/json"

// HTTP //////////////////////////////////////////

// AuthTokenRequest Http token request
type AuthTokenRequest struct {
	GrantType string `json:"grant_type"`
	Username  string `json:"username"`
	Password  string `json:"password"`
	ClientId  string `json:"client_id"`
}

// AuthTokenResponse Http token response
type AuthTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresAt   int64  `json:"expires_at"`
}

// WS //////////////////////////////////////////

const (
	// https://trueconf.ru/docs/chatbot-connector/ru/objects/#message-type
	MT_RESERVED int = iota
	MT_REQUEST      = 1
	MT_RESPONSE     = 2
)

type WSHeader struct {
	Type    uint32          `json:"type"` // message-type = mtREQUEST (by default)
	Id      uint32          `json:"id"`
	Method  string          `json:"method"`
	Payload json.RawMessage `json:"payload"`
}

// WSAuthPayload https://trueconf.ru/docs/chatbot-connector/ru/connect-and-auth/#%D0%B0%D0%B2%D1%82%D0%BE%D1%80%D0%B8%D0%B7%D0%B0%D1%86%D0%B8%D1%8F-%D0%BF%D0%BE%D0%B4%D0%BA%D0%BB%D1%8E%D1%87%D0%B5%D0%BD%D0%B8%D1%8F
type WSAuthPayload struct {
	Token         string `json:"token"`
	TokenType     string `json:"tokenType"`
	ReceiveUnread bool   `json:"receiveUnread"`
}

// WSGetChatsRequestPayload https://trueconf.ru/docs/chatbot-connector/ru/chats/#%D0%BF%D0%BE%D0%BB%D1%83%D1%87%D0%B8%D1%82%D1%8C-%D1%81%D0%BF%D0%B8%D1%81%D0%BE%D0%BA-%D1%87%D0%B0%D1%82%D0%BE%D0%B2
type WSGetChatsRequestPayload struct {
	Count uint32 `json:"count"`
	Page  uint32 `json:"page"`
}

type WSMessagePayload struct {
	ChatId    string                  `json:"chatId"`
	MessageId string                  `json:"messageId,omitzero"`
	Timestamp uint64                  `json:"timestamp,omitzero"`
	Author    WSMessageAuthorPayload  `json:"author,omitzero"`
	IsEdited  bool                    `json:"isEdited,omitzero"`
	Box       WSMessageBoxPayload     `json:"box,omitzero"`
	Type      uint32                  `json:"type,omitzero"` // https://trueconf.ru/docs/chatbot-connector/ru/objects/#EnvelopeTypeEnum
	Content   WSMessageContentPayload `json:"content,omitzero"`
}

type WSMessageAuthorPayload struct {
	Id   string `json:"id,omitzero"`
	Type uint32 `json:"type,omitzero"` // https://trueconf.ru/docs/chatbot-connector/ru/objects/#EnvelopeAuthorTypeEnum
}
type WSMessageBoxPayload struct { // https://trueconf.ru/docs/chatbot-connector/ru/objects/#EnvelopeBox
	Id       uint32 `json:"id,omitzero"`
	Position string `json:"position,omitzero"`
}
type WSMessageContentPayload struct {
	Text      string `json:"text,omitzero"`
	ParseMode string `json:"parseMode,omitzero"`
}

/*

{
  "type": 1,
  "id": 1,
  "method": "sendMessage",
  "payload": {
    "chatId": "c8c3eee8-9ad0-4638-9692-ad16391a4256",
    "content": {
      "text": "Hello, world!",
      "parseMode": "text"
    }
  }
}

*/

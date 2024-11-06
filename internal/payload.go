package internal

import "encoding/json"

type TypePayload string

const (
	KeyLogPayload   TypePayload = "kl"
	ShortcutPayload TypePayload = "sc"
)

type Payload struct {
	Version int             `json:"version"`
	Type    TypePayload     `json:"type"`
	Data    json.RawMessage `json:"data"` // why not json.RawMessage?
}

type KeylogPayloadV1 struct {
	KeyboardDeviceId string `json:"kID"`
	Code             uint16 `json:"c"`
}

type ShortcutPayloadV1 struct {
	KeyboardDeviceId string `json:"kID"`
	ShortcutId       int64  `json:"scID"`
}

func getPayload(typePayload TypePayload, data any) ([]byte, error) {
	db, err := json.Marshal(data)
	if err != nil {
		return []byte{}, err
	}
	p := Payload{Version: 1, Type: typePayload, Data: db}
	pb, err := json.Marshal(p)
	if err != nil {
		return []byte{}, err
	}
	return pb, nil
}

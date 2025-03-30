package internal

import "encoding/json"

// Payload to keylogme.com

type TypePayload string

const (
	TypePayloadKeylog      TypePayload = "kl"
	TypePayloadShortcut    TypePayload = "sc"
	TypePayloadLayerChange TypePayload = "lc"
	TypePayloadShiftState  TypePayload = "ss"
)

type Payload struct {
	Version int             `json:"version"`
	Type    TypePayload     `json:"type"`
	Data    json.RawMessage `json:"data"`
}

type KeylogPayload struct {
	KeyboardDeviceId string `json:"kID"`
	LayerId          int64  `json:"lID"`
	Code             uint16 `json:"c"`
}

type ShortcutPayload struct {
	KeyboardDeviceId string `json:"kID"`
	ShortcutId       string `json:"scID"`
}

type LayerChangePayload struct {
	KeyboardDeviceId string `json:"kID"`
	LayerId          int64  `json:"lID"`
}

type ShiftStatePayload struct {
	KeyboardDeviceId string `json:"kID"`
	Modifier         uint16 `json:"m"`
	Code             uint16 `json:"c"`
	Auto             bool   `json:"a"`
}

func GetPayload(typePayload TypePayload, data any) ([]byte, error) {
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

// Payload to logger

type TypePayloadLogger string

const (
	TypePayloadLoggerShortcut TypePayloadLogger = "sc"
)

type PayloadLogger struct {
	Version int               `json:"version"`
	Type    TypePayloadLogger `json:"type"`
	Data    json.RawMessage   `json:"data"`
}

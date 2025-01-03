package internal

import (
	"fmt"
	"log/slog"
	"time"
)

type KeylogMeStorage struct {
	sender *Sender
}

func MustGetNewKeylogMeStorage(origin, apiKey string) *KeylogMeStorage {
	sender := MustGetNewSender(origin, apiKey)
	return &KeylogMeStorage{sender: sender}
}

func (ks *KeylogMeStorage) SaveKeylog(deviceId string, keycode uint16) error {
	payloadBytes, err := getPayload(
		KeyLogPayload,
		KeylogPayloadV1{KeyboardDeviceId: deviceId, Code: keycode},
	)
	if err != nil {
		return err
	}
	return ks.sender.Send(payloadBytes)
}

func (ks *KeylogMeStorage) SaveShortcut(deviceId, shortcutId string) error {
	start := time.Now()
	defer func() {
		slog.Info(fmt.Sprintf("| %s | Shortcut %s\n", time.Since(start), shortcutId))
	}()
	pb, err := getPayload(
		ShortcutPayload,
		ShortcutPayloadV1{KeyboardDeviceId: deviceId, ShortcutId: shortcutId},
	)
	if err != nil {
		return err
	}
	return ks.sender.Send(pb)
}

func (ks *KeylogMeStorage) Close() error {
	return ks.sender.Close()
}

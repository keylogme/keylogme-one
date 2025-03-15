package internal

import (
	k1 "github.com/keylogme/keylogme-one"
)

type KeylogMeStorage struct {
	sender *Sender
}

func MustGetNewKeylogMeStorage(origin, apiKey string) *KeylogMeStorage {
	sender := MustGetNewSender(origin, apiKey)
	return &KeylogMeStorage{sender: sender}
}

func (ks *KeylogMeStorage) SaveKeylog(deviceId string, keycode uint16) error {
	pb, err := k1.GetPayload(
		k1.TypePayloadKeylog,
		k1.KeylogPayload{KeyboardDeviceId: deviceId, Code: keycode},
	)
	if err != nil {
		return err
	}
	return ks.sender.Send(pb)
}

func (ks *KeylogMeStorage) SaveShortcut(deviceId, shortcutId string) error {
	pb, err := k1.GetPayload(
		k1.TypePayloadShortcut,
		k1.ShortcutPayload{KeyboardDeviceId: deviceId, ShortcutId: shortcutId},
	)
	if err != nil {
		return err
	}
	return ks.sender.Send(pb)
}

func (ks *KeylogMeStorage) SaveLayerChange(deviceId string, layerId int64) error {
	pb, err := k1.GetPayload(
		k1.TypePayladLayerChange,
		k1.LayerChangePayload{KeyboardDeviceId: deviceId, LayerId: layerId},
	)
	if err != nil {
		return err
	}
	return ks.sender.Send(pb)
}

func (ks *KeylogMeStorage) SaveShiftState(deviceId string, modifier, code uint16, auto bool) error {
	pb, err := k1.GetPayload(
		k1.TypePayloadShiftState,
		k1.ShiftStatePayload{
			KeyboardDeviceId: deviceId,
			Modifier:         modifier,
			Code:             code,
			Auto:             auto,
		},
	)
	if err != nil {
		return err
	}
	return ks.sender.Send(pb)
}

func (ks *KeylogMeStorage) Close() error {
	return ks.sender.Close()
}

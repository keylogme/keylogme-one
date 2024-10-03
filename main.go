package main

import (
	"context"
	"os"

	"github.com/keylogme/zero-trust-logger/keylog"

	"github.com/keylogme/one-trust-logger/internal"
)

func main() {
	APIKEY := os.Args[1]
	ORIGIN_ENDPOINT := os.Args[2]

	// Get config
	config := keylog.Config{
		Devices: []keylog.DeviceInput{
			{Id: 1, Name: "foostan Corne"},
			{Id: 2, Name: "MOSART Semi. 2.4G INPUT DEVICE Mouse"},
			{Id: 2, Name: "Logitech MX Master 2S"},
			// {Id: 2, Name: "Wacom Intuos BT M Pen"},
		},
		Shortcuts: []keylog.Shortcut{
			{Id: 1, Values: []string{"J", "S"}, Type: keylog.SequentialShortcutType},
			{Id: 2, Values: []string{"J", "F"}, Type: keylog.SequentialShortcutType},
			{Id: 3, Values: []string{"J", "G"}, Type: keylog.SequentialShortcutType},
			{Id: 4, Values: []string{"J", "S", "G"}, Type: keylog.SequentialShortcutType},
		},
	}

	storage := internal.MustGetNewKeylogMeStorage(ORIGIN_ENDPOINT, APIKEY)
	defer storage.Close()

	_, cleanup := keylog.Start(context.Background(), storage, config)
	defer cleanup()
}

// func timeTrack(start time.Time, name string) {
// 	elapsed := time.Since(start)
// 	log.Printf("%s took %s", name, elapsed)
// }

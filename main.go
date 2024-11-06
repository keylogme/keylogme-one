package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/keylogme/zero-trust-logger/keylog"

	"github.com/keylogme/one-trust-logger/internal"
)

// install sudo apt-get install input-utils
// sudo lsinput | grep name
// sudo lsusb

// sudo lshw

func main() {
	APIKEY := os.Args[1]
	ORIGIN_ENDPOINT := os.Args[2]

	// Get config
	config := keylog.Config{
		Devices: []keylog.DeviceInput{
			{DeviceId: "1", Name: "foostan Corne"},
			{DeviceId: "2", Name: "MOSART Semi. 2.4G INPUT DEVICE Mouse"},
			{DeviceId: "2", Name: "Logitech MX Master 2S"},
			// {Id: 1, Name: "Microsoft Microsoft® 2.4GHz Transceiver v9.0 System Control"},
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

	_, cleanup := keylog.Start(storage, config)

	// Graceful shutdown
	ctxInt, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	// Wait for interrupt signal to gracefully shutdown the server with a timeout of 10 seconds.
	<-ctxInt.Done()
	cleanup()

	fmt.Println("Logger closed.")
}

// func timeTrack(start time.Time, name string) {
// 	elapsed := time.Since(start)
// 	log.Printf("%s took %s", name, elapsed)
// }

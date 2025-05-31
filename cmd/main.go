package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"time"

	k0 "github.com/keylogme/keylogme-zero"
	"github.com/keylogme/keylogme-zero/types"

	"github.com/keylogme/keylogme-one/internal"
)

// install sudo apt-get install input-utils
// sudo lsinput | grep name
// sudo lsusb

// sudo lshw

func main() {
	APIKEY := os.Args[1]
	ORIGIN_ENDPOINT := os.Args[2]

	//****************************************************

	// res, err := http.Get(fmt.Sprintf("%s/config?apikey=%s", ORIGIN_ENDPOINT, APIKEY))
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// print body as str
	// var config k0.Config
	// err = json.NewDecoder(res.Body).Decode(&config)
	// if err != nil {
	// 	log.Fatalf("Error decoding config: %v", err)
	// }

	config := k0.Config{
		Devices: []k0.DeviceInput{
			{
				DeviceId: "1",
				Name:     "built in",
				Layers:   []k0.LayerInput{},
				KeyloggerInput: types.KeyloggerInputAllOS{
					VendorID:  types.Hex(1),
					ProductID: types.Hex(2),
				},
			},
		},
		ShortcutGroups: []k0.ShortcutGroupInput{},
		ShiftState: k0.ShiftStateInput{
			ThresholdAuto: types.Duration{Duration: time.Duration(5 * time.Second)},
		},
	}

	// FIXME: check no duplicates of usb names of devices
	fmt.Println("Config:")
	fmt.Println("Devices:")
	for _, d := range config.Devices {
		fmt.Printf("%+v\n", d)
	}
	fmt.Println("Shortcut groups:")
	for _, sg := range config.ShortcutGroups {
		fmt.Printf("  %s %s :\n", sg.Id, sg.Name)
		for _, sc := range sg.Shortcuts {
			fmt.Printf("     %s %s %+v %s\n", sc.Id, sc.Name, sc.Codes, sc.Type)
		}
	}
	//****************************************************
	ctx, cancel := context.WithCancel(context.Background())

	storage := internal.MustGetNewKeylogMeStorage(ctx, ORIGIN_ENDPOINT, APIKEY)

	chEvt := make(chan k0.DeviceEvent)
	devices := []k0.Device{}
	for _, dev := range config.Devices {
		d := k0.GetDevice(ctx, dev, chEvt)
		devices = append(devices, *d)
	}
	sd := k0.MustGetNewShortcutsDetector(config.ShortcutGroups)

	ss := k0.NewShiftStateDetector(config.ShiftState)

	ld := k0.NewLayersDetector(config.Devices, config.ShiftState)

	k0.Start(chEvt, &devices, sd, ss, ld, storage)

	// Graceful shutdown
	ctxInt, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	<-ctxInt.Done() // blocks until process is interrupted
	cancel()
	time.Sleep(3 * time.Second) // graceful wait
	slog.Info("Logger closed.")
}

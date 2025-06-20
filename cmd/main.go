package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"time"

	k0 "github.com/keylogme/keylogme-zero"

	"github.com/keylogme/keylogme-one/internal"
)

// install sudo apt-get install input-utils
// sudo lsinput | grep name
// sudo lsusb

// sudo lshw

const (
	KEYLOGME_ENDPOINT = "https://keylogme.com/logger/v1"
)

func main() {
	// Get setup
	APIKEY := os.Getenv("KEYLOGME_ONE_API_KEY")
	if APIKEY == "" {
		log.Fatal("API_KEY env var is not set")
	}
	//****************************************************

	res, err := http.Get(fmt.Sprintf("%s/config?apikey=%s", KEYLOGME_ENDPOINT, APIKEY))
	if err != nil {
		log.Fatal(err)
	}
	if res.StatusCode != http.StatusOK {
		log.Fatalf("Error with api key : %s", res.Status)
	}
	// print body as str
	var config k0.Config
	err = json.NewDecoder(res.Body).Decode(&config)
	if err != nil {
		log.Fatalf("Error decoding config: %v", err)
	}
	slog.Info("Config:")
	slog.Info("Devices:")
	for _, d := range config.Devices {
		slog.Info(fmt.Sprintf("%+v\n", d))
	}
	slog.Info("Shortcut groups:")
	for _, sg := range config.ShortcutGroups {
		slog.Info(fmt.Sprintf("  %s %s :\n", sg.Id, sg.Name))
		for _, sc := range sg.Shortcuts {
			slog.Info(fmt.Sprintf("     %s %s %+v %s\n", sc.Id, sc.Name, sc.Codes, sc.Type))
		}
	}
	slog.Info("Shift state:")
	slog.Info(fmt.Sprintf("   %+v\n", config.ShiftState))

	slog.Info("Security:")
	slog.Info(fmt.Sprintf("   %+v\n", config.Security))
	//****************************************************
	ctx, cancel := context.WithCancel(context.Background())

	security := k0.NewSecurity(config.Security)

	storage := internal.MustGetNewKeylogMeStorage(ctx, KEYLOGME_ENDPOINT, APIKEY)

	chEvt := make(chan k0.DeviceEvent)
	for _, dev := range config.Devices {
		k0.GetDevice(ctx, dev, chEvt)
	}

	sd := k0.MustGetNewShortcutsDetector(config.ShortcutGroups)

	ss := k0.NewShiftStateDetector(config.ShiftState)

	ld := k0.NewLayersDetector(config.Devices, config.ShiftState)

	k0.Start(chEvt, security, sd, ss, ld, storage)

	// Graceful shutdown
	ctxInt, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	<-ctxInt.Done() // blocks until process is interrupted
	cancel()
	time.Sleep(3 * time.Second) // graceful wait
	slog.Info("Logger closed.")
}

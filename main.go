package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"

	"github.com/keylogme/keylogme-zero/keylog"

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

	res, err := http.Get(fmt.Sprintf("%s/config?apikey=%s", ORIGIN_ENDPOINT, APIKEY))
	if err != nil {
		log.Fatal(err)
	}
	var config keylog.Config
	err = json.NewDecoder(res.Body).Decode(&config)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Config:")
	fmt.Printf("Devices %+v\n", config.Devices)
	fmt.Println("Shortcut groups:")
	for _, sg := range config.ShortcutGroups {
		fmt.Printf("  %s %s :\n", sg.Id, sg.Name)
		for _, sc := range sg.Shortcuts {
			fmt.Printf("     %s %s %+v %s\n", sc.Id, sc.Name, sc.Codes, sc.Type)
		}
	}
	//****************************************************
	ctx, cancel := context.WithCancel(context.Background())

	storage := internal.MustGetNewKeylogMeStorage(ORIGIN_ENDPOINT, APIKEY)
	defer storage.Close()

	chEvt := make(chan keylog.DeviceEvent)
	devices := []keylog.Device{}
	for _, dev := range config.Devices {
		d := keylog.GetDevice(ctx, dev, chEvt)
		devices = append(devices, *d)
	}

	sd := keylog.MustGetNewShortcutsDetector(config.ShortcutGroups)

	keylog.Start(chEvt, &devices, sd, storage)

	// Graceful shutdown
	ctxInt, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	// Wait for interrupt signal to gracefully shutdown the server with a timeout of 10 seconds.
	<-ctxInt.Done()
	cancel()

	fmt.Println("Logger closed.")
}

// func timeTrack(start time.Time, name string) {
// 	elapsed := time.Since(start)
// 	log.Printf("%s took %s", name, elapsed)
// }

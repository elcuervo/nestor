package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/cretz/bine/tor"
	"github.com/ipsn/go-libtor"
)

const logo = `
                  _
  _ __   ___  ___| |_ ___  _ __
 | '_ \ / _ \/ __| __/ _ \| '__|
 | | | |  __/\__ \ || (_) | |
 |_| |_|\___||___/\__\___/|_|
     NEtwork Share via TOR
`

func main() {
	log.Println(logo)
	log.Println("Finding an available tor address..")

	t, err := tor.Start(nil, &tor.StartConf{ProcessCreator: libtor.Creator, DebugWriter: os.Stderr})

	if err != nil {
		log.Panicf("Unable to start Tor: %v", err)
	}

	defer t.Close()

	listenCtx, listenCancel := context.WithTimeout(context.Background(), 3*time.Minute)

	defer listenCancel()

	onion, err := t.Listen(listenCtx, &tor.ListenConf{RemotePorts: []int{80}})

	if err != nil {
		log.Panicf("Unable to create onion service: %v", err)
	}

	defer onion.Close()

	log.Printf("Go to http://%v.onion\n", onion.ID)
	log.Println("Press enter to exit")

	errCh := make(chan error, 1)

	go func() { errCh <- http.Serve(onion, http.FileServer(http.Dir("."))) }()
	go func() {
		fmt.Scanln()
		errCh <- nil
	}()

	if err = <-errCh; err != nil {
		log.Panicf("Failed serving: %v", err)
	}
}

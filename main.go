package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/cretz/bine/process/embedded"
	"github.com/cretz/bine/tor"
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
	fmt.Println(logo)
	fmt.Println("Finding an available tor address..")
	t, err := tor.Start(nil, &tor.StartConf{ProcessCreator: embedded.NewCreator()})
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
	fmt.Printf("Go to http://%v.onion\n", onion.ID)
	fmt.Println("Press enter to exit")
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

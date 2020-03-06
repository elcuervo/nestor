package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/briandowns/spinner"
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
	fmt.Println(logo)

	s := spinner.New(spinner.CharSets[4], 100*time.Millisecond)
	c := make(chan os.Signal, 2)

	s.Suffix = " Finding an available tor address."
	s.Start()

	go func() {
		<-c
		os.Exit(0)
	}()

	t, err := tor.Start(nil, &tor.StartConf{ProcessCreator: libtor.Creator})

	if err != nil {
		fmt.Errorf("Unable to start Tor: %v", err)
	}

	defer t.Close()

	listenCtx, listenCancel := context.WithTimeout(context.Background(), 3*time.Minute)

	defer listenCancel()

	onion, err := t.Listen(listenCtx, &tor.ListenConf{RemotePorts: []int{80}})

	if err != nil {
		fmt.Errorf("Unable to create onion service: %v", err)
	}

	defer onion.Close()

	s.Stop()

	fmt.Printf("Go to http://%v.onion\n", onion.ID)

	errCh := make(chan error, 1)

	go func() { errCh <- http.Serve(onion, http.FileServer(http.Dir("."))) }()

	if err = <-errCh; err != nil {
		fmt.Errorf("Failed serving: %v", err)
	}
}

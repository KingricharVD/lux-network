package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/216k155/lux-network"
)

var peerFile = flag.String("peerfile", "",
	"local file for storing known peers between runs.")
var connections = flag.Int("connections", 3,
	"number of connections to aim for.")

func main() {
	flag.Parse()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	log.SetFlags(log.Ldate | log.Lmicroseconds | log.Lshortfile)

	output := make(chan network.Message)
	n := network.New(network.Config{
		DesiredConnections: *connections,
		PeerStorageFile:    *peerFile,
		SeedHostnames: []string{
			"149.28.163.40",
			"207.148.83.62",
			"207.148.81.88",
			"149.28.172.147",
		},
		OutputChannel: output,
	})
	d := network.NewDispatcher(output)
	d.Subscribe("version", newConnectionHandler(n.SendChannel()))
	d.Subscribe("inv", newInventoryHandler())
	d.Run()

	go func() {
		sig := <-sigs

		fmt.Println()
		fmt.Println(sig)

		t := time.NewTimer(2 * time.Second)
		go func() {
			_ = <-t.C
			log.Println("shut down timed out; forcing exit")
			os.Exit(2)
		}()

		d.Unsubscribe("version")
		d.Unsubscribe("inv")
		n.Close()
		d.Close()
		close(output)

		os.Exit(1)
	}()

	http.Handle("/peers", n)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func newConnectionHandler(output chan<- network.Message) chan<- network.Message {
	input := make(chan network.Message, 1)
	go func() {
		for {
			m := <-input // receive version message.
			log.Printf("Received %q message from %q.", m.Type, m.Endpoint)
			output <- network.Message{
				Type:     "mempool",
				Endpoint: m.Endpoint,
			}
		}
	}()
	return input
}

func newInventoryHandler() chan<- network.Message {
	input := make(chan network.Message, 1)
	go func() {
		for {
			m := <-input // receive inventory message.
			log.Printf("Received %q message of size %d from %q.",
				m.Type, len(m.Data), m.Endpoint)
		}
	}()
	return input
}

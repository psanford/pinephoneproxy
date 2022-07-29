package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/godbus/dbus/v5"
)

var to = flag.String("to", "", "Phone to send message to")

func main() {
	flag.Parse()

	if *to == "" {
		log.Fatal("-to is required")
	}

	msg := strings.Join(flag.Args(), " ")
	if msg == "" {
		log.Fatalf("usage: %s -to <number> message text here", os.Args[0])
	}

	conn, err := dbus.ConnectSystemBus()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to connect to session bus:", err)
		os.Exit(1)
	}
	defer conn.Close()

	obj := conn.Object("org.ofono", "/quectelqmi_0")
	c := obj.Call("org.ofono.MessageManager.SendMessage", 0, to, msg)
	if c.Err != nil {
		panic(c.Err)
	}

	fmt.Printf("%+v\n", c)
}

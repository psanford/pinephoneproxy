package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/godbus/dbus/v5"
)

var postURL = flag.String("url", "", "URL to post messages to")
var username = flag.String("username", "", "Username")
var password = flag.String("password", "", "Username")
var slackToken = flag.String("slack-token", "", "Slack bot token")
var slackChannelID = flag.String("slack-channel", "", "Slack channel id")

func main() {
	flag.Parse()
	s := &server{
		inboundMsg: make(chan PostMsg, 100),
	}
	s.initCMDS()
	go s.runSlack()
	s.run()
}

type server struct {
	cmds       []cmd
	inboundMsg chan PostMsg
}

func (s *server) run() {
	conn, err := dbus.ConnectSystemBus()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to connect to session bus:", err)
		os.Exit(1)
	}
	defer conn.Close()

	if err = conn.AddMatchSignal(
		dbus.WithMatchObjectPath("/quectelqmi_0"),
		dbus.WithMatchInterface("org.ofono.MessageManager"),
		dbus.WithMatchMember("IncomingMessage"),
	); err != nil {
		panic(err)
	}

	c := make(chan *dbus.Signal, 10)
	conn.Signal(c)
	for v := range c {
		fmt.Printf("got sms msg %+v\n", v)
		message := v.Body[0].(string)
		metadata := v.Body[1].(map[string]dbus.Variant)
		sender := metadata["Sender"]
		sentTime := metadata["SentTime"]
		fmt.Printf("from=%s at=%s: %s\n", sender.String(), sentTime.String(), message)
		ts, err := time.Parse(time.RFC3339, sentTime.String())
		if err != nil {
			ts = time.Now()
		}

		msg := PostMsg{
			From: sender.String(),
			TS:   ts,
			Body: message,
		}

		s.sendMsg(msg)
	}

}

func (s *server) sendMsg(msg PostMsg) error {
	select {
	case s.inboundMsg <- msg:
	default:
	}

	payload, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("Marshal msg err: %s", err)
	}

	req, err := http.NewRequest("POST", *postURL, bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("NewRequest err: %s", err)
	}

	req.SetBasicAuth(*username, *password)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("Post error: %s", err)
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("Post error: %d %+v", resp.StatusCode, resp)
	}

	return nil

}

type PostMsg struct {
	From string    `json:"from"`
	To   string    `json:"to"`
	TS   time.Time `json:"ts"`
	Body string    `json:"body"`
}

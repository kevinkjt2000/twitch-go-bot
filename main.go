package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/spddl/go-twitch-ws"
	"nhooyr.io/websocket"
)

func main() {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	conn, _, err := websocket.Dial(ctx, "wss://eventsub.wss.twitch.tv/ws", nil)
	if err != nil {
		panic(err)
	}
	defer conn.CloseNow()
	defer conn.Close(websocket.StatusNormalClosure, "")

	irc, err := twitch.NewClient(&twitch.Client{
		Server:      "wss://irc-ws.chat.twitch.tv", // SSL, without SSL: ws://irc-ws.chat.twitch.tv
		User:        "",
		Oauth:       "", // without "oauth:" https://twitchapps.com/tmi/
		Debug:       true,
		BotVerified: false,                    // verified bots: Have higher chat limits than regular users.
		Channel:     []string{"shinybucket_"}, // only in Lowercase
	})
	if err != nil {
		panic(err)
	}
	defer irc.Close()

	irc.OnClearChatMessage = func(msg twitch.IRCMessage) {
		channel := msg.Params[0]
		msgline := msg.Params[1]

		if bytes.Equal(channel, []byte("#spddl")) {
			if bytes.Contains(msgline, []byte("hi")) {
				irc.Say("spddl", "Hi!", false) // only with creds
			}
		}
		log.Printf("%s - %s: %s\n", channel, msg.Tags["display-name"], msgline)
	}

	irc.Run()

outer:
	for {
		select {
		case <-interrupt:
			break outer
		default:
			msgType, data, err := conn.Read(ctx)
			if err != nil {
				panic(err)
			}
			fmt.Printf("%v %s\n", msgType, data)
		}
	}
}

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"os/signal"
	"strings"

	"github.com/caarlos0/env"
	"github.com/kevinkjt2000/twitch-go-bot/twitch"
	"nhooyr.io/websocket"
)

func main() {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var conf twitch.Config
	err := env.Parse(&conf)
	panicOnErr(err)
	client, err := twitch.NewClient(ctx, conf)
	panicOnErr(err)
	defer client.Close()

	broadcasterId, err := client.GetBroadcasterId("shinybucket_")
	panicOnErr(err)
	conn, _, err := websocket.Dial(ctx, "wss://eventsub.wss.twitch.tv/ws", nil)
	panicOnErr(err)
	defer conn.CloseNow()
	defer conn.Close(websocket.StatusNormalClosure, "")

outer:
	for {
		select {
		case <-interrupt:
			break outer
		default:
			msgType, data, err := conn.Read(ctx)
			panicOnErr(err)
			var tMsg twitch.Message
			err = json.Unmarshal(data, &tMsg)
			if err != nil {
				continue
			}
			switch tMsg.Metadata.MessageType {
			case "session_welcome":
				session := tMsg.Payload["session"].(map[string]interface{})
				sessionId := session["id"].(string)
				err = client.SubscribeToEvent(broadcasterId, sessionId)
				panicOnErr(err)
			case "notification":
				switch tMsg.Metadata.SubscriptionType {
				case "channel.channel_points_custom_reward_redemption.add":
					event := tMsg.Payload["event"].(map[string]interface{})
					reward := event["reward"].(map[string]interface{})
					switch reward["title"] {
					case "TTS":
						fmt.Printf("TTS event: %v\n", event)
						msg := event["user_input"].(string)
						_ = speak(msg) //TODO: refund user if festival fails
						//TODO: pause music?
					default: // Can safely ignore rewards that do not require an automated response
					}
					fmt.Printf("%s redeemed '%s'\n", event["user_login"], reward["title"])
				default:
					fmt.Printf("unimplemented subscription handle: %s", tMsg.Metadata.SubscriptionType)
				}
			case "session_keepalive":
			default:
				fmt.Printf("%v %s\n", msgType, data)
			}

			//TODO: handle disconnects
			//TODO: reconnect after a timeout of no messages
		}
	}
}

var festivalVoices []string

func init() {
	cmd := exec.Command("ls", "/usr/share/festival/voices/us")
	output, err := cmd.CombinedOutput()
	panicOnErr(err)
	fmt.Print(string(output))
	festivalVoices = strings.Split(string(output), "\n")
}

func speak(msg string) error {
	randVoice := festivalVoices[rand.Intn(len(festivalVoices))]
	cmd := exec.Command("festival", "--batch", fmt.Sprintf(`(voice_%s)`, randVoice), fmt.Sprintf(`(SayText "%s")`, msg))
	return cmd.Start()
}

func panicOnErr(err error) {
	if err != nil {
		panic(err)
	}
}

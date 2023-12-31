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
	"time"

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

	twitchMessages := twitchMessagesChannel(conn, ctx)
	keepAliveTimeoutSeconds := float64(15)
	for {
		select {
		case <-interrupt:
			fmt.Println("Interrupt detected!")
			return
		case <-time.After(time.Duration(keepAliveTimeoutSeconds) * time.Second):
			fmt.Printf("%v seconds passed without any message\n", keepAliveTimeoutSeconds)
			return
		case tMsg := <-twitchMessages:
			switch tMsg.Metadata.MessageType {
			case "session_welcome":
				session := tMsg.Payload["session"].(map[string]interface{})
				sessionId := session["id"].(string)
				err = client.SubscribeToEvent(broadcasterId, sessionId)
				panicOnErr(err)
				keepAliveTimeoutSeconds = session["keepalive_timeout_seconds"].(float64) + 2 // add a few seconds to be safe
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
			case "session_reconnect":
				// TODO MessageText {"metadata":{"message_id":"0249c605-e94c-47b4-8e7f-74f1b17f83e6","message_type":"session_reconnect","message_timestamp":"2023-12-04T18:20:44.155180678Z"},"payload":{"session":{"id":"AgoQcVTYThJiRFyIqybyp4l47RIGY2VsbC1i","status":"reconnecting","connected_at":"2023-12-03T23:26:36.10105215Z","keepalive_timeout_seconds":null,"reconnect_url":"wss://cell-b.eventsub.wss.twitch.tv/ws?challenge=cde23fff-2894-432d-93af-561bbb6f08c8\u0026id=AgoQcVTYThJiRFyIqybyp4l47RIGY2VsbC1i"}}}
			default:
				fmt.Printf("Unhandled twitch message: %s\n", tMsg)
			}
			//TODO: handle disconnects
		}
	}
}

func twitchMessagesChannel(conn *websocket.Conn, ctx context.Context) chan twitch.Message {
	twitchMessages := make(chan twitch.Message)

	go func() {
		for {
			select {
			case <-ctx.Done():
				fmt.Println("Closing twitch messages channel")
				close(twitchMessages)
				return
			default:
				_, data, err := conn.Read(ctx) // the first return value is always "MessageText"
				if err != nil {
					continue
				}
				var tMsg twitch.Message
				err = json.Unmarshal(data, &tMsg)
				if err != nil {
					continue
				}
				twitchMessages <- tMsg
			}
		}
	}()

	return twitchMessages
}

var festivalVoices []string

func init() {
	cmd := exec.Command("ls", "/usr/share/festival/voices/us")
	output, err := cmd.CombinedOutput()
	panicOnErr(err)
	festivalVoices = strings.Split(string(output), "\n")
}

func speak(msg string) error {
	randVoice := festivalVoices[rand.Intn(len(festivalVoices))]
	fmt.Println("using random voice ", randVoice)
	cmd := exec.Command("festival", "--batch", fmt.Sprintf(`(voice_%s)`, randVoice), fmt.Sprintf(`(SayText "%s")`, msg))
	return cmd.Start()
}

func panicOnErr(err error) {
	if err != nil {
		panic(err)
	}
}

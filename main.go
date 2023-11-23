package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/signal"

	"github.com/caarlos0/env"
	"github.com/kevinkjt2000/twitch-go-bot/twitch"
	"nhooyr.io/websocket"
)

func getBroadcasterId(authToken string, username string, conf twitch.Config) (broadcasterId string, err error) {
	Url, err := url.Parse("https://api.twitch.tv/helix/users")
	if err != nil {
		return
	}
	params := url.Values{}
	params.Add("login", username)
	Url.RawQuery = params.Encode()
	req, err := http.NewRequest("GET", Url.String(), nil)
	if err != nil {
		return
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authToken))
	req.Header.Set("Client-Id", conf.ClientId)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}
	var users TwitchUsersData
	err = json.Unmarshal(data, &users)
	if err != nil {
		return
	}
	broadcasterId = users.Data[0].Id
	return
}

type TwitchUser struct {
	Id    string `json:"id"`
	Login string `json:"login"`
}

type TwitchUsersData struct {
	Data []TwitchUser `json:"data"`
}

func main() {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	var conf twitch.Config
	err := env.Parse(&conf)
	panicOnErr(err)

	authCode, err := twitch.Authenticate(conf)
	panicOnErr(err)
	twitchToken, err := twitch.GetTokenFromAuthCode(authCode, conf)
	panicOnErr(err)
	broadcasterId, err := getBroadcasterId(twitchToken, "shinybucket_", conf)
	panicOnErr(err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	conn, _, err := websocket.Dial(ctx, "wss://eventsub.wss.twitch.tv/ws", nil)
	panicOnErr(err)
	defer conn.CloseNow()
	defer conn.Close(websocket.StatusNormalClosure, "")

	client, err := twitch.NewClient(conf)
	panicOnErr(err)
	defer client.Close()

outer:
	for {
		select {
		case <-interrupt:
			break outer
		default:
			msgType, data, err := conn.Read(ctx)
			panicOnErr(err)
			var tMsg TwitchMessage
			err = json.Unmarshal(data, &tMsg)
			if err != nil {
				continue
			}
			switch tMsg.Metadata.MessageType {
			case "session_welcome":
				session := tMsg.Payload["session"].(map[string]interface{})
				sessionId := session["id"].(string)
				err = SubscribeToEvent(broadcasterId, sessionId, twitchToken, conf)
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
						_ = msg // TODO speak the user's message with festival
						// TODO pause music?
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

			//TODO handle disconnects
			//TODO reconnect after a timeout of no messages
		}
	}
}

func panicOnErr(err error) {
	if err != nil {
		panic(err)
	}
}

type SubscriptionCondition struct {
	BroadcasterUserId string `json:"broadcaster_user_id,omitempty"`
}

func SubscribeToEvent(broadcasterId string, sessionId string, token string, conf twitch.Config) error {
	var buf bytes.Buffer
	_ = json.NewEncoder(&buf).Encode(Subscription{
		Condition: SubscriptionCondition{
			BroadcasterUserId: broadcasterId,
		},
		Transport: SubscriptionTransport{
			Method:    "websocket",
			SessionId: sessionId,
		},
		Type:    "channel.channel_points_custom_reward_redemption.add",
		Version: "1",
	})
	req, _ := http.NewRequest("POST", "https://api.twitch.tv/helix/eventsub/subscriptions", &buf)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("Client-Id", conf.ClientId)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	fmt.Println("response Status:", resp.Status)
	fmt.Println("response Headers:", resp.Header)
	body, _ := io.ReadAll(resp.Body)
	fmt.Println("response Body:", string(body))
	return nil
}

type SubscriptionTransport struct {
	Method    string `json:"method"`
	SessionId string `json:"session_id"`
}

type Subscription struct {
	Condition SubscriptionCondition `json:"condition"`
	Transport SubscriptionTransport `json:"transport"`
	Type      string                `json:"type"`
	Version   string                `json:"version"`
}

type SessionInfo struct {
	Id                      string  `json:"id"`
	Status                  string  `json:"status"`
	ConnectedAt             string  `json:"connected_at"`
	KeepaliveTimeoutSeconds int64   `json:"keepalive_timeout_seconds"`
	ReconnectUrl            *string `json:"reconnect_url,omitempty"`
}

type WelcomeMessage struct {
	Session SessionInfo `json:"session"`
}

type TwitchMetadata struct {
	MessageId        string `json:"message_id"`
	MessageType      string `json:"message_type"`
	MessageTimestamp string `json:"message_timestamp"`
	SubscriptionType string `json:"subscription_type,omitempty"`
}

type TwitchMessage struct {
	Metadata TwitchMetadata `json:"metadata"`
	Payload  map[string]interface{}
}

type OauthTokenRequest struct {
	ClientId string `json:"client_id"`
}

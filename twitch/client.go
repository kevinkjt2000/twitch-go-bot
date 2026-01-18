package twitch

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/spddl/go-twitch-ws"
	"golang.org/x/oauth2"
	otwitch "golang.org/x/oauth2/twitch"
)

type Event interface{}

type Client interface {
	Close()
	GetBroadcasterId(username string) (string, error)
	Reconnect()
	SubscribeToEvent(broadcasterId string, sessionId string) error
}

type websocketClient struct {
	config     Config
	httpClient *http.Client
	ircClient  *twitch.Client
}

// Reconnect asynchronously attempts to re-establish connection.
func (w websocketClient) Reconnect() {
	w.ircClient.CloseAndReconnect()
}

func (w websocketClient) doRequest(method string, url string, body io.Reader) ([]byte, int, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Client-Id", w.config.ClientId)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := w.httpClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	return data, resp.StatusCode, nil
}

func (w websocketClient) Close() {
	w.ircClient.Close()
}

func (w websocketClient) GetBroadcasterId(username string) (string, error) {
	Url, err := url.Parse("https://api.twitch.tv/helix/users")
	if err != nil {
		return "", err
	}
	params := url.Values{}
	params.Add("login", username)
	Url.RawQuery = params.Encode()
	data, _, err := w.doRequest("GET", Url.String(), nil)
	if err != nil {
		return "", err
	}
	var users UsersData
	err = json.Unmarshal(data, &users)
	if err != nil {
		return "", err
	}
	if len(users.Data) == 0 {
		return "", errors.New("twitch: no matching users")
	}
	return users.Data[0].Id, nil
}

func (w websocketClient) SubscribeToEvent(broadcasterId string, sessionId string) error {
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
	_, status, err := w.doRequest("POST", "https://api.twitch.tv/helix/eventsub/subscriptions", &buf)
	if err != nil {
		return err
	}
	if status != http.StatusAccepted {
		return fmt.Errorf("twitch: failed subscription %d", status)
	}
	return nil
}

func NewClient(ctx context.Context, conf Config) (Client, error) {
	token, err := AcquireToken(ctx, conf)
	if err != nil {
		return nil, err
	}
	oauthClient, err := NewAuthClient(ctx, conf, token)
	if err != nil {
		return nil, err
	}

	ircClient, err := twitch.NewClient(&twitch.Client{
		Server:      "wss://irc-ws.chat.twitch.tv",
		User:        "shinybotwatch",
		Oauth:       token.AccessToken, // without "oauth:" https://twitchapps.com/tmi/
		Debug:       false,
		BotVerified: false, // verified bots: Have higher chat limits than regular users.
		Channel:     []string{"shinybucket_"},
	})
	if err != nil {
		return nil, err
	}
	ircClient.OnConnect = func(connected bool) {
		fmt.Printf("Connecting to IRC with %v\n", connected)
	}
	ircClient.OnPrivateMessage = func(msg twitch.IRCMessage) {
		channel := msg.Params[0][1:]
		msgline := msg.Params[1]
		squashedMsgline := bytes.ReplaceAll(msgline, []byte(" "), []byte(""))
		if bytes.Contains(squashedMsgline, []byte("(╯°□°)╯︵┻━┻")) {
			fmt.Println("Table flipping detected... flipping back")
			ircClient.Say(string(channel), "┬─┬ ノ( ゜-゜ノ)", false)
		} else if bytes.Equal(squashedMsgline, []byte("!discord")) {
			ircClient.Say(string(channel), "https://discord.gg/CgnaKXDnar", false)
		} else if bytes.Equal(squashedMsgline, []byte("!modpack")) {
			ircClient.Say(string(channel), "Check out GTNH's wiki here https://wiki.gtnewhorizons.com/wiki/Main_Page", false)
		} else if bytes.Equal(squashedMsgline, []byte("!textures")) {
			ircClient.Say(string(channel), "Using F32 packs and outlined ores from https://gtnh.miraheze.org/wiki/Resource_Packs", false)
		} else if bytes.Equal(squashedMsgline, []byte("!youtube")) {
			ircClient.Say(string(channel), "http://www.youtube.com/@shinybucket", false)
		}
	}
	ircClient.Run()
	return &websocketClient{
		config:     conf,
		httpClient: oauthClient,
		ircClient:  ircClient,
	}, nil
}

func createOauthClient(conf Config) oauth2.Config {
	oauthConf := oauth2.Config{
		ClientID:     conf.ClientId,
		ClientSecret: conf.ClientSecret,
		Endpoint:     otwitch.Endpoint,
		RedirectURL:  "http://localhost:3000",
		Scopes: []string{
			// https://dev.twitch.tv/docs/authentication/scopes/
			"channel:manage:redemptions",
			"chat:edit",
			"chat:read",
			"channel:moderate",
			"whispers:read",
			"whispers:edit",
		},
	}
	return oauthConf
}

type SubscriptionCondition struct {
	BroadcasterUserId string `json:"broadcaster_user_id,omitempty"`
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

type Metadata struct {
	MessageId        string `json:"message_id"`
	MessageType      string `json:"message_type"`
	MessageTimestamp string `json:"message_timestamp"`
	SubscriptionType string `json:"subscription_type,omitempty"`
}

type Message struct {
	Metadata Metadata `json:"metadata"`
	Payload  map[string]interface{}
}

type Config struct {
	ClientId     string `env:"TWITCH_CLIENT_ID,required"`
	ClientSecret string `env:"TWITCH_CLIENT_SECRET,required"`
}

type User struct {
	Id    string `json:"id"`
	Login string `json:"login"`
}

type UsersData struct {
	Data []User `json:"data"`
}

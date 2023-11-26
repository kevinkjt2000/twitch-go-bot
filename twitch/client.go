package twitch

import (
	"bytes"
	"context"
	"encoding/json"
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
	Poll() (Event, error)
	SubscribeToEvent(broadcasterId string, sessionId string) error
}

type websocketClient struct {
	config     Config
	httpClient *http.Client
	ircClient  *twitch.Client
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
	req, err := http.NewRequest("GET", Url.String(), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Client-Id", w.config.ClientId)
	resp, err := w.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	var users UsersData
	err = json.Unmarshal(data, &users)
	if err != nil {
		return "", err
	}
	// TODO raise error when no user is found
	return users.Data[0].Id, nil
}

func (w websocketClient) Poll() (Event, error) {
	return nil, nil
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
	req, _ := http.NewRequest("POST", "https://api.twitch.tv/helix/eventsub/subscriptions", &buf)
	req.Header.Set("Client-Id", w.config.ClientId)
	req.Header.Set("Content-Type", "application/json")
	resp, err := w.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("twitch: failed subscription %s", resp.Status)
	}
	return nil
}

func NewClient(ctx context.Context, conf Config) (Client, error) {
	oauthConf := creatOauthClient(conf)
	code, err := authenticate(oauthConf)
	if err != nil {
		return nil, err
	}
	token, err := oauthConf.Exchange(ctx, code)
	if err != nil {
		return nil, err
	}
	oauthClient := oauthConf.Client(ctx, token)

	ircClient, err := twitch.NewClient(&twitch.Client{
		Server:      "wss://irc-ws.chat.twitch.tv",
		User:        "",
		Oauth:       "", // without "oauth:" https://twitchapps.com/tmi/
		Debug:       true,
		BotVerified: false, // verified bots: Have higher chat limits than regular users.
		Channel:     []string{"shinybucket_"},
	})
	if err != nil {
		return nil, err
	}
	ircClient.Run()
	return &websocketClient{
		config:     conf,
		httpClient: oauthClient,
		ircClient:  ircClient,
	}, nil
}

func creatOauthClient(conf Config) oauth2.Config {
	oauthConf := oauth2.Config{
		ClientID:     conf.ClientId,
		ClientSecret: conf.ClientSecret,
		Endpoint:     otwitch.Endpoint,
		RedirectURL:  "http://localhost:3000",
		Scopes: []string{
			"channel:manage:redemptions",
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

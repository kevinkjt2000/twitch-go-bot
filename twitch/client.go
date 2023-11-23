package twitch

import (
	"github.com/spddl/go-twitch-ws"
)

type Event interface{}

type Client interface {
	Close()
	Poll() (Event, error)
}

type websocketClient struct {
	ircClient *twitch.Client
}

func (w websocketClient) Close() {
	w.ircClient.Close()
}

func (w websocketClient) Poll() (Event, error) {
	return nil, nil
}

func NewClient(conf Config) (Client, error) {
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
		ircClient: ircClient,
	}, nil
}

type Config struct {
	ClientId     string `env:"TWITCH_CLIENT_ID,required"`
	ClientSecret string `env:"TWITCH_CLIENT_SECRET,required"`
}

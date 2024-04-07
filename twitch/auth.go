package twitch

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"sync"

	"github.com/kevinkjt2000/twitch-go-bot/internal"
	"golang.org/x/oauth2"
)

func AcquireToken(ctx context.Context, conf Config) (*oauth2.Token, error) {
	data, file_err := os.ReadFile(".twitch_token")
	if file_err != nil {
		//Token file was missing, so we contact auth servers
		oauthConf := createOauthClient(conf)
		code, err := authenticate(oauthConf)
		if err != nil {
			return nil, err
		}
		token, err := oauthConf.Exchange(ctx, code)
		if err != nil {
			return nil, err
		}
		tokenData, err := json.Marshal(token)
		if err != nil {
			return nil, err
		}
		os.WriteFile(".twitch_token", tokenData, 0600)
		return token, nil
	}

	var token oauth2.Token
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, err
	}
	return &token, nil
}

func NewAuthClient(ctx context.Context, conf Config, token *oauth2.Token) (*http.Client, error) {
	oauthConf := createOauthClient(conf)
	return oauthConf.Client(ctx, token), nil
}

// Returns an authentication code that may be used to request an OAuth token
func authenticate(conf oauth2.Config) (authCode string, err error) {
	csrfToken, err := internal.GenerateRandomStringURLSafe(16)
	if err != nil {
		return
	}
	authCodeUrl := conf.AuthCodeURL(csrfToken)
	wg := sync.WaitGroup{}
	wg.Add(1)
	authCallbackHandler := func(w http.ResponseWriter, r *http.Request) {
		values := r.URL.Query()
		if csrfToken != values.Get("state") {
			panic("possible CSRF attack")
		}
		authCode = values.Get("code")
		_, _ = io.WriteString(w, "Authorization code stored successfully. You may now close this page.")
		wg.Done()
	}
	redirectUrl, err := url.Parse(conf.RedirectURL)
	if err != nil {
		return
	}
	listener, err := net.Listen("tcp", redirectUrl.Host)
	if err != nil {
		return
	}
	server := &http.Server{Addr: listener.Addr().String()}
	http.HandleFunc("/"+redirectUrl.Path, authCallbackHandler)
	go func() {
		if err := server.Serve(listener); err != http.ErrServerClosed {
			panic(err)
		}
	}()

	err = internal.Open(authCodeUrl)
	if err != nil {
		return
	}
	wg.Wait()
	_ = server.Close()
	return
}

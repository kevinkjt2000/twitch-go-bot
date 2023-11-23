package twitch

import (
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/kevinkjt2000/twitch-go-bot/internal"
)

// Returns an authentication code that may be used to request an OAuth token
func Authenticate(conf Config) (authCode string, err error) {
	csrfToken, err := internal.GenerateRandomStringURLSafe(16)
	if err != nil {
		return
	}
	wg := sync.WaitGroup{}
	wg.Add(1)
	authCallbackHandler := func(w http.ResponseWriter, r *http.Request) {
		values := r.URL.Query()
		if csrfToken != values.Get("state") {
			panic("possible CSRF attack")
		}
		authCode = values.Get("code")
		_, _ = io.WriteString(w, "Authorization token stored successfully. You may now close this page.")
		wg.Done()
	}
	// TODO parse host and path as a url.URL from config struct
	listener, err := net.Listen("tcp", "localhost:3000")
	if err != nil {
		return
	}
	server := &http.Server{Addr: listener.Addr().String()}
	http.HandleFunc("/", authCallbackHandler)
	go func() {
		if err := server.Serve(listener); err != http.ErrServerClosed {
			panic(err)
		}
	}()

	scopes := []string{
		"channel:manage:redemptions",
		"chat:read",
		"channel:moderate",
		"whispers:read",
		"whispers:edit",
	}
	authorizationURL, err := url.Parse("https://id.twitch.tv/oauth2/authorize")
	if err != nil {
		return
	}
	params := url.Values{}
	params.Add("client_id", conf.ClientId)
	params.Add("redirect_uri", "http://localhost:3000")
	params.Add("response_type", "code")
	params.Add("scope", strings.Join(scopes, " "))
	params.Add("state", csrfToken)
	authorizationURL.RawQuery = params.Encode()
	err = internal.Open(authorizationURL.String())
	if err != nil {
		return
	}
	wg.Wait()
	_ = server.Close()
	return
}

func GetTokenFromAuthCode(authCode string, conf Config) (string, error) {
	data := url.Values{}
	data.Add("client_id", conf.ClientId)
	data.Add("client_secret", conf.ClientSecret)
	data.Add("code", authCode)
	data.Add("grant_type", "authorization_code")
	data.Add("redirect_uri", "http://localhost:3000")
	req, err := http.NewRequest("POST", "https://id.twitch.tv/oauth2/token", strings.NewReader(data.Encode()))
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	buf, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	var oauthResp TwitchOauthResponse
	err = json.Unmarshal(buf, &oauthResp)
	if err != nil {
		return "", err
	}
	return oauthResp.AccessToken, nil
}

type TwitchOauthResponse struct {
	AccessToken  string   `json:"access_token"`
	ExpiresIn    int64    `json:"expires_in"`
	RefreshToken string   `json:"refresh_token"`
	Scope        []string `json:"scope"`
	TokenType    string   `json:"token_type"`
}

package twitch

import (
	"io"
	"net"
	"net/http"
	"net/url"
	"sync"

	"github.com/kevinkjt2000/twitch-go-bot/internal"
	"golang.org/x/oauth2"
)

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
		_, _ = io.WriteString(w, "Authorization token stored successfully. You may now close this page.")
		wg.Done()
	}
	// TODO parse host and path as a url.URL from config struct
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

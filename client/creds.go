package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
)

var userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:144.0) Gecko/20100101 Firefox/144.0"

type getCredsFunc func(string) (string, string, string, error)

// GetUserAgentFromBrowser - запускает сервер и при первом открытии браузера считывает строку user-agent и закрывает соединение.
//
// Parameters:
//   - listen: адрес и порт, на котором запускать сервер.
func GetUserAgentFromBrowser(listen string) {
	if listen == "" {
		listen = ":9000"
	}
	chUserAgent := make(chan string)
	defer close(chUserAgent)

	userAgentHandler := func(_ http.ResponseWriter, r *http.Request) {
		chUserAgent <- r.UserAgent()
	}
	server := http.Server{
		Handler: http.HandlerFunc(userAgentHandler),
		Addr:    listen,
	}

	go func() {
		for newUserAgent := range chUserAgent {
			if newUserAgent == "" {
				fmt.Println("Empty user-agent received, try again")
				continue
			}
			userAgent = newUserAgent
			err := server.Close()
			if err != nil {
				log.Panicf("Failed to close server: %v", err)
			}
		}
	}()
	formattedListen := listen
	if strings.HasPrefix(formattedListen, ":") {
		formattedListen = "127.0.0.1" + formattedListen
	}
	formattedListen = "http://" + formattedListen
	fmt.Printf("open %s for fix user-agent\n", formattedListen)
	err := server.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Panicf("Failed to start server: %v", err)
	}
	fmt.Printf("Got new user-agent: %s\n", userAgent)
}

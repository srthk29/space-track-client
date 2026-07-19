package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os"

	"github.com/joho/godotenv"
	"golang.org/x/net/publicsuffix"
)

func main() {
	ctx := context.Background()

	err := godotenv.Load()
	if err != nil {
		panic(err)
	}

	c := NewHttpClient()

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		"https://www.space-track.org/basicspacedata/query/class/gp/favorites/Geosynchronous/EPOCH/%3Enow-0.5/format/3le/limit/1",
		nil)
	if err != nil {
		panic(err)
	}

	resp, err := c.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Non-OK Status: %s\n", resp.Status)
	}

	if _, err := io.Copy(os.Stdout, resp.Body); err != nil {
		panic(err)
	}
}

func noCookieJar() {
	ts := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			if cookie, err := r.Cookie("Cookie"); err != nil {
				fmt.Println("You don't have a cookie. Here's one for you. Don't forget to keep it.")
				http.SetCookie(w, &http.Cookie{Name: "Cookie", Value: "Peanut Butter"})
			} else {
				fmt.Printf("You have %s cookie!\n", cookie.Value)
			}
		}))
	defer ts.Close()

	client := &http.Client{}

	if _, err := client.Get(ts.URL); err != nil {
		panic(err)
	}

	if _, err := client.Get(ts.URL); err != nil {
		panic(err)
	}
}

func withCookieJar() {
	ts := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			if cookie, err := r.Cookie("Cookie"); err != nil {
				fmt.Println("You don't have a cookie. Here's one for you. Don't forget to keep it.")
				http.SetCookie(w, &http.Cookie{Name: "Cookie", Value: "Peanut Butter"})
			} else {
				fmt.Printf("You have %s cookie!\n", cookie.Value)
			}
		}))
	defer ts.Close()

	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		panic(err)
	}

	client := &http.Client{
		Jar: jar,
	}

	if _, err = client.Get(ts.URL); err != nil {
		panic(err)
	}

	if _, err = client.Get(ts.URL); err != nil {
		panic(err)
	}
}

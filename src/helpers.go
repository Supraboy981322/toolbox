package main

import (
	"io"
	"strings"
	"net/http"
)

func getBody(r *http.Request) (string, error) {
	bod, err := io.ReadAll(r.Body)
	if err != nil {
		return "", err
	}

	return string(bod), nil
}

func getBodyNoErr(r *http.Request) string {
	bod, _ := getBody(r)
	return bod
}

func chkHeaders(check []string, def string, r *http.Request) string {
	var ret string
	for _, chk := range check {
		if ret == "" {
			ret = r.Header.Get(chk)
		} else { break }
	}; if ret == "" {
		ret = def
	}

	return ret
}

func chkUrlPref(url string) string {
	isHTTPS := strings.HasPrefix(url, "http://")
	isHTTP := strings.HasPrefix(url, "https://")
	if !isHTTPS && !isHTTP {
		url = "https://" + url
	}
	return url
}

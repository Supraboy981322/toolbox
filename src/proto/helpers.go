package main

import (
	"io"
	"strings"
	"net/http"
)

//just gets the body of a response
func getBody(r *http.Request) (string, error) {
	//read body
	bod, err := io.ReadAll(r.Body)
	if err != nil { return "", err }

	//return it as string
	return string(bod), nil
}

//gets the body and ignores err 
func getBodyNoErr(r *http.Request) string {
	bod, _ := getBody(r)
	return bod
}

//loops through slice of headers,
//  returns value of first non-empty header,
//    defaults to input arg if none matched
func chkHeaders(check []string, def string, r *http.Request) string {
	var val string
	for _, chk := range check {
		//if empty, get header
		if val == "" {
			val = r.Header.Get(chk)
		} else { break } //end loop otherwise
	}; if val == "" {
		//if empty,
		//  set value to default
		val = def
	}

	return val
}

//validate url prefix
func chkUrlPref(url string) string {
	//check if https
	isHTTPS := strings.HasPrefix(url, "https://")
	//check if http
	isHTTP := strings.HasPrefix(url, "http://")
	//if neither, prefix with `https://`
	if !isHTTPS && !isHTTP {
		url = "https://" + url
	}

	//return the url
	return url
}

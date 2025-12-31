package main

import (
	"io"
	"io/fs"
	"errors"
	"strings"
	"net/http"
	"path/filepath"
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

func chkFile(file string) (string, error) {
	res, err := chkIsDir(file)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			if filepath.Ext(res) == "" {
				res += ".elh"
				return chkFile(res)
			} else if filepath.Ext(res) == ".elh" {
				nS := strings.Split(filepath.Base(res), ".")
				n := strings.Join(nS[:len(nS)-1], ".")
				res = filepath.Dir(res)
				res = filepath.Join(res, n+".html")
				return chkFile(res)
			} else { return file, fs.ErrNotExist }
		} else { return file, err }
	}
	return res, nil
}

func chkIsDir(file string) (string, error) {
	fi, err := webUIdir.Open(file)
	if err != nil { return file, err }

	st, err := fi.Stat()
	if err != nil { return file, err }

	if st.IsDir() { return filepath.Join(file, "index"), fs.ErrNotExist }

	return file, nil
}

func fileExists(filePath string) bool {
	_, err := webUIdir.Open(filePath)
	return !errors.Is(err, fs.ErrNotExist)
}

package main

import (
	"os"
	"io"
	"fmt"
	"bytes"
	"strings"
	"strconv"
	"net/url"
	"math/big"
	"net/http"
	"io/ioutil"
	"crypto/rand"
	"encoding/json"
	"github.com/charmbracelet/log"
	"github.com/Supraboy981322/gomn"
)

var (
	port int
	config gomn.Map
	chars = []string{
		"a", "b",	"c", "d", "e", "f", "g",
		"h", "i", "j", "k", "l", "m", "n",
		"o", "p", "q", "r", "s", "t", "u",
		"v", "w", "x", "y", "z", "A", "B",
		"C", "D", "E", "F", "G", "H", "I",
		"J", "K", "L", "M", "N", "O", "P",
		"Q", "R", "S", "T", "U", "V", "W",
		"X", "Y", "Z", "0", "9", "8", "7",
		"6", "5", "4", "3", "2", "1", "!",
		"@", "#", "$", "%", "^", "&", "*",
		"(", ")", "-", "_", "=", "+", "[",
		"]", "{", "}", "|", "\\", ";", ":",
		"'", "\"", "<", ">", "/", "?", ".",
		",",
	}
)

func main() {
	log.SetLevel(log.DebugLevel)
	if confBytes, err := os.ReadFile("config.gomn"); err != nil {
		log.Fatalf("failed to read config:  %v", err)
	} else { log.Debug("reading config")
		log.Info("parsing config...")

		var ok bool
		if config, err = gomn.Parse(string(confBytes)); err != nil {
			log.Fatalf("failed to parse config:  %v", err)
		} else { log.Debug("success mapping config") }

		var deLvl string
		if deLvl, ok = config["log level"].(string); ok {
			switch deLvl {
			case "debug":
				log.SetLevel(log.DebugLevel)
			case "info": 
				log.SetLevel(log.InfoLevel)
			case "warn": 
				log.SetLevel(log.WarnLevel)
			case "error":
				log.SetLevel(log.ErrorLevel)
			case "fatal":
				log.SetLevel(log.FatalLevel)
			default:
				log.Fatal("invalid log level")
			}
			log.Infof("log level set to:  %s", deLvl)
		} else { log.Fatal("failed to get log level") }

		if port, ok = config["port"].(int); !ok {
			log.Fatal("failed to get server port")
		} else { log.Debug("success reading server port") }

		log.Info("success parsing config")
	}

	http.HandleFunc("/", pageHandler)
	log.Infof("listening on port:  %d", port)
	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(port), nil))
}

func pageHandler(w http.ResponseWriter, r *http.Request) {
	var selfHan bool;var resp string
	log.Infof("[req]: %s", r.URL.Path[1:])
	switch r.URL.Path[1:] {
	case "no":
		resp = noReq()
	case "discord":
		resp = discord(r)
	case "pass", "ranPass", "ran", "random", "password", "ranpass":
		resp = ranPass(r)
	case "de-shortener", "de-shorten", "de-short", "deshort", "deshorten", "deshortener":
		original := r.Header.Get("og")
		resp = deShortenURL(original)
	default:
		resp = "foo"
	}
	if !selfHan {
		w.Write([]byte(fmt.Sprint(resp,"\n")))
	}
}

func deShortenURL(original string) string {
	if original == "" {
		return "no shortened url provided"
	}

	client := &http.Client{
		CheckRedirect: 
			func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
	}

	var loc string
	currentURL := original
	for {
		req, err := http.NewRequest("HEAD", currentURL, nil)
		if err != nil {
			return err.Error()
		}

		resp, err := client.Do(req)
		if err != nil {
			uerr, ok := err.(*url.Error)
			if ok && uerr.Err == http.ErrUseLastResponse {
				err = nil
			} else if err != nil {
				return err.Error()
			}
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		resp.Body.Close()

		if resp.StatusCode >= 300 && resp.StatusCode < 400 {
			location, err := resp.Location()
			if err != nil {
				return err.Error()
			}
			loc = location.String()
			currentURL = loc
		} else {
			break
		}
	}
	return loc
}

func noReq() string {
	resp, err := http.Get("https://naas.isalman.dev/no")
	if err != nil {
		return err.Error()
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "recieved bad status code from api"
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err.Error()
	}

	type noJSON struct {
		Reason string `json:"reason"`
	}

	var no noJSON
	if err := json.Unmarshal(body, &no); err != nil {
		return err.Error()
	}

	return no.Reason
}

func discord(r *http.Request) string {
	var ok bool;var body string
	webhook :=  r.Header.Get("webhook")
	if webhook == "" {
		if webhook, ok = config["discord webhook"].(string); !ok || webhook == "your discord webhook" {
			return "no discord webhook provided or set in config"
		}
	}
	switch r.Method {
	case http.MethodGet:
		body = r.Header.Get("body")
	case http.MethodPost:
		bodyByte, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return err.Error()
		}
		body = string(bodyByte)
	default:
		return "method not allowed"
	}

	if body == "" {
		return "no body provided"
	}

	payload := map[string]interface{}{
		"content": body,
	}
	//json, eww....
	data, err := json.Marshal(payload)
	if err != nil {
		return err.Error()
	}
	resp, err := http.Post(webhook, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return err.Error()
	}
	defer resp.Body.Close()
	return "sent"
}

func ranPass(r *http.Request) string {
	lenStr := r.Header.Get("len")
	if len(lenStr) >= 18 {
		log.Warnf("overflow attempt: ip%s",
			r.RemoteAddr)
		return "request denied; possible overflow attempt"
	}
	if lenStr == "" {
		lenStr = "16"
	}

	charSet := []string{}
	if charsRaw := r.Header.Get("bld"); charsRaw == "" {
		if charsRaw = r.Header.Get("build"); charsRaw == "" {
			charSet = chars 
		} else { charSet = strings.Split(charsRaw, "") }
	} else { charSet = strings.Split(charsRaw, "") }

	l, err := strconv.ParseInt(lenStr, 10, 64)
	if err != nil {
		return err.Error()
	}

	if l < 0 {
		l = -l
	}
	if l > 56527 {
		log.Debug("req denied")
		return "What the hell would you need a random string longer than 56527 characters for?"
	}
	var res string
	var i int64
	for i = 0; i < l; i++ {
		bigInt := big.NewInt(int64(len(charSet)))
		in, err := rand.Int(rand.Reader, bigInt)
		if err != nil {
			return err.Error()
		}
		ranDig := int(in.Int64())
		res += charSet[ranDig]
	}

	return res
}

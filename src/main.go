package main

import (
	"os"
	"io"
	"strconv"
	"net/url"
	"net/http"
	"encoding/json"
	"github.com/charmbracelet/log"
	"github.com/Supraboy981322/gomn"
)

var (
	port int
	config gomn.Map
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
	log.Infof("[req]: %s", r.URL.Path[1:])
	switch r.URL.Path[1:] {
	case "no":
		w.Write([]byte(noReq()))
	case "discord":
		w.Write([]byte(discord()))
	case "de-shortener":
		w.Write([]byte(deShortenURL(r.Header.Get("og"))))
	default:
		w.Write([]byte("foo"))
	}
}

func deShortenURL(original string) string {
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
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

func discord() string {
	return "TODO"
}

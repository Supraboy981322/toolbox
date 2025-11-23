package main

import (
	"os"
	"strconv"
	"net/http"
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
	switch r.URL.Path[1:] {
	case "de-shortener":
		w.Write([]byte(deShortenURL(r.Header.Get("short"))))
	default:
		w.Write([]byte("foo"))
	}
}

func deShortenURL(original string) string {
  return original
}

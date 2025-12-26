package main

import (
	"io"
	"os"
	"fmt"
	"time"
	"strings"
	"strconv"
	"net/http"
	"math/big"
	"encoding/json"
	"path/filepath"
	"github.com/charmbracelet/log"
	"github.com/Supraboy981322/gomn"
	elh "github.com/Supraboy981322/ELH"
)

var (
	port int
	useWebUI bool
	config gomn.Map
	endPtMap map[string]map[string]string
	srvErr = http.StatusInternalServerError

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

	//the webui just uses Bash,
	//  so I'm only using a Bash
	//    runner for ELH
	registry = map[string]elh.Runner{
		"bash": &elh.ExternalRunner{
			CmdName: "bash",
			Args:    []string{},
			Timeout: 5 * time.Second,
			Env:     os.Environ(),
		},
	}
)

func init() {
	var ok bool
	var err error
	log.SetLevel(log.DebugLevel)

	elh.WebDir = "web"

	log.Info("reading config...")
	if config, err = gomn.ParseFile("config.gomn"); err != nil {
		log.Fatalf("failed to read config:  %v", err)
	} else { log.Debug("read config")	}

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

	//check if dashboard is enabled
	//  (I know, this looks highly compressed... because it is)
	if dashBoard, ok := config["dashboard"].(gomn.Map); ok {
		if useWebUI, ok = dashBoard["enable"].(bool); !ok {
			log.Fatal("dashboard --> enable is invalid")
		} else if useWebUI { log.Debug("dashboard is enabled")
		} else { log.Warn("web ui is disabled") }
	} else { log.Fatal("failed to parse dashboard config") } 

	//set the port from config
	if port, ok = config["port"].(int); !ok {
		log.Fatal("failed to get server port")
	} else { log.Debug("success reading server port") }

	//not used yet, but maps custom endpoints
	ptMapTmp := make(map[string]map[string]string)
	if endPtsRaw, ok := config["endpoints"].(gomn.Map); ok {
		log.Debug("found custom endpoints")

		for ptRaw, mpRaw := range endPtsRaw {
			ptMap := make(map[string]string)

			var mp gomn.Map
			if mp, ok = mpRaw.(gomn.Map); !ok {
				log.Fatal("failed to assert endpoint map type")
			} else { log.Debug("asserted endpoint map") } 

			for keyRaw, valRaw := range mp {
				if key, ok := keyRaw.(string); ok {

					if valS, ok := valRaw.(string); ok {
						ptMap[key] = valS
					}	else {
						if valR, ok := valRaw.([]rune); ok {
							ptMap[key] = string(valR)
						} else { log.Fatalf("invalid endpoint map value", valRaw) }
					}
					
				} else { log.Fatalf("invalid endpoint map key:  ", keyRaw) }
			}

			if pt, ok := ptRaw.(string); ok {
				ptMapTmp[pt] = ptMap
			} else { log.Fatal("failed to assert endpoint to string") }

		}; endPtMap = ptMapTmp
	} else { log.Debug("no custom endpoints defined") }

	log.Info("startup done.")
}

func main() {
	http.HandleFunc("/", pageHandler)
	log.Infof("listening on port:  %d", port)
	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(port), nil))
}


func pageHandler(w http.ResponseWriter, r *http.Request) {
	var selfHan bool;var resp string
	{ //check if num (math/big for very large nums) 
		codeStr := r.URL.Path[1:]
		n := new(big.Int)
		if _, ok := n.SetString(codeStr, 10); ok {
			log.Infof("[req]: %s", codeStr)
			statCode(w, codeStr)
			return
		}
	}
	switch strings.ToLower(r.URL.Path[1:]) {
	case "no":
		resp = noReq()
	case "e", "echo":
		resp = echo(r)
	case "time":
		resp = timeFunc(r)
	case "discord":
		resp = discord(r)
	case "pass", "ran", "random", "password", "ranpass":
		resp = ranPass(r)
	case "de-shortener", "de-shorten", "de-short", "deshort", "deshorten", "deshortener":
		validHeaders := []string{
			"og", "original",
			"url", "uniform resource locator", "u",
			"address", "short", "shortened", "a",
		}
		original := chkHeaders(validHeaders, "", r)
		if original == "" {
			bod, err := io.ReadAll(r.Body)
			if err != nil {
				resp = err.Error()
				break
			}; original = string(bod)
		};original = chkUrlPref(original)
		
		resp = deShortenURL(original)
	case "headers":
		resp = headers(r)
		w.Header().Set("Content-Type", "text/json")
	case "md", "markdown":
		resp = md(r)
/*	case "elh", "ELH":   //scrapped, it enables remote
		resp = elhFunc(r)*/  //  code execution
	case "syntax", "highlight":
		var err error
		resp, err = highlightCode(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	case "yt-dlp", "ytdlp", "yt", "youtube": 
		ytDlp(w, r)
		return
	default:
		if r.URL.Path == "/" {
			r.URL.Path = "/index"
		}
		if useWebUI { web(w, r)
		} else { http.Error(w, "404 not found", 404) }
		return
	}
	log.Infof("[req]: %s", r.URL.Path[1:])
	if !selfHan {
		w.Write([]byte(fmt.Sprint(resp,"\n")))
	}
}

/*func endPt(pt map[string]string) string {
	var url string
	if url, ok := pt["source"].(string); !ok {
		log.Errorf("failed to get source. pt:  %v", pt)
		return "failed to get source"
	}
}*/

func web(w http.ResponseWriter, r *http.Request) {
	//crappy temporary workaround for
	//  reading headers when using elh
	dir := filepath.Dir(r.URL.Path[1:])
	filePath := filepath.Join(dir, "headers.json")
	log.Warn(filePath)
	ext := filepath.Ext(r.URL.Path)
	if ext == ".elh" || ext == "" {
		jsonHeaders, err := json.Marshal(r.Header)
		if err != nil {
			log.Errorf("%v", err)
		}
		err = os.WriteFile(filePath, jsonHeaders, 0644)
		if err != nil { log.Errorf("%v", err) }
	}


	resp, err := elh.ServeWithRegistry(w, r, registry)
	if err != nil { log.Error(err) }
	log.Infof("[req]: %s", resp)

	if ext == ".elh" || ext == "" {
		err = os.Remove(filePath)
		if err != nil { log.Errorf("%v", err) }
	}
}

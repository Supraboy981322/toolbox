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
	serverName string
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
	case "help":
		helpHan(w, r)
		selfHan = true
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

/* TODO: user-defined endpoints */
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

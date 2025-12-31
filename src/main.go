package main

import (
	"io"
	"os"
	"fmt"
	"time"
	"bytes"
	"io/fs"
	"embed"
	"errors"
	"syscall"
	"strings"
	"strconv"
	"net/http"
	"math/big"
	"os/signal"
//	"encoding/json"
	"path/filepath"
	"github.com/charmbracelet/log"
	"github.com/Supraboy981322/gomn"
	elh "github.com/Supraboy981322/ELH"
)

//go:embed web/*
var webUIdir embed.FS

var (
	port int
	useWebUI bool
	config gomn.Map
	tmpWebDir string
	serverName string
	endPtMap map[string]map[string]string
	srvErr = http.StatusInternalServerError
	
	suppNoExt = []string{
		".elh",
		".html",
	}

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
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	defer func() {
		log.Debug("cleaning up...")
		os.RemoveAll(tmpWebDir)
		log.Warn("exiting...")
	}()
	go func() {
		_ = <-sigs
		log.Debug("cleaning up...")
		os.RemoveAll(tmpWebDir)
		log.Warn("exiting...")
		os.Exit(0)
	}()
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
	var resp string
	//get the requested file
	file := r.URL.Path[1:]
	if file == "" { file = "index" }
	file = filepath.Join("web", file)

	file, err := chkFile(file)
	if errors.Is(err, fs.ErrNotExist) {
		http.Error(w, "404 not found", 404)
		return
	}

	//get the extension of the requested file
	ext := filepath.Ext(file)	
	if ext == "" { //if there is no ext
		//check against list of ext which can
		//  have no ext in url
		for i := 0; i < len(suppNoExt); i++ {
			checkFile := fmt.Sprintf("%s%s", file, suppNoExt[i])
			_, err := chkFile(file)
			if err == nil { //if the file exists
				file = checkFile //assume it's the correct one
				ext = suppNoExt[i]
				break
			} else if !errors.Is(err, os.ErrNotExist) {
				http.Error(w, "cannot check if file exists! Schrodinger's file:  "+err.Error(), http.StatusInternalServerError)
				resp = "500; Schrodinger's file"
			}
		}
	}
	fileByte, err := webUIdir.ReadFile(file)
	if err != nil {
		http.Error(w, "read file:  "+err.Error(), http.StatusInternalServerError)
		resp = "500; err reading file"
	}
	fileStr := string(fileByte)
	var result string
	//if the file is elh, parse it
	if ext == ".elh" {
		result, err = elh.RenderWithRegistry(fileStr, registry, r)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			fmt.Fprintf(w, "There appears to be an error in the `.elh` file %s", file)
			resp = "500; problem with elh file"
			log.Error(err)
		}
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintln(w, result)
	} else {
		fileReader := bytes.NewReader(fileByte)
		http.ServeContent(w, r, file, time.Now(), fileReader)
	}

	//colorize response log string
	if resp == "" { resp = "\033[32m"+file+"\033[0m"
	} else { resp = "\033[31m"+resp+"\033[0m" }

	//colorize file log string
	file = "\033[35m"+file+"\033[0m"

	//build string
	logStr := "\033[1;34m[req]:\033[0m "
	logStr += file+" | "
	logStr += "\033[1m[resp]:\033[0m "+resp
	//log it
	log.Print(logStr)
}

package main

import (
//	"os"
	"io"
	"fmt"
	"time"
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
	//	elh "github.com/Supraboy981322/ELH"
	"github.com/Supraboy981322/gomn"
	"github.com/gomarkdown/markdown"
)

var (
	port int
	config gomn.Map
	endPtMap map[string]map[string]string

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

func init() {
	var ok bool
	var err error
	log.SetLevel(log.DebugLevel)

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
	
	if port, ok = config["port"].(int); !ok {
		log.Fatal("failed to get server port")
	} else { log.Debug("success reading server port") }

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

func getBody(r *http.Request) (string, error) {
	bod, err := io.ReadAll(r.Body)
	if err != nil {
		return "", err
	}

	return string(bod), nil
}

func pageHandler(w http.ResponseWriter, r *http.Request) {
	var selfHan bool;var resp string
	switch r.URL.Path[1:] {

	case "no":
		resp = noReq()
	case "time":
		resp = timeFunc(r)
	case "discord":
		resp = discord(r)
	case "pass", "ranPass", "ran", "random", "password", "ranpass":
		resp = ranPass(r)
	case "de-shortener", "de-shorten", "de-short", "deshort", "deshorten", "deshortener":
		original := r.Header.Get("og")
		resp = deShortenURL(original)
	case "headers":
		resp = headers(r)
	case "md", "markdown":
		resp = md(r)
	default:
		bhtm(w, r)
		return
	}
	log.Infof("[req]: %s", r.URL.Path[1:])
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

func headers(r *http.Request) string {
	jsonHeaders, err := json.Marshal(r.Header)
	if err != nil {
		return err.Error()
	}
	return string(jsonHeaders)
}

func timeFunc(r *http.Request) string {
	curTime := time.Now()
	var res []string
	opts := []string{
		"month", "mon",
		"day", "d",
		"hour", "h",
		"min", "minute", "m",
		"sec", "second", "s",
		"year", "y",
		"day of week", "dow", "weekday",
		"nanoseconds", "nc",
		"miliseconds", "ms",
		"utc",
		"unix",
		"loc", "location",
		"fmt", "format",
		"rfc", "RFC3339", "rfc3389",
	}
	for _, opt := range opts {
		set := r.Header.Get(opt)
		if set != "" {
			var newVal string
			switch opt {
			case "month", "mon":
				newVal = curTime.Month().String()
			case "day", "d":
				newVal = strconv.Itoa(curTime.Day())
			case "hour", "h":
				newVal = strconv.Itoa(curTime.Hour())
			case "min", "minute", "m":
				newVal = strconv.Itoa(curTime.Minute())
			case "sec", "second", "s":
				newVal = strconv.Itoa(curTime.Second())
			case "year", "y":
				newVal = strconv.Itoa(curTime.Year())
			case "day of week", "dow", "weekday":
				newVal = curTime.Weekday().String()
			case "loc", "location":
				newVal = curTime.Location().String()
			case "utc":
				newVal = curTime.UTC().String()
			case "unix":
				newVal = strconv.FormatInt(curTime.Unix(), 10)
			case "fmt", "format":
				newVal = curTime.Format(set)
			case "rfc", "RFC3339":
				newVal = curTime.Format(time.RFC3339)
			default:
				continue
			}
			res = append(res, newVal)
		}
	}
	if len(res) == 0 {
		res = append(res, curTime.String())
	}
	return strings.Join(res, " ")
}

func md(r *http.Request) string {
	var mdStr string
	for _, chk := range []string{"md", "markdown"} {
		if mdStr == "" {
			mdStr = r.Header.Get(chk)
		} else { break }
	}; if mdStr == "" {
		if bod, err := getBody(r); err == nil {
			if bod != "" {
				mdStr = bod 
			} else { return "no input" }
		} else { return err.Error() }
	}

	res := markdown.ToHTML([]byte(mdStr), nil, nil)

	return string(res)
}

/*func endPt(pt map[string]string) string {
	var url string
	if url, ok := pt["source"].(string); !ok {
		log.Errorf("failed to get source. pt:  %v", pt)
		return "failed to get source"
	}
}*/

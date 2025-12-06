package main

import (
	"os"
	"io"
//	"fmt"
	"time"
	"bytes"
//	"context"
	"errors"
	"strings"
	"strconv"
	"net/url"
	"os/exec"
	"math/big"
	"net/http"
	"io/ioutil"
	"crypto/rand"
	"encoding/json"
  "github.com/charmbracelet/log"
	"github.com/alecthomas/chroma"
	"github.com/gomarkdown/markdown"
	elh "github.com/Supraboy981322/ELH"
	"github.com/alecthomas/chroma/lexers"
	"github.com/alecthomas/chroma/styles"
	"github.com/alecthomas/chroma/formatters/html"
)

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
	
	if loc == "" {
		loc = original
	};return loc
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

//scrapped because I realized this
//  enables remote code execution
/*func elhFunc(r *http.Request) string {
	src := chkHeaders([]string{
			"src", "source", "s"}, getBodyNoErr(r), r)
	if src == "" { return "no input" }

	res, err := elh.Render(src, r)
	if err != nil { return err.Error() }

	return res
}*/

func highlightCode(r *http.Request) (string, error) {
	//get language from headers 
	var lang string
	lang = chkHeaders([]string{
			"l", "lang", "language"}, "", r)
	//return err if no language
	if lang == "" { return "", errors.New("need language") }

	//get syle from headers
	var style *chroma.Style 
	if s := chkHeaders([]string{
				"style", "theme", "t"}, "", r); s != "" {
		style = styles.Get(s)
	} else { style = styles.Fallback }
	
	//get the source code from headers,
	//  fallback to body
	code := chkHeaders([]string{
			"code", "src", "source", 
			"c", "s"}, getBodyNoErr(r), r)
	//return err if no code 
	if code == "" { return "", errors.New("no code input") }

	//set the lexer for language 
	lexer := lexers.Get(lang)
	if lexer == nil {
		lexer = lexers.Fallback
	}
	
	//set the formatter for html output
	formatter := html.New(html.WithClasses(true))
	iterator, err := lexer.Tokenise(nil, code)
	if err != nil { return "", err }
 
	var res string

	//format the code to html
	var buf bytes.Buffer
	err = formatter.Format(&buf, style, iterator)
	if err != nil { return "", err }
	res += buf.String()

	//generate the css
	var cssBuf bytes.Buffer
	err = formatter.WriteCSS(&cssBuf, style)
	if err != nil { return "", err }
	res += cssBuf.String()

	//return result
	return res, nil
}

func ytDlp(w http.ResponseWriter, r *http.Request) {
	//let client know it's about to 
	//  recieve raw binary data
	w.Header().Set("Content-Type", "application./octet-stream")

	//get the format from headers,
	//  defaults to mp4
	format := chkHeaders([]string{
			"fmt", "format", "f",
		}, "mp4", r)

	//get quality arg from headers
	//  defaults to `bestvideo+bestaudio/best`
	quality := chkHeaders([]string{
			"quality", "qual", "q",
		}, "bestvideo+bestaudio/best", r)

	//get the url from headers,
	//  with fallback to the req body
	url := chkHeaders([]string{
			"url", "source", "src", "addr",
			"u", "address", "video", "song",
			"v",
		}, getBodyNoErr(r), r)

	//quickly return err if no url 
	if url == "" {
		http.Error(w, "no url provided", http.StatusBadRequest)
		return
	}

	//args passed to yt-dlp
	args := []string{
		url,
		"-o", "-",
		"-q",
		"--recode-video", format,
		"-f", quality,
	}
	
	//yt-dlp cmd
	cmd := exec.Command("yt-dlp", args...)

	//create stdout buffer
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		http.Error(w, err.Error(), srvErr)
		return
	}; defer stdout.Close()

	//a multi-buffer output of
	//  cmd stderr
	var clientMsgBuff bytes.Buffer
	errBuff := io.MultiWriter(os.Stderr, &clientMsgBuff)
	cmd.Stderr = errBuff

	//exec cmd
	if err := cmd.Start(); err != nil {
		http.Error(w, err.Error(), srvErr)
		return
	}

	//stream yt-dlp output to client
	if _, err := io.Copy(w, stdout); err != nil {
		http.Error(w, err.Error(), srvErr)
		return
	}

	if err = cmd.Wait(); err != nil {
		//err buffer to string 
		errMsg := clientMsgBuff.String()

		//remove the `ERROR: ` part
		//  of yt-dlp output
		indx := strings.IndexRune(errMsg, ' ')
		if indx != -1 { errMsg = errMsg[indx+1:] }

		//remove newline
		//  (yt-dlp inserts double newline)
		errMsg = strings.ReplaceAll(errMsg, "\n", "")

		//send err
		http.Error(w, errMsg, srvErr)
		return 
	}
}

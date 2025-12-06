package main

import (
	"os"
	"io"
//	"fmt"
	"time"
	"bytes"
//	"context"
	"errors"
	"slices"
	"strings"
	"strconv"
	"net/url"
	"os/exec"
	"math/big"
	"net/http"
	//"io/ioutil"
	"crypto/rand"
	"encoding/json"
//  "github.com/charmbracelet/log"
	"github.com/alecthomas/chroma"
	"github.com/gomarkdown/markdown"
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

//this one looks a little ugly, I know
//  I'm writing this comment when I'm
//    about to make an attempt to clean-up
func discord(r *http.Request) string {
	var ok bool
	webhook := chkHeaders([]string {
			"webhook", "w", "h", "hook"}, "", r)
	if webhook == "" {
		badWebHooks := []string{"your discord webhook", "", " "}
		webhook, ok = config["discord webhook"].(string)
		if !ok || slices.Contains(badWebHooks, webhook) {
			return "no discord webhook provided or set in config"
		}
	}

	body := chkHeaders([]string{
			"body", "bod", "b", "m", "msg",
			"message", "text", "t"}, getBodyNoErr(r), r)
	if body == "" {	return "no message provided" }

	//construct payload
	payload := map[string]interface{}{
		"content": body,
	}

	//json, eww....
	data, err := json.Marshal(payload)
	if err != nil { return err.Error() }

	//send request
	conType := "application/json"
	datBuff := bytes.NewBuffer(data)
	_, err = http.Post(webhook, conType, datBuff)
	if err != nil {	return err.Error() }

	return "sent"
}

//random password generator
func ranPass(r *http.Request) string {
	//get the requested length, 
	//  fallback to 16
	lenStr := chkHeaders([]string{
			}, "16", r)
	if len(lenStr) >= 18 {
		return "denied; too long"
	}

	//get the set of characters
	charsRaw := chkHeaders([]string{
			"bld", "build", "chars", "c",
			"chars", "char", "charaters"}, "", r)

	//use default set if blank,
	//  and split into slice
	var charSet []string
	if charsRaw == "" { charSet = chars
	} else { charSet = strings.Split(charsRaw, "") }

	//convert the length string
	//  to a 64 bit integer 
	l, err := strconv.ParseInt(lenStr, 10, 64)
	if err != nil { return err.Error() }

	//so I can import
	//  one less module
	if l < 0 {
		l = -l
	}
	
	//because who needs a random string 
	//  any longer than 56527 characters?
	if l > 56527 {
		return "What the hell would you need "+
				"a cryptographically random string "+
				"longer than 56527 characters for?"
	}

	//actually generate
	var res string
	var i int64
	for i = 0; i < l; i++ {
		//convert to big.Int (for crypto/rand) 
		bigInt := big.NewInt(int64(len(charSet)))

		//generate random integer
		in, err := rand.Int(rand.Reader, bigInt)
		if err != nil { return err.Error() }

		//convert to regular integer
		ranDig := int(in.Int64())

		//add char of random index
		//  to result
		res += charSet[ranDig]
	}

	//finally,
	//  return the result
	return res
}

//no comments needed, just returns
//  headers as json
func headers(r *http.Request) string {
	jsonHeaders, err := json.Marshal(r.Header)
	if err != nil {
		return err.Error()
	}
	return string(jsonHeaders)
}

//needlessly long function to return
//  the time (because why not? I'm bored)
func timeFunc(r *http.Request) string {
	//current time
	curTime := time.Now()
	//placeholder for result
	var res []string

	//valid options
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

	//get headers
	//  (I know, eww, long switch statement)
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
			//add to result slice
			res = append(res, newVal)
		}
	}

	//if no valid headers sent,
	//  default to the current time
	if len(res) == 0 {
		res = append(res, curTime.String())
	}

	//return result as one string
	return strings.Join(res, " ")
}

func md(r *http.Request) string {
	//get md from headers, fallback
	//  to body, return err if empty
	mdStr := chkHeaders([]string{
		"md", "markdown"}, getBodyNoErr(r), r)
	if mdStr == "" { return "no input" }

	//render HTML as md 
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

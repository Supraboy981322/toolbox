package main

import (
	"os"
	"io"
	"fmt"
	"time"
	"bytes"
	"errors"
	"slices"
	"strings"
	"strconv"
	"net/url"
	"os/exec"
	"math/big"
	"net/http"
	"crypto/rand"
	"encoding/json"
//  "github.com/charmbracelet/log" //used to need this, might need again 
	"github.com/alecthomas/chroma"
	"github.com/gomarkdown/markdown"
	"github.com/alecthomas/chroma/lexers"
	"github.com/alecthomas/chroma/styles"
	"github.com/alecthomas/chroma/formatters/html"
)

//url de-shortener
func deShortenURL(original string) string {
	//immediately err if no url 
	if original == "" {	return "url provided"	}

	//create client that just checks
	//  for redirects
	client := &http.Client{
		CheckRedirect: 
			func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
	}

	var loc string //to store final url
	currentURL := original //start at input url

	//infinite loop only broken by
	//  hitting final address
	for {
		//create request with for current url 
		req, err := http.NewRequest("HEAD", currentURL, nil)
		if err != nil {
			return err.Error()
		}

		//send request 
		resp, err := client.Do(req)
		if err != nil {
			//check what the err is
			uerr, ok := err.(*url.Error)
			//clear err if it's a redirection
			if ok && uerr.Err == http.ErrUseLastResponse {
				err = nil
			//otherwise, return err
			} else if err != nil { return err.Error()	}
		}

		//discard response body
		_, _ = io.Copy(io.Discard, resp.Body)
		resp.Body.Close()

		//move to next request if redirected
		if resp.StatusCode >= 300 && resp.StatusCode < 400 {
			location, err := resp.Location()
			if err != nil {
				return err.Error()
			}
			loc = location.String()
			currentURL = loc
		} else {
			//otherwise,
			//  break the loop
			break
		}
	}
	
	//if never redirected,
	//  final url is original url
	if loc == "" { loc = original }

	//otherwise,
  //  return final
	return loc
}

//NaaS
//  (returns parsed json reason as plain-text)
func noReq() string {
	//yes, I'm just using their server,
	//  I'm not packaging a JS project with this
	resp, err := http.Get("https://naas.isalman.dev/no")
	if err != nil {	return err.Error() }
	defer resp.Body.Close()

	//ensure api is fine
	if resp.StatusCode != http.StatusOK {
		return "recieved bad status code from api"
	}

	//get the response
	body, err := io.ReadAll(resp.Body)
	if err != nil { return err.Error() }

	//for parsing the json 
	type noJSON struct {
		Reason string `json:"reason"`
	}

	//unmarshal into json
	var no noJSON
	err = json.Unmarshal(body, &no)
	if err != nil {	return err.Error() }

	//return reason value
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
		"len", "l", "length"}, "16", r)
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
	format := chkHeaders([]string{
			"format", "f", "fmt"}, "key-value", r)

	var res string

	switch strings.ToLower(format) {
   case "json", "j":
		res += "{\n"
		for key, vals := range r.Header {
			res += "  \""+key+"\": [\n"
			for _, val := range vals {
				res += "    \""+val+"\",\n"
			}
			res = res[0:len(res)-2]
			res += "\n  ],\n"
		};res = res[0:len(res)-2]+"\n}\n"

   case "gomn", "g":
		for key, vals := range r.Header {
			res += "[\""+key+"\"] := {\n"
			for _, val := range vals {
				res += "  \""+val+"\",\n"
			}
			res += "}\n"
		}

	 case "kv", "k-v", "k_v", "k v": fallthrough
	 case "key-value", "key value", "key_value":
		for key, vals := range r.Header {
			res += key+": "
			for _, val := range vals {
				res += val+" "
			}
			res += "\n"
		}

	 default: return "invalid format\n  Accepts: 'json' or 'key-value'"
	}

	//remove trailing newline
	res = res[0:len(res)-1]
	return res
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
	//  defaults to webm
	format := chkHeaders([]string{
			"fmt", "format", "f",
		}, "webm", r)
	
	outHeader := fmt.Sprintf("attachment; filename=\"yt-dlpServer_%s.%s\"",
				time.Now().Format("2006-01-02_15-04-05"), format)
	w.Header().Set("Content-Disposition", outHeader)

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

  extraArgsR := chkHeaders([]string{
      "a", "args", "arg",
    }, "", r)

  extraArgs := strings.Split(extraArgsR, ";")
	fmt.Println(extraArgs)

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
	};args = append(args, extraArgs...)
	
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

		var indx int
		for _, l := range strings.Split(errMsg, "\n") {
			//remove the error type part
			//  of yt-dlp output
			indx = strings.IndexRune(l, ':')
			if indx == -1 { continue }
			errTyp := l[0:indx]
			errMsg = l[indx+1:]
			if errTyp == "ERROR" { break }
		}

		//remove newline
		//  (yt-dlp inserts double newline)
		errMsg = strings.ReplaceAll(errMsg, "\n", "")

		//send err
		http.Error(w, errMsg, srvErr)
		return 
	}
}

func statCode(w http.ResponseWriter, c string) {
	hj, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "couldn't hijack connction, "+
					"(can't bypass RFC status code limits)", 500)
		return
	}

	hj_conn, hj_rw_buf, err := hj.Hijack()
	if err != nil {
		http.Error(w, err.Error(), 500); return 
	};defer hj_conn.Close()
	
	lines := []string{
		fmt.Sprintf("HTTP/1.1 %s status code %s", c, c),
		"Server: hijacked (to bypass golang RFC)",
		"Content-Type: text/plain",
		fmt.Sprintf("Content-Length: %d", len(c)),
		"",
		c,
	};for _, li := range lines {
		fmt.Fprintf(hj_rw_buf, li+"\r\n")
	};hj_rw_buf.Flush()
}

func echo(r *http.Request) string {
	bod, err := io.ReadAll(r.Body)
	if err != nil { return err.Error() }

	return string(bod)
}

func rand_bytes(r *http.Request) ([]byte, error) {
	b, err := io.ReadAll(r.Body)
	if err != nil { return nil, err }

	l, err := strconv.Atoi(string(b))
	if err != nil { return nil, errors.New("not a number") }

	res := make([]byte, l)
	_, err = rand.Read(res)
	if err != nil { return nil, err }	

	return res, nil
}

func helpHan(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("want_html") != "" {
		w.Write([]byte("todo: html help menu"))
	} else {
		w.Header().Set("Content-Type", "text/plain")
		lines := []string{
			serverName+" --> help",
			"\tendpoints:",
			"\t\t\"help\": this",
			"\t\t\theaders:",
			"\t\t\t\t\"want_html\": bool, set to any value for true",
			"\t\t\"no\": No as a Service",
			"\t\t\theaders:",
			"\t\t\t\tN/A",
			"\t\t\"echo\", \"e\": echo request body",
			"\t\t\theaders:",
			"\t\t\t\tN/A",
			"\t\t\"time\": return the time",
			"\t\t\theaders:",
			"\t\t\t\t\"month\": include month (bool, any value is true)",
			"\t\t\t\t\"day\": include day (bool, any value is true)",
			"\t\t\t\t\"hour\": include the hour (bool, any value is true)",
			"\t\t\t\t\"minute\": include the minute (bool, any value is true)",
			"\t\t\t\t\"second\": include the second (bool, any value is true)",
			"\t\t\t\t\"year\": include the year (bool, any value is true)",
			"\t\t\t\t\"day of week\": include the weekday (bool, any value is true)",
			"\t\t\t\t\"nanoseconds\": include nanoseconds (bool, any value is true)",
			"\t\t\t\t\"miliseconds\": include miliseconds (bool, any value is true)",
			"\t\t\t\t\"utc\": include return in UTC format (bool, any value is true)",
			"\t\t\t\t\"unix\": include return in UNIX standard format (bool, any value is true)",
			"\t\t\t\t\"fmt\": include custom format (string, value is passed to time function as a format)",
			"\t\t\t\t\"rfc\": include time in RFC format (bool, any value is true)",
			"\t\t\"discord\": send a message to a Discord webhook",
			"\t\t\theaders:",
			"\t\t\t\t\"webhook\": override webhook in config (string, used as webhook url)",
			"\t\t\t\t\"message\": the message that's sent to Discord",
			"\t\t\t\t\tdefault: uses request body if message header is empty",
			"\t\t\"random\": random string generator",
			"\t\t\tnotes:",
			"\t\t\t\tgolang's crypto/rand library can be slow on some machines, so a sufficiently long string may take a while to generate", 
			"\t\t\theaders:",
			"\t\t\t\t\"length\": the length of the string",
			"\t\t\t\t\tdefault: if length unspecified, uses 16",
			"\t\t\t\t\ttype: int",
			"\t\t\t\t\tpurpose: determines how many times the random string builder func iterates",
			"\t\t\t\t\"characters\": custom set of characters to use",
			"\t\t\t\t\ttype: string",
			"\t\t\t\t\tpurpose: the output of a cryptographically random number generator is used to index a string for a char",
			"\t\t\t\t\tdefault: abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0987654321!@#$%^&*()-_=+[]{}|\\;:'\"<>/?.,",
	}
		for _, li := range lines {
			li = strings.ReplaceAll(li, "\t", "  ", )
			w.Write([]byte(li+"\n"))
		}
	}
}

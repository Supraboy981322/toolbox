package main

import (
	"os"
	"io"
	"sync"
	"bufio"
	"bytes"
	"strings"
	"os/exec"
	"net/http"
  "github.com/charmbracelet/log" //used to need this, might need again 
	elh "github.com/Supraboy981322/ELH"
)

var srvErr = http.StatusInternalServerError

var hub = newSSEHub()

type (
	sseClient struct {
		ch   chan string
		done chan struct{}
	}
	sseHub struct{
		mut sync.Mutex
		cli *sseClient
	}
)

func newSSEHub() *sseHub {
	return &sseHub{}
}

func (h *sseHub) reg() *sseClient {
	h.mut.Lock()
	defer h.mut.Unlock()
	c := &sseClient{
		ch: make(chan string, 16),
		done: make(chan struct{})
	}
	h.client = c
	return c
}

func (h *sseHub) unreg() {
	h.mut.Lock()
	defer h.mut.Unlock()
	if h.cli != nil {
		close(h.cli.done)
		close(h.cli.ch)
		h.cli = nil
	}
}

func (h *sseHub) bc(line string) {
	h.mut.Lock()
	defer h.mut.Unlock()
	if h.cli != nil {
		select {
		 case h.cli.ch <- line:
		 default: //drops slow client
		}
	}
}

func sseHan(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", srvErr)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	cli := hub.reg()
	defer hub.unreg()

	keepAlive := time.NewTicker(15 * time.Second)
	defer keepAlive.Stop()

	for {
		select {
		 case <-client.done:
			return
		 case line, ok := <-client.ch:
			if !ok {
				return
			}
			fmt.Fprintf(w, "data:  %s\n\n", line)
			flusher.Flush()
		 case <-keepAlive.C:
			fmt.Fprint(w, ": keep-alive\n\n")
			flusher.Flush()
		 case <-r.Context().Done():
			return
		}
	}
}

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request){
		if r.URL.Path == "/yt-dlp" {
			ytDlp(w, r)
			return
		}
		p, err := elh.Serve(w,r)
		log.Infof("%s  ;  %v", p, err)
	})
	log.Info("listening...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func ytDlp(w http.ResponseWriter, r *http.Request) {
	var err error
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
		"--newline",
//		"-q",
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
	stderrPipe, err := cmd.StderrPipe()
	if err != nil { log.Errorf("cmd.StderrPipe()   %v", err) }
	var clientMsgBuff bytes.Buffer
	var errBuff io.Writer
//	errBuff := io.MultiWriter(os.Stderr, &clientMsgBuff)
//	cmd.Stderr = errBuff

	//exec cmd
	if err = cmd.Start(); err != nil {
		http.Error(w, err.Error(), srvErr)
		return
	}

	//stream yt-dlp output to client
	go func() {
		if _, err = io.Copy(w, stdout); err != nil {
			//http.Error(w, err.Error(), srvErr)
			return
		}
	}()

	progChan := make(chan string)
	go func() {
	//	defer close(progChan)
		errBuff = io.MultiWriter(os.Stderr, &clientMsgBuff)
		scanner := bufio.NewScanner(stderrPipe)
		for scanner.Scan() {
			line := scanner.Text()
			_, _ = errBuff.Write(append([]byte(line), '\n'))

			progChan <- line
		}
/*		if err == nil {
			close(progChan)
		}*/
	}()

	go func() {
		for p := range progChan {
			log.Print("progChan: ", p)
		}
	}()

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
		close(progChan) //close progress channel last
		return
	}
	//close progress channel last
	//  (defer causes panic on err)
	close(progChan)
}

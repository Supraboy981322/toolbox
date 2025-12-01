package main

/*Bash embeded in HTML
 *  (just a stripped-down version of ELH)*/

import (
	"io"
	"os"
	"fmt"
	"time"
	"bytes"
	"errors"
	"strings"
	"context"
	"os/exec"
	"net/http"
	"encoding/json"
	"path/filepath"
	"github.com/charmbracelet/log"
)

type (
	Runner interface {
		Run(workingDir string, code string, tmp *os.File) (stdout string, stderr string, err error)
	}
	ExternalRunner struct {
		CmdName string
		Args []string
		Timeout time.Duration
		Env []string
		WorkDir string
	}
	ReqProps struct {
		Method string      `json:"method"`
		URL    string      `json:"url"`
		Host   string      `json:"host"`
		RemoteAddr string  `json:"remote_addr"`
		Header http.Header `json:"header"`
		QParams map[string]string `json:"query_params"`
		Body   json.RawMessage `json:"body,omitempty"`
	}
)

var (
	registery = map[string]Runner{
		"bash": &ExternalRunner{
			CmdName: "bash",
			Args:    []string{"-u"},
			Timeout: 5 * time.Second,
			Env:     os.Environ(),
			WorkDir: "",
		},
	}
)

func bhtm(w http.ResponseWriter, r *http.Request) {
	var file string
	if r.URL.Path[0] == '/' {
		file = "web"+r.URL.Path
	} else {
		file = "web/"+r.URL.Path
	}
	if file == "web/" {
		file = "web/index"
	} else if file[len(file)-1:] == "/" {
		file = fmt.Sprintf("%sindex", string(file[1:]))
		file, _ = checkIsDir(file)
	} else {
		file, _ = checkIsDir(file)
	}
	
	ext := filepath.Ext(file)
	if ext == "" {
		for _, chkExt := range []string{".bhtm", ".bhtml", ".html"} {
			checkFile := fmt.Sprintf("%s%s", file, chkExt)
			if _, err := os.Stat(checkFile); err == nil {
				file = checkFile
				ext = chkExt
				break
			} else if !errors.Is(err, os.ErrNotExist) {
				log.Error(err)
				retSrvrErr(w)
				break
			}
		}
	}

	log.Info("[req]: "+file)

	_, err := os.Stat(file)
	if !errors.Is(err, os.ErrNotExist) {
		fileByte, err := os.ReadFile(file)
		if err != nil {
			log.Error(err)
			retSrvrErr(w)
			return
		}
		fileStr := string(fileByte)
		var result string
		if ext == ".bhtm" || ext == ".bhtml" {
			filePathABS, err := filepath.Abs(file)
			workingDir := filepath.Dir(filePathABS)
			if err != nil {
				log.Error(err)
				retSrvrErr(w)
				return
			}
			result, err = parseAndRun(workingDir, fileStr, r)
			if err != nil {
				log.Error(err)
				retSrvrErr(w)
				w.Write([]byte("there appears to be a problem with the "+ext+" file "+file))
				return
			}
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(result))
			return
		} else {
			fileReader := bytes.NewReader(fileByte)
			http.ServeContent(w, r, file, time.Now(), fileReader)
		}
	} else {
		http.Error(w, "404 forbidden", http.StatusForbidden)
		return
	}
}

func checkIsDir(file string) (string, error) {
	fileInfo, err := os.Stat(file)
	if err != nil {
		return file, err
	}
	if fileInfo.IsDir() {
		file = fmt.Sprintf("%s/index", file)
	}
	return file, nil
}

func retSrvrErr(w http.ResponseWriter) {
	http.Error(w, "Internal Server Error", http.StatusInternalServerError)
}

func (r *ExternalRunner) Run(workingDir string, code string, tmp *os.File) (string, string, error) {
	r.WorkDir = workingDir
	var err error
	ctx := context.Background()
	if r.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, r.Timeout)
		defer cancel()
	}
	
	tmpName := tmp.Name()

	defer func() {
		tmp.Close()
		_ = os.Remove(tmpName)
	}()

	if _, err = io.WriteString(tmp, code); err != nil {
		return "", "", err
	}

	if err = tmp.Sync(); err != nil {
		return "", "", err
	}

	if err = tmp.Close(); err != nil {
		return "", "", err
	}

	args := append([]string{}, r.Args...)
	args = append(args, tmpName)
	cmd := exec.CommandContext(ctx, r.CmdName, args...)

	if r.WorkDir != "" {
		cmd.Dir = r.WorkDir
	}

	if r.Env != nil {
		cmd.Env = r.Env
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()

	stderrStr := stderr.String()
	stdoutStr := stdout.String()
	
	if ctx.Err() == context.DeadlineExceeded {
		return stdoutStr, stderrStr, errors.New("exec exceeded timeout")
	}

	if err != nil {
		return stdoutStr, stderrStr, err
	}

	return stdoutStr, stderrStr, nil
}

func parseAndRun(workingDir string, src string, req *http.Request) (string, error) {
	var out strings.Builder
	i := 0;n := len(src)
	for {
		rel := strings.Index(src[i:], "<$")
		if rel < 0 {
			out.WriteString(src[i:])
			break
		}
		
		start := i + rel
		out.WriteString(src[i:start])
		if start+2 >= n {
			out.WriteString("<$")
			i = start + 2
			continue
		}

		j := start + 2;k := j
		for k < n {
			c := src[k]
			if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') {
				k++
				continue
			}
			break
		}
		if k == j {
			out.WriteString("<$")
			i = j
			continue
		}
		lang := src[j:k]
		codeStart := k
		if codeStart < n && src[codeStart] == ' ' {
			codeStart++
		}
		if codeStart >= n {
			out.WriteString(src[start:])
			break
		}
		endRel := strings.Index(src[codeStart:], "$>")
		if endRel < 0 {
			out.WriteString(src[start:])
			break
		}
		end := codeStart + endRel
		code := src[codeStart:end]

		r, ok := registery[strings.ToLower(lang)]
		if !ok {
			return "", errors.New("unkown lang:  "+lang)
		}

		tmpDir, err := os.MkdirTemp("", "snippet*")
		if err != nil {
			return "", err
		}
		
		tmpLib, err := genLib(lang, req, tmpDir)
		if err != nil {
			return "", errors.New("generate lib:  "+err.Error())
		}

		fileName := fmt.Sprint(filepath.Base(tmpDir))
		file, err := os.Create(filepath.Join(tmpDir, fileName))
		if err != nil {
			return "", err 
		}

		code = formatCode(code, tmpLib, req)

		stdout, stderr, err := r.Run(workingDir, code, file)
		if err != nil {
			idx := strings.IndexRune(stderr, ' ')
			if idx != -1 {
				stderr = stderr[idx+1:]
			}
			fmtdErr := 	"runner failed ("+err.Error()+"):  `"+stderr+"`"
			return "", errors.New(fmtdErr)
		}

		out.WriteString(stdout)
		i = end + 2
		
		if err := os.Remove(tmpLib); err != nil {
			return "", err
		}
	}

	return out.String(), nil
}

func genLib(lang string, r *http.Request, tmpDir string) (string, error) {
	source, err := os.Open("libs/bhtm.bash")
	if err != nil {
		return "", err
	}
	defer source.Close()

	jsonHeaders, err := json.Marshal(r.Header)
	if err != nil {
		return "", err
	}

	err = os.WriteFile(filepath.Join(tmpDir,"headers.json"), jsonHeaders, 0644)
	if err != nil {
		return "", err
	}

	tmpLib, err := os.Create(filepath.Join(tmpDir, "bhtm"))
	if err != nil {
		return "", err
	}
	defer tmpLib.Close()

	_, err = io.Copy(tmpLib, source)
	if err != nil {
		return "", err
	}

	return tmpLib.Name(), nil
}

func formatCode(code string, tmpLib string, r *http.Request) string {
	libPath := filepath.Dir(tmpLib)
	reqProps := map[string]string{
		"Method":     r.Method,
		"Url":        r.URL.String(),
		"Host":       r.Host,
		"RemoteAddr": r.RemoteAddr,
	}
	for key, value := range reqProps {
		code = key+"=\""+value+"\"\n"+code
	}

	code = `Headers="$(cat '` +
				filepath.Join(libPath, "headers.json") +
				"')\"\n" + code
	code = "source " + tmpLib + "\n" + code
	return "#!/usr/bin/env bash\n" + code
}

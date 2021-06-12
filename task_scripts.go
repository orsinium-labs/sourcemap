package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sync"

	"go.uber.org/zap"
)

var rex = regexp.MustCompile(`//#\s*sourceMappingURL=(.*\.map)[\s\x00$]`)

// TaskScripts consumes JS scripts and extracts source map URLs
type TaskScripts struct {
	Logger  *zap.Logger
	In      chan *url.URL
	Out     chan *url.URL
	Visited map[string]struct{}
	Mutex   *sync.Mutex
}

func (TaskScripts) Name() string {
	return "scripts"
}

func (task *TaskScripts) Finish() {
	close(task.Out)
}

func (task *TaskScripts) URLs() <-chan *url.URL {
	return task.In
}

func (task *TaskScripts) Run(surl *url.URL) error {
	if task.visited(surl.String()) {
		return nil
	}
	resp, err := http.Get(surl.String())
	if err != nil {
		return fmt.Errorf("make http request: %v", err)
	}
	if resp.StatusCode >= 300 {
		return fmt.Errorf("invalid response: %s", resp.Status)
	}

	// get source map url from headers
	murl := resp.Header.Get("SourceMap")
	if murl == "" {
		murl = resp.Header.Get("X-SourceMap")
	}
	if murl == "" {
		// get source map url from comments
		murl, err = task.find(rex, resp.Body)
		if err != nil {
			return fmt.Errorf("read response body: %v", err)
		}
	}

	if murl != "" {
		murl, err := surl.Parse(murl)
		if err != nil {
			return fmt.Errorf("parse source map url: %v", err)
		}
		task.Out <- murl
		return nil
	}
	task.Logger.Debug("no source map found", zapURL(surl))
	return nil
}

func (task *TaskScripts) visited(url string) bool {
	task.Mutex.Lock()
	defer task.Mutex.Unlock()
	_, visited := task.Visited[url]
	if visited {
		return true
	}
	task.Visited[url] = struct{}{}
	return false
}

func (TaskScripts) find(rex *regexp.Regexp, stream io.Reader) (string, error) {
	prev := make([]byte, 1024)
	for {
		curr := make([]byte, 1024)
		n, err := stream.Read(curr)
		if n == 0 {
			return "", nil
		}
		if err != nil && err != io.EOF {
			return "", err
		}
		match := rex.FindSubmatch(append(prev, curr...))
		if match != nil {
			return string(match[1]), nil
		}
		prev = curr
	}
}

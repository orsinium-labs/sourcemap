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

func (c *Collector) runScripts() {
	wg := sync.WaitGroup{}
	for i := 0; i < c.Workers; i++ {
		c.spawn(&wg, c.workerScripts)
	}
	wg.Wait()
	close(c.maps)
}

func (c *Collector) workerScripts() {
	for url := range c.scripts {
		err := c.handleScript(url)
		if err != nil {
			c.Logger.Error("script handler error", zap.Error(err))
		}
	}
}

func (c *Collector) handleScript(surl *url.URL) error {
	c.Logger.Debug("checking script", zapURL(surl))
	resp, err := http.Get(surl.String())
	if err != nil {
		return fmt.Errorf("make http request: %v", err)
	}

	// get source map url from headers
	murl := resp.Header.Get("SourceMap")
	if murl == "" {
		murl = resp.Header.Get("X-SourceMap")
	}
	if murl == "" {
		// get source map url from comments
		murl, err = c.find(rex, resp.Body)
		if err != nil {
			return fmt.Errorf("read response body: %v", err)
		}
	}

	if murl != "" {
		murl, err := surl.Parse(murl)
		if err != nil {
			return fmt.Errorf("parse source map url: %v", err)
		}
		c.maps <- murl
		return nil
	}
	c.Logger.Debug("no source map found", zapURL(surl))
	return nil
}

func (Collector) find(rex *regexp.Regexp, stream io.Reader) (string, error) {
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

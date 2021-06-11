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

func (c *Collector) runScripts() {
	wg := sync.WaitGroup{}
	wg.Add(c.Workers)
	for i := 0; i < c.Workers; i++ {
		go c.workerScripts()
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
		murl = c.find(rex, resp.Body)
	}

	if murl != "" {
		murl, err := surl.Parse(murl)
		if err != nil {
			return fmt.Errorf("parse source map url: %v", err)
		}
		c.maps <- murl
		return nil
	}

	return nil
}

func (Collector) find(rex *regexp.Regexp, stream io.Reader) string {
	prev := make([]byte, 1024)
	curr := make([]byte, 1024)
	for {
		_, err := stream.Read(curr)
		if err == io.EOF {
			return ""
		}
		match := rex.FindSubmatch(append(prev, curr...))
		if match != nil {
			return string(match[1])
		}
		prev = curr
	}
}

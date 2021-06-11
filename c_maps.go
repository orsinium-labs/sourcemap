package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"sync"

	"go.uber.org/zap"
)

func (c *Collector) runMaps() {
	wg := sync.WaitGroup{}
	for i := 0; i < c.Workers; i++ {
		c.spawn(&wg, c.workerMaps)
	}
	wg.Wait()
	close(c.infos)
}

func (c *Collector) workerMaps() {
	for url := range c.maps {
		err := c.handleMap(url)
		if err != nil {
			c.Logger.Error("map handler error", zap.Error(err), zapURL(url))
		}
	}
}

func (c *Collector) handleMap(surl *url.URL) error {
	c.Logger.Debug("reading map", zapURL(surl))
	resp, err := http.Get(surl.String())
	if err != nil {
		return fmt.Errorf("make http request: %v", err)
	}
	if resp.StatusCode >= 300 {
		return fmt.Errorf("invalid response: %s", resp.Status)
	}

	var m SourceMap
	err = json.NewDecoder(resp.Body).Decode(&m)
	if err != nil {
		return fmt.Errorf("read JSON: %v", err)
	}

	for i, fname := range m.FileNames {
		fname = strings.ReplaceAll(fname, "../", ".")
		fname = strings.ReplaceAll(fname, "webpack://", "")
		fname = strings.ReplaceAll(fname, "://", "")
		fname = path.Join(c.Output, surl.Hostname(), fname)

		if i >= len(m.Contents) {
			return errors.New("sources is longer than sourcesContent")
		}
		if strings.HasPrefix(fname, "external ") {
			return errors.New("external source maps unsupported")
		}

		parent, _ := path.Split(fname)
		err = os.MkdirAll(parent, 0770)
		if err != nil {
			return fmt.Errorf("create dir: %v", err)
		}

		c.Logger.Debug("writing file", zap.String("path", fname))
		err = os.WriteFile(fname, []byte(m.Contents[i]), 0660)
		if err != nil {
			return fmt.Errorf("write file: %v", err)
		}
	}

	c.infos <- surl
	return nil
}

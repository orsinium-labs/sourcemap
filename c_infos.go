package main

import (
	"fmt"
	"net/url"
	"os"
	"path"

	"go.uber.org/zap"
)

func (c *Collector) runInfos() {
	for url := range c.infos {
		err := c.handleInfo(url)
		if err != nil {
			c.Logger.Error("info handler error", zap.Error(err))
		}
	}
}

func (c *Collector) handleInfo(surl *url.URL) error {
	fpath := path.Join(c.Output, surl.Hostname(), "_sources.txt")
	c.Logger.Debug("writing _sources.txt", zap.String("path", fpath))
	f, err := os.OpenFile(fpath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0660)
	if err != nil {
		return fmt.Errorf("open _sources.txt: %v", err)
	}
	defer f.Close()
	_, err = f.WriteString(surl.String() + "\n")
	if err != nil {
		return fmt.Errorf("write _sources.txt: %v", err)
	}
	return nil
}

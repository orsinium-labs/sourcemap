package main

import (
	"fmt"
	"net/http"
	"net/url"
	"sync"

	"github.com/PuerkitoBio/goquery"
	"go.uber.org/zap"
)

func (c *Collector) runPages() {
	wg := sync.WaitGroup{}
	wg.Add(c.Workers)
	for i := 0; i < c.Workers; i++ {
		go c.workerPages()
	}
	wg.Wait()
	close(c.scripts)
}

func (c *Collector) workerPages() {
	for url := range c.pages {
		err := c.handlePage(url)
		if err != nil {
			c.Logger.Error("page handler error", zap.Error(err))
		}
	}
}

func (c *Collector) handlePage(purl *url.URL) error {
	c.Logger.Debug("checking page", zapURL(purl))
	resp, err := http.Get(purl.String())
	if err != nil {
		return fmt.Errorf("make http request: %v", err)
	}
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return fmt.Errorf("make goquery doc: %v", err)
	}

	var qerr error
	doc.Find("script").Each(func(i int, s *goquery.Selection) {
		surl, _ := s.Attr("src")
		if surl != "" {
			surl, err := purl.Parse(surl)
			if err != nil {
				qerr = err
				return
			}
			c.scripts <- surl
		}
		surl, _ = s.Attr("data-src")
		if surl != "" {
			surl, err := purl.Parse(surl)
			if err != nil {
				qerr = err
				return
			}
			c.scripts <- surl
		}
	})
	if qerr != nil {
		return fmt.Errorf("parse script url: %v", qerr)
	}

	return nil
}

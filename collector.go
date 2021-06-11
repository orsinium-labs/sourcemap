package main

import (
	"net/url"
	"sync"

	"go.uber.org/zap"
)

func zapURL(u *url.URL) zap.Field {
	return zap.String("url", u.String())
}

type SourceMap struct {
	FileNames []string `json:"sources"`
	Contents  []string `json:"sourcesContent"`
}

type Collector struct {
	Logger  *zap.Logger
	Output  string
	Workers int
	Debug   bool

	pages   chan *url.URL
	scripts chan *url.URL
	maps    chan *url.URL
	infos   chan *url.URL
}

func (c *Collector) Init() {
	c.pages = make(chan *url.URL)
	c.scripts = make(chan *url.URL)
	c.maps = make(chan *url.URL)
	c.infos = make(chan *url.URL)
}

func (c *Collector) Add(purl string) error {
	parsed, err := url.Parse(purl)
	if err != nil {
		return err
	}
	c.pages <- parsed
	return nil
}

func (c *Collector) Close() {
	close(c.pages)
}

func (c *Collector) Run() {
	wg := sync.WaitGroup{}
	c.spawn(&wg, c.runPages)
	c.spawn(&wg, c.runScripts)
	c.spawn(&wg, c.runMaps)
	c.runInfos()
	wg.Wait()
}

func (Collector) spawn(wg *sync.WaitGroup, fn func()) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		fn()
	}()
}

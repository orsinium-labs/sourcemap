package main

import (
	"regexp"
	"sync"

	"github.com/gocolly/colly/v2"
	"go.uber.org/zap"
)

var rex = regexp.MustCompile(`//\s*sourceMappingURL\s*=\s*(.*)\s*$`)

type RawMap struct {
	Content []byte
	Host    string
}

type Explorer struct {
	URLs   <-chan string
	Maps   chan<- RawMap
	Logger *zap.Logger

	baseC   *colly.Collector
	scriptC *colly.Collector
	mapC    *colly.Collector
}

func (ex *Explorer) log(msg string, err error) {
	if err != nil {
		ex.Logger.Error(msg, zap.Error(err))
	}
}

func (ex *Explorer) Init() {
	ex.baseC = colly.NewCollector()
	ex.scriptC = ex.baseC.Clone()
	ex.mapC = ex.baseC.Clone()

	// extract scripts
	ex.baseC.OnResponse(func(resp *colly.Response) {
		f := zap.String("url", resp.Request.URL.String())
		ex.Logger.Debug("checking page", f)
	})
	ex.baseC.OnHTML("script[src]", func(el *colly.HTMLElement) {
		url := el.Request.AbsoluteURL(el.Attr("src"))
		ex.log("script collector", ex.scriptC.Visit(url))
	})

	// detect source map for the given script
	ex.scriptC.OnResponse(func(resp *colly.Response) {
		f := zap.String("url", resp.Request.URL.String())
		ex.Logger.Debug("checking script", f)
		var h string
		h = resp.Headers.Get("X-SourceMap")
		if h != "" {
			ex.log("map collector", ex.mapC.Visit(h))
			return
		}
		h = resp.Headers.Get("SourceMap")
		if h != "" {
			ex.log("map collector", ex.mapC.Visit(h))
			return
		}
		match := rex.FindSubmatch(resp.Body)
		if match != nil {
			ex.log("map collector", ex.mapC.Visit(string(match[1])))
			return
		}
		ex.Logger.Debug("no source map found", f)
	})

	// emit source map
	ex.mapC.OnResponse(func(resp *colly.Response) {
		f := zap.String("url", resp.Request.URL.String())
		ex.Logger.Debug("source map found", f)
		if resp.StatusCode != 200 {
			ex.Logger.Warn("cannot get source map", f)
			return
		}
		ex.Logger.Info("source map detected", f)
		ex.Maps <- RawMap{
			Content: resp.Body,
			Host:    resp.Request.URL.Hostname(),
		}
	})
}

func (ex *Explorer) Run() {
	wg := sync.WaitGroup{}
	for url := range ex.URLs {
		wg.Add(1)
		go func(url string) {
			defer wg.Done()
			ex.log("base collector", ex.baseC.Visit(url))
		}(url)
	}
	wg.Wait()
}

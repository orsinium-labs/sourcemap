package main

import (
	"net/url"
	"sync"

	"go.uber.org/zap"
)

func zapURL(u *url.URL) zap.Field {
	return zap.String("url", u.String())
}

type Task interface {
	Name() string
	URLs() <-chan *url.URL
	Run(*url.URL) error
	Finish()
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

	pages chan *url.URL
}

func (c *Collector) Init() {
	c.pages = make(chan *url.URL)
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
	scripts := make(chan *url.URL)
	maps := make(chan *url.URL)
	infos := make(chan *url.URL)

	c.spawn(&wg, c.runWorkers, &TaskPages{
		Logger: c.Logger,
		In:     c.pages,
		Out:    scripts,
	})
	c.spawn(&wg, c.runWorkers, &TaskScripts{
		Logger: c.Logger,
		In:     scripts,
		Out:    maps,
	})
	c.spawn(&wg, c.runWorkers, &TaskMaps{
		Logger: c.Logger,
		Output: c.Output,
		In:     maps,
		Out:    infos,
	})
	c.worker(&TaskInfos{
		Logger: c.Logger,
		Output: c.Output,
		In:     infos,
	})
	wg.Wait()
}

func (Collector) spawn(wg *sync.WaitGroup, fn func(Task), task Task) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		fn(task)
	}()
}

func (c *Collector) runWorkers(task Task) {
	wg := sync.WaitGroup{}
	for i := 0; i < c.Workers; i++ {
		c.spawn(&wg, c.worker, task)
	}
	wg.Wait()
	task.Finish()
}

func (c *Collector) worker(task Task) {
	for url := range task.URLs() {
		err := task.Run(url)
		if err != nil {
			c.Logger.Error(
				"task error",
				zap.Error(err),
				zap.String("task", task.Name()),
				zapURL(url),
			)
		}
	}
}

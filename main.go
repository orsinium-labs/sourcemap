package main

import (
	"bufio"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/spf13/pflag"
	"go.uber.org/zap"
)

func emit_urls(logger *zap.Logger, urls chan string) {
	stat, err := os.Stdin.Stat()
	if err != nil {
		logger.Error("cannot access stdin", zap.Error(err))
		return
	}
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			url := scanner.Text()
			url = strings.TrimSpace(url)
			if url != "" {
				urls <- url
			}
		}
		if err := scanner.Err(); err != nil {
			logger.Error("cannot read stdin", zap.Error(err))
			return
		}
	}
	close(urls)
}

func main() {
	// parse CLI arguments
	var err error
	var output string
	flags := pflag.NewFlagSet(os.Args[0], pflag.ExitOnError)
	flags.StringVar(&output, "output", "sources", "directory where to write the results to")
	err = flags.Parse(os.Args[1:])
	if err != nil {
		log.Printf("cannot create logger %v:", err)
		return
	}

	// create logger
	c := zap.NewDevelopmentConfig()
	c.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	logger, err := c.Build(zap.AddStacktrace(zap.PanicLevel))
	if err != nil {
		log.Printf("cannot create logger %v:", err)
		return
	}
	defer func() {
		err := logger.Sync()
		if err != nil {
			log.Printf("cannot sync logs: %v", err)
		}
	}()

	wg := sync.WaitGroup{}
	urls := make(chan string)
	maps := make(chan RawMap)

	// read URLs from stdin
	wg.Add(1)
	go func() {
		defer wg.Done()
		emit_urls(logger, urls)
	}()

	// start explorer, discover source maps
	ex := Explorer{
		URLs:   urls,
		Maps:   maps,
		Logger: logger,
	}
	ex.Init()
	wg.Add(1)
	go func() {
		defer wg.Done()
		ex.Run()
	}()

	// run parser, parse source maps and save resulting files
	p := Parser{
		Maps:   maps,
		Logger: logger,
		Output: output,
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		p.Run()
	}()

	logger.Info("running")
	wg.Wait()
	logger.Info("finished")
}

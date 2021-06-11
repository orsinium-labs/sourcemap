package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/spf13/pflag"
	"go.uber.org/zap"
)

func emit_urls(collector *Collector) error {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return fmt.Errorf("access stdin: %v", err)
	}
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			url := scanner.Text()
			url = strings.TrimSpace(url)
			if url != "" {
				err = collector.Add(url)
				if err != nil {
					return fmt.Errorf("add url into collector: %v", err)
				}
			}
		}
		err := scanner.Err()
		if err != nil {
			return fmt.Errorf("read stdin: %v", err)
		}
	}
	collector.Close()
	return nil
}

func main() {
	collector := Collector{}

	// parse CLI arguments
	flags := pflag.NewFlagSet(os.Args[0], pflag.ExitOnError)
	flags.StringVar(&collector.Output, "output", "sources", "directory where to write the results to")
	flags.IntVar(&collector.Workers, "workers", 20, "how many workers to start for each step")
	flags.BoolVar(&collector.Debug, "debug", false, "show debug log messages")
	err := flags.Parse(os.Args[1:])
	if err != nil {
		log.Printf("cannot create logger %v:", err)
		return
	}

	// create logger
	c := zap.NewDevelopmentConfig()
	if collector.Debug {
		c.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	} else {
		c.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	}
	collector.Logger, err = c.Build(zap.AddStacktrace(zap.PanicLevel))
	if err != nil {
		log.Printf("cannot create logger %v:", err)
		return
	}
	collector.Logger.Debug("debug messages enabled")
	defer func() {
		err := collector.Logger.Sync()
		if err != nil {
			log.Printf("cannot sync logs: %v", err)
		}
	}()

	wg := sync.WaitGroup{}
	collector.Init()
	wg.Add(1)
	go func() {
		defer wg.Done()
		collector.Run()
	}()

	collector.Logger.Info("running")
	err = emit_urls(&collector)
	if err != nil {
		collector.Logger.Error("cannot emit urls", zap.Error(err))
	}
	wg.Wait()
	collector.Logger.Info("finished")
}

package main

import (
	"bufio"
	"log"
	"os"
	"sync"

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
			urls <- scanner.Text()
		}
		if err := scanner.Err(); err != nil {
			logger.Error("cannot read stdin", zap.Error(err))
			return
		}
	}
}

func main() {
	wg := sync.WaitGroup{}
	urls := make(chan string)
	maps := make(chan RawMap)
	logger, err := zap.NewDevelopment()
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

	wg.Add(1)
	go func() {
		defer wg.Done()
		emit_urls(logger, urls)
	}()

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

	p := Parser{
		Maps:   maps,
		Logger: logger,
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

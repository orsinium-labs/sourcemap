package main

import (
	"fmt"
	"net/url"
	"os"
	"path"

	"go.uber.org/zap"
)

// TaskMaps consumes source maps URLs and saves them into _sources.txt file.
type TaskInfos struct {
	Logger *zap.Logger
	Output string
	In     chan *url.URL
}

func (TaskInfos) Name() string {
	return "infos"
}

func (task *TaskInfos) Finish() {}

func (task *TaskInfos) URLs() <-chan *url.URL {
	return task.In
}

func (task *TaskInfos) Run(surl *url.URL) error {
	fpath := path.Join(task.Output, surl.Hostname(), "_sources.txt")
	task.Logger.Debug("writing _sources.txt", zap.String("path", fpath))
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

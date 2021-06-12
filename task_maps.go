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
)

// TaskMaps consumes source maps URLs and extracts them into the file system
type TaskMaps struct {
	Output string
	In     chan *url.URL
	Out    chan *url.URL
}

func (TaskMaps) Name() string {
	return "maps"
}

func (task *TaskMaps) Finish() {
	close(task.Out)
}

func (task *TaskMaps) URLs() <-chan *url.URL {
	return task.In
}

func (task *TaskMaps) Run(surl *url.URL) error {
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
		fname = path.Join(task.Output, surl.Hostname(), fname)

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

		err = os.WriteFile(fname, []byte(m.Contents[i]), 0660)
		if err != nil {
			return fmt.Errorf("write file: %v", err)
		}
	}

	task.Out <- surl
	return nil
}

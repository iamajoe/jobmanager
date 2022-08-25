package status

import (
	"net/http"
	"time"
)

type StatusJob struct {
	isRunning bool
	isDone    bool
	err       error

	CheckInterval time.Duration
	Url           string
	Updater       func(status int, err error)
}

func (j *StatusJob) GetStatus() (bool, bool, error) {
	return j.isRunning, j.isDone, j.err
}

func (j *StatusJob) Start() {
	if j.isRunning {
		return
	}

	j.isRunning = true
	j.loop()
}

func (j *StatusJob) loop() {
	if !j.isRunning || j.isDone {
		return
	}

	// we need an updater, if we don't, no point in fetching
	if j.Updater != nil || j.Url == "" {
		// get the response status code to inform
		resp, err := http.Get(j.Url)
		j.Updater(resp.StatusCode, err)
	}

	// wait to run it again
	time.Sleep(j.CheckInterval)
	j.loop()
}

func (j *StatusJob) Stop() {
	j.isRunning = false
}

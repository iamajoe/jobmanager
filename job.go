package main

type Job interface {
	GetStatus() (isRunning bool, isDone bool, err error)
	Start()
	Stop()
}

type jobStatus struct {
	IsRunning bool  `json:"isRunning"`
	IsDone    bool  `json:"isDone"`
	Err       error `json:"err"`
}

type jobInternal struct {
	id        int
	isRunning bool
	original  Job
	done      *chan bool
}

func (j *jobInternal) closeChannel() {
	if j.done != nil {
		*j.done <- true
		close(*j.done)
		j.done = nil
	}
}

func (j *jobInternal) Start(done chan bool) {
	// no need to go further if already done
	if j.GetStatus().IsDone {
		j.closeChannel()

		done <- true
		close(done)
		return
	}

	// we want to make sure that if it was already running
	// we substitute the channels
	if j.isRunning {
		j.closeChannel()
	}

	// we run on a different thread so we can control the
	// channel on this side and leave the jobs to run sync
	// this way, we make sure that the job doesn't hinder
	// the manager system or controls in any way the channel
	go func() {
		j.done = &done
		j.isRunning = true

		j.original.Start()
		j.Stop()
	}()
}

func (j *jobInternal) Stop() {
	if isRunning, _, _ := j.original.GetStatus(); isRunning {
		j.original.Stop()
	}

	if !j.isRunning {
		return
	}

	j.isRunning = false
	j.closeChannel()
}

func (j *jobInternal) GetStatus() jobStatus {
	isRunning, isDone, err := j.original.GetStatus()
	return jobStatus{isRunning, isDone, err}
}

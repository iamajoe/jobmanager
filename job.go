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

func (j *jobInternal) Start(done chan bool) {
	// no need to go further if already done
	if j.GetStatus().IsDone {
		done <- true
		return
	}

	// make sure it is stopped first, we want to make sure
	// the channels went through in case we are starting
	// on top of another start
	j.Stop()

	// we run on a different thread so we can control the
	// channel on this side and leave the jobs to run sync
	// this way, we make sure that the job doesn't hinder
	// the manager system or controls in any way the channel
	go func() {
		j.done = &done
		j.isRunning = true

		j.original.Start()
		*j.done <- true
		j.done = nil
	}()
}

func (j *jobInternal) Stop() {
	if !j.isRunning {
		return
	}

	if j.done != nil {
		*j.done <- true
		j.done = nil
	}

	j.original.Stop()
	j.isRunning = false
}

func (j *jobInternal) GetStatus() jobStatus {
	isRunning, isDone, err := j.original.GetStatus()
	return jobStatus{isRunning, isDone, err}
}

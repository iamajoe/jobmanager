package main

import (
	"math"
	"sync"
)

type JobManager struct {
	idCount     int
	isRunning   bool
	maxDone     int
	maxWorkers  int
	queueLine   []jobInternal
	runningPool map[int]jobInternal
	doneJobs    []jobInternal
	processWg   *sync.WaitGroup
}

func (m *JobManager) addToDone(job jobInternal) {
	if !job.GetStatus().IsDone {
		return
	}

	// make sure the job isn't in any other list
	m.Remove(job.id)

	// check if already in the list
	for _, doneJob := range m.doneJobs {
		if doneJob.id == job.id {
			return
		}
	}

	m.doneJobs = append(m.doneJobs, job)

	// good habit to clean the list so that the system doesn't
	// become a huge memory hog, so we limit this to a specific number
	if len(m.doneJobs) > m.maxDone {
		diff := len(m.doneJobs) - m.maxDone
		m.doneJobs = m.doneJobs[diff:]
	}
}

func (m *JobManager) removeFromPool(id int) {
	if m.runningPool == nil {
		return
	}

	if job, ok := m.runningPool[id]; ok {
		job.Stop()
		delete(m.runningPool, id)
	}

	// now that the job is removed, we have space in the pool
	m.runNewJob()
}

func (m *JobManager) addToPool(job jobInternal) {
	if job.GetStatus().IsDone {
		m.addToDone(job)
		return
	}

	if m.runningPool == nil {
		return
	}

	// make sure it is not in queue
	m.removeFromQueue(job.id)

	m.runningPool[job.id] = job
}

func (m *JobManager) removeFromQueue(id int) {
	newList := []jobInternal{}

	for _, job := range m.queueLine {
		if job.id == id {
			continue
		}

		newList = append(newList, job)
	}

	m.queueLine = newList
}

func (m *JobManager) addToQueue(job jobInternal, isStart bool) {
	if job.GetStatus().IsDone {
		m.addToDone(job)
		return
	}

	// make sure it is not in pool
	m.removeFromPool(job.id)

	// check if already in the queue
	for _, queueJob := range m.queueLine {
		if queueJob.id == job.id {
			return
		}
	}

	if isStart {
		m.queueLine = append([]jobInternal{job}, m.queueLine...)
	} else {
		m.queueLine = append(m.queueLine, job)
	}

	// make sure the running pool is full
	m.runNewJob()
}

func (m *JobManager) runNewJob() {
	if m.runningPool == nil {
		return
	}

	if len(m.runningPool) >= m.maxWorkers || len(m.queueLine) == 0 || !m.isRunning {
		return
	}

	job := m.queueLine[0]
	m.addToPool(job)

	// set the job running on a different thread because we want
	// different workers, one per job
	go func() {
		done := make(chan bool, 1)
		job.Start(done)
		<-done

		// the add will make sure it is not done
		m.addToQueue(job, true)
	}()

	// do we still have slots for new jobs?
	m.runNewJob()
}

func (m *JobManager) Remove(id int) {
	m.removeFromQueue(id)
	m.removeFromPool(id)
}

func (m *JobManager) Add(job Job) int {
	// reset the id count when too big so we don't have issues
	m.idCount = m.idCount + 1
	if m.idCount >= math.MaxInt-1 {
		m.idCount = 0
	}

	jobInt := jobInternal{
		id:       m.idCount,
		original: job,
	}
	m.addToQueue(jobInt, false)

	return jobInt.id
}

func (m *JobManager) GetStatus() map[int]jobStatus {
	list := make(map[int]jobStatus)

	// iterate the various lists to fetch the status
	for _, job := range m.doneJobs {
		list[job.id] = job.GetStatus()
	}

	if m.runningPool != nil {
		for _, job := range m.runningPool {
			list[job.id] = job.GetStatus()
		}
	}
	for _, job := range m.queueLine {
		list[job.id] = job.GetStatus()
	}

	return list
}

func (m *JobManager) Start(maxWorkers int, maxDone int) {
	// keep the process running so that the goroutines don´t fall
	if m.processWg == nil {
		m.processWg = &sync.WaitGroup{}
		m.processWg.Add(1)
	}

	m.maxWorkers = maxWorkers
	m.maxDone = maxDone
	if maxDone > math.MaxInt || maxDone <= 0 {
		m.maxDone = math.MaxInt - 1
	}

	m.isRunning = true

	// make sure that the pool of running jobs is full and wait
	m.runNewJob()
	m.processWg.Wait()
}

func (m *JobManager) Stop() {
	m.isRunning = false

	// get all running jobs back to the queue line
	if m.runningPool != nil {
		for _, job := range m.runningPool {
			m.addToQueue(job, true)
		}
	}

	// end the process
	m.processWg.Done()
	m.processWg = nil
}

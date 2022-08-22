package main

import (
	"reflect"
	"sync"
	"testing"
)

func TestJobManager_addToDone(t *testing.T) {
	type fields struct {
		maxDone     int
		queueLine   []jobInternal
		runningPool map[int]jobInternal
		doneJobs    []jobInternal
	}
	type args struct {
		job jobInternal
	}
	type testStruct struct {
		name              string
		fields            fields
		args              args
		expectQueueLength int
		expectInRunning   bool
		expectDoneLength  int
		expectDoneStatus  bool
	}

	tests := []testStruct{
		{
			name:              "runs and adds to done when done",
			fields:            fields{maxDone: 1},
			args:              args{jobInternal{id: 11, original: &testJob{isDone: true}}},
			expectQueueLength: 0,
			expectInRunning:   false,
			expectDoneLength:  1,
			expectDoneStatus:  true,
		},
		func() testStruct {
			job := jobInternal{id: 12, original: &testJob{isDone: true}}
			runningPool := make(map[int]jobInternal)
			runningPool[job.id] = job

			return testStruct{
				name:              "runs and removes from running list",
				fields:            fields{maxDone: 1, runningPool: runningPool},
				args:              args{job},
				expectQueueLength: 0,
				expectInRunning:   false,
				expectDoneLength:  1,
				expectDoneStatus:  true,
			}
		}(),
		func() testStruct {
			job := jobInternal{id: 13, original: &testJob{isDone: true}}

			return testStruct{
				name:              "runs and removes from queue list",
				fields:            fields{maxDone: 1, queueLine: []jobInternal{job}},
				args:              args{job},
				expectQueueLength: 0,
				expectInRunning:   false,
				expectDoneLength:  1,
				expectDoneStatus:  true,
			}
		}(),
		func() testStruct {
			job0 := jobInternal{id: 14, original: &testJob{isDone: true}}
			job1 := jobInternal{id: 15, original: &testJob{isDone: true}}

			return testStruct{
				name:              "runs and removes from done list if max is over",
				fields:            fields{maxDone: 1, doneJobs: []jobInternal{job0}},
				args:              args{job1},
				expectQueueLength: 0,
				expectInRunning:   false,
				expectDoneLength:  1,
				expectDoneStatus:  true,
			}
		}(),
		{
			name:             "ignores job if not done",
			fields:           fields{maxDone: 1},
			args:             args{jobInternal{id: 16, original: &testJob{isDone: false}}},
			expectDoneLength: 0,
			expectDoneStatus: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &JobManager{
				maxDone:     tt.fields.maxDone,
				queueLine:   tt.fields.queueLine,
				runningPool: tt.fields.runningPool,
				doneJobs:    tt.fields.doneJobs,
			}
			m.addToDone(tt.args.job)

			if len(m.doneJobs) != tt.expectDoneLength ||
				len(m.queueLine) != tt.expectQueueLength {
				t.Error("arrays are not of the right length")
			}

			if m.runningPool != nil {
				if _, ok := m.runningPool[tt.args.job.id]; ok != tt.expectInRunning {
					t.Errorf("in running = %v, want %v", ok, tt.expectInRunning)
				}
			}

			if tt.expectDoneLength > 0 {
				if m.doneJobs[0].id != tt.args.job.id {
					t.Errorf("job id = %v, want %v", m.doneJobs[0].id, tt.args.job.id)
				}

				if m.doneJobs[0].GetStatus().IsDone != tt.expectDoneStatus {
					t.Errorf("job IsDone = %v, want %v", m.doneJobs[0].GetStatus().IsDone, tt.expectDoneStatus)
				}
			}
		})
	}
}

func TestJobManager_removeFromPool(t *testing.T) {
	type fields struct {
		maxWorkers  int
		queueLine   []jobInternal
		runningPool map[int]jobInternal
		doneJobs    []jobInternal
	}
	type args struct {
		job jobInternal
	}
	type testStruct struct {
		name              string
		fields            fields
		args              args
		expectQueueLength int
		expectInRunning   int
		expectJobRunning  bool
		expectDoneLength  int
		expectDoneStatus  bool
	}

	tests := []testStruct{
		func() testStruct {
			job0 := jobInternal{id: 10, original: &testJob{}}
			job1 := jobInternal{id: 11, original: &testJob{}}

			runningPool := make(map[int]jobInternal)
			runningPool[job0.id] = job0
			runningPool[job1.id] = job1

			return testStruct{
				name:              "runs and removes from running list",
				fields:            fields{runningPool: runningPool},
				args:              args{job1},
				expectQueueLength: 0,
				expectInRunning:   job0.id,
				expectJobRunning:  false,
				expectDoneLength:  0,
				expectDoneStatus:  false,
			}
		}(),
		func() testStruct {
			job := jobInternal{id: 12, original: &testJob{isRunning: true}}
			runningPool := make(map[int]jobInternal)
			runningPool[job.id] = job

			return testStruct{
				name:              "runs, removes from running list and stops the job",
				fields:            fields{runningPool: runningPool},
				args:              args{job},
				expectQueueLength: 0,
				expectInRunning:   -1,
				expectJobRunning:  false,
				expectDoneLength:  0,
				expectDoneStatus:  false,
			}
		}(),
		func() testStruct {
			job := jobInternal{id: 13, original: &testJob{isRunning: true}}
			runningPool := make(map[int]jobInternal)

			return testStruct{
				name:              "runs and ignores if the job isn't on the running pool",
				fields:            fields{runningPool: runningPool},
				args:              args{job},
				expectQueueLength: 0,
				expectInRunning:   -1,
				expectJobRunning:  true,
				expectDoneLength:  0,
				expectDoneStatus:  false,
			}
		}(),
		// TODO: test to make sure that the worker is full
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &JobManager{
				maxWorkers:  tt.fields.maxWorkers,
				queueLine:   tt.fields.queueLine,
				runningPool: tt.fields.runningPool,
				doneJobs:    tt.fields.doneJobs,
			}
			m.removeFromPool(tt.args.job.id)

			if len(m.doneJobs) != tt.expectDoneLength ||
				len(m.queueLine) != tt.expectQueueLength {
				t.Error("arrays are not of the right length")
			}

			if m.runningPool != nil {
				if _, ok := m.runningPool[tt.args.job.id]; ok {
					t.Error("it wasnt removed from the pool")
				}

				if tt.expectInRunning != -1 {
					if _, ok := m.runningPool[tt.expectInRunning]; !ok {
						t.Errorf("in running = %v, want %v", ok, tt.expectInRunning)
					}
				}
			}

			if tt.args.job.GetStatus().IsDone != tt.expectDoneStatus {
				t.Errorf("job IsDone = %v, want %v", tt.args.job.GetStatus().IsDone, tt.expectDoneStatus)
			}

			if tt.args.job.GetStatus().IsRunning != tt.expectJobRunning {
				t.Errorf("job IsRunning = %v, want %v", tt.args.job.GetStatus().IsRunning, tt.expectJobRunning)
			}
		})
	}
}

func TestJobManager_addToPool(t *testing.T) {
	type fields struct {
		idCount     int
		isRunning   bool
		maxDone     int
		maxWorkers  int
		queueLine   []jobInternal
		runningPool map[int]jobInternal
		doneJobs    []jobInternal
		processWg   *sync.WaitGroup
	}
	type args struct {
		job jobInternal
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &JobManager{
				idCount:     tt.fields.idCount,
				isRunning:   tt.fields.isRunning,
				maxDone:     tt.fields.maxDone,
				maxWorkers:  tt.fields.maxWorkers,
				queueLine:   tt.fields.queueLine,
				runningPool: tt.fields.runningPool,
				doneJobs:    tt.fields.doneJobs,
				processWg:   tt.fields.processWg,
			}
			m.addToPool(tt.args.job)
		})
	}
}

func TestJobManager_removeFromQueue(t *testing.T) {
	type fields struct {
		idCount     int
		isRunning   bool
		maxDone     int
		maxWorkers  int
		queueLine   []jobInternal
		runningPool map[int]jobInternal
		doneJobs    []jobInternal
		processWg   *sync.WaitGroup
	}
	type args struct {
		id int
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &JobManager{
				idCount:     tt.fields.idCount,
				isRunning:   tt.fields.isRunning,
				maxDone:     tt.fields.maxDone,
				maxWorkers:  tt.fields.maxWorkers,
				queueLine:   tt.fields.queueLine,
				runningPool: tt.fields.runningPool,
				doneJobs:    tt.fields.doneJobs,
				processWg:   tt.fields.processWg,
			}
			m.removeFromQueue(tt.args.id)
		})
	}
}

func TestJobManager_addToQueue(t *testing.T) {
	type fields struct {
		idCount     int
		isRunning   bool
		maxDone     int
		maxWorkers  int
		queueLine   []jobInternal
		runningPool map[int]jobInternal
		doneJobs    []jobInternal
		processWg   *sync.WaitGroup
	}
	type args struct {
		job     jobInternal
		isStart bool
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &JobManager{
				idCount:     tt.fields.idCount,
				isRunning:   tt.fields.isRunning,
				maxDone:     tt.fields.maxDone,
				maxWorkers:  tt.fields.maxWorkers,
				queueLine:   tt.fields.queueLine,
				runningPool: tt.fields.runningPool,
				doneJobs:    tt.fields.doneJobs,
				processWg:   tt.fields.processWg,
			}
			m.addToQueue(tt.args.job, tt.args.isStart)
		})
	}
}

func TestJobManager_runNewJob(t *testing.T) {
	type fields struct {
		idCount     int
		isRunning   bool
		maxDone     int
		maxWorkers  int
		queueLine   []jobInternal
		runningPool map[int]jobInternal
		doneJobs    []jobInternal
		processWg   *sync.WaitGroup
	}
	tests := []struct {
		name   string
		fields fields
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &JobManager{
				idCount:     tt.fields.idCount,
				isRunning:   tt.fields.isRunning,
				maxDone:     tt.fields.maxDone,
				maxWorkers:  tt.fields.maxWorkers,
				queueLine:   tt.fields.queueLine,
				runningPool: tt.fields.runningPool,
				doneJobs:    tt.fields.doneJobs,
				processWg:   tt.fields.processWg,
			}
			m.runNewJob()
		})
	}
}

func TestJobManager_Remove(t *testing.T) {
	type fields struct {
		idCount     int
		isRunning   bool
		maxDone     int
		maxWorkers  int
		queueLine   []jobInternal
		runningPool map[int]jobInternal
		doneJobs    []jobInternal
		processWg   *sync.WaitGroup
	}
	type args struct {
		id int
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &JobManager{
				idCount:     tt.fields.idCount,
				isRunning:   tt.fields.isRunning,
				maxDone:     tt.fields.maxDone,
				maxWorkers:  tt.fields.maxWorkers,
				queueLine:   tt.fields.queueLine,
				runningPool: tt.fields.runningPool,
				doneJobs:    tt.fields.doneJobs,
				processWg:   tt.fields.processWg,
			}
			m.Remove(tt.args.id)
		})
	}
}

func TestJobManager_Add(t *testing.T) {
	type fields struct {
		idCount     int
		isRunning   bool
		maxDone     int
		maxWorkers  int
		queueLine   []jobInternal
		runningPool map[int]jobInternal
		doneJobs    []jobInternal
		processWg   *sync.WaitGroup
	}
	type args struct {
		job Job
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   int
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &JobManager{
				idCount:     tt.fields.idCount,
				isRunning:   tt.fields.isRunning,
				maxDone:     tt.fields.maxDone,
				maxWorkers:  tt.fields.maxWorkers,
				queueLine:   tt.fields.queueLine,
				runningPool: tt.fields.runningPool,
				doneJobs:    tt.fields.doneJobs,
				processWg:   tt.fields.processWg,
			}
			if got := m.Add(tt.args.job); got != tt.want {
				t.Errorf("JobManager.Add() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestJobManager_GetStatus(t *testing.T) {
	type fields struct {
		idCount     int
		isRunning   bool
		maxDone     int
		maxWorkers  int
		queueLine   []jobInternal
		runningPool map[int]jobInternal
		doneJobs    []jobInternal
		processWg   *sync.WaitGroup
	}
	tests := []struct {
		name   string
		fields fields
		want   map[int]jobStatus
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &JobManager{
				idCount:     tt.fields.idCount,
				isRunning:   tt.fields.isRunning,
				maxDone:     tt.fields.maxDone,
				maxWorkers:  tt.fields.maxWorkers,
				queueLine:   tt.fields.queueLine,
				runningPool: tt.fields.runningPool,
				doneJobs:    tt.fields.doneJobs,
				processWg:   tt.fields.processWg,
			}
			if got := m.GetStatus(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("JobManager.GetStatus() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestJobManager_Start(t *testing.T) {
	type fields struct {
		idCount     int
		isRunning   bool
		maxDone     int
		maxWorkers  int
		queueLine   []jobInternal
		runningPool map[int]jobInternal
		doneJobs    []jobInternal
		processWg   *sync.WaitGroup
	}
	type args struct {
		maxWorkers int
		maxDone    int
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &JobManager{
				idCount:     tt.fields.idCount,
				isRunning:   tt.fields.isRunning,
				maxDone:     tt.fields.maxDone,
				maxWorkers:  tt.fields.maxWorkers,
				queueLine:   tt.fields.queueLine,
				runningPool: tt.fields.runningPool,
				doneJobs:    tt.fields.doneJobs,
				processWg:   tt.fields.processWg,
			}
			m.Start(tt.args.maxWorkers, tt.args.maxDone)
		})
	}
}

func TestJobManager_Stop(t *testing.T) {
	type fields struct {
		idCount     int
		isRunning   bool
		maxDone     int
		maxWorkers  int
		queueLine   []jobInternal
		runningPool map[int]jobInternal
		doneJobs    []jobInternal
		processWg   *sync.WaitGroup
	}
	tests := []struct {
		name   string
		fields fields
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &JobManager{
				idCount:     tt.fields.idCount,
				isRunning:   tt.fields.isRunning,
				maxDone:     tt.fields.maxDone,
				maxWorkers:  tt.fields.maxWorkers,
				queueLine:   tt.fields.queueLine,
				runningPool: tt.fields.runningPool,
				doneJobs:    tt.fields.doneJobs,
				processWg:   tt.fields.processWg,
			}
			m.Stop()
		})
	}
}

package main

import (
	"encoding/json"
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
		func() testStruct {
			job := jobInternal{id: 11, original: &testJob{isDone: false}}
			runningPool := make(map[int]jobInternal)

			return testStruct{
				name:              "runs and adds to the pool when not done",
				fields:            fields{maxDone: 1, runningPool: runningPool},
				args:              args{job},
				expectQueueLength: 0,
				expectInRunning:   true,
				expectDoneLength:  0,
				expectDoneStatus:  false,
			}
		}(),
		func() testStruct {
			job := jobInternal{id: 12, original: &testJob{isDone: true}}
			runningPool := make(map[int]jobInternal)

			return testStruct{
				name:              "runs and doesn´t add to the pool when done",
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
			runningPool := make(map[int]jobInternal)

			return testStruct{
				name:              "runs and adds to the pool and removes from queue",
				fields:            fields{maxDone: 1, runningPool: runningPool, queueLine: []jobInternal{job}},
				args:              args{job},
				expectQueueLength: 0,
				expectInRunning:   false,
				expectDoneLength:  1,
				expectDoneStatus:  true,
			}
		}(),
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &JobManager{
				maxDone:     tt.fields.maxDone,
				queueLine:   tt.fields.queueLine,
				runningPool: tt.fields.runningPool,
				doneJobs:    tt.fields.doneJobs,
			}
			m.addToPool(tt.args.job)

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

func TestJobManager_removeFromQueue(t *testing.T) {
	type fields struct {
		queueLine []jobInternal
	}
	type args struct {
		job jobInternal
	}
	type testStruct struct {
		name              string
		fields            fields
		args              args
		expectInQueue     bool
		expectQueueLength int
		expectJobRunning  bool
		expectDoneLength  int
	}

	tests := []testStruct{
		func() testStruct {
			job0 := jobInternal{id: 10, original: &testJob{}}
			job1 := jobInternal{id: 11, original: &testJob{}}

			return testStruct{
				name:              "runs and removes from queue list",
				fields:            fields{queueLine: []jobInternal{job0, job1}},
				args:              args{job1},
				expectInQueue:     false,
				expectQueueLength: 1,
				expectJobRunning:  false,
				expectDoneLength:  0,
			}
		}(),
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &JobManager{
				queueLine: tt.fields.queueLine,
			}
			m.removeFromQueue(tt.args.job.id)

			found := false
			for _, val := range m.queueLine {
				if val.id == tt.args.job.id {
					found = true

					if !tt.expectInQueue {
						t.Error("job shouldn't be in queue")
					}
				}
			}

			if !found && tt.expectInQueue {
				t.Error("job should be in queue")
			}

			if len(m.doneJobs) != tt.expectDoneLength ||
				len(m.queueLine) != tt.expectQueueLength {
				t.Error("arrays are not of the right length")
			}

			if m.runningPool != nil {
				if _, ok := m.runningPool[tt.args.job.id]; ok {
					t.Error("it wasnt removed from the pool")
				}
			}

			if tt.args.job.GetStatus().IsRunning != tt.expectJobRunning {
				t.Errorf("job IsRunning = %v, want %v", tt.args.job.GetStatus().IsRunning, tt.expectJobRunning)
			}
		})
	}
}

func TestJobManager_addToQueue(t *testing.T) {
	type fields struct {
		maxDone     int
		queueLine   []jobInternal
		runningPool map[int]jobInternal
		doneJobs    []jobInternal
	}
	type args struct {
		job     jobInternal
		isStart bool
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
		func() testStruct {
			job := jobInternal{id: 11, original: &testJob{isDone: false}}
			runningPool := make(map[int]jobInternal)

			return testStruct{
				name:              "runs and adds to the queue when not done, to the back",
				fields:            fields{maxDone: 1, runningPool: runningPool},
				args:              args{job, false},
				expectQueueLength: 1,
				expectInRunning:   false,
				expectDoneLength:  0,
				expectDoneStatus:  false,
			}
		}(),
		func() testStruct {
			job := jobInternal{id: 12, original: &testJob{isDone: false}}
			runningPool := make(map[int]jobInternal)

			return testStruct{
				name:              "runs and adds to the queue when not done, on the start",
				fields:            fields{maxDone: 1, runningPool: runningPool},
				args:              args{job, true},
				expectQueueLength: 1,
				expectInRunning:   false,
				expectDoneLength:  0,
				expectDoneStatus:  false,
			}
		}(),
		func() testStruct {
			job := jobInternal{id: 13, original: &testJob{isDone: true}}
			runningPool := make(map[int]jobInternal)

			return testStruct{
				name:              "runs and doesn´t add to the queue when done",
				fields:            fields{maxDone: 1, runningPool: runningPool},
				args:              args{job, false},
				expectQueueLength: 0,
				expectInRunning:   false,
				expectDoneLength:  1,
				expectDoneStatus:  true,
			}
		}(),
		func() testStruct {
			job := jobInternal{id: 14, original: &testJob{isDone: false}}
			runningPool := make(map[int]jobInternal)
			runningPool[job.id] = job

			return testStruct{
				name:              "runs and adds to the queue and removes from pool",
				fields:            fields{maxDone: 1, runningPool: runningPool, queueLine: []jobInternal{job}},
				args:              args{job, false},
				expectQueueLength: 1,
				expectInRunning:   false,
				expectDoneLength:  0,
				expectDoneStatus:  false,
			}
		}(),

		// TODO: make sure the running pool is full
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &JobManager{
				maxDone:     tt.fields.maxDone,
				queueLine:   tt.fields.queueLine,
				runningPool: tt.fields.runningPool,
				doneJobs:    tt.fields.doneJobs,
			}
			m.addToQueue(tt.args.job, tt.args.isStart)

			if len(m.doneJobs) != tt.expectDoneLength {
				t.Errorf("done array = %v, want %v", len(m.doneJobs), tt.expectDoneLength)
				return
			}

			if len(m.queueLine) != tt.expectQueueLength {
				t.Errorf("queue array = %v, want %v", len(m.queueLine), tt.expectQueueLength)
				return
			}

			if tt.expectQueueLength > 0 {
				if tt.args.isStart && m.queueLine[0].id != tt.args.job.id {
					t.Error("job was not at the start of the queue line")
				} else if !tt.args.isStart && m.queueLine[len(m.queueLine)-1].id != tt.args.job.id {
					t.Error("job was not at the end of the queue line")
				}
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
		queueLine   []jobInternal
		runningPool map[int]jobInternal
		doneJobs    []jobInternal
	}
	type testStruct struct {
		name   string
		fields fields
		want   string
	}

	tests := []testStruct{
		func() testStruct {
			job0 := jobInternal{id: 11, original: &testJob{isDone: false}}
			job1 := jobInternal{id: 12, original: &testJob{isDone: true}}
			job2 := jobInternal{id: 13, original: &testJob{isDone: false}}
			job3 := jobInternal{id: 14, original: &testJob{isDone: false}}

			runningPool := make(map[int]jobInternal)
			runningPool[job0.id] = job0

			return testStruct{
				name: "example 1",
				fields: fields{
					queueLine:   []jobInternal{job2, job3},
					runningPool: runningPool,
					doneJobs:    []jobInternal{job1},
				},
				want: "{\"11\":{\"isRunning\":false,\"isDone\":false,\"err\":null},\"12\":{\"isRunning\":false,\"isDone\":true,\"err\":null},\"13\":{\"isRunning\":false,\"isDone\":false,\"err\":null},\"14\":{\"isRunning\":false,\"isDone\":false,\"err\":null}}",
			}
		}(),
		func() testStruct {
			job0 := jobInternal{id: 11, original: &testJob{isDone: false}}
			job1 := jobInternal{id: 12, original: &testJob{isDone: true}}
			job2 := jobInternal{id: 13, original: &testJob{isDone: true}}
			job3 := jobInternal{id: 14, original: &testJob{isDone: false}}

			runningPool := make(map[int]jobInternal)
			runningPool[job0.id] = job0

			return testStruct{
				name: "example 2",
				fields: fields{
					queueLine:   []jobInternal{job2},
					runningPool: runningPool,
					doneJobs:    []jobInternal{job1, job3},
				},
				want: "{\"11\":{\"isRunning\":false,\"isDone\":false,\"err\":null},\"12\":{\"isRunning\":false,\"isDone\":true,\"err\":null},\"13\":{\"isRunning\":false,\"isDone\":true,\"err\":null},\"14\":{\"isRunning\":false,\"isDone\":false,\"err\":null}}",
			}
		}(),
		func() testStruct {
			job0 := jobInternal{id: 11, original: &testJob{isDone: false}}
			job1 := jobInternal{id: 12, original: &testJob{isDone: true}}
			job2 := jobInternal{id: 13, original: &testJob{isDone: true}}
			job3 := jobInternal{id: 14, original: &testJob{isDone: false}}

			runningPool := make(map[int]jobInternal)

			return testStruct{
				name: "example 3",
				fields: fields{
					queueLine:   []jobInternal{job0, job2},
					runningPool: runningPool,
					doneJobs:    []jobInternal{job1, job3},
				},
				want: "{\"11\":{\"isRunning\":false,\"isDone\":false,\"err\":null},\"12\":{\"isRunning\":false,\"isDone\":true,\"err\":null},\"13\":{\"isRunning\":false,\"isDone\":true,\"err\":null},\"14\":{\"isRunning\":false,\"isDone\":false,\"err\":null}}",
			}
		}(),
		func() testStruct {
			runningPool := make(map[int]jobInternal)

			return testStruct{
				name: "example 4",
				fields: fields{
					queueLine:   []jobInternal{},
					runningPool: runningPool,
					doneJobs:    []jobInternal{},
				},
				want: "{}",
			}
		}(),
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &JobManager{
				queueLine:   tt.fields.queueLine,
				runningPool: tt.fields.runningPool,
				doneJobs:    tt.fields.doneJobs,
			}

			// marshall the result because it is easier to compare
			gotRaw, err := json.Marshal(m.GetStatus())
			if err != nil {
				t.Error(err)
				return
			}
			got := string(gotRaw)

			if !reflect.DeepEqual(got, tt.want) {
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

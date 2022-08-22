package main

import (
	"errors"
	"reflect"
	"testing"
	"time"
)

type testJob struct {
	isRunning bool
	isDone    bool
	err       error
}

func (j *testJob) GetStatus() (bool, bool, error) {
	return j.isRunning, j.isDone, j.err
}
func (j *testJob) Start() {
	j.isRunning = true
	time.Sleep(time.Second)
}
func (j *testJob) Stop() {
	j.isRunning = false
}

func Test_jobInternal_Start(t *testing.T) {
	type fields struct {
		isRunning            bool
		original             Job
		runningBeforeChannel bool
		waitForChannel       bool
		done                 chan bool
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{
			"was not running",
			fields{false, &testJob{false, false, nil}, true, true, make(chan bool, 1)},
		},
		{
			"was running",
			fields{true, &testJob{true, false, nil}, true, true, make(chan bool, 1)},
		},
		{
			"was done",
			fields{false, &testJob{false, true, nil}, false, true, make(chan bool, 1)},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			j := &jobInternal{
				isRunning: tt.fields.isRunning,
				original:  tt.fields.original,
			}

			j.Start(tt.fields.done)

			// wait a bit for the go routine to actually run
			time.Sleep(time.Millisecond * 200)

			// before setting the signal ok, we want to make sure all is running
			if j.isRunning != tt.fields.runningBeforeChannel {
				t.Errorf("before channel = %v, want %v", j.isRunning, tt.fields.runningBeforeChannel)
			}

			if IsRunning, _, _ := j.original.GetStatus(); IsRunning != tt.fields.runningBeforeChannel {
				t.Errorf("before channel = %v, want %v", IsRunning, tt.fields.runningBeforeChannel)
			}

			// wait for done signal
			if tt.fields.waitForChannel {
				<-tt.fields.done
			}

			// after all done set it as not running
			if j.GetStatus().IsRunning != false {
				t.Errorf("after channel = %v, want %v", j.GetStatus().IsRunning, false)
			}

			if IsRunning, _, _ := j.original.GetStatus(); IsRunning != false {
				t.Errorf("after channel = %v, want %v", IsRunning, false)
			}
		})
	}
}

func Test_jobInternal_Stop(t *testing.T) {
	type fields struct {
		isRunning      bool
		original       Job
		waitForChannel bool
		done           chan bool
	}

	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			"was not running",
			fields{false, &testJob{false, false, nil}, false, make(chan bool, 1)},
			false,
		},
		{
			"was running",
			fields{true, &testJob{true, false, nil}, true, make(chan bool, 1)},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			j := &jobInternal{
				isRunning: tt.fields.isRunning,
				original:  tt.fields.original,
				done:      &tt.fields.done,
			}

			j.Stop()

			// wait for done signal
			if tt.fields.waitForChannel {
				<-tt.fields.done
			}

			if j.isRunning != tt.want {
				t.Errorf("jobInternal.Stop() = %v, want %v", j.isRunning, tt.want)
			}

			if IsRunning, _, _ := j.original.GetStatus(); IsRunning != tt.want {
				t.Errorf("jobInternal.original.Stop() = %v, want %v", IsRunning, tt.want)
			}
		})
	}
}

func Test_jobInternal_GetStatus(t *testing.T) {
	tests := []struct {
		name string
		job  Job
		want jobStatus
	}{
		{
			"running, not done and no error",
			&testJob{true, false, nil},
			jobStatus{true, false, nil},
		},
		{
			"not running, done and no error",
			&testJob{false, true, nil},
			jobStatus{false, true, nil},
		},
		{
			"not running, not done and error",
			&testJob{false, false, errors.New("foo")},
			jobStatus{false, false, errors.New("foo")},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			j := &jobInternal{original: tt.job}
			if got := j.GetStatus(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("jobInternal.GetStatus() = %v, want %v", got, tt.want)
			}
		})
	}
}

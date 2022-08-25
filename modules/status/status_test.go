package status

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestStatusJob_Start(t *testing.T) {
	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		code, _ := strconv.Atoi(r.URL.Query().Get("code"))
		w.WriteHeader(code)
		w.Write([]byte(""))
	}))
	defer svr.Close()

	type fields struct {
		CheckInterval time.Duration
		Url           string
		Updater       func(status int, err error)
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{
			name: "runs with a 404",
			fields: fields{
				CheckInterval: time.Second,
				Url:           fmt.Sprintf("%s?code=%d", svr.URL, 404),
			},
		},
		{
			name: "runs with a 200",
			fields: fields{
				CheckInterval: time.Second,
				Url:           fmt.Sprintf("%s?code=%d", svr.URL, 200),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lastStatus := ""

			j := &StatusJob{
				CheckInterval: tt.fields.CheckInterval,
				Url:           tt.fields.Url,
				Updater: func(status int, err error) {
					lastStatus = fmt.Sprintf("%d", status)
				},
			}

			go j.Start()
			time.Sleep(time.Millisecond * 100)
			j.isRunning = false

			// lets make sure that the data was updated
			rawCode := strings.ReplaceAll(tt.fields.Url, fmt.Sprintf("%s?code=", svr.URL), "")
			if rawCode != lastStatus {
				t.Errorf("code = %v, want %v", rawCode, lastStatus)
			}
		})
	}
}

func TestStatusJob_GetStatus(t *testing.T) {
	type fields struct {
		isRunning bool
		isDone    bool
		err       error
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{"runs with isRunning", fields{true, false, nil}},
		{"runs with isDone", fields{false, true, nil}},
		{"runs with err", fields{false, true, errors.New("example")}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			j := &StatusJob{
				isRunning: tt.fields.isRunning,
				isDone:    tt.fields.isDone,
				err:       tt.fields.err,
			}

			gotRunning, gotDone, err := j.GetStatus()

			if err != tt.fields.err {
				t.Errorf("StatusJob.GetStatus() error = %v, want %v", err, tt.fields.err)
			}
			if gotRunning != tt.fields.isRunning {
				t.Errorf("StatusJob.GetStatus() got = %v, want %v", gotRunning, tt.fields.isRunning)
			}
			if gotDone != tt.fields.isDone {
				t.Errorf("StatusJob.GetStatus() got1 = %v, want %v", gotDone, tt.fields.isDone)
			}
		})
	}
}

func TestStatusJob_loop(t *testing.T) {
	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		code, _ := strconv.Atoi(r.URL.Query().Get("code"))
		w.WriteHeader(code)
		w.Write([]byte(""))
	}))
	defer svr.Close()

	type fields struct {
		isRunning     bool
		isDone        bool
		err           error
		CheckInterval time.Duration
		Url           string
		Updater       func(status int, err error)
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{
			name: "runs and loops more than once",
			fields: fields{
				isRunning:     true,
				isDone:        false,
				err:           nil,
				CheckInterval: time.Millisecond,
				Url:           fmt.Sprintf("%s?code=%d", svr.URL, 200),
			},
		},
		{
			name: "doesnt run if done",
			fields: fields{
				isRunning:     true,
				isDone:        true,
				err:           nil,
				CheckInterval: time.Millisecond,
				Url:           fmt.Sprintf("%s?code=%d", svr.URL, 200),
			},
		},
		{
			name: "doesnt run if not running",
			fields: fields{
				isRunning:     true,
				isDone:        false,
				err:           nil,
				CheckInterval: time.Millisecond,
				Url:           fmt.Sprintf("%s?code=%d", svr.URL, 200),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loopCount := 0
			lastStatus := ""

			j := &StatusJob{
				isRunning:     tt.fields.isRunning,
				isDone:        tt.fields.isDone,
				err:           tt.fields.err,
				CheckInterval: tt.fields.CheckInterval,
				Url:           tt.fields.Url,
				Updater: func(status int, err error) {
					loopCount += 1
					lastStatus = fmt.Sprintf("%d", status)
				},
			}

			go j.loop()
			time.Sleep(time.Millisecond * 100)
			j.isRunning = false

			if tt.fields.isDone || !tt.fields.isRunning {
				if lastStatus != "" {
					t.Errorf("code = %v, want %v", "", lastStatus)
				}

				if loopCount != 0 {
					t.Errorf("loopCount = %v, want %v", loopCount, 0)
				}
			} else {
				rawCode := strings.ReplaceAll(tt.fields.Url, fmt.Sprintf("%s?code=", svr.URL), "")
				if rawCode != lastStatus {
					t.Errorf("code = %v, want %v", rawCode, lastStatus)
				}

				timeExpCount := int(tt.fields.CheckInterval)/int(time.Millisecond) - 1
				if loopCount < timeExpCount {
					t.Errorf("loopCount = %v, want %v", loopCount, timeExpCount)
				}
			}

		})
	}
}

func TestStatusJob_Stop(t *testing.T) {
	type fields struct {
		isRunning bool
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{name: "runs and stops from running", fields: fields{isRunning: true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			j := &StatusJob{
				isRunning: tt.fields.isRunning,
			}

			j.Stop()

			if j.isRunning != false {
				t.Errorf("isRunning = %v, want %v", j.isRunning, false)
			}
		})
	}
}

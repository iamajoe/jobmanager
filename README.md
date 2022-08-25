# Job manager

## Example: Manager

```go
m := &JobManager{}

// start with a maximum of workers / job threads and limit 
// the done so that you don't fill up memory without a need
go m.Start(4, 100)

// add jobs to the queue, they will be automatically set
// to start if manager is already running
id := m.Add(&job{})

// get the status of all the jobs in the manager
data := m.GetStatus()

// stop at any time without losing queues or data
m.Stop()
```

## Example: Job

A very simple example is the [status checker module](modules/status.go).

## Test

```
go fmt
go vet
go test
```

## TODO

- setup tests
- add modules: scraper
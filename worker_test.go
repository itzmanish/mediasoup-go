package mediasoup

import (
	"os"
	"time"
)

func init() {
	os.Setenv("DEBUG_COLORS", "false")
}

func CreateTestWorker(options ...Option) *Worker {
	options = append([]Option{WithLogLevel("debug"), WithLogTags([]WorkerLogTag{"info"})}, options...)

	worker, err := NewWorker(options...)
	if err != nil {
		panic(err)
	}
	return worker
}

func Wait(d time.Duration) {
	time.Sleep(d)
}

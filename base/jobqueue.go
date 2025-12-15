package base

import "time"

type JobQueue struct {
	jobs         chan func()
	limitsPerMin int
}

func NewJobQueue(limitsPerMin int) *JobQueue {
	return &JobQueue{
		jobs:         make(chan func(), 100),
		limitsPerMin: limitsPerMin,
	}
}

func (jq *JobQueue) Start() {
	go func() {
		ticker := time.NewTicker(time.Minute / time.Duration(jq.limitsPerMin))
		defer ticker.Stop()
		for job := range jq.jobs {
			<-ticker.C
			go job()
		}
	}()
}

func (jq *JobQueue) Enqueue(job func()) {
	jq.jobs <- job
}

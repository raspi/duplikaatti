package main

import (
	"fmt"
	"sync"
	"os"
	"crypto/sha256"
	"io"
	"time"
	"log"
)

type hasherWorkerResult struct {
	Hash string
	Info FileInfo
}

type ReadOperationType uint8

const (
	READ_FIRST ReadOperationType = iota
	READ_LAST                    = iota + 1
	READ_WHOLE                   = iota + 1
)

type hasherWorker struct {
	Errors         chan error
	Jobs           chan FileInfo
	Results        chan hasherWorkerResult
	Wg             *sync.WaitGroup
	readSize       int64
	readType       ReadOperationType
	ticker         *time.Ticker
	totalStartTime *time.Time
	startTime      *time.Time
}

func NewBytesWorker(t *time.Ticker, st *time.Time, workerCount int, readSize int64, rt ReadOperationType) hasherWorker {
	now := time.Now()

	w := hasherWorker{
		Jobs:           make(chan FileInfo, workerCount*2),
		Results:        make(chan hasherWorkerResult, 100),
		Errors:         make(chan error),
		Wg:             &sync.WaitGroup{},
		readSize:       readSize,
		readType:       rt,
		ticker:         t,
		totalStartTime: st,
		startTime:      &now,
	}

	for i := 0; i < workerCount; i++ {
		log.Printf(`Starting worker..`)
		go w.worker()
	}

	return w
}

func (w *hasherWorker) worker() {
	buf := make([]byte, w.readSize)

	for job := range w.Jobs {
		f, err := os.Open(job.Path)
		if err != nil {
			w.Errors <- err
			w.Wg.Done()
			continue
		}

		h := sha256.New()

		if w.readType == READ_LAST {
			f.Seek(-w.readSize, io.SeekEnd)
		}

		for {

			rb, err := f.Read(buf)
			if err != nil {
				if err == io.EOF {
					break
				}

				w.Errors <- err
				w.Wg.Done()
				continue
			}

			if rb < 0 {
				w.Errors <- fmt.Errorf(`rb was < 0`)
			}

			if int64(rb) > w.readSize {
				w.Errors <- fmt.Errorf(`rb was > read size`)
			}

			select {
			case <-w.ticker.C:

				readType := ``

				switch w.readType {
				case READ_FIRST:
					readType = `First bytes`
				case READ_LAST:
					readType = `Last bytes`
				case READ_WHOLE:
					readType = `Hashing`
				}

				log.Printf(`[%v] %v %v %v`, time.Since(*w.totalStartTime).Truncate(time.Second), time.Since(*w.startTime).Truncate(time.Second), readType, job.Path)
			default:

			}

			// Calculate checksum further
			wb, err := h.Write(buf[0:rb])
			if err != nil {
				w.Errors <- err
				w.Wg.Done()
				continue
			}

			if wb < 0 {
				w.Errors <- fmt.Errorf(`wb was < 0`)
			}

			if int64(wb) > w.readSize {
				w.Errors <- fmt.Errorf(`wb was > read size`)
			}

			if w.readType == READ_FIRST {
				break
			}

		}

		f.Close()

		w.Results <- hasherWorkerResult{
			Hash: fmt.Sprintf(`%x`, h.Sum(nil)),
			Info: job,
		}

		w.Wg.Done()
	}

	log.Printf(`Stopping worker..`)

}

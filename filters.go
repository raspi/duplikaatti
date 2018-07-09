package main

import (
	"time"
	"github.com/OneOfOne/xxhash"
	"fmt"
)

// Remove orphans
func filterByRemoveOrphans(m map[uint64][]string) {
	for fSize, files := range m {
		if len(files) < 2 {
			delete(m, fSize)
		}
	}
}

// Job for worker
type byteWorkerJob struct {
	Filename string
}

// Result of a worker
type byteWorkerResult struct {
	Filename string
	Checksum uint64
	ReadedBytes uint64
}

// Worker for reading N bytes from start/end
func byteReaderWorker(readSize uint64, fn func(string, uint64) ([]byte, error), jobs <-chan byteWorkerJob, results chan<- byteWorkerResult) {
	for j := range jobs {
		b, err := fn(j.Filename, readSize)
		if err != nil {
			panic(err)
		}

		h := xxhash.New64()
		n, err := h.Write(b)

		if err != nil {
			panic(err)
		}

		if n < 0 {
			panic(`wat??`)
		}

		results <- byteWorkerResult{
			Filename: j.Filename,
			Checksum: h.Sum64(),
			ReadedBytes: uint64(n),
		}

	}
}

// Read N bytes with given func
func filterByFuncBytes(startedTime time.Time, m map[uint64][]string, readSize uint64, refFunc func(string, uint64) ([]byte, error)) {
	fileCount, _ := getFileCount(m)
	ticker := time.NewTicker(time.Second * 1)
	defer ticker.Stop()
	startTime := time.Now()
	index := uint64(0)
	readedBytes := uint64(0)
	lastReadedBytes := uint64(0)

	fmt.Printf("\r  Initializing..")

	go func() {
		for {
			select {
			case <-ticker.C:
				diff := readedBytes - lastReadedBytes
				percent := (float64(index) * float64(100.0)) / float64(fileCount)
				totalDur := time.Since(startedTime).Truncate(time.Second)
				dur := time.Since(startTime).Truncate(time.Second)
				fmt.Printf("\r  %v %v/%v (%07.3f%%) %v %v %v/sec", totalDur, index, fileCount, percent, dur, bytesToHuman(readedBytes), bytesToHuman(diff))
				lastReadedBytes = readedBytes
			default:
				// do nothing
			}
		}
	}()


	for fSize, files := range m {
		hashMap := make(map[uint64][]string)

		jobs := make(chan byteWorkerJob, len(files))
		results := make(chan byteWorkerResult, cap(jobs))

		workerCount := 2

		if fSize <= MEBIBYTE {
			// Start more workers for smaller files
			workerCount = 10
		}

		// start N byte reader workers
		for i := 0; i < workerCount; i++ {
			go byteReaderWorker(readSize, refFunc, jobs, results)
		}

		for _, file := range files {
			jobs <- byteWorkerJob{
				Filename: file,
			}
		}
		close(jobs)

		// Collect results
		for i := 0; i < cap(results); i++ {
			res := <-results
			hashMap[res.Checksum] = append(hashMap[res.Checksum], res.Filename)
			index += 1
			readedBytes += res.ReadedBytes
		}
		close(results)

		var keepFiles []string

		for _, files := range hashMap {
			if len(files) > 1 {
				keepFiles = append(keepFiles, files...)
			}
		}

		m[fSize] = keepFiles
	}
}

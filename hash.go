package main

import (
	"os"
	"io"
	"github.com/pkg/errors"
	"fmt"
	"crypto/sha256"
	"time"
)

// For better readability
type checkSum string

type hashWorkerJob struct {
	Filename string
}

type hashWorkerResult struct {
	Filename    string
	Checksum    checkSum
	ReadedBytes uint64
}

type duplicateFile struct {
	Size     uint64
	CheckSum checkSum
	Keep     string   // File to keep
	Remove   []string // Files to remove
}

// Worker for hashing file
func hashWorker(hashingRefFn func(string) ([]byte, uint64, uint64, error), jobs <-chan hashWorkerJob, results chan<- hashWorkerResult) {
	for j := range jobs {
		// Hash
		b, totalRead, _, err := hashingRefFn(j.Filename)
		if err != nil {
			panic(err)
		}

		checkSum := checkSum(fmt.Sprintf(`%x`, b))

		results <- hashWorkerResult{
			Filename:    j.Filename,
			Checksum:    checkSum,
			ReadedBytes: totalRead,
		}
	}
}

// hash one file
func hashFile(file string) (b []byte, tr uint64, tw uint64, err error) {
	f, err := os.Open(file)
	if err != nil {
		return
	}
	defer f.Close()

	h := sha256.New()
	buf := make([]byte, MEBIBYTE)
	for {
		rn, err := f.Read(buf)
		if err != nil {
			if err == io.EOF {
				break
			}

			return nil, tr, tw, err
		}
		if rn < 0 {
			return nil, tr, tw, errors.New(fmt.Sprintf(`read bytes less than 0`))
		}

		tr += uint64(rn) // total read

		wn, err := h.Write(buf)
		if err != nil {
			return nil, tr, tw, err
		}
		if wn < 0 {
			return nil, tr, tw, errors.New(fmt.Sprintf(`write bytes less than 0`))
		}
		tw += uint64(wn) // total write
	}

	return h.Sum(nil), tr, tw, nil

}

// Calculate file hashes with N workers
func calculateHashes(startedTime time.Time, m map[uint64][]string) (dupeMap map[uint64]map[checkSum][]string) {
	fileCount, _ := getFileCount(m)
	dupeMap = make(map[uint64]map[checkSum][]string, 0)
	index := 0
	readedBytes := uint64(0)
	processingSize := uint64(0)

	ticker := time.NewTicker(time.Second * 1)
	defer ticker.Stop()
	startTime := time.Now()

	fmt.Printf("\r  Initializing..")

	go func() {
		for {
			select {
			case <-ticker.C:
				percent := (float64(index) * float64(100.0)) / float64(fileCount)
				totalDur := time.Since(startedTime).Truncate(time.Second)
				dur := time.Since(startTime).Truncate(time.Second)
				fmt.Printf("\r  %v %v/%v (%07.3f%%) %v %v Size: %v", totalDur, index, fileCount, percent, dur, bytesToHuman(readedBytes), bytesToHuman(processingSize))
			default:
				// do nothing
			}
		}
	}()

	for fSize, files := range m {
		processingSize = fSize
		hashMap := make(map[checkSum][]string)

		jobs := make(chan hashWorkerJob, len(files))
		results := make(chan hashWorkerResult, cap(jobs))

		workerCount := 4

		if fSize <= MEBIBYTE {
			// Start more workers for smaller files
			workerCount = 10
		}

		if fSize >= GIBIBYTE {
			workerCount = 2
		}

		// start N workers
		for i := 0; i < workerCount; i++ {
			go hashWorker(hashFile, jobs, results)
		}

		// Send file to worker
		for _, file := range files {
			jobs <- hashWorkerJob{
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

		for csum, files := range hashMap {
			if len(files) > 1 {
				if dupeMap[fSize] == nil {
					dupeMap[fSize] = make(map[checkSum][]string, 0)
				}

				dupeMap[fSize][csum] = files
			}
		}
	}

	return dupeMap
}

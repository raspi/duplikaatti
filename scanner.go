package main

import (
	"log"
	"time"
)

type DupeScanner struct {
	files       []fileInfo
	ticker      *time.Ticker
	workerCount int
	startTime   *time.Time
}

func New(ticker *time.Ticker, startTime *time.Time, workerCount int) DupeScanner {
	d := DupeScanner{
		files:       []fileInfo{},
		ticker:      ticker,
		workerCount: workerCount,
		startTime:   startTime,
	}

	return d
}

func (ds *DupeScanner) GetFileCount() (l int, ts uint64) {
	for _, f := range ds.files {
		ts += uint64(f.Size)
	}

	return len(ds.files), ts
}

// RemoveFileOrphans removes all files that do not share file sizes
func (ds *DupeScanner) RemoveFileOrphans() {
	fileSizeCounts := map[uint64]uint64{}

	for _, f := range ds.files {
		fileSizeCounts[f.Size]++
	}

	fileSizeHasDupes := map[uint64]bool{}

	for fileSize, fileCount := range fileSizeCounts {
		if fileCount > 1 {
			fileSizeHasDupes[fileSize] = true
		}
	}

	fileSizeCounts = nil

	var newFiles []fileInfo

	for _, f := range ds.files {
		_, ok := fileSizeHasDupes[f.Size]

		if !ok {
			continue
		}

		newFiles = append(newFiles, f)

	}

	fileSizeHasDupes = nil

	ds.files = newFiles
}

func (ds *DupeScanner) startWorker(readSize int64, rt ReadOperationType) hasherWorker {
	worker := NewBytesWorker(ds.ticker, ds.startTime, ds.workerCount, readSize, rt)

	worker.Wg.Add(1)

	fileCount, _ := ds.GetFileCount()

	go func(w *hasherWorker) {
		for _, file := range ds.files {
			fileCount--
			w.Wg.Add(1)
			w.Jobs <- file
		}

		w.Wg.Done()
	}(&worker) // /func

	go func(w *hasherWorker) {
		for e := range w.Errors {
			log.Printf(`error: %v`, e)
		}
	}(&worker)

	go func(w *hasherWorker) {
		// Wait jobs to finish
		w.Wg.Wait()

		for {
			if len(w.Results) == 0 && len(w.Jobs) == 0 {
				break
			}

			time.Sleep(time.Millisecond * 10)
		}

		close(w.Jobs)
		close(w.Results)
		close(w.Errors)
	}(&worker) // /func

	return worker
}

func (ds *DupeScanner) RemoveBasedOnBytes(readSize int64, rt ReadOperationType) {
	fbw := ds.startWorker(readSize, rt)

	hashMap := map[string][]uint64{}
	for res := range fbw.Results {
		hashMap[res.Hash] = append(hashMap[res.Hash], res.Info.INode)
	}

	keepInodes := make(map[uint64]bool)
	for _, inodes := range hashMap {
		if len(inodes) > 1 {
			for _, inode := range inodes {
				keepInodes[inode] = true
			}
		}
	}

	hashMap = nil

	var newFiles []fileInfo
	for _, file := range ds.files {
		_, ok := keepInodes[file.INode]
		if !ok {
			continue
		}

		newFiles = append(newFiles, file)
	}
	keepInodes = nil

	ds.files = newFiles

	ds.RemoveFileOrphans()

}

// Hash whole files
func (ds *DupeScanner) HashDuplicates(readSize int64) (m map[string]map[uint64][]fileInfo) {
	m = make(map[string]map[uint64][]fileInfo)

	fbw := ds.startWorker(readSize, READ_WHOLE)

	for res := range fbw.Results {
		if m[res.Hash] == nil {
			m[res.Hash] = make(map[uint64][]fileInfo)
		}
		m[res.Hash][res.Info.Size] = append(m[res.Hash][res.Info.Size], res.Info)
	}

	return m

}

func (ds *DupeScanner) Reset() {
	ds.files = nil
}

func (ds *DupeScanner) ReportStats() {
	fileCount, totalSize := ds.GetFileCount()
	log.Printf(`%v files %v`, fileCount, bytesToHuman(totalSize))
}

func (ds *DupeScanner) AddFile(info fileInfo) (err error) {
	ds.files = append(ds.files, info)
	return nil
}

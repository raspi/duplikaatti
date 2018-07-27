package main

import (
	"os"
	"log"
	"runtime"
	"github.com/raspi/dirscanner"
	"time"
	"math"
	"flag"
	"path/filepath"
	"fmt"
	"syscall"
)

const (
	VERSION  = `2.0.0`
	AUTHOR   = `Pekka Järvinen`
	YEAR     = 2018
	HOMEPAGE = `https://github.com/raspi/duplikaatti`
)

const (
	MEBIBYTE = 1048576
	GIBIBYTE = 1073741824
)

func getFilterFunc() dirscanner.FileValidatorFunction {
	return func(path string, info os.FileInfo, stat syscall.Stat_t) bool {
		return info.Mode().IsRegular() && info.Size() != 0
	}
}

type KeepFile struct {
	Priority uint8
	INode    uint64
}

func main() {
	readSize := int64(MEBIBYTE)

	actuallyRemove := false
	flag.BoolVar(&actuallyRemove, `remove`, false, `Actually remove files.`)

	flag.Usage = func() {
		f := filepath.Base(os.Args[0])

		fmt.Fprintf(flag.CommandLine.Output(), "Duplicate file remover (version %v)\n", VERSION)
		fmt.Fprintf(flag.CommandLine.Output(), "Removes duplicate files. Algorithm idea from rdfind.\n")
		fmt.Fprintf(flag.CommandLine.Output(), "\n")

		fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s [options] <directories>:\n", f)
		fmt.Fprintf(flag.CommandLine.Output(), "\nParameters:\n")
		flag.PrintDefaults()
		fmt.Fprintf(flag.CommandLine.Output(), "\n")

		fmt.Fprintf(flag.CommandLine.Output(), "Examples:\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  Test what would be removed:\n")
		fmt.Fprintf(flag.CommandLine.Output(), "    %v /home/raspi/storage /mnt/storage\n", f)
		fmt.Fprintf(flag.CommandLine.Output(), "  Remove files:\n")
		fmt.Fprintf(flag.CommandLine.Output(), "    %v -remove /home/raspi/storage /mnt/storage\n", f)
		fmt.Fprintf(flag.CommandLine.Output(), "\n")

		ai := 1
		fmt.Fprintf(flag.CommandLine.Output(), "Algorithm:\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  %v. Get file list from given directory list.\n", ai)
		ai++
		fmt.Fprintf(flag.CommandLine.Output(), "  %v. Remove all orphans (only one file with same size).\n", ai)
		ai++
		fmt.Fprintf(flag.CommandLine.Output(), "  %v. Read first %v bytes (%v) of files.\n", ai, readSize, bytesToHuman(uint64(readSize)))
		ai++
		fmt.Fprintf(flag.CommandLine.Output(), "  %v. Remove all orphans.\n", ai)
		ai++
		fmt.Fprintf(flag.CommandLine.Output(), "  %v. Read last %v bytes (%v) of files.\n", ai, readSize, bytesToHuman(uint64(readSize)))
		ai++
		fmt.Fprintf(flag.CommandLine.Output(), "  %v. Remove all orphans.\n", ai)
		ai++
		fmt.Fprintf(flag.CommandLine.Output(), "  %v. Hash whole files.\n", ai)
		ai++
		fmt.Fprintf(flag.CommandLine.Output(), "  %v. Select first file with checksum X not to be removed and add rest of the files to a duplicates list.\n", ai)
		ai++
		fmt.Fprintf(flag.CommandLine.Output(), "  %v. Remove duplicates.\n", ai)
		ai++
		fmt.Fprintf(flag.CommandLine.Output(), "\n")

		fmt.Fprintf(flag.CommandLine.Output(), "(c) %v %v- / %v\n", AUTHOR, YEAR, HOMEPAGE)
		fmt.Fprintf(flag.CommandLine.Output(), "\n")
	}

	flag.Parse()

	if flag.NArg() == 0 {
		flag.Usage()
		os.Exit(1)
	}

	if !isPowerOfTwo(uint64(readSize)) {
		fmt.Printf(`readSize (%v) is not power of two`, readSize)
		os.Exit(1)
	}

	dirs := flag.Args()

	// Check that all given arguments are directories
	for _, dir := range dirs {
		_, err := isDirectory(dir)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}

	if actuallyRemove {
		log.Printf("ACTUALLY DELETING FILES! PRESS CTRL+C TO ABORT!")
	} else {
		log.Printf("Note: Not actually deleting files (dry run)")
	}

	// Ticker for stats
	ticker := time.NewTicker(time.Second * 1)
	defer ticker.Stop()
	now := time.Now()
	workerCount := runtime.NumCPU()

	dupes := New(ticker, &now, workerCount)

	filterFunc := getFilterFunc()

	// look-up table for inodes
	seenInodes := map[uint64]bool{}

	log.Printf(`Generating file list..`)

	// First get a recursive file listing
	for _, dir := range dirs {
		scanner := dirscanner.New()

		err := scanner.Init(workerCount*2, filterFunc)
		if err != nil {
			panic(err)
		}

		err = scanner.ScanDirectory(dir)
		if err != nil {
			panic(err)
		}

		prio := uint8(math.MaxUint8)
		lastDir := ``
		lastFile := ``
		fileCount := 0

	scanloop:
		for {
			select {

			case <-scanner.Finished: // Finished getting file list
				log.Printf(`got all files`)
				break scanloop

			case e, ok := <-scanner.Errors: // Error happened, handle, discard or abort
				if ok {
					log.Printf(`got error: %v`, e)
					//s.Aborted <- true // Abort
				}


			case info, ok := <-scanner.Information: // Got information where worker is currently
				if ok {
					lastDir = info.Directory
				}


			case <-ticker.C: // Display some progress stats
				log.Printf(`%v Files scanned: %v Last file: %#v Dir: %#v`, time.Since(now).Truncate(time.Second), fileCount, lastFile, lastDir)

			case res, ok := <-scanner.Results:
				if ok {
					fileCount++
					lastFile = res.Path

					_, iok := seenInodes[res.Stat.Ino]

					if !iok {
						seenInodes[res.Stat.Ino] = true
						dupes.AddFile(newFileInfo(prio, res.Path, res.Stat.Ino, res.FileInfo))
					}
				}
			}
		}

		scanner.Close()

		prio--
	} // End of recursive scan

	// Now we have list of files sorted by key = file size

	log.Printf(`File list generated..`)

	dupes.ReportStats()
	log.Printf(`Removing orphans..`)
	dupes.RemoveFileOrphans()
	log.Printf(`Getting file information..`)
	dupes.ReportStats()
	log.Printf(`Reading first bytes..`)
	dupes.RemoveBasedOnBytes(readSize, READ_FIRST)
	dupes.ReportStats()
	log.Printf(`Reading last bytes..`)
	dupes.RemoveBasedOnBytes(readSize, READ_LAST)
	dupes.ReportStats()

	deletedCount := uint64(0)
	deletedSize := uint64(0)

	log.Printf(`Hashing files..`)

	hashed := dupes.HashDuplicates(readSize)
	dupes.Reset()

	for _, v := range GetDuplicateList(hashed) {
		for idx, f := range v {
			if idx == 0 {
				log.Printf(`Keeping %v`, f.Path)
				continue
			}

			deletedSize += uint64(f.Info.Size())
			deletedCount++

			log.Printf(`Deleting %v`, f.Path)

			if actuallyRemove {
				err := os.Remove(f.Path)

				if err != nil {
					log.Printf(`%v`, err)
				}
			}
		}
	}

	log.Printf(`Deleted %v files, %v`, deletedCount, bytesToHuman(deletedSize))
	log.Printf(`Took %v`, time.Since(now).Truncate(time.Second))
	log.Printf(`Done.`)

}

func GetDuplicateList(m map[string]map[int64][]FileInfo) (dupes [][]FileInfo) {
	for _, sizeKey := range m {
		for _, files := range sizeKey {

			var selected []FileInfo

			// Find best candidate
			bestCandidateFile := KeepFile{
				Priority: 0,
				INode:    math.MaxUint64,
			}

			for _, file := range files {
				if file.INode < bestCandidateFile.INode && file.Priority >= bestCandidateFile.Priority {
					bestCandidateFile.INode = file.INode
					bestCandidateFile.Priority = file.Priority
				}
			}

			// List what to keep and discard
			var keep FileInfo
			var discard []FileInfo

			for _, file := range files {
				if file.INode == bestCandidateFile.INode {
					keep = file
				} else {
					discard = append(discard, file)
				}
			}

			selected = append(selected, keep)
			selected = append(selected, discard...)

			dupes = append(dupes, selected)

		}
	}

	return dupes
}

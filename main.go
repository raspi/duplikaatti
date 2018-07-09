package main

import (
	"os"
	"fmt"
	"time"
	"flag"
	"path/filepath"
)

const (
	VERSION  = `1.0.0`
	AUTHOR   = `Pekka Järvinen`
	YEAR     = 2018
	HOMEPAGE = `https://github.com/raspi/duplikaatti`
)

func main() {
	readSize := uint64(MEBIBYTE)

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
		fmt.Fprintf(flag.CommandLine.Output(), "  %v. Remove all non-unique files (files in same path).\n", ai)
		ai++
		fmt.Fprintf(flag.CommandLine.Output(), "  %v. Remove all orphans (only one file with same size).\n", ai)
		ai++
		fmt.Fprintf(flag.CommandLine.Output(), "  %v. Read first %v bytes (%v) of files.\n", ai, readSize, bytesToHuman(readSize))
		ai++
		fmt.Fprintf(flag.CommandLine.Output(), "  %v. Remove all orphans.\n", ai)
		ai++
		fmt.Fprintf(flag.CommandLine.Output(), "  %v. Read last %v bytes (%v) of files.\n", ai, readSize, bytesToHuman(readSize))
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

	if !isPowerOfTwo(readSize) {
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

	fileList := make(map[uint64][]string, 0)
	//                   |      List of files
	//                    ----->File size

	if actuallyRemove {
		fmt.Printf("ACTUALLY DELETING FILES! PRESS CTRL+C TO ABORT!\n")
	} else {
		fmt.Printf("Note: Not actually deleting files (dry run)\n")
	}

	fmt.Printf("Generating file list (1/7)..\n")

	// Ticker for stats
	ticker := time.NewTicker(time.Second * 1)
	defer ticker.Stop()
	startTime := time.Now()

	// Generate file listing
	for _, dir := range dirs {
		err := listFiles(ticker, startTime, fileList, dir)
		if err != nil {
			panic(err)
		}
	}
	fmt.Printf("\n")

	fileCount, totalSize := getFileCount(fileList)
	fmt.Printf("File count after file list generation: %v (%v)\n", fileCount, bytesToHuman(totalSize))

	if fileCount < 2 {
		fmt.Printf("No possible duplicates.\n")
		os.Exit(1)
	}

	// If user gave colliding paths (ex. /storage & /storage/foo) by accident, remove all non-unique files
	fmt.Printf("Removing non-unique files (2/7)..\n")
	filterByUnique(fileList)

	fileCount, totalSize = getFileCount(fileList)
	fmt.Printf("File count after removing non-unique files: %v (%v)\n", fileCount, bytesToHuman(totalSize))

	if fileCount < 2 {
		fmt.Printf("No possible duplicates.\n")
		os.Exit(1)
	}

	fmt.Printf("Removing orphans..\n")
	filterByRemoveOrphans(fileList)

	fileCount, totalSize = getFileCount(fileList)
	fmt.Printf("File count after removing orphans: %v (%v)\n", fileCount, bytesToHuman(totalSize))

	if fileCount < 2 {
		fmt.Printf("No possible duplicates.\n")
		os.Exit(1)
	}

	// Read first N bytes of files
	fmt.Printf("Reading first %v bytes (%v) ~%v (3/7)..\n", readSize, bytesToHuman(readSize), bytesToHuman(readSize*fileCount))
	filterByFuncBytes(startTime, fileList, readSize, GetFirstBytes)

	fileCount, totalSize = getFileCount(fileList)
	fmt.Printf("\nFile count after reading first bytes of files: %v (%v)\n", fileCount, bytesToHuman(totalSize))

	if fileCount < 2 {
		fmt.Printf("No possible duplicates.\n")
		os.Exit(1)
	}

	fmt.Printf("Removing orphans..\n")
	filterByRemoveOrphans(fileList)
	fileCount, totalSize = getFileCount(fileList)
	fmt.Printf("File count after removing orphans: %v (%v)\n", fileCount, bytesToHuman(totalSize))

	if fileCount < 2 {
		fmt.Printf("No possible duplicates.\n")
		os.Exit(1)
	}

	// Read last N bytes of files
	fmt.Printf("Reading last %v bytes (%v) ~%v (4/7)..\n", readSize, bytesToHuman(readSize), bytesToHuman(readSize*fileCount))
	filterByFuncBytes(startTime, fileList, readSize, GetLastBytes)

	fileCount, totalSize = getFileCount(fileList)
	fmt.Printf("\nFile count after reading last bytes of files: %v (%v)\n", fileCount, bytesToHuman(totalSize))

	if fileCount < 2 {
		fmt.Printf("No possible duplicates.\n")
		os.Exit(1)
	}

	fmt.Printf("Removing orphans..\n")
	filterByRemoveOrphans(fileList)

	fileCount, totalSize = getFileCount(fileList)
	fmt.Printf("File count after removing orphans: %v (%v)\n", fileCount, bytesToHuman(totalSize))

	if fileCount < 2 {
		fmt.Printf("No possible duplicates.\n")
		os.Exit(1)
	}

	// Calculate hashes of files that are left
	fmt.Printf("Hashing (5/7)..\n")
	dupes := calculateHashes(startTime, fileList)

	fmt.Printf("\nCollecting duplicates (6/7..)\n")

	var duplicateFileList []duplicateFile

	for fSize, same := range dupes {
		for csum, files := range same {
			var df duplicateFile

			df.CheckSum = csum
			df.Size = fSize

			for idx, file := range files {
				if idx == 0 {
					// Keep first file
					df.Keep = file
					continue
				}

				df.Remove = append(df.Remove, file)
			}

			duplicateFileList = append(duplicateFileList, df)
		}
	}

	fmt.Printf("Removing duplicates (7/7)..\n")
	removeFileCount := uint64(0) // How many files are deleted
	removeFileSize := uint64(0) // How much space is freed

	// Remove files
	for _, d := range duplicateFileList {
		for _, f := range d.Remove {
			s, err := os.Stat(f)
			if err != nil {
				panic(err)
			}

			if uint64(s.Size()) != d.Size {
				fmt.Printf(`File size mismatch: %v`, f)
				fmt.Printf(`Aborting.`)
				os.Exit(1)
			}
		}

		fmt.Printf("  Checksum: %v Size: %v bytes (%v)\n", d.CheckSum, d.Size, bytesToHuman(d.Size))
		fmt.Printf("  Keep: %v\n", d.Keep)

		// Remove files
		for _, f := range d.Remove {
			removeFileCount++
			removeFileSize += d.Size
			fmt.Printf("  Remove: %v\n", f)
			if actuallyRemove {
				// Remove file
				os.Remove(f)
			}
		}

		fmt.Printf("\n")
	}

	fmt.Printf("Took %v\n", time.Since(startTime).Truncate(time.Second))

	fmt.Printf("Removed file count: %v Size: %v\n", removeFileCount, bytesToHuman(removeFileSize))
	fmt.Printf("Done.\n")

}

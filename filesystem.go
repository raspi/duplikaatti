package main

import (
	"os"
	"fmt"
	"path/filepath"
	"github.com/pkg/errors"
	"io"
	"time"
)

// Get list of files from given directory
func listFiles(ticker *time.Ticker, startTime time.Time, fileMap map[uint64][]string, root string) (err error) {
	f, err := os.Open(root)

	if err != nil {
		if os.IsPermission(err) {
			// Skip if there's no permission
			return nil
		}
		return err
	}

	fInfo, err := f.Readdir(-1)
	f.Close()
	if err != nil {
		if os.IsPermission(err) {
			// Skip if there's no permission
			return nil
		}
		return err
	}

	for _, file := range fInfo {
		fpath := filepath.Join(f.Name(), file.Name())

		if file.Mode().IsRegular() {
			// is file
			fs := uint64(file.Size())
			if fs == 0 {
				continue
			}

			if fileMap[fs] == nil {
				fileMap[fs] = []string{}
			}
			fileMap[fs] = append(fileMap[fs], fpath)
		} else if file.IsDir() {
			// is directory
			listFiles(ticker, startTime, fileMap, fpath)
		}

		// Report stats
		select {
		case <-ticker.C:
			dur := time.Since(startTime).Truncate(time.Second)
			fmt.Printf("\r  %v Dir: %v", dur, root)
		default:
			// do nothing
		}

	}

	return nil
}

// is directory and exists
func isDirectory(dir string) (b bool, err error) {
	fi, err := os.Stat(dir)

	if err != nil {
		if os.IsNotExist(err) {
			return false, err
		}
	}

	if !fi.IsDir() {
		return false, errors.New(fmt.Sprintf(`Not a directory: %v`, dir))
	}

	return true, nil
}

// Read first N bytes of file
func GetFirstBytes(fileName string, size uint64) (b []byte, err error) {
	f, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	buf := make([]byte, size)
	n, err := f.Read(buf)
	if err != nil {
		return nil, err
	}

	if n < 0 {
		panic(`wat??`)
	}

	if uint64(n) > size {
		panic(`wat??`)
	}

	return buf, nil
}

// Read last N bytes of file
func GetLastBytes(fileName string, size uint64) (b []byte, err error) {
	f, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// Go to end of file
	f.Seek(-int64(size), io.SeekEnd)

	buf := make([]byte, size)
	n, err := f.Read(buf)
	if err != nil {
		return nil, err
	}

	if n < 0 {
		panic(`wat??`)
	}

	if uint64(n) > size {
		panic(`wat??`)
	}

	return buf, nil
}

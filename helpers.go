package main

import (
	"fmt"
	"math"
)

const (
	MEBIBYTE = 1048576
	GIBIBYTE = 1073741824
)

func isPowerOfTwo(n uint64) bool {
	return (n != 0) && (n != 1) && ((n & (n - 1)) == 0)
}

// Convert 1024 to '1 KiB' etc
func bytesToHuman(src uint64) string {
	if src < 10 {
		return fmt.Sprintf("%d B", src)
	}

	s := float64(src)
	base := float64(1024)
	sizes := []string{"B", "KiB", "MiB", "GiB", "TiB", "PiB", "EiB"}

	e := math.Floor(math.Log(s) / math.Log(base))
	suffix := sizes[int(e)]
	val := math.Floor(s / math.Pow(base, e) * 10 + 0.5) / 10
	f := "%.0f %s"
	if val < 10 {
		f = "%.1f %s"
	}

	return fmt.Sprintf(f, val, suffix)
}

func filterByUnique(m map[uint64][]string) {
	for fSize, files := range m {

		var keepFiles []string

		for _, file := range files {

			exists := false
			for _, f := range keepFiles {
				if file == f {
					exists = true
					break
				}
			}

			if !exists {
				keepFiles = append(keepFiles, file)
			}
		}

		m[fSize] = keepFiles
	}

}

// Get file and total size count
func getFileCount(m map[uint64][]string) (fc uint64, tfs uint64) {
	for fileSize, files := range m {
		fCount := uint64(len(files))
		fc += fCount
		tfs += uint64(fileSize) * fCount
	}

	return fc, tfs
}

package main

import (
	"fmt"
	"math"
	"os"
)

func isPowerOfTwo(n uint64) bool {
	if n == 0 || n == 1 {
		return false
	}
	return (n & (n - 1)) == 0
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
	val := math.Floor(s/math.Pow(base, e)*10+0.5) / 10
	f := "%.0f %s"
	if val < 10 {
		f = "%.1f %s"
	}

	return fmt.Sprintf(f, val, suffix)
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
		return false, fmt.Errorf(`not a directory: %v`, dir)
	}

	return true, nil
}

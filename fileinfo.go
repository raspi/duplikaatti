package main

import (
	"github.com/raspi/dirscanner"
)

type fileInfo struct {
	Priority uint8  // Priority
	Path     string // Path to file
	INode    uint64 // INode
	Size     uint64 // File size
}

func newFileInfo(priority uint8, info dirscanner.FileInformation) fileInfo {
	return fileInfo{
		Priority: priority,
		Path:     info.Path,
		INode:    info.Identifier,
		Size:     info.Size,
	}
}

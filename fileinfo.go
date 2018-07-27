package main

import "os"

type FileInfo struct {
	Priority uint8  // Priority
	Path     string // Path to file
	INode    uint64
	Info     os.FileInfo
}

func newFileInfo(priority uint8, path string, inode uint64, info os.FileInfo) FileInfo {
	return FileInfo{
		Priority: priority,
		Path:     path,
		INode:    inode,
		Info:     info,
	}
}

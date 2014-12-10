package fsdiffer

import (
	"os"
	"path/filepath"
	"time"

	"github.com/coreos/fleet/log"
)

type ChangeType uint8

const (
	Added ChangeType = iota
	Modified
	Deleted
)

type FSChange struct {
	Path string
	ChangeType
}

type FileInfo struct {
	Path string
	os.FileInfo
}

func fsWalker(fileInfos map[string]FileInfo) filepath.WalkFunc {

	return func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		fileInfos[path] = FileInfo{Path: path, FileInfo: info}
		return nil
	}
}

func FSDiff(A string, B string) ([]FSChange, error) {
	changes := []FSChange{}
	fileInfosA := make(map[string]FileInfo)
	fileInfosB := make(map[string]FileInfo)
	err := filepath.Walk(A, fsWalker(fileInfosA))
	if err != nil {
		return nil, err
	}
	err = filepath.Walk(B, fsWalker(fileInfosB))
	if err != nil {
		return nil, err
	}

	//fmt.Printf("fileInfosA: %v\n", fileInfosA)
	//fmt.Printf("fileInfosB: %v\n", fileInfosB)

	for _, infoB := range fileInfosB {
		relpath, _ := filepath.Rel(B, infoB.Path)
		infoA, ok := fileInfosA[filepath.Join(A, relpath)]
		if !ok {
			changes = append(changes, FSChange{Path: relpath, ChangeType: Added})
		} else {
			// tar time is with the second precision. TODO now add 1 second to the modtime.
			if infoA.Size() != infoB.Size() || infoA.ModTime().Add(time.Second).Before(infoB.ModTime()) {
				log.V(1).Infof("relpath: %s, infoA.Size(): %d, infoB.Size(): %d, infoA.ModTime(): %s, infoB.ModTime():%s\n", relpath, infoA.Size(), infoB.Size(), infoA.ModTime(), infoB.ModTime())
				changes = append(changes, FSChange{Path: relpath, ChangeType: Modified})
			}
		}
	}
	for _, infoA := range fileInfosA {
		relpath, _ := filepath.Rel(A, infoA.Path)
		_, ok := fileInfosB[filepath.Join(B, relpath)]
		if !ok {
			changes = append(changes, FSChange{Path: relpath, ChangeType: Deleted})
		}
	}
	return changes, nil
}

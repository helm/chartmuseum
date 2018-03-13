package storage

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

type (
	// Object is a generic representation of a storage object
	Object struct {
		Path         string
		Content      []byte
		LastModified time.Time
	}

	// ObjectSliceDiff provides information on what has changed since last calling ListObjects
	ObjectSliceDiff struct {
		Change  bool
		Removed []Object
		Added   []Object
		Updated []Object
	}

	// Backend is a generic interface for storage backends
	Backend interface {
		ListObjects(prefix string) ([]Object, error)
		GetObject(path string) (Object, error)
		PutObject(path string, content []byte) error
		DeleteObject(path string) error
	}
)

// HasExtension determines whether or not an object contains a file extension
func (object Object) HasExtension(extension string) bool {
	return filepath.Ext(object.Path) == fmt.Sprintf(".%s", extension)
}

// GetObjectSliceDiff takes two objects slices and returns an ObjectSliceDiff
func GetObjectSliceDiff(os1 []Object, os2 []Object) ObjectSliceDiff {
	var diff ObjectSliceDiff
	for _, o1 := range os1 {
		found := false
		for _, o2 := range os2 {
			if o1.Path == o2.Path {
				found = true
				if !o1.LastModified.Equal(o2.LastModified) {
					diff.Updated = append(diff.Updated, o2)
				}
				break
			}
		}
		if !found {
			diff.Removed = append(diff.Removed, o1)
		}
	}
	for _, o2 := range os2 {
		found := false
		for _, o1 := range os1 {
			if o2.Path == o1.Path {
				found = true
				break
			}
		}
		if !found {
			diff.Added = append(diff.Added, o2)
		}
	}
	diff.Change = len(diff.Removed)+len(diff.Added)+len(diff.Updated) > 0
	return diff
}

func cleanPrefix(prefix string) string {
	return strings.Trim(prefix, "/")
}

func removePrefixFromObjectPath(prefix string, path string) string {
	if prefix == "" {
		return path
	}
	path = strings.Replace(path, fmt.Sprintf("%s/", prefix), "", 1)
	return path
}

func objectPathIsInvalid(path string) bool {
	return strings.Contains(path, "/") || path == ""
}

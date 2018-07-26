/*
Copyright The Helm Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package storage

import (
	"fmt"
	"path/filepath"
	"sort"
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

	ObjectSorter struct {
		objects []Object
		by      func(o1, o2 *Object) bool
	}

	// By is the type of a "less" function that defines the ordering of its Planet arguments.
	By func(o1, o2 *Object) bool

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

// Len is part of sort.Interface.
func (o *ObjectSorter) Len() int {
	return len(o.objects)
}

// Swap is part of sort.Interface.
func (o *ObjectSorter) Swap(i, j int) {
	o.objects[i], o.objects[j] = o.objects[j], o.objects[i]
}

// Less is part of sort.Interface. It is implemented by calling the "by" closure in the sorter.
func (o *ObjectSorter) Less(i, j int) bool {
	return o.by(&o.objects[i], &o.objects[j])
}

// Sort is a method on the function type, By, that sorts the argument slice according to the function.
func (by By) Sort(planets []Object) {
	ps := &ObjectSorter{
		objects: planets,
		by:      by, // The Sort method's receiver is the function (closure) that defines the sort order.
	}
	sort.Sort(ps)
}

// GetObjectSliceDiff takes two objects slices and returns an ObjectSliceDiff
func GetObjectSliceDiff(os1 []Object, os2 []Object) ObjectSliceDiff {
	var diff ObjectSliceDiff
	for _, o1 := range os1 {
		found := false
		for _, o2 := range os2 {
			if o1.Path == o2.Path {
				found = true
				// ignore milliseconds due to helm
				if o1.LastModified.Sub(o2.LastModified) > time.Duration(time.Second*1) {
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

package repo

import (
	"encoding/json"
	"errors"
	"fmt"
)

var (
	// MetaFileExtension is the file extension used for meta files
	MetaFileExtension = "tgz.meta"

	// MetaFileContentType is the http content-type header for meta files
	MetaFileContentType = "application/json"

	// ErrorInvalidMetaFile is raised when a meta file is invalid
	ErrorInvalidMetaFile = errors.New("invalid meta file")
)

// MetaFilenameFromNameVersion returns a meta filename from a name and version
func MetaFilenameFromNameVersion(name string, version string) string {
	filename := fmt.Sprintf("%s-%s.%s", name, version, MetaFileExtension)
	return filename
}

// MetaFilenameFromContent returns a meta filename from binary content
func MetaFilenameFromContent(content []byte) (string, error) {
	var contentJSON map[string]interface{}
	json.Unmarshal(content[:], &contentJSON)

	if contentJSON["version"] == nil || contentJSON["name"] == nil {
		return "", ErrorInvalidMetaFile
	}
	filename := MetaFilenameFromNameVersion(fmt.Sprintf("%v", contentJSON["name"]), fmt.Sprintf("%v", contentJSON["version"]))
	return filename, nil
}

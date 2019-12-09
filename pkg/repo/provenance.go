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

package repo

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

	"helm.sh/helm/v3/pkg/provenance"
	"regexp"
)

var (
	// ProvenanceFileExtension is the file extension used for provenance files
	ProvenanceFileExtension = "tgz.prov"

	// ProvenanceFileContentType is the http content-type header for provenance files
	ProvenanceFileContentType = "application/pgp-signature"

	// ErrorInvalidProvenanceFile is raised when a provenance file is invalid
	ErrorInvalidProvenanceFile = errors.New("invalid provenance file")
)

// ProvenanceFilenameFromNameVersion returns a provenance filename from a name and version
func ProvenanceFilenameFromNameVersion(name string, version string) string {
	filename := fmt.Sprintf("%s-%s.%s", name, version, ProvenanceFileExtension)
	return filename
}

// ProvenanceFilenameFromContent returns a provenance filename from binary content
func ProvenanceFilenameFromContent(content []byte) (string, error) {
	contentStr := string(content[:])

	hasPGPBegin := strings.HasPrefix(contentStr, "-----BEGIN PGP SIGNED MESSAGE-----")
	nameMatch := regexp.MustCompile("\nname:[ *](.+)").FindStringSubmatch(contentStr)
	versionMatch := regexp.MustCompile("\nversion:[ *](.+)").FindStringSubmatch(contentStr)

	if !hasPGPBegin || len(nameMatch) != 2 || len(versionMatch) != 2 {
		return "", ErrorInvalidProvenanceFile
	}

	filename := ProvenanceFilenameFromNameVersion(nameMatch[1], versionMatch[1])
	return filename, nil
}

func provenanceDigestFromContent(content []byte) (string, error) {
	digest, err := provenance.Digest(bytes.NewBuffer(content))
	return digest, err
}

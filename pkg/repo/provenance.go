package repo

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

	"k8s.io/helm/pkg/provenance"
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

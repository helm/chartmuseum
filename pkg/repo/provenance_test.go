package repo

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type ProvenanceTestSuite struct {
	suite.Suite
}

func (suite *ProvenanceTestSuite) TestProvenanceFileFilenameFromContent() {
	goodContent := []byte(`-----BEGIN PGP SIGNED MESSAGE-----
Hash: SHA512

name: mychart
version: 0.1.0

...
files:
  mychart-0.1.0.tgz: sha256:5c824605d676f5244aaf70d889f4e58f953308c426f2fa8f970e8fd580eaf363
-----BEGIN PGP SIGNATURE-----

wsBcBAEBCgAQBQJZuxVACRCEO7+YH8GHYgAAtVMIAEIKSyWH9hb3y/ck6Dwg2Y6v
6i0kP3L9iCyyTp64XJYiuipdhUO/XK0CxRcLqLa0I5qu658XeU/Qxwb1GTgPoP52
BCyiJVOY5aXl0SJa+jXHliDak7fgZjUHCtp1HBEKX2uRrx57tTkIjZr7pitt/OwI
bRz9OXHQe9+fhtAZo5DPtMd53UQ2uRc7xft9HxnwlDEWrBfH6CUNlhbdtKRR5n0s
FUyR0Eszw/x3No0DdPuH3fo0ShamW9eOFnXIgWqvaeSJthTC5WO5mlSGNEunJKft
HjQLzdEWppyu55ZS6/oIJdVC2GjUa/PZmKkhYwsMvaWYv+jZWFfhZn8fPYEF0qI=
=/cXn
-----END PGP SIGNATURE-----`)
	 goodContentWithMatainerName := []byte(`-----BEGIN PGP SIGNED MESSAGE-----
Hash: SHA512

description: Buildpack application builder for Hephy Workflow.
home: https://github.com/teamhephy/slugbuilder
maintainers:
- - email: team@teamhephy.com
  name: Team Hephy
name: mychart
version: 0.1.0

...
files:
  mychart-0.1.0.tgz: sha256:5c824605d676f5244aaf70d889f4e58f953308c426f2fa8f970e8fd580eaf363
-----BEGIN PGP SIGNATURE-----

wsBcBAEBCgAQBQJZuxVACRCEO7+YH8GHYgAAtVMIAEIKSyWH9hb3y/ck6Dwg2Y6v
6i0kP3L9iCyyTp64XJYiuipdhUO/XK0CxRcLqLa0I5qu658XeU/Qxwb1GTgPoP52
BCyiJVOY5aXl0SJa+jXHliDak7fgZjUHCtp1HBEKX2uRrx57tTkIjZr7pitt/OwI
bRz9OXHQe9+fhtAZo5DPtMd53UQ2uRc7xft9HxnwlDEWrBfH6CUNlhbdtKRR5n0s
FUyR0Eszw/x3No0DdPuH3fo0ShamW9eOFnXIgWqvaeSJthTC5WO5mlSGNEunJKft
HjQLzdEWppyu55ZS6/oIJdVC2GjUa/PZmKkhYwsMvaWYv+jZWFfhZn8fPYEF0qI=
=/cXn
-----END PGP SIGNATURE-----`)
	badContentNoBeginPGP := []byte("badbadverybad")
	badContentNoChartName := []byte(`-----BEGIN PGP SIGNED MESSAGE-----
version: 0.1.0`)
	badContentNoChartVersion := []byte(`-----BEGIN PGP SIGNED MESSAGE-----
name: mychart`)

	filename, err := ProvenanceFilenameFromContent(goodContent)
	suite.Nil(err, "no error getting filename from good content")
	suite.Equal("mychart-0.1.0.tgz.prov", filename, "filename generated from good content")

	filename, err = ProvenanceFilenameFromContent(goodContentWithMatainerName)
	suite.Nil(err, "no error getting filename from good content")
	suite.Equal("mychart-0.1.0.tgz.prov", filename, "filename generated from good content with maintainer name field")

	_, err = ProvenanceFilenameFromContent(badContentNoBeginPGP)
	suite.Equal(ErrorInvalidProvenanceFile, err, "ErrorInvalidProvenanceFile from bad content, no begin pgp")

	_, err = ProvenanceFilenameFromContent(badContentNoChartName)
	suite.Equal(ErrorInvalidProvenanceFile, err, "ErrorInvalidProvenanceFile from bad content, no name")

	_, err = ProvenanceFilenameFromContent(badContentNoChartVersion)
	suite.Equal(ErrorInvalidProvenanceFile, err, "ErrorInvalidProvenanceFile from bad content, no version")
}

func TestProvenanceTestSuite(t *testing.T) {
	suite.Run(t, new(ProvenanceTestSuite))
}

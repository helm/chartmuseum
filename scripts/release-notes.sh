#!/usr/bin/env bash

# Copyright The Helm Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

RELEASE=${RELEASE:-$2}
PREVIOUS_RELEASE=${PREVIOUS_RELEASE:-$1}

## Ensure Correct Usage
if [[ -z "${PREVIOUS_RELEASE}" || -z "${RELEASE}" ]]; then
  echo Usage:
  echo ./scripts/release-notes.sh v3.0.0 v3.1.0
  echo or
  echo PREVIOUS_RELEASE=v3.0.0
  echo RELEASE=v3.1.0
  echo ./scripts/release-notes.sh
  exit 1
fi

## validate git tags
for tag in $RELEASE $PREVIOUS_RELEASE; do
  OK=$(git tag -l ${tag} | wc -l)
  if [[ "$OK" == "0" ]]; then
    echo ${tag} is not a valid release version
    exit 1
  fi
done

## Check for hints that checksum files were downloaded
## from `make fetch-dist`
if [[ ! -e "./_dist/chartmuseum-${RELEASE}-darwin-amd64.tar.gz.sha256sum" ]]; then
  echo "checksum file ./_dist/chartmuseum-${RELEASE}-darwin-amd64.tar.gz.sha256sum not found in ./_dist/"
  echo "Did you forget to run \`make fetch-dist\` first ?"
  exit 1
fi

## Generate CHANGELOG from git log
CHANGELOG=$(git log --no-merges --pretty=format:'- %s %H (%aN)' ${PREVIOUS_RELEASE}..${RELEASE})
if [[ ! $? -eq 0 ]]; then
  echo "Error creating changelog"
  echo "try running \`git log --no-merges --pretty=format:'- %s %H (%aN)' ${PREVIOUS_RELEASE}..${RELEASE}\`"
  exit 1
fi

## guess at MAJOR / MINOR / PATCH versions
MAJOR=$(echo ${RELEASE} | sed 's/^v//' | cut -f1 -d.)
MINOR=$(echo ${RELEASE} | sed 's/^v//' | cut -f2 -d.)
PATCH=$(echo ${RELEASE} | sed 's/^v//' | cut -f3 -d.)

## Print release notes to stdout
cat <<EOF
## ${RELEASE}

ChartMuseum ${RELEASE} is a feature release. This release, we focused on <insert focal point>. Users are encouraged to upgrade for the best experience.

The community keeps growing, and we'd love to see you there!

- Join the discussion in [Kubernetes Slack](https://kubernetes.slack.com):
  - `#chartmuseum` for discussing PRs, code, bugs, or just to hang out
- Hang out at the Helm Public Developer Call: Thursday, 9:30 Pacific via [Zoom](https://zoom.us/j/696660622)

## Notable Changes

- Add list of
- notable changes here

## Installation and Upgrading

Download ChartMuseum ${RELEASE}. The common platform binaries are here:

- [MacOS amd64](https://get.helm.sh/chartmuseum-${RELEASE}-darwin-amd64.tar.gz) ([archive sig](https://get.helm.sh/chartmuseum-${RELEASE}-darwin-amd64.tar.gz.sig) / [checksum](https://get.helm.sh/chartmuseum-${RELEASE}-darwin-amd64.tar.gz.sha256sum) / [checksum sig](https://get.helm.sh/chartmuseum-${RELEASE}-darwin-amd64.tar.gz.sha256sum.sig) / $(cat _dist/chartmuseum-${RELEASE}-darwin-amd64.tar.gz.sha256sum | awk '{print $1}'))
- [Linux amd64](https://get.helm.sh/chartmuseum-${RELEASE}-linux-amd64.tar.gz) ([archive sig](https://get.helm.sh/chartmuseum-${RELEASE}-linux-amd64.tar.gz.sig) / [checksum](https://get.helm.sh/chartmuseum-${RELEASE}-linux-amd64.tar.gz.sha256sum) / [checksum sig](https://get.helm.sh/chartmuseum-${RELEASE}-linux-amd64.tar.gz.sha256sum.sig) /  $(cat _dist/chartmuseum-${RELEASE}-linux-amd64.tar.gz.sha256sum | awk '{print $1}'))
- [Linux arm](https://get.helm.sh/chartmuseum-${RELEASE}-linux-arm.tar.gz) ([archive sig](https://get.helm.sh/chartmuseum-${RELEASE}-linux-arm.tar.gz.sig) / [checksum](https://get.helm.sh/chartmuseum-${RELEASE}-linux-arm.tar.gz.sha256sum) / [checksum sig](https://get.helm.sh/chartmuseum-${RELEASE}-linux-arm.tar.gz.sha256sum.sig) / $(cat _dist/chartmuseum-${RELEASE}-linux-arm.tar.gz.sha256sum | awk '{print $1}'))
- [Linux arm64](https://get.helm.sh/chartmuseum-${RELEASE}-linux-arm64.tar.gz) ([archive sig](https://get.helm.sh/chartmuseum-${RELEASE}-linux-arm64.tar.gz.sig) / [checksum](https://get.helm.sh/chartmuseum-${RELEASE}-linux-arm64.tar.gz.sha256sum) / [checksum sig](https://get.helm.sh/chartmuseum-${RELEASE}-linux-arm64.tar.gz.sha256sum.sig) / $(cat _dist/chartmuseum-${RELEASE}-linux-arm64.tar.gz.sha256sum | awk '{print $1}'))
- [Linux i386](https://get.helm.sh/chartmuseum-${RELEASE}-linux-386.tar.gz) ([archive sig](https://get.helm.sh/chartmuseum-${RELEASE}-linux-386.tar.gz.sig) / [checksum](https://get.helm.sh/chartmuseum-${RELEASE}-linux-386.tar.gz.sha256sum) / [checksum sig](https://get.helm.sh/chartmuseum-${RELEASE}-linux-386.tar.gz.sha256sum.sig) /  $(cat _dist/chartmuseum-${RELEASE}-linux-386.tar.gz.sha256sum | awk '{print $1}'))
- [Linux mips64le](https://get.helm.sh/chartmuseum-${RELEASE}-linux-mips64le.tar.gz) ([archive sig](https://get.helm.sh/chartmuseum-${RELEASE}-linux-mips64le.tar.gz.sig) / [checksum](https://get.helm.sh/chartmuseum-${RELEASE}-linux-mips64le.tar.gz.sha256sum) / [checksum sig](https://get.helm.sh/chartmuseum-${RELEASE}-linux-mips64le.tar.gz.sha256sum.sig) / $(cat _dist/chartmuseum-${RELEASE}-linux-mips64le.tar.gz.sha256sum | awk '{print $1}'))
- [Linux ppc64le](https://get.helm.sh/chartmuseum-${RELEASE}-linux-ppc64le.tar.gz) ([archive sig](https://get.helm.sh/chartmuseum-${RELEASE}-linux-ppc64le.tar.gz.sig) / [checksum](https://get.helm.sh/chartmuseum-${RELEASE}-linux-ppc64le.tar.gz.sha256sum) / [checksum sig](https://get.helm.sh/chartmuseum-${RELEASE}-linux-ppc64le.tar.gz.sha256sum.sig) / $(cat _dist/chartmuseum-${RELEASE}-linux-ppc64le.tar.gz.sha256sum | awk '{print $1}'))
- [Linux s390x](https://get.helm.sh/chartmuseum-${RELEASE}-linux-s390x.tar.gz) ([archive sig](https://get.helm.sh/chartmuseum-${RELEASE}-linux-s390x.tar.gz.sig) / [checksum](https://get.helm.sh/chartmuseum-${RELEASE}-linux-s390x.tar.gz.sha256sum) / [checksum sig](https://get.helm.sh/chartmuseum-${RELEASE}-linux-s390x.tar.gz.sha256sum.sig) / $(cat _dist/chartmuseum-${RELEASE}-linux-s390x.tar.gz.sha256sum | awk '{print $1}'))
- [Windows amd64](https://get.helm.sh/chartmuseum-${RELEASE}-windows-amd64.zip) ([archive sig](https://get.helm.sh/chartmuseum-${RELEASE}-windows-amd64.zip.sig) / [checksum](https://get.helm.sh/chartmuseum-${RELEASE}-windows-amd64.zip.sha256sum) / [checksum sig](https://get.helm.sh/chartmuseum-${RELEASE}-windows-amd64.zip.sha256sum.sig) / $(cat _dist/chartmuseum-${RELEASE}-windows-amd64.zip.sha256sum | awk '{print $1}'))

You can download the SBOM for this release in SPDX format [here](https://get.helm.sh/chartmuseum-${RELEASE}.spdx).

You can use a [script to install](https://raw.githubusercontent.com/helm/chartmuseum/main/scripts/get-chartmuseum) on any system with \`bash\`.

## What's Next

- ${MAJOR}.${MINOR}.$(expr ${PATCH} + 1) will contain only bug fixes.
- ${MAJOR}.$(expr ${MINOR} + 1).${PATCH} is the next feature release.

## Changelog

${CHANGELOG}
EOF

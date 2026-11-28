#!/usr/bin/env bash

set -euo pipefail
: ${VERSION:?"VERSION environment variable is not set"}
IMAGE_REF=${IMAGE_REF:-ghcr.io/helm/chartmuseum:${VERSION}}

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd $DIR/../
mkdir -p ./_dist/
pushd ./_dist/

# Initialize the configuration file
cat << EOF > .sbom.yaml
---
namespace: https://get.helm.sh/chartmuseum-${VERSION}.spdx
license: Apache-2.0
name: ChartMuseum
artifacts:
  - type: directory
    source: ..
EOF

for file in $(ls *.{gz,zip}); 
    do echo "Adding ${file} to SBOM"
    echo "  - type: file"  >> .sbom.yaml
    echo "    source: ${file}" >> .sbom.yaml
done

printf 'Adding image %s\n' "$IMAGE_REF"
printf '  - type: image\n    source: %s\n' "$IMAGE_REF" >> .sbom.yaml

echo "Wrote configuration file:"
cat .sbom.yaml

bom generate -c .sbom.yaml -o chartmuseum-${VERSION}.spdx

rm .sbom.yaml
popd
echo "SBOM written to _dist/chartmuseum-${VERSION}.spdx"

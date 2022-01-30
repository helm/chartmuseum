#!/usr/bin/env bash

RELEASE=${RELEASE:-$2}

cd _dist

# Initialize the configuration file
cat << EOF > .sbom.yaml
---
namespace: https://get.helm.sh/chartmuseum-${RELEASE}.spdx
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

echo "Adding image ghcr.io/helm/chartmuseum:${RELEASE}"
echo "  - type: image" >> .sbom.yaml
echo "    source: ghcr.io/helm/chartmuseum:${RELEASE}" >> .sbom.yaml

echo "Wrote configuration file:"
cat .sbom.yaml

bom generate -c .sbom.yaml -o chartmuseum-${RELEASE}.spdx

echo "SBOM written to _dist/chartmuseum-${RELEASE}.spdx"
rm .sbom.yaml


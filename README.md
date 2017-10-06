# ChartMuseum
<img align="right" src="https://github.com/chartmuseum/chartmuseum/raw/master/logo.png">

[![CircleCI](https://circleci.com/gh/chartmuseum/chartmuseum.svg?style=svg)](https://circleci.com/gh/chartmuseum/chartmuseum)
[![Go Report Card](https://goreportcard.com/badge/github.com/chartmuseum/chartmuseum)](https://goreportcard.com/report/github.com/chartmuseum/chartmuseum)
[![GoDoc](https://godoc.org/github.com/chartmuseum/chartmuseum?status.svg)](https://godoc.org/github.com/chartmuseum/chartmuseum)
<sub>**_"Preserve your precious artifacts... in the cloud!"_**<sub>

*ChartMuseum* is an open-source **[Helm Chart Repository](https://github.com/kubernetes/helm/blob/master/docs/chart_repository.md)** written in Go (Golang), with support for cloud storage backends, including [Google Cloud Storage](https://cloud.google.com/storage/) and [Amazon S3](https://aws.amazon.com/s3/).

Works as a valid Helm Chart Repository, and also provides an API for uploading new chart packages to storage etc.

<img width="60" align="right" src="https://github.com/golang-samples/gopher-vector/raw/master/gopher-side_color.png">
<img width="20" align="right" src="https://github.com/golang-samples/gopher-vector/raw/master/gopher-side_color.png">

Powered by some great Go technology:
- [Kubernetes Helm](https://github.com/kubernetes/helm) - for working with charts, generating repository index
- [Gin Web Framework](https://github.com/gin-gonic/gin) - for HTTP routing
- [cli](https://github.com/urfave/cli) - for command line option parsing
- [zap](https://github.com/uber-go/zap) - for logging

## Things that have been said in Helm land
>"Finally!!" 

>"ChartMuseum is awesome"

>"This is awesome!"

>"Oh yes!!!! I’ve been waiting for this for so long. Makes life much easier, especially for the index.yaml creation!"

>"I was thinking about writing one of these up myself. This is perfect! thanks!"

>"I am jumping for joy over ChartMuseum, a full-fledged Helm repository server with upload!"

>"This is really cool ... We currently have a process that generates the index file and then uploads, so this is nice"

>"Really a good idea ... really really great, thanks again. I can use nginx to hold the repos and the museum to add/delete the chart. That's a whole life cycle management of chart with the current helm"

>"thanks for building the museum!"

## API
### Helm Chart Repository
- `GET /index.yaml` - retrieved when you run `helm repo add chartmuseum http://localhost:8080/`
- `GET /charts/mychart-0.1.0.tgz` - retrieved when you run `helm install chartmuseum/mychart`
- `GET /charts/mychart-0.1.0.tgz.prov` - retrieved when you run `helm install` with the `--verify` flag

### Chart Manipulation
- `POST /api/charts` - upload a new chart version
- `POST /api/prov` - upload a new provenance file
- `DELETE /api/charts/<name>/<version>` - delete a chart version (and corresponding provenance file)
- `GET /api/charts` - list all charts
- `GET /api/charts/<name>` - list all versions of a chart
- `GET /api/charts/<name>/<version>` - describe a chart version

## Uploading a Chart Package
<sub>*Follow **"How to Run"** section below to get ChartMuseum up and running at ht<span>tp:/</span>/localhost:8080*<sub>

First create `mychart-0.1.0.tgz` using the [Helm CLI](https://docs.helm.sh/using_helm/#installing-helm):
```
cd mychart/
helm package .
```

Upload `mychart-0.1.0.tgz`:
```bash
curl --data-binary "@mychart-0.1.0.tgz" http://localhost:8080/api/charts
```

If you've signed your package and generated a [provenance file](https://github.com/kubernetes/helm/blob/master/docs/provenance.md), upload it with:
```bash
curl --data-binary "@mychart-0.1.0.tgz.prov" http://localhost:8080/api/prov
```

## Installing Charts into Kubernetes
Add the URL to your *ChartMuseum* installation to the local repository list:
```bash
helm repo add chartmuseum http://localhost:8080
```

Search for charts:
```bash
helm search chartmuseum/
```

Install chart:
```bash
helm install chartmuseum/mychart
```

## How to Run
### CLI
#### Installation
Install the binary:
```bash
# on Linux
curl -LO https://s3.amazonaws.com/chartmuseum/release/latest/bin/linux/amd64/chartmuseum

# on macOS
curl -LO https://s3.amazonaws.com/chartmuseum/release/latest/bin/darwin/amd64/chartmuseum

chmod +x ./chartmuseum
mv ./chartmuseum /usr/local/bin
```
Using `latest` in URLs above will get the latest binary (built from master branch).

Replace `latest` with `$(curl -s https://s3.amazonaws.com/chartmuseum/release/stable.txt)` to automatically determine the latest stable release (e.g. `v0.1.0`).

Show all CLI options with `chartmuseum --help` and determine version with `chartmuseum --version`

#### Using with Amazon S3
Make sure your environment is properly setup to access `my-s3-bucket`
```bash
chartmuseum --debug --port=8080 \
  --storage="amazon" \
  --storage-amazon-bucket="my-s3-bucket" \
  --storage-amazon-prefix="" \
  --storage-amazon-region="us-east-1"
```

#### Using with Google Cloud Storage
Make sure your environment is properly setup to access `my-gcs-bucket`
```bash
chartmuseum --debug --port=8080 \
  --storage="google" \
  --storage-google-bucket="my-gcs-bucket" \
  --storage-google-prefix=""
```

#### Using with local filesystem storage
Make sure you have read-write access to `./chartstorage` (will create if doesn't exist)
```bash
chartmuseum --debug --port=8080 \
  --storage="local" \
  --storage-local-rootdir="./chartstorage"
```

### Docker Image
Available via [Docker Hub](https://hub.docker.com/r/chartmuseum/chartmuseum/).

Example usage (S3):
```bash
docker run --rm -it \
  -p 8080:8080 \
  -v ~/.aws:/root/.aws:ro \
  chartmuseum/chartmuseum:latest \
  --debug --port=8080 \
  --storage="amazon" \
  --storage-amazon-bucket="my-s3-bucket" \
  --storage-amazon-prefix="" \
  --storage-amazon-region="us-east-1"
```

If you're having access issues, you can validate your AWS credentials with something like
```bash
docker run --rm \
  -v ~/.aws:/root/.aws:ro \
  --entrypoint sh \
  chartmuseum/chartmuseum:latest \
  -c 'apk add --update py-pip &&
      pip install awscli &&
      aws s3 ls my-s3-bucket --region=us-east-1'
```

### Helm Chart
There is a [Helm chart for *ChartMuseum*](https://github.com/kubernetes/charts/tree/master/incubator/chartmuseum) itself which can be found in the official Kubernetes Charts repository.

You can also view it on [KubeApps](https://kubeapps.com/charts/incubator/chartmuseum).

Please note that for now, this **should only be used for testing purposes**. An [emptyDir volume](https://kubernetes.io/docs/concepts/storage/volumes/#emptydir) is currently being used for storage, which means your .tgzs will disappear when the pod is removed. If you can help get this to work with persistent storage or any of the cloud storage options, please submit a PR to kubernetes/charts. Thanks!

## Notes on index.yaml
The repository index (index.yaml) is dynamically generated based on packages found in storage. If you store your own version of index.yaml, it will be completely ignored.

`GET /index.yaml` occurs when you run `helm repo add chartmuseum http://localhost:8080/` or `helm repo update`.

If you manually add/remove a .tgz package from storage, it will be immediately reflected in `GET /index.yaml`.

You are no longer required to maintain your own version of index.yaml using `helm repo index --merge`.

## Mirroring the official Kubernetes repositories
Please see `scripts/mirror_k8s_repos.sh` for an example of how to download all .tgz packages from the official Kubernetes repositories (both stable and incubator).

You can then use *ChartMuseum* to serve up an internal mirror:
```
scripts/mirror_k8s_repos.sh
chartmuseum --debug --port=8080 --storage="local" --storage-local-rootdir="./mirror"
 ```

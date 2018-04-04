# ChartMuseum
<img align="right" src="https://github.com/kubernetes-helm/chartmuseum/raw/master/logo.png">

[![CircleCI](https://circleci.com/gh/kubernetes-helm/chartmuseum.svg?style=svg)](https://circleci.com/gh/kubernetes-helm/chartmuseum)
[![Go Report Card](https://goreportcard.com/badge/github.com/kubernetes-helm/chartmuseum)](https://goreportcard.com/report/github.com/kubernetes-helm/chartmuseum)
[![GoDoc](https://godoc.org/github.com/kubernetes-helm/chartmuseum?status.svg)](https://godoc.org/github.com/kubernetes-helm/chartmuseum)
<sub>**_"Preserve your precious artifacts... in the cloud!"_**<sub>

*ChartMuseum* is an open-source **[Helm Chart Repository](https://github.com/kubernetes/helm/blob/master/docs/chart_repository.md)** written in Go (Golang), with support for cloud storage backends, including [Google Cloud Storage](https://cloud.google.com/storage/), [Amazon S3](https://aws.amazon.com/s3/), [Microsoft Azure Blob Storage](https://azure.microsoft.com/en-us/services/storage/blobs/), and [Alibaba Cloud OSS Storage](https://www.alibabacloud.com/product/oss).

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

### Server Info
- `GET /` - HTML welcome page
- `GET /health` - returns 200 OK

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

Both files can also be uploaded at once (or one at a time) on the `/api/charts` route using the `multipart/form-data` format:

```bash
curl -F "chart=@mychart-0.1.0.tgz" -F "prov=@mychart-0.1.0.tgz.prov" http://localhost:8080/api/charts
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

# on Windows
curl -LO https://s3.amazonaws.com/chartmuseum/release/latest/bin/windows/amd64/chartmuseum

chmod +x ./chartmuseum
mv ./chartmuseum /usr/local/bin
```
Using `latest` in URLs above will get the latest binary (built from master branch).

Replace `latest` with `$(curl -s https://s3.amazonaws.com/chartmuseum/release/stable.txt)` to automatically determine the latest stable release (e.g. `v0.5.1`).

Determine your version with `chartmuseum --version`.

#### Configuration
Show all CLI options with `chartmuseum --help`. Common configurations can be seen below.

All command-line options can be specified as environment variables, which are defined by the command-line option, capitalized, with all `-`'s replaced with `_`'s.

For example, the env var `STORAGE_AMAZON_BUCKET` can be used in place of `--storage-amazon-bucket`.

#### Using with Amazon S3
Make sure your environment is properly setup to access `my-s3-bucket`
```bash
chartmuseum --debug --port=8080 \
  --storage="amazon" \
  --storage-amazon-bucket="my-s3-bucket" \
  --storage-amazon-prefix="" \
  --storage-amazon-region="us-east-1"
```
You need at least the following permissions inside your IAM Policy
```yaml
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "AllowListObjects",
      "Effect": "Allow",
      "Action": [
        "s3:ListBucket"
      ],
      "Resource": "arn:aws:s3:::my-s3-bucket"
    },
    {
      "Sid": "AllowObjectsCRUD",
      "Effect": "Allow",
      "Action": [
        "s3:DeleteObject",
        "s3:GetObject",
        "s3:PutObject"
      ],
      "Resource": "arn:aws:s3:::my-s3-bucket/*"
    }
  ]
}
```

#### Using with Google Cloud Storage
Make sure your environment is properly setup to access `my-gcs-bucket`
```bash
chartmuseum --debug --port=8080 \
  --storage="google" \
  --storage-google-bucket="my-gcs-bucket" \
  --storage-google-prefix=""
```

#### Using with Microsoft Azure Blob Storage

Make sure your environment is properly setup to access `mycontainer`.

To do so, you must set the following env vars:
- `AZURE_STORAGE_ACCOUNT`
- `AZURE_STORAGE_ACCESS_KEY`

```bash
chartmuseum --debug --port=8080 \
  --storage="microsoft" \
  --storage-microsoft-container="mycontainer" \
  --storage-microsoft-prefix=""
```

#### Using with Alibaba Cloud OSS Storage

Make sure your environment is properly setup to access `my-oss-bucket`.

To do so, you must set the following env vars:
- `ALIBABA_CLOUD_ACCESS_KEY_ID`
- `ALIBABA_CLOUD_ACCESS_KEY_SECRET`

```bash
chartmuseum --debug --port=8080 \
  --storage="alibaba" \
  --storage-alibaba-bucket="my-oss-bucket" \
  --storage-alibaba-prefix="" \
  --storage-alibaba-endpoint="oss-cn-beijing.aliyuncs.com"
```

#### Using with local filesystem storage
Make sure you have read-write access to `./chartstorage` (will create if doesn't exist)
```bash
chartmuseum --debug --port=8080 \
  --storage="local" \
  --storage-local-rootdir="./chartstorage"
```

#### Basic Auth
If both of the following options are provided, basic http authentication will protect all routes:
- `--basic-auth-user=<user>` - username for basic http authentication
- `--basic-auth-pass=<pass>` - password for basic http authentication

You may want basic auth to only be applied to operations that can change Charts, i.e. PUT, POST and DELETE.  So to avoid basic auth on GET operations use

- `--auth-anonymous-get` - allow anonymous GET operations

#### HTTPS
If both of the following options are provided, the server will listen and serve HTTPS:
- `--tls-cert=<crt>` - path to tls certificate chain file
- `--tls-key=<key>` - path to tls key file

#### Just generating index.yaml
You can specify the `--gen-index` option if you only wish to use _ChartMuseum_ to generate your index.yaml file. Note that this will only work with `--depth=0`.

The contents of index.yaml will be printed to stdout and the program will exit. This is useful if you are satisfied with your current Helm CI/CD process and/or don't want to monitor another webservice.

#### Other CLI options
- `--log-json` - output structured logs as json
- `--disable-api` - disable all routes prefixed with /api
- `--allow-overwrite` - allow chart versions to be re-uploaded
- `--chart-url=<url>` - absolute url for .tgzs in index.yaml
- `--storage-amazon-endpoint=<endpoint>` - alternative s3 endpoint
- `--storage-amazon-sse=<algorithm>` - s3 server side encryption algorithm
- `--chart-post-form-field-name=<field>` - form field which will be queried for the chart file content
- `--prov-post-form-field-name=<field>` - form field which will be queried for the provenance file content
- `--index-limit=<number>` - limit the number of parallel indexers
- `--context-path=<path>` - base context path (new root for application routes)
- `--depth=<number>` - levels of nested repos for multitenancy

Available via [Docker Hub](https://hub.docker.com/r/chartmuseum/chartmuseum/).

Example usage (S3):
```bash
docker run --rm -it \
  -p 8080:8080 \
  -e PORT=8080 \
  -e DEBUG=1 \
  -e STORAGE="amazon" \
  -e STORAGE_AMAZON_BUCKET="my-s3-bucket" \
  -e STORAGE_AMAZON_PREFIX="" \
  -e STORAGE_AMAZON_REGION="us-east-1" \
  -v ~/.aws:/root/.aws:ro \
  chartmuseum/chartmuseum:latest
```

### Helm Chart
There is a [Helm chart for *ChartMuseum*](https://github.com/kubernetes/charts/tree/master/incubator/chartmuseum) itself which can be found in the official Kubernetes Charts repository.

You can also view it on [Kubeapps Hub](https://hub.kubeapps.com/charts/incubator/chartmuseum).

To install:
```bash
helm repo add incubator https://kubernetes-charts-incubator.storage.googleapis.com
helm install incubator/chartmuseum
```

If interested in making changes, please submit a PR to kubernetes/charts. Before doing any work, please check for any [currently open pull requests](https://github.com/kubernetes/charts/pulls?q=is%3Apr+is%3Aopen+chartmuseum). Thanks!

## Multitenancy
Multitenancy is supported with the `--depth` flag.

To begin, start with a directory structure such as
```
charts
├── org1
│   ├── repoa
│   │   └── nginx-ingress-0.9.3.tgz
├── org2
│   ├── repob
│   │   └── chartmuseum-0.4.0.tgz
```

This represents a storage layout appropriate for `--depth=2`. The organization level can be eliminated by using `--depth=1`. The default depth is 0 (singletenant server).

Start the server with `--depth=2`, pointing to the `charts/` directory:
```
chartmuseum --debug --depth=2 --storage="local" --storage-local-rootdir=./charts
```

This example will provide two separate Helm Chart Repositories at the following locations:
- `http://localhost:8080/org1/repoa`
- `http://localhost:8080/org2/repob`

This should work with all supported storage backends.

To use the chart manipulation routes, simply place the name of the repo directly after "/api" in the route:

```bash
curl -F "chart=@mychart-0.1.0.tgz" http://localhost:8080/api/org1/repoa/charts
```


## Notes on index.yaml
The repository index (index.yaml) is dynamically generated based on packages found in storage. If you store your own version of index.yaml, it will be completely ignored.

`GET /index.yaml` occurs when you run `helm repo add chartmuseum http://localhost:8080` or `helm repo update`.

If you manually add/remove a .tgz package from storage, it will be immediately reflected in `GET /index.yaml`.

You are no longer required to maintain your own version of index.yaml using `helm repo index --merge`.

The `--gen-index` CLI option (described above) can be used to generate and print index.yaml to stdout.

## Mirroring the official Kubernetes repositories
Please see `scripts/mirror_k8s_repos.sh` for an example of how to download all .tgz packages from the official Kubernetes repositories (both stable and incubator).

You can then use *ChartMuseum* to serve up an internal mirror:
```
scripts/mirror_k8s_repos.sh
chartmuseum --debug --port=8080 --storage="local" --storage-local-rootdir="./mirror"
 ```

## Community
You can reach the *ChartMuseum* community and developers in the [Kubernetes Slack](https://slack.k8s.io) **#chartmuseum** channel.


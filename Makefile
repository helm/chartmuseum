# Change this and commit to create new release
VERSION=0.12.0
REVISION := $(shell git rev-parse --short HEAD;)

MOD_PROXY_URL ?= https://gocenter.io

CM_LOADTESTING_HOST ?= http://localhost:8080

.PHONY: bootstrap
bootstrap: export GO111MODULE=on
bootstrap: export GOPROXY=$(MOD_PROXY_URL)
bootstrap:
	@go mod download && go mod vendor

.PHONY: build
build: build-linux build-mac build-windows

build-windows: export GOOS=windows
build-windows: export GOARCH=amd64
build-windows: export GO111MODULE=on
build-windows: export GOPROXY=$(MOD_PROXY_URL)
build-windows:
	go build -v --ldflags="-w -X main.Version=$(VERSION) -X main.Revision=$(REVISION)" \
		-o bin/windows/amd64/chartmuseum cmd/chartmuseum/main.go  # windows
	sha256sum bin/windows/amd64/chartmuseum || shasum -a 256 bin/windows/amd64/chartmuseum

build-linux: export GOOS=linux
build-linux: export GOARCH=amd64
build-linux: export CGO_ENABLED=0
build-linux: export GO111MODULE=on
build-linux: export GOPROXY=$(MOD_PROXY_URL)
build-linux:
	go build -v --ldflags="-w -X main.Version=$(VERSION) -X main.Revision=$(REVISION)" \
		-o bin/linux/amd64/chartmuseum cmd/chartmuseum/main.go  # linux
	sha256sum bin/linux/amd64/chartmuseum || shasum -a 256 bin/linux/amd64/chartmuseum

build-armv7: export GOOS=linux
build-armv7: export GOARCH=arm
build-armv7: export GOARM=7
build-armv7: export CGO_ENABLED=0
build-armv7: export GO111MODULE=on
build-armv7: export GOPROXY=$(MOD_PROXY_URL)
build-armv7:
	go build -v --ldflags="-w -X main.Version=$(VERSION) -X main.Revision=$(REVISION)" \
		-o bin/linux/armv7/chartmuseum cmd/chartmuseum/main.go  # linux

build-mac: export GOOS=darwin
build-mac: export GOARCH=amd64
build-mac: export CGO_ENABLED=0
build-mac: export GO111MODULE=on
build-mac: export GOPROXY=$(MOD_PROXY_URL)
build-mac:
	go build -v --ldflags="-w -X main.Version=$(VERSION) -X main.Revision=$(REVISION)" \
		-o bin/darwin/amd64/chartmuseum cmd/chartmuseum/main.go # mac osx
	sha256sum bin/darwin/amd64/chartmuseum || shasum -a 256 bin/darwin/amd64/chartmuseum

.PHONY: clean
clean:
	@git status --ignored --short | grep '^!! ' | sed 's/!! //' | xargs rm -rf

.PHONY: setup-test-environment
setup-test-environment:
	@./scripts/setup_test_environment.sh

.PHONY: test
test: export CGO_ENABLED=0
test: export GO111MODULE=on
test: export GOPROXY=$(MOD_PROXY_URL)
test: setup-test-environment
	@./scripts/test.sh

.PHONY: startloadtest
startloadtest:
	@cd loadtesting && pipenv install
	@cd loadtesting && pipenv run locust --host $(CM_LOADTESTING_HOST)

.PHONY: covhtml
covhtml:
	@go tool cover -html=.cover/cover.out

.PHONY: acceptance
acceptance: setup-test-environment
	@./scripts/acceptance.sh

.PHONY: run
run:
	@rm -rf .chartstorage/
	@bin/darwin/amd64/chartmuseum --debug --port=8080 --storage="local" \
		--storage-local-rootdir=".chartstorage/"

.PHONY: tree
tree:
	@tree -I vendor

# https://github.com/hirokidaichi/goviz/pull/8
.PHONY: goviz
goviz:
	#@go get -u github.com/RobotsAndPencils/goviz
	@goviz -i helm.sh/chartmuseum/cmd/chartmuseum -l | dot -Tpng -o goviz.png

.PHONY: release-latest
release-latest:
	@scripts/release.sh latest

.PHONY: release-stable
release-stable:
	@scripts/release.sh $(VERSION)

.PHONY: version-released
version-released:
	@scripts/version_released.sh $(VERSION)

.PHONY: get-version
get-version:
	@echo $(VERSION)

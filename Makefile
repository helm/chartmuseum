# Change this and commit to create new release
VERSION=0.8.1
REVISION := $(shell git rev-parse --short HEAD;)

CM_LOADTESTING_HOST ?= http://localhost:8080

.PHONY: bootstrap
bootstrap:
	@go mod download && go mod vendor

.PHONY: build
build: build-linux build-mac build-windows

build-windows: export GOARCH=amd64
build-windows:
	@GOOS=windows go build -mod=vendor -v --ldflags="-w -X main.Version=$(VERSION) -X main.Revision=$(REVISION)" \
		-o bin/windows/amd64/chartmuseum cmd/chartmuseum/main.go  # windows

build-linux: export GOARCH=amd64
build-linux: export CGO_ENABLED=0
build-linux:
	@GOOS=linux go build -mod=vendor -v --ldflags="-w -X main.Version=$(VERSION) -X main.Revision=$(REVISION)" \
		-o bin/linux/amd64/chartmuseum cmd/chartmuseum/main.go  # linux

build-mac: export GOARCH=amd64
build-mac: export CGO_ENABLED=0
build-mac:
	@GOOS=darwin go build -mod=vendor -v --ldflags="-w -X main.Version=$(VERSION) -X main.Revision=$(REVISION)" \
		-o bin/darwin/amd64/chartmuseum cmd/chartmuseum/main.go # mac osx

.PHONY: clean
clean:
	@git status --ignored --short | grep '^!! ' | sed 's/!! //' | xargs rm -rf

.PHONY: setup-test-environment
setup-test-environment:
	@./scripts/setup_test_environment.sh

.PHONY: test
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
	@goviz -i github.com/helm/chartmuseum/cmd/chartmuseum -l | dot -Tpng -o goviz.png

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

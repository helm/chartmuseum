# Change this and commit to create new release
VERSION=0.5.1
REVISION := $(shell git rev-parse --short HEAD;)

HAS_DEP := $(shell command -v dep;)
HAS_PIP := $(shell command -v pip;)
HAS_PIPENV := $(shell command -v pipenv;)
HAS_VENV := $(shell command -v virtualenv;)
HAS_GOVIZ := $(shell command -v goviz;)
HAS_DOT := $(shell command -v dot;)
HAS_AWS := $(shell command -v aws;)

CM_LOADTESTING_HOST ?= http://localhost:8080

.PHONY: bootstrap
bootstrap:
ifndef HAS_DEP
	@go get -u github.com/golang/dep/cmd/dep
endif
	@dep ensure -v -vendor-only

.PHONY: build
build: build_linux build_mac build_windows

build_windows: export GOARCH=amd64
build_windows:
	@GOOS=windows go build -v --ldflags="-w -X main.Version=$(VERSION) -X main.Revision=$(REVISION)" \
		-o bin/windows/amd64/chartmuseum cmd/chartmuseum/main.go  # windows

build_linux: export GOARCH=amd64
build_linux: export CGO_ENABLED=0
build_linux:
	@GOOS=linux go build -v --ldflags="-w -X main.Version=$(VERSION) -X main.Revision=$(REVISION)" \
		-o bin/linux/amd64/chartmuseum cmd/chartmuseum/main.go  # linux

build_mac: export GOARCH=amd64
build_mac: export CGO_ENABLED=0
build_mac:
	@GOOS=darwin go build -v --ldflags="-w -X main.Version=$(VERSION) -X main.Revision=$(REVISION)" \
		-o bin/darwin/amd64/chartmuseum cmd/chartmuseum/main.go # mac osx

.PHONY: clean
clean:
	@git status --ignored --short | grep '^!! ' | sed 's/!! //' | xargs rm -rf

.PHONY: setup-test-environment
setup-test-environment:
ifndef HAS_PIP
	@sudo apt-get update && sudo apt-get install -y python-pip
endif
ifndef HAS_VENV
	@sudo pip install virtualenv
endif
	@./scripts/setup_test_environment.sh

.PHONY: test
test: setup-test-environment
	@./scripts/test.sh

.PHONY: testcloud
testcloud: export TEST_CLOUD_STORAGE=1
testcloud: test

.PHONY: startloadtest
startloadtest:
ifndef HAS_PIPENV
	@sudo pip install pipenv
endif
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
ifndef HAS_GOVIZ
	@go get -u github.com/RobotsAndPencils/goviz
endif
ifndef HAS_DOT
	@sudo apt-get update && sudo apt-get install -y graphviz
endif
	@goviz -i github.com/kubernetes-helm/chartmuseum/cmd/chartmuseum -l | dot -Tpng -o goviz.png

.PHONY: release
release:
ifndef HAS_AWS
	@sudo pip install awscli
endif
	@scripts/release.sh $(VERSION)

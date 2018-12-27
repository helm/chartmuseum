.PHONY: clean
clean:
	git status --ignored --short | grep '^!! ' | sed 's/!! //' | xargs rm -rf

.PHONY: covhtml
covhtml:
	open coverage.html

.PHONY: ci-setup
ci-setup:
	@mkdir -p $(HOME)/.oci
	@echo $(ORACLE_CONFIG) | base64 --decode --ignore-garbage > $(HOME)/.oci/config
	@echo $(ORACLE_KEY) | base64 --decode --ignore-garbage > $(HOME)/.oci/oci_api_key.pem
	@chmod go-rwx $(HOME)/.oci/*
	@echo $(GCLOUD_SERVICE_KEY) | base64 --decode --ignore-garbage > $(HOME)/gcp-key.json

.PHONY: test
test:
	rm -rf .test/ && mkdir .test/
	go test -v -covermode=atomic -coverprofile=coverage.out .
	go tool cover -html=coverage.out -o=coverage.html

.PHONY: testcloud
testcloud: export TEST_CLOUD_STORAGE=1
testcloud: test

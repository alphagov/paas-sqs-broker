.PHONY: unit
unit:
	ginkgo $(COMMAND) -r --skipPackage=testing/integration $(PACKAGE)

.PHONY: test
test:
	go test ./...

.PHONY: generate
generate:
	go generate ./...

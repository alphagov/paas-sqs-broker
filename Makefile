.PHONY: unit
unit:
	ginkgo $(COMMAND) -r --skipPackage=testing/integration $(PACKAGE)

.PHONY: test
test:
	go test -mod=vendor ./...

.PHONY: generate
generate:
	go generate ./...

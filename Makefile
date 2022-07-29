
.PHONY: test
test:
	ginkgo -v -r -nodes=2

.PHONY: generate
generate:
	go generate ./...

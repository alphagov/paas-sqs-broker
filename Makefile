
.PHONY: test
test:
	go run github.com/onsi/ginkgo/v2/ginkgo -v -r -nodes=2

.PHONY: generate
generate:
	go generate ./...

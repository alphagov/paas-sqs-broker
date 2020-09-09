
.PHONY: test
test:
	go run github.com/onsi/ginkgo/ginkgo -v ./...

.PHONY: generate
generate:
	go generate ./...

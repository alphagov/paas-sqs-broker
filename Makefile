
.PHONY: test
test:
	go run github.com/onsi/ginkgo/ginkgo -v -mod=vendor ./...

.PHONY: generate
generate:
	go generate ./...

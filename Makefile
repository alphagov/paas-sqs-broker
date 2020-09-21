
.PHONY: test
test:
	go run github.com/onsi/ginkgo/ginkgo -v -mod=vendor -nodes=2 -stream ./...

.PHONY: generate
generate:
	go generate ./...

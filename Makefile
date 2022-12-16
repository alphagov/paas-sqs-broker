
.PHONY: test
test:
	go run github.com/onsi/ginkgo/v2/ginkgo -v -r -nodes=2

.PHONY: generate
generate:
	go generate ./...

.PHONY: build_amd64
build_amd64:
	mkdir -p amd64
	GOOS=linux GOARCH=amd64 go build -o amd64/sqs-broker

.PHONY: bosh_scp
bosh_scp: build_amd64
	./scripts/bosh-scp.sh

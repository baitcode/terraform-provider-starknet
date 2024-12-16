default: fmt lint install generate

build:
	go build -v ./...

install: build
	go install -v ./...
	rm ~/.terraform.d/plugins/local/baitcode/starknet/1.0.0/darwin_arm64/terraform-provider-starknet | true
	cp ~/go/bin/terraform-provider-starknet ~/.terraform.d/plugins/local/baitcode/starknet/1.0.0/darwin_arm64/terraform-provider-starknet

lint:
	golangci-lint run

generate:
	cd tools; go generate ./...

fmt:
	gofmt -s -w -e .

test:
	go test -v -cover -timeout=120s -parallel=10 ./...

testacc:
	TF_ACC=1 go test -v -cover -timeout 120m ./...

.PHONY: fmt lint test testacc build install generate

GITCOMMIT := $(shell git rev-parse HEAD)
GITDATE := $(shell git show -s --format='%ct')

LDFLAGSSTRING +=-X main.GitCommit=$(GITCOMMIT)
LDFLAGSSTRING +=-X main.GitDate=$(GITDATE)
LDFLAGS := -ldflags "$(LDFLAGSSTRING)"

exchange-wallet-service:
	env GO111MODULE=on go build -v $(LDFLAGS) ./cmd/exchange-wallet-service

clean:
	rm exchange-wallet-service

test:
	go test -v ./...

protogo:
	sh ./sh/go_compile.sh

lint:
	golangci-lint run ./...

.PHONY: \
	exchange-wallet-service \
	clean \
	test \
	protogo \
	lint
ifeq ($(shell uname),Linux)
	OS=linux
else
	OS=darwin
endif

ifeq ($(shell uname -m),aarch64)
    ARCH=arm64
else ifeq ($(shell uname -m),arm64)
    ARCH=arm64
else
    ARCH=amd64
endif

default:
	make clean
	go build -o builds/webp-server-$(OS)-$(ARCH) .
	ls builds

all:
	make clean
	./scripts/build.sh $(OS) $(ARCH)

tools-dir:
	mkdir -p tools/bin

install-staticcheck: tools-dir
	GOBIN=`pwd`/tools/bin go install honnef.co/go/tools/cmd/staticcheck@latest
	curl -sfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh| sh -s -- -b ./tools/bin v1.59.1

static-check: install-staticcheck
	#S1000,SA1015,SA4006,SA4011,S1023,S1034,ST1003,ST1005,ST1016,ST1020,ST1021
	tools/bin/staticcheck -checks all,-ST1000 ./...
	GO111MODULE=on tools/bin/golangci-lint run -v $$($(PACKAGE_DIRECTORIES)) --config .golangci.yml

test:
	go test -v -coverprofile=coverage.txt -covermode=atomic ./...

clean:
	rm -rf prefetch remote-raw exhaust tools coverage.txt metadata exhaust_test

docker:
	DOCKER_BUILDKIT=1 docker build -t webpsh/webps .

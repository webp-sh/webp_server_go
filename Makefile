ifeq ($(shell uname),Linux)
	OS=linux
else
	OS=darwin
endif

ifeq ($(shell uname -m),aarch64)
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

test:
	go test -v -coverprofile=coverage.txt -covermode=atomic

clean:
	rm -rf builds
	rm -rf prefetch

docker:
	DOCKER_BUILDKIT=1 docker build -t webpsh/webps .
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

all: build
build: 
	./scripts/build.sh $(OS) $(ARCH)
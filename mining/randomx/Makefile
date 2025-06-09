# RandomX CGO build configuration
RANDOMX_DIR := ../../third_party/randomx
RANDOMX_LIB := $(RANDOMX_DIR)/build/librandomx.a

.PHONY: all clean build-deps test

all: build-deps
	go build -tags cgo .

build-deps: $(RANDOMX_LIB)

$(RANDOMX_LIB):
	./build.sh

clean:
	rm -rf $(RANDOMX_DIR)/build
	go clean

test: build-deps
	go test -tags cgo -v . 
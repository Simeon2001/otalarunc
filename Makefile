# Name of the final binary
BINARY_NAME = alpinecell

# Go source files
SRC = alpinecell.go container.go randNo.go spawnuser.go cleaner.go mountDir.go utils/cgroup.go utils/devices.go

# Where to install the binary (must be in $PATH)
INSTALL_DIR = /usr/local/bin

all: build

build:
	go build -o $(BINARY_NAME) .


install: build
	sudo cp $(BINARY_NAME) $(INSTALL_DIR)

container-test: build
	./$(BINARY_NAME) run /bin/sh

clean:
	rm -f $(BINARY_NAME)

.PHONY: all build install clean container-test

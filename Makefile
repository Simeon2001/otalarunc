# Name of the final binary
BINARY_NAME = alpinecell

# Go source files
SRC = alpinecell.go container.go randNo.go  spawnuser.go

# Where to install the binary (must be in $PATH)
INSTALL_DIR = /usr/local/bin

all: build

build:
	go build -o $(BINARY_NAME) $(SRC)

install: build
	sudo cp $(BINARY_NAME) $(INSTALL_DIR)

clean:
	rm -f $(BINARY_NAME)

.PHONY: all build install clean

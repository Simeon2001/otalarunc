package utils

import (
	"bytes"
	"log"
	"os"
)

func ensureResolvConf(path string) {
	defaultContent := []byte("nameserver 8.8.8.8\nnameserver 8.8.4.4\n")

	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		// File does not exist: create it with default content
		log.Printf("%s does not exist, creating with default nameservers...", path)
		if err := os.WriteFile(path, defaultContent, 0644); err != nil {
			log.Fatalf("Failed to create %s: %v", path, err)
		}
		return
	} else if err != nil {
		log.Fatalf("Error checking %s: %v", path, err)
	}

	if info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		// Read the file content
		content, err := os.ReadFile(path)
		if err != nil {
			log.Fatalf("Failed to read %s: %v", path, err)
		}

		if bytes.Contains(content, []byte("nameserver")) {
			log.Printf("%s already contains nameserver entries, nothing to do.", path)
			return
		}

		// No nameserver found, overwrite
		log.Printf("%s missing nameserver entries, overwriting...", path)
		if err := os.WriteFile(path, defaultContent, 0644); err != nil {
			log.Fatalf("Failed to overwrite %s: %v", path, err)
		}
	} else {
		log.Fatalf("%s exists but is not a regular file or symlink", path)
	}
}

func main() {
	ensureResolvConf("/etc/resolv.conf")
}

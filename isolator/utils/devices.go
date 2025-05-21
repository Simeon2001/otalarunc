package utils

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

func CreateDeviceNodesAndMount(rootfs string) error {

	deviceList := [7]string{"null", "zero", "full", "random", "urandom", "tty", "console"}

	// Create each device node
	for _, device := range deviceList {
		hostDevicepath := filepath.Join("/dev", device)
		containerDevicepath := filepath.Join(rootfs, "/dev", device)

		if _, err := os.Stat(containerDevicepath); os.IsNotExist(err) {
			file, err := os.Create(containerDevicepath)
			if err != nil {
				return fmt.Errorf("failed to create %s: %w", containerDevicepath, err)
			}
			file.Close()
		}
		// mount --bind "/dev/$d" "$ROOTFS/dev/$d"
		if err := unix.Mount(hostDevicepath, containerDevicepath, "", unix.MS_BIND, ""); err != nil {
			return fmt.Errorf("bind mount failed: %v -> %v: %w", hostDevicepath, containerDevicepath, err)
		}
		log.Printf("[+] Bound %s -> %s", hostDevicepath, containerDevicepath)

	}

	if err := createDevSymlinks(rootfs); err != nil {
		return err
	}

	return nil
}

// symlink for stdin, stdout, stderr in /dev
func createDevSymlinks(rootfs string) error {
	dev := filepath.Join(rootfs, "dev")

	links := []struct {
		linkName string
		target   string
	}{
		{"stdin", "../proc/self/fd/0"},
		{"stdout", "../proc/self/fd/1"},
		{"stderr", "../proc/self/fd/2"},
	}

	for _, l := range links {
		fullLinkPath := filepath.Join(dev, l.linkName)

		// Remove existing file if any (e.g. from extracted rootfs)
		if err := os.RemoveAll(fullLinkPath); err != nil {
			return err
		}

		// Create the symlink
		if err := os.Symlink(l.target, fullLinkPath); err != nil {
			return err
		}
	}

	return nil
}

func MaskPaths() error {
	var maskedPaths = []string{
		"/proc/acpi",
		"/proc/kcore",
		"/proc/keys",
		"/proc/latency_stats",
		"/proc/sched_debug",
		"/proc/scsi",
		"/proc/timer_list",
		"/proc/timer_stats",
		"/sys/devices/virtual/powercap",
		"/sys/firmware",
		"/sys/fs/selinux",
		"/sys/devices/virtual/powercap",
	}
	for _, p := range maskedPaths {
        fi, err := os.Stat(p)
        if os.IsNotExist(err) {
            continue // only mask existing paths
        }
        if err != nil {
            return fmt.Errorf("stat %q: %w", p, err)
        }

        if fi.IsDir() {
            if err := unix.Mount(
                "tmpfs", p, "tmpfs",
                unix.MS_RDONLY|unix.MS_NOSUID|unix.MS_NODEV|unix.MS_NOEXEC,
                "mode=755,size=0",
            ); err != nil {
                return fmt.Errorf("mount tmpfs on %q: %w", p, err)
            }
        } else {
            if err := unix.Mount(
                "/dev/null", p, "",
                unix.MS_BIND, "",
            ); err != nil {
                return fmt.Errorf("bind-mount /dev/null on %q: %w", p, err)
            }
        }
    }
    return nil
}

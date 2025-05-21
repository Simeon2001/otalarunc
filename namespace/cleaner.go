package namespace

import (
	"log"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

func clean(mountedProjectDir string, rootfs string) {
	log.Println("[✅] Cleaning up mounts...")

	// Unmount proc
	procPath := filepath.Join(rootfs, "proc")
	if err := unix.Unmount(procPath, 0); err != nil {
		log.Printf("[❌] Failed to unmount proc: %v", err)
	}

	// Unmount bind mount
	mountedProjectDirPath := filepath.Join(rootfs, mountedProjectDir)
	if err := unix.Unmount(mountedProjectDirPath, unix.MNT_DETACH); err != nil {
		log.Printf("[❌] Failed to unmount bind path: %v", err)
	} else {
		log.Printf("[✅] Unmounted %s", mountedProjectDir)
	}

	// Remove bind mount directory
	if err := os.RemoveAll(mountedProjectDirPath); err != nil {
		log.Printf("[❌] Failed to delete %s: %v", mountedProjectDir, err)
	} else {
		log.Printf("[✅] Cleaned up %s", mountedProjectDir)
	}

	// Unmount rootfs
	if _, err := os.Stat(rootfs); err == nil {
		if err := unix.Unmount(rootfs, unix.MNT_DETACH); err != nil {
			log.Printf("[❌] Failed to unmount rootfs: %v", err)
		} else {
			log.Printf("[✅] Unmounted %s", rootfs)
		}
	} else {
		log.Printf("[❌] Rootfs path %s does not exist or can't be accessed: %v", rootfs, err)
	}

	log.Println("[✅] Finished unmounting — now reaping zombies...")

	// Small delay just to allow child exit cleanup if needed
	time.Sleep(2 * time.Second)

	// Reap zombies
	zombieCount := 0
	for {
		var ws unix.WaitStatus
		pid, err := unix.Wait4(-1, &ws, 0, nil)
		if err != nil {
			log.Printf("[❌] Reaper finished, collected %d zombies", zombieCount)
			break
		}
		zombieCount++
		log.Printf("[✅] Reaped zombie process with pid %d", pid)
	}

}

func killer(pid int) {
	// Find process
	process, err := os.FindProcess(pid)
	if err != nil {
		log.Printf("[❌] Error finding process: %v\n", err)
		return
	}
	// Send signal
	err = process.Signal(syscall.SIGKILL)
	if err != nil {
		log.Printf("[❌] Failed to send signal: %v\n", err)
		return
	}

	log.Printf("Sent %v to process %d\n", "SIGKILL", pid)

}

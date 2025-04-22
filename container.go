package main

import (
	"context"
	"golang.org/x/sys/unix"
	"log"
	"os"
	"os/exec"
	"time"
)

func spawnContainer() {
	must("set hostname", unix.Sethostname([]byte("otala-runc")))
	// get the current working directory
	cwd, err := os.Getwd()
	must("getting current dir", err)
	log.Printf("here the path you want to copy: %s\n", cwd)

	// Define rootfs path
	// rootfs := "/home/ciscoquan/alpine"
	rootfs := "/home/ciscoquan/Desktop/project-back/training/bocker-master/lizrice/alpine-rootfs/"
	mountedProjectDir := randNowithDash()
	bindDest := rootfs + mountedProjectDir
	// Check if the rootfs directory exists
	if _, err := os.Stat(rootfs); os.IsNotExist(err) {
		log.Printf("[-] Rootfs directory %s does not exist\n", rootfs)
		os.Exit(1)
	}
	// Make mount namespace private
	must("namespace private mount error: ", unix.Mount("", "/", "", unix.MS_PRIVATE|unix.MS_REC, ""))

	// Mount the rootfs as a bind mount (required for pivot_root)
	must("bind mount of rootfs error: ", unix.Mount(rootfs, rootfs, "", unix.MS_BIND|unix.MS_REC, ""))

	// Create a directory to hold the old root (inside the new root)
	putOld := rootfs + "/.pivot_old"
	must("creating dir for old root error: ", os.MkdirAll(putOld, 0700))

	// Bind mount host folder
	must("create Bind mount host folder error: ", os.MkdirAll(bindDest, 0700))
	must("bind mount the source to the target dest error: ", unix.Mount(cwd, bindDest, "", unix.MS_BIND, ""))

	// change to the new rootfs
	must("chdir error: ", os.Chdir(rootfs))

	// do the pivot_root
	must("pivot_root failed", unix.PivotRoot(".", ".pivot_old"))
	must("changing root dir gone wrong: ", unix.Chdir("/"))
	must("proc didnt mount: ", unix.Mount("proc", "proc", "proc", 0, ""))

	// Unmount the old root and remove the directory
	must("unmount old root failed: ", unix.Unmount("/.pivot_old", unix.MNT_DETACH))
	must("remove pivot_old dir failed: ", os.RemoveAll("/.pivot_old"))
	// Move to our mounted project folder
	must("chdir to where cwd dir are failed: ", os.Chdir(mountedProjectDir))

	cmd := exec.Command(os.Args[2], os.Args[3:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), "PATH=/bin:/usr/bin:/sbin:/usr/sbin")

	// Create a context that cancels after 20 minutes
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel() // Always call cancel to release resources

	// Create a channel to signal when command completes
	cmdDone := make(chan error)

	// Start a goroutine to run the command and signal when done
	go func() {
		cmdDone <- cmd.Run()
	}()

	// Wait for either command completion or timeout
	select {
	case err := <-cmdDone:
		if err != nil {
			log.Printf("command execution failed: %v", err)
		}
	case <-ctx.Done():
		log.Println("Operation timed out or was canceled:", ctx.Err())
		// Kill the command
		if err := cmd.Process.Kill(); err != nil {
			log.Printf("killing cmd failed: %v\n", err)
		}
	}

	// Cleanup always runs - moved outside the select statement
	log.Println("[+] Cleaning up mounts...")

	// Unmount in reverse order (most specific to least specific)
	// First unmount proc
	if err := unix.Unmount("/proc", 0); err != nil {
		log.Printf("[-] Failed to unmount proc: %v", err)
	}

	// Then unmount the project directory
	if err := unix.Unmount(mountedProjectDir, unix.MNT_DETACH); err != nil {
		log.Printf("[-] Failed to unmount bind path: %v", err)
	} else {
		log.Printf("[+] Unmounted %s", mountedProjectDir)
	}

	// Remove the mounted project directory
	if err := os.RemoveAll(mountedProjectDir); err != nil {
		log.Printf("[-] Delete mountedProjectDir failed: %v", err)
	} else {
		log.Printf("[+] Cleaned up %s", mountedProjectDir)
	}

	// Finally unmount rootfs, check if it exists first
	if _, err := os.Stat(rootfs); err == nil {
		if err := unix.Unmount(rootfs, unix.MNT_DETACH); err != nil {
			log.Printf("[-] Failed to unmount rootfs: %v", err)
		} else {
			log.Printf("[+] Unmounted %s", rootfs)
		}
	} else {
		log.Printf("[!] Rootfs path %s does not exist or can't be accessed: %v", rootfs, err)
	}

	log.Println("[+] Cleaned up mounts")

}

//tempDir := "/tmp/alpine-root"
//// tempDir := filepath.Join(os.TempDir(), fmt.Sprintf("alpine-root"))
//_ = os.RemoveAll(tempDir)
//must("extracting and creating tootfs failed", extractRootfs(tempDir))

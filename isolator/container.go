package isolator

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/Simeon2001/AlpineCell/isolator/utils"
	"golang.org/x/sys/unix"
)

func SpawnContainer() {
	log.Println("[✅] Spawning container...")
	log.Println(os.Args)
	// Ensure that we have received enough arguments
	if len(os.Args) < 4 {
		log.Fatal("Expected at least one argument")
	}

	// get the current working directory
	cwd, err := os.Getwd()
	must("getting current dir", err)
	log.Printf("here the path you want to copy: %s\n", cwd)

	// Define rootfs path
	rootfs := "/home/ciscoquan/alpine"
	// rootfs := "/home/ciscoquan/Desktop/project-back/training/bocker-master/lizrice/alpine-rootfs/"
	mountedProjectDir := os.Args[2]
	bindDest := filepath.Join(rootfs, mountedProjectDir) // rootfs + mountedProjectDir
	// Check if the rootfs directory exists
	if _, err := os.Stat(rootfs); os.IsNotExist(err) {
		log.Printf("[❌] Rootfs directory %s does not exist\n", rootfs)
		os.Exit(1)
	}

	// Make mount namespace private
	must("namespace private mount error: ", unix.Mount("", "/", "", unix.MS_PRIVATE|unix.MS_REC, ""))

	// Mount the rootfs as a bind mount (required for pivot_root)
	must("bind mount of rootfs error: ", unix.Mount(rootfs, rootfs, "", unix.MS_BIND|unix.MS_REC, ""))

	// mounted proc, dev, sys and devicesnode
	// mounter(rootfs, mountedProjectDir)
	mounter(rootfs)

	// Create a directory to hold the old root (inside the new root)
	putOld := filepath.Join(rootfs, ".pivot_old") //rootfs + "/.pivot_old"
	must("creating dir for old root error: ", os.MkdirAll(putOld, 0700))

	// Bind mount host folder
	must("create Bind mount host folder error: ", os.MkdirAll(bindDest, 0700))
	must("bind mount the source to the target dest error: ", unix.Mount(cwd, bindDest, "", unix.MS_BIND, ""))

	// change to the new rootfs
	must("chdir error: ", os.Chdir(rootfs))

	// do the pivot_root
	must("pivot_root failed", unix.PivotRoot(".", ".pivot_old"))
	must("changing root dir gone wrong: ", unix.Chdir("/"))
	must("masked path failed: ", utils.MaskPaths())
	// must(" Failed to remount: ", MakeFilesystemsReadOnly())

	// Unmount the old root and remove the directory
	must("unmount old root failed: ", unix.Unmount("/.pivot_old", unix.MNT_DETACH))
	must("remove pivot_old dir failed: ", os.RemoveAll("/.pivot_old"))
	// Move to our mounted project folder
	must("chdir to where cwd dir are failed: ", os.Chdir(mountedProjectDir))

	cmdPath, err := exec.LookPath(os.Args[3])
	log.Println("cmdpath : ", cmdPath)
	must("cmdpath error: ", err)

	// Build args for the target command (os.Args[2:] = command and its args)
	cmdArgs := os.Args[4:]
	log.Println("cmdArgs : ", cmdArgs)

	argv := append([]string{cmdPath}, cmdArgs...)
	log.Println(argv)

	// Build the command with the full path and args
	env := append(os.Environ(), "PATH=/bin:/usr/bin:/sbin:/usr/sbin", "TERM=xterm")
	// setSid, err := unix.Setsid()
	// log.Println("[✅] child pid: ", setSid)
	must("creating a new session failed: ", err)

	must("command  Exec error: ", unix.Exec(cmdPath, argv, env))

}

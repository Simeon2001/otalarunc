package namespace

import (
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

func Stage1UserNS(dirPath string) {
	// Configure signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	var processID int
	log.Println("[✅] Creating container with user namespace...")
	rootfs := "/home/ciscoquan/alpine"
	// rootfs := "/home/ciscoquan/Desktop/project-back/training/bocker-master/lizrice/alpine-rootfs/"
	mountedProjectDir := "MDIR-" + dirPath
	cmd := exec.Command("/proc/self/exe", append([]string{"child", mountedProjectDir}, os.Args[2:]...)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Set up syscall attributes for the new process with all namespaces at once

	cmd.SysProcAttr = &syscall.SysProcAttr{
		// syscall.CLONE_NEWUTS which mean new hostname
		// syscall.CLONE_NEWPID which mean new processid
		// syscall.CLONE_NEWNET which mean new network stack
		// syscall.CLONE_NEWNS which mean new mount filesystem
		// for /sys to mount make sure you use NET namespace
		// for cgroup to mount make sure you use CGROUP namespace
		Cloneflags: unix.CLONE_NEWUSER | unix.CLONE_NEWNS | unix.CLONE_NEWUTS | unix.CLONE_NEWIPC | unix.CLONE_NEWPID | unix.CLONE_NEWCGROUP | unix.CLONE_NEWNET,

		// UidMappings: []syscall.SysProcIDMap{
		// 	{ContainerID: 0, HostID: 1000, Size: 1},       // Map root in container to host UID 1000
		// 	// {ContainerID: 1, HostID: 100000, Size: 65536}, // Map UID 1+ in container to host UID 100000+
		// },
		// // Corresponds to: --map-groups 0:1000:1 --map-groups 1:100000:65536
		// // Assuming host GID 1000 maps to container GID 0 (root group)
		// GidMappings: []syscall.SysProcIDMap{
		// 	{ContainerID: 0, HostID: 1000, Size: 1},       // Map root group in container to host GID 1000
		// 	// {ContainerID: 1, HostID: 100000, Size: 65536}, // Map GID 1+ in container to host GID 100000+
		// },

		UidMappings: []syscall.SysProcIDMap{
			{
				ContainerID: 0,
				HostID:      os.Getuid(),
				Size:        1,
			},
		},
		GidMappings: []syscall.SysProcIDMap{
			{
				ContainerID: 0,
				HostID:      os.Getgid(),
				Size:        1,
			},
		},
	}

	must("executing child process failed", cmd.Start())
	processID = cmd.Process.Pid
	log.Printf("[✅] Child PID after running child: %d", processID)

	// Set up a goroutine to handle termination signals
	go func(pid int, sigChan chan os.Signal) {
		sig := <-sigChan
		log.Printf("[⚠️] Received signal %v. Shutting down container...", sig)
		killer(pid)
		log.Println("[✅] Cleanup complete")
		// Wait a moment for cleanup to complete
		time.Sleep(500 * time.Millisecond)
		os.Exit(0)
	}(processID, sigChan)

	// Wait for the child process to complete
	err := cmd.Wait()
	if err != nil {
		log.Printf("[❌] Child process exited with error: %v", err)
	} else {
		log.Println("[✅] Container exited successfully")
	}

	// Clean up resources
	clean(mountedProjectDir, rootfs)
	log.Println("[✅] All resources cleaned up")

}

func must(reply string, err error) {
	if err != nil {
		log.Printf("[❌] %s: %v", reply, err)
		os.Exit(1)
	}
}

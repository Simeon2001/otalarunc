package main

import (
	"fmt"
	"golang.org/x/sys/unix"
	"log"
	"os"
	"os/exec"
	"syscall"
)

func stage1UserNS() {

	log.Println("[+] Creating container with user namespace...")

	cmd := exec.Command("/proc/self/exe", append([]string{"child"}, os.Args[2:]...)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Set up syscall attributes for the new process with all namespaces at once
	cmd.SysProcAttr = &syscall.SysProcAttr{
		// syscall.CLONE_NEWUTS which mean new hostname
		// syscall.CLONE_NEWPID which mean new processid
		// syscall.CLONE_NEWNET which mean new network stack
		// syscall.CLONE_NEWNS which mean new mount filesystem
		Cloneflags: unix.CLONE_NEWUSER | unix.CLONE_NEWNS | unix.CLONE_NEWUTS | unix.CLONE_NEWIPC | unix.CLONE_NEWPID,
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

	// Execute the child process
	must("executing child process failed", cmd.Run())

	fmt.Println("Container exited")
}

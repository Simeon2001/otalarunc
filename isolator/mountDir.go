package isolator

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/Simeon2001/AlpineCell/isolator/utils"
	"golang.org/x/sys/unix"
)

// func mounter(rootfs, dirName string) {
func mounter(rootfs string) {
	// Create necessary directories
	dirs := []string{
		"/dev", "/dev/pts", "/dev/mqueue", "/dev/shm",
		"/sys", "/sys/fs/cgroup", "/run", "/proc",
		"/proc/acpi", "/proc/scsi", "/tmp",
	}
	for _, dir := range dirs {
		devDir := filepath.Join(rootfs, dir)
		if err := os.MkdirAll(devDir, 0755); err != nil {
			log.Printf("Warning: Failed to create directory %s: %v", dir, err)
		}
	}

	// Mount filesystems
	conProc := filepath.Join(rootfs, "/proc")
	conDev := filepath.Join(rootfs, "/dev")
	conSys := filepath.Join(rootfs, "/sys")
	cgroupPath := filepath.Join(conSys, "/fs/cgroup")
	must("proc mount failed: ", unix.Mount("proc", conProc, "proc", 0, ""))
	must(" Failed to remount: ", makeFilesystemsReadOnly(rootfs))
	// Try mounting sysfs (works in rootful), else bind-mount /sys
	must("mount sysfs", unix.Mount("sysfs", conSys, "sysfs", unix.MS_NOSUID|unix.MS_NOEXEC|unix.MS_NODEV|unix.MS_RDONLY, ""))
	log.Println("Mounted /sys successfully")
	must("mount /sys/fs/cgroup", unix.Mount("cgroup2", cgroupPath, "cgroup2", unix.MS_NOSUID|unix.MS_NODEV|unix.MS_NOEXEC|unix.MS_RDONLY, "nsdelegate,memory_recursiveprot"))
	must("mount /dev as tmpfs", unix.Mount("tmpfs", conDev, "tmpfs", unix.MS_NOSUID, "mode=755,size=65536k"))
	for _, dir := range []string{"/dev/pts", "/dev/mqueue", "/dev/shm"} {
		deviceDir := filepath.Join(rootfs, dir)
		must("Failed to create device directory %s after mounting tmpfs: ", os.MkdirAll(deviceDir, 0755))
	}

	// More mounts...
	conPts := filepath.Join(rootfs, "/dev/pts")
	conMqueue := filepath.Join(rootfs, "/dev/mqueue")
	conShm := filepath.Join(rootfs, "/dev/shm")
	mountOptions := fmt.Sprintf("ptmxmode=0666,mode=0620,gid=%d", os.Getgid())
	must("mount /dev/pts", unix.Mount("devpts", conPts, "devpts", unix.MS_NOSUID|unix.MS_NOEXEC, mountOptions))
	// Create symlink for /dev/ptmx
	ptmxPath := filepath.Join(rootfs, "/dev/ptmx")
	os.Remove(ptmxPath) // Remove if exists
	must("create ptmx symlink", os.Symlink("pts/ptmx", ptmxPath))
	must("mount /dev/mqueue", unix.Mount("mqueue", conMqueue, "mqueue", unix.MS_NOSUID|unix.MS_NODEV|unix.MS_NOEXEC, ""))
	must("mount /dev/shm", unix.Mount("tmpfs", conShm, "tmpfs", unix.MS_NOSUID|unix.MS_NODEV|unix.MS_NOEXEC, "size=64000k"))

	// Create device nodes
	must("device mount error: ", utils.CreateDeviceNodesAndMount(rootfs))
	setupEtcFiles(rootfs)

	// utils.SetupCgroups(dirName)


}

func makeFilesystemsReadOnly(rootfs string) error {
	// List of filesystems to remount as read-only
	// "readonlyPaths":["/proc/asound","/proc/bus","/proc/fs","/proc/irq","/proc/sys","/proc/sysrq-trigger"]
	readOnlyFs := []string{
		// "/sys",
		"/proc/sys",
		"/proc/sysrq-trigger",
		"/proc/irq",
		"/proc/bus",
		"/proc/asound",
		"/proc/fs",
	}

	for _, fs := range readOnlyFs {
		fsPath := filepath.Join(rootfs, fs)
		_, err := os.Stat(fsPath)
		if os.IsNotExist(err) {
			return fmt.Errorf("file does not exist")
		}
		if err != nil {
			return fmt.Errorf("stat %q: %w", fs, err)
		}

		if err := unix.Mount(fsPath, fsPath, "", unix.MS_BIND|unix.MS_REC, ""); err != nil {
			// fmt.Printf("Warning: Failed to remount %s as read-only: %v", fs, err)
			return fmt.Errorf("Failed to bindmount %s as read-only: %w", fs, err)
		} else {
			log.Printf("[+] Remounted %s as read-only", fs)
			// "ro,nosuid,nodev,noexec,relatime" relatime
			if err := unix.Mount("", fsPath, "", unix.MS_REMOUNT|unix.MS_RDONLY|unix.MS_NOSUID|unix.MS_NODEV|unix.MS_NOEXEC|unix.MS_RELATIME, ""); err != nil {
				return fmt.Errorf("Warning: Failed to remount %s as read-only: %w", fs, err)
			}

		}
	}
	return nil
}

func setupEtcFiles(rootfs string) {
	// 1) helper tmpfs mount point under rootfs
	baseTmp := filepath.Join(rootfs, "tmp", "tmpfs-etc")
	must("mkdir tmpfs-etc", os.MkdirAll(baseTmp, 0700))
	must("mount tmpfs for /etc files", unix.Mount("tmpfs", baseTmp, "tmpfs", unix.MS_NOSUID|unix.MS_NODEV, "size=64k,mode=700"))

	// file names we need
	etcFiles := []string{"resolv.conf", "hosts", "hostname"}
	for _, name := range etcFiles {
		// path inside tmpfs
		tmpFile := filepath.Join(baseTmp, name)
		must("create "+tmpFile, func() error {
			f, err := os.Create(tmpFile)
			if err != nil {
				return err
			}
			return f.Close()
		}())

		// target path under rootfs
		target := filepath.Join(rootfs, "etc", name)
		// ensure the parent dir exists
		must("mkdir parent of "+target, os.MkdirAll(filepath.Dir(target), 0755))

		// bind-mount the tmpfs file over the real etc file
		must("bind-mount "+name, unix.Mount(tmpFile, target, "", unix.MS_PRIVATE|unix.MS_BIND, ""))
	}

	// 2) set hostname inside container
	must("set hostname", unix.Sethostname([]byte("otala-runc")))

	// 3) write default hosts + resolv.conf
	hostsPath := filepath.Join(rootfs, "etc", "hosts")
	hostsData := []byte("127.0.0.1 localhost\n127.0.0.1 otala-runc\n")
	must("write hosts", os.WriteFile(hostsPath, hostsData, 0644))

	resolvPath := filepath.Join(rootfs, "etc", "resolv.conf")
	resolvData := []byte("nameserver 8.8.8.8\nnameserver 1.1.1.1\n")
	must("write resolv.conf", os.WriteFile(resolvPath, resolvData, 0644))
}

func must(reply string, err error) {
	if err != nil {
		log.Printf("[❌] %s: %v", reply, err)
		os.Exit(1)
	}
}

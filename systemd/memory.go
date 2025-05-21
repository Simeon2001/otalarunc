package systemd

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/godbus/dbus/v5"
)

func SystemdManager(containerName string, memoryBytes string) (error, bool, string){
	const (
		// systemd D-Bus interface
		SystemdUserService   = "org.freedesktop.systemd1"
		SystemdUserPath      = "/org/freedesktop/systemd1"
		SystemdUserInterface = "org.freedesktop.systemd1.Manager"
	)

	memBytes, err := strconv.ParseUint(memoryBytes, 10, 64)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Memory limit must be a number: %v\n", err)
		os.Exit(1)
	}


	// Connect to the user's session bus
	conn, err := dbus.ConnectSessionBus()
	defer conn.Close()
	if err != nil {
		return fmt.Errorf("Failed to connect to session bus: %v", err), false, ""
	}


	// Get systemd manager object
	systemd := conn.Object(SystemdUserService, dbus.ObjectPath(SystemdUserPath))

	// Create unit name - scope units end with .scope
	unitName := fmt.Sprintf("%s.scope", containerName)

	var unitPath dbus.ObjectPath

	// Try to get the unit — if it doesn't exist, we skip stopping
	err = systemd.Call("org.freedesktop.systemd1.Manager.GetUnit", 0, unitName).Store(&unitPath)
	if err != nil {
		fmt.Printf("Unit %s does not exist, continuing...\n", unitName)
	} else{
		return fmt.Errorf("containerName existed already in cgroup SystemdUserService"), true, ""
	}

	// The correct D-Bus signature is a(sv)
	properties := []struct {
		Name  string
		Value dbus.Variant
	}{
		// {
		// 	Name:  "Description",
		// 	Value: dbus.MakeVariant(fmt.Sprintf("Scope for %s", containerName)),
		// },
		{
			Name:  "MemoryMax",
			Value: dbus.MakeVariant(memBytes), // uint64
		},
		{
			Name:  "MemorySwapMax",
			Value: dbus.MakeVariant(memBytes), // uint64
		},
		{
			Name:  "PIDs",
			Value: dbus.MakeVariant([]uint32{uint32(os.Getpid())}),
		},
	}

	// For a(sa(sv)) — no auxiliary units
	var aux []struct {
		Name       string
		Properties []struct {
			Name  string
			Value dbus.Variant
		}
	}


	// Call StartTransientUnit
	var jobPath dbus.ObjectPath

	err = systemd.Call(
		SystemdUserInterface+".StartTransientUnit",
		0,
		unitName,   // unit name
		"replace",  // mode - "replace" is usually what you want
		properties, // properties - matches a(sv)
		aux,        // aux (unused, pass empty) - matches a(sa(sv))
	).Store(&jobPath)

	if err != nil {
		return fmt.Errorf("Failed to start transient unit: %v\n", err), false, ""
	}

	log.Printf("Creating scope unit: %s\n", unitName)
	log.Printf("Memory limits: %v bytes\n", memoryBytes)

	// Give systemd a moment to set up the cgroup
	time.Sleep(500 * time.Millisecond)

	// Find and verify the cgroup path
	cgroupPath, err := findCgroupPath(containerName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to find cgroup path: %v\n", err)
	} else {
		// Check memory.max
		log.Printf("%s is the cgroup path \n", cgroupPath)
		memoryMax, err := readMemoryMax(cgroupPath)
		if err != nil {
			return fmt.Errorf("Failed to read memory.max: %v\n", err), false, ""
		} else {
			log.Printf("Verified memory.max: %s\n", memoryMax)
		}
	}

	return nil, false, cgroupPath

}

package main

import (
	"log"
	"github.com/Simeon2001/AlpineCell/namespace"
	"github.com/Simeon2001/AlpineCell/systemd"
)

func InitProcess() {
	uniqueID, err := generateUniqueID()
	if err != nil {
		must("Generating uniqueID err: ", err)
	}
	containerName := "otalacon-" + uniqueID
	memoryAllocoy := "104857600"

	err, boolValue, cgroupPath := systemd.SystemdManager(containerName, memoryAllocoy)
	if err != nil{
		if boolValue == true{
			InitProcess()
		}else{
			must("systemd error", err)
		}
	}
	log.Printf("your cgroup Path: %s\n", cgroupPath)

	namespace.Stage1UserNS(uniqueID)



}

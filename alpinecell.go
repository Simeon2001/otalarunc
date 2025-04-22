package main

import (
	"log"
	"os"
)

//
//import (
//	"embed"
//	"log"
//	"os"
//)
//
////go:embed otala-alpine.tar.gz
//var alpineFS embed.FS

func main() {
	switch os.Args[1] {
	case "run":
		stage1UserNS()
	case "child":
		spawnContainer()
	//case "child":
	//	containerInitProcess()
	default:
		panic("unknown command")
	}
}
func must(reply string, err error) {
	if err != nil {
		log.Printf("🔥 %s: %v", reply, err)
		os.Exit(1)
	}
}

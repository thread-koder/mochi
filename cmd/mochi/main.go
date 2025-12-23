package main

import (
	"fmt"
	"os"
)

var (
	Version   = "dev"
	BuildTime = "unknown"
	GitCommit = "unknown"
)

func main() {
	fmt.Printf("Mochi - Kubernetes Resource Optimization Platform\n")
	fmt.Printf("Version: %s\n", Version)
	fmt.Printf("Build Time: %s\n", BuildTime)
	fmt.Printf("Git Commit: %s\n", GitCommit)
	fmt.Println("\nMochi is starting...")

	os.Exit(0)
}

// +build mage

package main

import (
	"fmt"
	"os/exec"
)

// Default target to run when none is specified
// If not set, running mage will list available targets
// var Default = Build

// A build step that requires additional params, or platform specific steps for example
func Build() error {
	fmt.Print("Building...")
	cmd := exec.Command("bash", "-c", "KO_DOCKER_REPO=kind.local ko resolve --platform linux/amd64 -f config > artifacts/knative-mesh.yaml")
	return run(cmd)
}

// A custom install step if you need your bin someplace other than go/bin
func Install() error {
	fmt.Print("Installing Knative Mesh...")
	cmd := exec.Command("kubectl", "apply", "-f", "artifacts/knative-mesh.yaml")
	return run(cmd)
}

func Init() error {
	fmt.Print("Initializing Knative Mesh...")
	cmd := exec.Command("kubectl", "apply", "-f", "artifacts/kmesh.yaml")
	return run(cmd)
}

func run(cmd *exec.Cmd) error {
	err := cmd.Run()
	if err != nil{
		return err
	}
	fmt.Println("Done")
	return nil
}

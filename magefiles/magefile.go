//go:build mage

package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

// Default target
var Default = CI

// CI runs the full pipeline — lint, vet, staticcheck, test
func CI() {
	mg.Deps(Tidy, Fmt, Vet, Staticcheck, Test)
}

// Tidy verifies go.mod is tidy and there are no uncommitted changes
func Tidy() error {
	fmt.Println(">> go mod tidy")
	if err := sh.Run("go", "mod", "tidy"); err != nil {
		return err
	}
	out, err := sh.Output("git", "diff", "--exit-code")
	if err != nil {
		fmt.Println("go.mod or go.sum is not tidy:")
		fmt.Println(out)
		return err
	}
	return nil
}

// Fmt checks that all Go files are formatted
func Fmt() error {
	fmt.Println(">> gofmt")
	out, err := sh.Output("gofmt", "-l", ".")
	if err != nil {
		return err
	}
	if strings.TrimSpace(out) != "" {
		fmt.Println("Files not formatted:")
		fmt.Println(out)
		return fmt.Errorf("formatting check failed")
	}
	return nil
}

// Vet runs go vet
func Vet() error {
	fmt.Println(">> go vet")
	return sh.Run("go", "vet", "./...")
}

// Staticcheck installs and runs staticcheck
func Staticcheck() error {
	fmt.Println(">> staticcheck")
	if err := sh.Run("go", "install", "honnef.co/go/tools/cmd/staticcheck@latest"); err != nil {
		return err
	}
	return sh.Run("staticcheck", "./...")
}

// Test runs all tests with coverage
func Test() error {
	fmt.Println(">> go test")
	if err := sh.Run("go", "test", "./...", "-coverprofile=coverage.out", "-covermode=atomic"); err != nil {
		return err
	}
	if _, err := os.Stat("coverage.out"); os.IsNotExist(err) {
		return fmt.Errorf("coverage.out not generated")
	}
	return nil
}

// Lint is an alias for Staticcheck
func Lint() error {
	return Staticcheck()
}

// Private configures Go environment for private modules
// Mirrors the CI step — useful for local dev with private repos
func Private() error {
	fmt.Println(">> configuring private module access")
	pat := os.Getenv("GH_PAT")
	if pat == "" {
		return fmt.Errorf("GH_PAT environment variable not set")
	}
	cmds := [][]string{
		{"git", "config", "--global",
			fmt.Sprintf("url.https://%s@github.com/.insteadOf", pat),
			"https://github.com/"},
		{"go", "env", "-w", "GOPRIVATE=github.com/ntlaletsi70/*"},
		{"go", "env", "-w", "GONOSUMDB=github.com/ntlaletsi70/*"},
		{"go", "env", "-w", "GOPROXY=direct"},
		{"go", "env", "-w", "GOTOOLCHAIN=local"},
	}
	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return err
		}
	}
	return nil
}

// Build compiles the project
func Build() error {
	fmt.Println(">> go build")
	return sh.Run("go", "build", "./...")
}

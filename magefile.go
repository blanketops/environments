//go:build mage

/*
Copyright 2026 The BlanketOps Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Magefile provides development workflow targets for blanketops-environments.
//
// Usage:
//
//	mage vendor     — tidy and vendor all dependencies
//	mage tidy       — tidy go.mod and go.sum only
//	mage build      — build all packages
//	mage vet        — run go vet
//	mage lint       — run staticcheck
//	mage clean      — remove vendor directory
//	mage ci         — tidy + build + vet (CI gate)
package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

// goFlags are passed to all Go commands to ensure vendor is used when present.
const vendorFlags = "-mod=vendor"

// Vendor tidies go.mod and go.sum then vendors all dependencies into ./vendor.
// After running this once, all subsequent builds read from disk with no
// network access required.
func Vendor() error {
	fmt.Println(">> tidy")
	if err := sh.Run("go", "mod", "tidy"); err != nil {
		return err
	}
	fmt.Println(">> vendor")
	return sh.Run("go", "mod", "vendor")
}

// Tidy runs go mod tidy without vendoring. Use when you want to update
// go.mod and go.sum but are not ready to re-vendor yet.
func Tidy() error {
	fmt.Println(">> tidy")
	return sh.Run("go", "mod", "tidy")
}

// Tools installs all development tools from the vendor directory.
// Run this once after restoring vendor to make mage, staticcheck,
// and gomarkdoc available on PATH.
func Tools() error {
	fmt.Println(">> install tools")
	tools := []string{
		"github.com/princjef/gomarkdoc/cmd/gomarkdoc",
		"honnef.co/go/tools/cmd/staticcheck",
	}
	for _, t := range tools {
		fmt.Printf("   installing %s\n", t)
		if err := sh.Run("go", "install", vendorFlags, t); err != nil {
			return fmt.Errorf("failed to install %s: %w", t, err)
		}
	}
	return nil
}

// Build compiles all packages. Uses vendor if present.
func Build() error {
	fmt.Println(">> build")
	flags := buildFlags()
	return sh.Run("go", "build", flags, "./...")
}

// Vet runs go vet across all packages.
func Vet() error {
	fmt.Println(">> vet")
	flags := buildFlags()
	return sh.Run("go", "vet", flags, "./...")
}

// Lint runs staticcheck across all packages.
func Lint() error {
	fmt.Println(">> lint")
	flags := buildFlags()
	if err := ensureTool("staticcheck"); err != nil {
		return err
	}
	return sh.Run("staticcheck", flags, "./...")
}

// Clean removes the vendor directory.
func Clean() error {
	fmt.Println(">> clean vendor")
	return os.RemoveAll("vendor")
}

// CI runs the full gate: tidy → build → vet. Used in CI pipelines.
// Does not vendor — CI should restore from cache or vendor should be
// committed to the repo.
func CI() {
	mg.SerialDeps(Tidy, Build, Vet)
}

// buildFlags returns -mod=vendor when a vendor directory exists, otherwise
// returns an empty string so Go falls back to the module cache.
func buildFlags() string {
	if _, err := os.Stat("vendor"); err == nil {
		return vendorFlags
	}
	return ""
}

// ensureTool checks that a binary is available on PATH before running it,
// returning a clear error rather than a confusing exec failure.
func ensureTool(name string) error {
	if _, err := exec.LookPath(name); err != nil {
		return fmt.Errorf("%s not found on PATH — run: mage tools", name)
	}
	return nil
}

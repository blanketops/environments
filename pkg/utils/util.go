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

package utils

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	contractv1 "github.com/blanketops/environments-contract/blanketops/environments/v1alpha1"
	"github.com/blanketops/environments/pkg/apis/build/domain"
)

// -----------------------------------------------------------------------------
// Build Execution Hashing
// -----------------------------------------------------------------------------
//
// The execution hash MUST reflect semantic intent only.
// It must NOT depend on Kubernetes storage format or metadata.
func ComputeExecutionHash(
	spec *contractv1.BuildSpec,
	trigger domain.TriggerContext,
) (string, error) {

	if spec == nil {
		return "", fmt.Errorf("build spec is nil")
	}

	type hashInput struct {
		Spec    *contractv1.BuildSpec `json:"spec"`
		Trigger domain.TriggerContext `json:"trigger"`
	}

	in := hashInput{
		Spec:    spec,
		Trigger: trigger,
	}

	raw, err := json.Marshal(in)
	if err != nil {
		return "", fmt.Errorf("failed to marshal hash input: %w", err)
	}

	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}

//
// -----------------------------------------------------------------------------
// Kubernetes helpers
// -----------------------------------------------------------------------------
//

// CreateOrPatch creates obj if it doesn't exist, or leaves it unchanged if
// it does — a thin wrapper around controllerutil.CreateOrUpdate with a
// no-op mutate func.
func CreateOrPatch(ctx context.Context, c client.Client, obj client.Object) error {
	_, err := controllerutil.CreateOrUpdate(ctx, c, obj, func() error { return nil })
	if err != nil {
		return fmt.Errorf(
			"createOrPatch failed for %s/%s: %w",
			obj.GetNamespace(),
			obj.GetName(),
			err,
		)
	}
	return nil
}

//
// -----------------------------------------------------------------------------
// Git helpers
// -----------------------------------------------------------------------------
//

// RunGit runs a git command in dir (or the current directory if dir is
// empty) and returns its combined stdout/stderr output, trimmed.
func RunGit(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

//
// -----------------------------------------------------------------------------
// Image helpers
// -----------------------------------------------------------------------------
//

// ParseImage splits an image reference on its last colon into repository
// and tag, defaulting to tag "latest" when no colon is present.
func ParseImage(image string) (string, string) {
	for i := len(image) - 1; i >= 0; i-- {
		if image[i] == ':' {
			return image[:i], image[i+1:]
		}
	}
	return image, "latest"
}

// PointerBool returns a pointer to b.
func PointerBool(b bool) *bool {
	return &b
}

//
// -----------------------------------------------------------------------------
// Debug helpers
// -----------------------------------------------------------------------------
//

// PrintRepoTree prints an indented directory tree of basePath to stdout,
// for debugging what a cloned repository actually contains.
func PrintRepoTree(basePath string) error {
	fmt.Printf("\n[bootstrap] Repository tree: %s\n", basePath)
	return filepath.Walk(basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, _ := filepath.Rel(basePath, path)
		if relPath == "." {
			fmt.Println(".")
			return nil
		}

		prefix := ""
		depth := len(filepath.SplitList(relPath))
		for i := 0; i < depth-1; i++ {
			prefix += "│   "
		}
		if info.IsDir() {
			fmt.Printf("%s├── %s/\n", prefix, info.Name())
		} else {
			fmt.Printf("%s├── %s\n", prefix, info.Name())
		}
		return nil
	})
}

// ShortHash truncates hash to its first 12 characters, or returns it
// unchanged if it's already shorter.
func ShortHash(hash string) string {
	if len(hash) > 12 {
		return hash[:12]
	}
	return hash
}

// RunGitWithEnv runs a git command with additional environment variables.
func RunGitWithEnv(dir string, env []string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	cmd.Env = append(os.Environ(), env...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

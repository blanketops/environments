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

package runtime

import "os"

type Context struct {
	Environment string
	ClusterName string

	// ---- Secrets (names) ----
	GitSSHSecretName         string
	ContainerRegistrySecret  string
	CrossplaneProviderSecret string
	WebhookURLSecret         string

	// ---- Explicit URLs ----
	GitHubWebhookURL string

	Extra map[string]string
}

func FromEnv() *Context {
	return &Context{
		Environment: getEnv("BLANKETOPS_ENV", "dev"),
		ClusterName: getEnv("BLANKETOPS_CLUSTER", "local"),

		GitSSHSecretName:         getEnv("BLANKETOPS_GIT_SSH_SECRET", "git-ssh-credentials"),
		ContainerRegistrySecret:  getEnv("BLANKETOPS_REGISTRY_SECRET", "registry-credentials"),
		CrossplaneProviderSecret: getEnv("BLANKETOPS_CROSSPLANE_SECRET", "default"),
		WebhookURLSecret:         getEnv("BLANKETOPS_WEBHOOK_SECRET", "hookurl"),

		// 🔒 locked-in webhook ingress
		GitHubWebhookURL: getEnv("BLANKETOPS_GITHUB_WEBHOOK_URL", ""),

		Extra: map[string]string{},
	}
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

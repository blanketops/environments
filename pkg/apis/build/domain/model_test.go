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

package domain

import (
	"encoding/json"
	"testing"
)

func TestImageWithDigest(t *testing.T) {
	const digest = "sha256:c02393e2773174f790cbf0262a8954a2c9aa19812883d8ea7c5924fecccf4509"

	tests := []struct {
		name   string
		image  string
		digest string
		want   string
	}{
		{name: "tagged image", image: "ghcr.io/acme/app:1111111", digest: digest, want: "ghcr.io/acme/app:1111111@" + digest},
		{name: "untagged image", image: "ghcr.io/acme/app", digest: digest, want: "ghcr.io/acme/app@" + digest},
		{name: "registry with port", image: "localhost:5000/app:v1", digest: digest, want: "localhost:5000/app:v1@" + digest},
		{name: "existing digest is replaced", image: "ghcr.io/acme/app:v1@sha256:old", digest: digest, want: "ghcr.io/acme/app:v1@" + digest},
		{name: "no digest keeps the image", image: "ghcr.io/acme/app:v1", want: "ghcr.io/acme/app:v1"},
		{name: "no digest drops nothing but a stale digest", image: "ghcr.io/acme/app:v1@sha256:old", want: "ghcr.io/acme/app:v1"},
		{name: "no image", digest: digest, want: ""},
		{name: "nothing", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ImageWithDigest(tt.image, tt.digest); got != tt.want {
				t.Errorf("ImageWithDigest(%q, %q) = %q, want %q", tt.image, tt.digest, got, tt.want)
			}
		})
	}
}

// TestBuildStatus_ImageRoundTrips locks in the serialised key: status.contract
// is written and read back as JSON by separate reconcilers.
func TestBuildStatus_ImageRoundTrips(t *testing.T) {
	in := BuildStatus{Triggered: true, Success: true, ExecutionRef: "run-1", Image: "ghcr.io/acme/app:v1@sha256:abc"}

	raw, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var asMap map[string]any
	if err := json.Unmarshal(raw, &asMap); err != nil {
		t.Fatalf("Unmarshal into map: %v", err)
	}
	if asMap["Image"] != in.Image {
		t.Errorf("serialised Image = %v, want %q", asMap["Image"], in.Image)
	}

	var out BuildStatus
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if out != in {
		t.Errorf("round trip = %+v, want %+v", out, in)
	}

	// A contract written before the field existed still decodes.
	var legacy BuildStatus
	if err := json.Unmarshal([]byte(`{"Triggered":true,"Success":false,"ExecutionRef":"run-0"}`), &legacy); err != nil {
		t.Fatalf("Unmarshal legacy: %v", err)
	}
	if legacy.Image != "" || legacy.ExecutionRef != "run-0" {
		t.Errorf("legacy decode = %+v", legacy)
	}
}

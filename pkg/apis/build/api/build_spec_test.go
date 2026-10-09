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

package api

import (
	"context"
	"errors"
	"testing"

	"github.com/go-logr/logr"
	shipwrightv1alpha1 "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	buildv1 "github.com/blanketops/environments-api/api/environments/v1alpha1"
	"github.com/blanketops/environments/pkg/apis/build/domain"
	buildResolution "github.com/blanketops/environments/resolution/build/resolve"
)

type createBuildSpecFunc func(domain.BuildSpec, *buildResolution.ResolvedBuild) (*shipwrightv1alpha1.Build, error)

// buildSpecProviders lists every provider's CreateBuildSpec so the trigger
// handling is asserted identically across backends.
func buildSpecProviders() map[string]createBuildSpecFunc {
	log := logr.Discard()
	return map[string]createBuildSpecFunc{
		"buildah":    NewBuildahProvider(nil, nil, log, nil).CreateBuildSpec,
		"kaniko":     NewKanikoProvider(nil, nil, log, nil).CreateBuildSpec,
		"buildpacks": NewBuildpacksProvider(nil, nil, log, nil).CreateBuildSpec,
	}
}

func newResolvedBuild(annotations map[string]string) *buildResolution.ResolvedBuild {
	return &buildResolution.ResolvedBuild{
		Build: &buildv1.Build{
			ObjectMeta: metav1.ObjectMeta{Name: "app", Namespace: "default", Annotations: annotations},
		},
	}
}

func newDomainSpec(image string) domain.BuildSpec {
	return domain.BuildSpec{
		SourceURL:    "https://github.com/acme/app.git",
		Revision:     "main",
		StrategyName: "strategy",
		StrategyKind: "ClusterBuildStrategy",
		Image:        image,
	}
}

func TestCreateBuildSpec_TriggerSHA(t *testing.T) {
	const sha = "1111111aaaaaaa"

	tests := []struct {
		name         string
		annotations  map[string]string
		image        string
		wantRevision string
		wantImage    string
	}{
		{
			name:         "no annotations keeps spec values",
			image:        "ghcr.io/acme/app:main",
			wantRevision: "main",
			wantImage:    "ghcr.io/acme/app:main",
		},
		{
			name:         "empty sha keeps spec values",
			annotations:  map[string]string{triggerSHAAnnotation: ""},
			image:        "ghcr.io/acme/app:main",
			wantRevision: "main",
			wantImage:    "ghcr.io/acme/app:main",
		},
		{
			name:         "sha replaces revision and tag",
			annotations:  map[string]string{triggerSHAAnnotation: sha},
			image:        "ghcr.io/acme/app:main",
			wantRevision: sha,
			wantImage:    "ghcr.io/acme/app:" + sha,
		},
		{
			name:         "sha is appended to an untagged image",
			annotations:  map[string]string{triggerSHAAnnotation: sha},
			image:        "ghcr.io/acme/app",
			wantRevision: sha,
			wantImage:    "ghcr.io/acme/app:" + sha,
		},
		{
			name:         "registry port is not mistaken for a tag",
			annotations:  map[string]string{triggerSHAAnnotation: sha},
			image:        "localhost:5000/app",
			wantRevision: sha,
			wantImage:    "localhost:5000/app:" + sha,
		},
	}

	for provider, create := range buildSpecProviders() {
		for _, tt := range tests {
			t.Run(provider+"/"+tt.name, func(t *testing.T) {
				got, err := create(newDomainSpec(tt.image), newResolvedBuild(tt.annotations))
				if err != nil {
					t.Fatalf("CreateBuildSpec: %v", err)
				}
				if got.Spec.Source.Revision == nil || *got.Spec.Source.Revision != tt.wantRevision {
					t.Errorf("revision = %v, want %q", got.Spec.Source.Revision, tt.wantRevision)
				}
				if got.Spec.Output.Image != tt.wantImage {
					t.Errorf("image = %q, want %q", got.Spec.Output.Image, tt.wantImage)
				}
			})
		}
	}
}

func TestCreateBuildSpec_RequiresSourceAndImage(t *testing.T) {
	for provider, create := range buildSpecProviders() {
		t.Run(provider, func(t *testing.T) {
			spec := newDomainSpec("ghcr.io/acme/app:main")
			spec.SourceURL = ""
			if _, err := create(spec, newResolvedBuild(nil)); err == nil {
				t.Error("expected an error for an empty source URL")
			}
			if _, err := create(newDomainSpec(""), newResolvedBuild(nil)); err == nil {
				t.Error("expected an error for an empty image")
			}
		})
	}
}

// -----------------------------------------------------------------------------
// Retries
//
// A retry is requested by changing the build.blanketops.dev/retry-attempt
// annotation on the Build. The annotation is part of the execution hash, so
// each attempt gets its own BuildRun while a repeated reconcile of the same
// attempt reuses the existing one.
// -----------------------------------------------------------------------------

const retryAttemptAnnotation = "build.blanketops.dev/retry-attempt"

func newRunScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := buildv1.AddToScheme(scheme); err != nil {
		t.Fatalf("AddToScheme(build): %v", err)
	}
	if err := shipwrightv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("AddToScheme(shipwright): %v", err)
	}
	return scheme
}

// runProviders lists every Provider against the same client so retry
// handling is asserted identically across backends.
func runProviders(c client.Client, scheme *runtime.Scheme) map[string]Provider {
	log := logr.Discard()
	return map[string]Provider{
		"buildah":    NewBuildahProvider(c, scheme, log, nil),
		"kaniko":     NewKanikoProvider(c, scheme, log, nil),
		"buildpacks": NewBuildpacksProvider(c, scheme, log, nil),
	}
}

// newRetryBuild returns a resolved Build with a retry policy, as the
// resolver would produce it, and no annotations.
func newRetryBuild() *buildResolution.ResolvedBuild {
	return &buildResolution.ResolvedBuild{
		Build: &buildv1.Build{
			ObjectMeta: metav1.ObjectMeta{Name: "app", Namespace: "default", UID: "uid-app"},
		},
		Spec: &buildResolution.ResolvedBuildSpec{
			Image:    "ghcr.io/acme/app:main",
			Source:   buildResolution.ResolvedSource{URL: "https://github.com/acme/app.git", Revision: "main"},
			Strategy: buildResolution.ResolvedStrategy{Name: "strategy", StrategyKind: "ClusterBuildStrategy"},
			Policy: &buildResolution.ResolvedBuildPolicy{
				Retry: &buildResolution.ResolvedRetryPolicy{OnFailure: true, MaxAttempts: 3},
			},
		},
	}
}

func setAnnotation(rb *buildResolution.ResolvedBuild, key, value string) {
	if rb.Build.Annotations == nil {
		rb.Build.Annotations = map[string]string{}
	}
	rb.Build.Annotations[key] = value
}

// listRuns returns the BuildRuns carrying the Build's name label, which is
// the selector the retry accounting counts attempts with.
func listRuns(t *testing.T, c client.Client, rb *buildResolution.ResolvedBuild) []shipwrightv1alpha1.BuildRun {
	t.Helper()
	var runs shipwrightv1alpha1.BuildRunList
	if err := c.List(context.Background(), &runs,
		client.InNamespace(rb.Build.Namespace),
		client.MatchingLabels{"build.blanketops.dev/name": rb.Build.Name},
	); err != nil {
		t.Fatalf("list buildruns: %v", err)
	}
	return runs.Items
}

func mustRun(t *testing.T, p Provider, rb *buildResolution.ResolvedBuild) domain.BuildResult {
	t.Helper()
	res, err := p.Run(context.Background(), rb, newDomainSpec(rb.Spec.Image))
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !res.Triggered {
		t.Fatalf("Run: Triggered = false, want true")
	}
	return res
}

func TestExtractTriggerContext_RetryAttempt(t *testing.T) {
	tests := []struct {
		name        string
		annotations map[string]string
		want        domain.TriggerContext
	}{
		{
			name: "no annotations is a manual trigger without a retry",
			want: domain.TriggerContext{Type: "manual"},
		},
		{
			name:        "retry attempt alone keeps the manual default",
			annotations: map[string]string{retryAttemptAnnotation: "2"},
			want:        domain.TriggerContext{Type: "manual", RetryAttempt: "2"},
		},
		{
			name:        "empty retry attempt is no retry",
			annotations: map[string]string{retryAttemptAnnotation: ""},
			want:        domain.TriggerContext{Type: "manual"},
		},
		{
			name: "retry attempt is carried with the trigger metadata",
			annotations: map[string]string{
				triggerTypeAnnotation:  "push",
				triggerRefAnnotation:   "refs/heads/main",
				triggerSHAAnnotation:   "1111111aaaaaaa",
				retryAttemptAnnotation: "3",
			},
			want: domain.TriggerContext{Type: "push", Ref: "refs/heads/main", SHA: "1111111aaaaaaa", RetryAttempt: "3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractTriggerContext(newResolvedBuild(tt.annotations).Build)
			if got != tt.want {
				t.Errorf("ExtractTriggerContext() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestRun_RetryAttemptCreatesNewBuildRun(t *testing.T) {
	scheme := newRunScheme(t)
	for name := range runProviders(nil, scheme) {
		t.Run(name, func(t *testing.T) {
			c := fake.NewClientBuilder().WithScheme(scheme).Build()
			p := runProviders(c, scheme)[name]
			rb := newRetryBuild()

			first := mustRun(t, p, rb)
			if got := len(listRuns(t, c, rb)); got != 1 {
				t.Fatalf("after first run: %d BuildRuns, want 1", got)
			}

			// Each bumped attempt is a new execution identity.
			refs := map[string]bool{first.ExecutionRef: true}
			for i, attempt := range []string{"2", "3"} {
				setAnnotation(rb, retryAttemptAnnotation, attempt)
				res := mustRun(t, p, rb)
				if refs[res.ExecutionRef] {
					t.Fatalf("attempt %s reused BuildRun %q", attempt, res.ExecutionRef)
				}
				refs[res.ExecutionRef] = true
				if got, want := len(listRuns(t, c, rb)), i+2; got != want {
					t.Fatalf("after attempt %s: %d BuildRuns, want %d", attempt, got, want)
				}
			}
		})
	}
}

func TestRun_SameRetryAttemptIsIdempotent(t *testing.T) {
	scheme := newRunScheme(t)
	for name := range runProviders(nil, scheme) {
		t.Run(name, func(t *testing.T) {
			c := fake.NewClientBuilder().WithScheme(scheme).Build()
			p := runProviders(c, scheme)[name]
			rb := newRetryBuild()
			setAnnotation(rb, retryAttemptAnnotation, "2")

			first := mustRun(t, p, rb)
			second := mustRun(t, p, rb)

			if first.ExecutionRef != second.ExecutionRef {
				t.Errorf("ExecutionRef changed across identical runs: %q then %q", first.ExecutionRef, second.ExecutionRef)
			}
			if got := len(listRuns(t, c, rb)); got != 1 {
				t.Errorf("%d BuildRuns, want 1", got)
			}
		})
	}
}

func TestRun_RetryBuildRunIdentity(t *testing.T) {
	scheme := newRunScheme(t)
	for name := range runProviders(nil, scheme) {
		t.Run(name, func(t *testing.T) {
			c := fake.NewClientBuilder().WithScheme(scheme).Build()
			p := runProviders(c, scheme)[name]
			rb := newRetryBuild()
			setAnnotation(rb, retryAttemptAnnotation, "2")

			res := mustRun(t, p, rb)

			runs := listRuns(t, c, rb)
			if len(runs) != 1 {
				t.Fatalf("%d BuildRuns, want 1", len(runs))
			}
			run := runs[0]
			if run.Name != res.ExecutionRef {
				t.Errorf("BuildRun name = %q, want ExecutionRef %q", run.Name, res.ExecutionRef)
			}
			hash := run.Annotations["build.blanketops.dev/hash"]
			if hash == "" {
				t.Fatal("BuildRun has no build.blanketops.dev/hash annotation")
			}
			short := run.Labels["build-hash"]
			if short == "" || hash[:len(short)] != short {
				t.Errorf("build-hash label %q is not a prefix of hash %q", short, hash)
			}
			if want := rb.Build.Name + "-" + short; run.Name != want {
				t.Errorf("BuildRun name = %q, want %q", run.Name, want)
			}
			if run.Labels["shipwright-build"] != rb.Build.Name {
				t.Errorf("shipwright-build label = %q, want %q", run.Labels["shipwright-build"], rb.Build.Name)
			}
			if run.Spec.BuildRef == nil || run.Spec.BuildRef.Name != rb.Build.Name {
				t.Errorf("BuildRef = %+v, want name %q", run.Spec.BuildRef, rb.Build.Name)
			}
			if len(run.OwnerReferences) != 1 || run.OwnerReferences[0].UID != rb.Build.UID {
				t.Errorf("owner references = %+v, want the Build CR", run.OwnerReferences)
			}
		})
	}
}

func TestRun_RetryKeepsTriggeredCommit(t *testing.T) {
	const sha = "1111111aaaaaaa"

	scheme := newRunScheme(t)
	for name := range runProviders(nil, scheme) {
		t.Run(name, func(t *testing.T) {
			c := fake.NewClientBuilder().WithScheme(scheme).Build()
			p := runProviders(c, scheme)[name]
			rb := newRetryBuild()
			setAnnotation(rb, triggerTypeAnnotation, "push")
			setAnnotation(rb, triggerSHAAnnotation, sha)

			first := mustRun(t, p, rb)
			setAnnotation(rb, retryAttemptAnnotation, "2")
			retry := mustRun(t, p, rb)

			if first.ExecutionRef == retry.ExecutionRef {
				t.Fatalf("retry reused BuildRun %q", retry.ExecutionRef)
			}

			var ship shipwrightv1alpha1.Build
			if err := c.Get(context.Background(), client.ObjectKeyFromObject(rb.Build), &ship); err != nil {
				t.Fatalf("get shipwright build: %v", err)
			}
			if ship.Spec.Source.Revision == nil || *ship.Spec.Source.Revision != sha {
				t.Errorf("revision after retry = %v, want %q", ship.Spec.Source.Revision, sha)
			}
			if want := "ghcr.io/acme/app:" + sha; ship.Spec.Output.Image != want {
				t.Errorf("image after retry = %q, want %q", ship.Spec.Output.Image, want)
			}
		})
	}
}

func TestRun_NewCommitAfterRetriesStartsNewBuildRun(t *testing.T) {
	scheme := newRunScheme(t)
	for name := range runProviders(nil, scheme) {
		t.Run(name, func(t *testing.T) {
			c := fake.NewClientBuilder().WithScheme(scheme).Build()
			p := runProviders(c, scheme)[name]
			rb := newRetryBuild()
			setAnnotation(rb, triggerSHAAnnotation, "1111111aaaaaaa")
			setAnnotation(rb, retryAttemptAnnotation, "2")
			retried := mustRun(t, p, rb)

			// A new push arrives while the retry annotation is still set.
			setAnnotation(rb, triggerSHAAnnotation, "2222222bbbbbbb")
			next := mustRun(t, p, rb)

			if retried.ExecutionRef == next.ExecutionRef {
				t.Errorf("new commit reused BuildRun %q", next.ExecutionRef)
			}
			if got := len(listRuns(t, c, rb)); got != 2 {
				t.Errorf("%d BuildRuns, want 2", got)
			}
		})
	}
}

func TestRun_RetryCreateFailureIsNotTriggered(t *testing.T) {
	errCreate := errors.New("simulated apiserver error")

	scheme := newRunScheme(t)
	for name := range runProviders(nil, scheme) {
		t.Run(name, func(t *testing.T) {
			failing := false
			c := fake.NewClientBuilder().WithScheme(scheme).WithInterceptorFuncs(interceptor.Funcs{
				Create: func(ctx context.Context, cl client.WithWatch, obj client.Object, opts ...client.CreateOption) error {
					if _, ok := obj.(*shipwrightv1alpha1.BuildRun); ok && failing {
						return errCreate
					}
					return cl.Create(ctx, obj, opts...)
				},
			}).Build()
			p := runProviders(c, scheme)[name]
			rb := newRetryBuild()
			mustRun(t, p, rb)

			setAnnotation(rb, retryAttemptAnnotation, "2")
			failing = true
			res, err := p.Run(context.Background(), rb, newDomainSpec(rb.Spec.Image))
			if !errors.Is(err, errCreate) {
				t.Fatalf("Run error = %v, want %v", err, errCreate)
			}
			if res.Triggered || res.ExecutionRef != "" {
				t.Errorf("failed retry reported Triggered=%v ExecutionRef=%q", res.Triggered, res.ExecutionRef)
			}
			if got := len(listRuns(t, c, rb)); got != 1 {
				t.Fatalf("%d BuildRuns after failed retry, want 1", got)
			}

			// The same attempt succeeds once the API server recovers.
			failing = false
			mustRun(t, p, rb)
			if got := len(listRuns(t, c, rb)); got != 2 {
				t.Errorf("%d BuildRuns after recovery, want 2", got)
			}
		})
	}
}

func TestTeardown_RemovesEveryRetryBuildRun(t *testing.T) {
	scheme := newRunScheme(t)
	for name := range runProviders(nil, scheme) {
		t.Run(name, func(t *testing.T) {
			c := fake.NewClientBuilder().WithScheme(scheme).Build()
			p := runProviders(c, scheme)[name]
			rb := newRetryBuild()
			mustRun(t, p, rb)
			for _, attempt := range []string{"2", "3"} {
				setAnnotation(rb, retryAttemptAnnotation, attempt)
				mustRun(t, p, rb)
			}
			if got := len(listRuns(t, c, rb)); got != 3 {
				t.Fatalf("%d BuildRuns before teardown, want 3", got)
			}

			// Teardown is idempotent: the second call finds nothing to delete.
			for i := 0; i < 2; i++ {
				if err := p.Teardown(context.Background(), rb); err != nil {
					t.Fatalf("Teardown #%d: %v", i+1, err)
				}
			}

			if got := len(listRuns(t, c, rb)); got != 0 {
				t.Errorf("%d BuildRuns after teardown, want 0", got)
			}
			var ship shipwrightv1alpha1.Build
			err := c.Get(context.Background(), client.ObjectKeyFromObject(rb.Build), &ship)
			if client.IgnoreNotFound(err) != nil || err == nil {
				t.Errorf("shipwright Build after teardown: err = %v, want not found", err)
			}
		})
	}
}

// -----------------------------------------------------------------------------
// Declared values only
//
// A Shipwright object carries what the Build contract declares. An undeclared
// secret or service account is left off, not written with an empty name.
// -----------------------------------------------------------------------------

func TestCreateBuildSpec_CredentialsOnlyWhenDeclared(t *testing.T) {
	tests := []struct {
		name         string
		cloneSecret  string
		pushSecret   string
		wantClone    string
		wantPush     string
		wantNilClone bool
		wantNilPush  bool
	}{
		{name: "both declared", cloneSecret: "git-creds", pushSecret: "registry-creds", wantClone: "git-creds", wantPush: "registry-creds"},
		{name: "public source, private registry", pushSecret: "registry-creds", wantNilClone: true, wantPush: "registry-creds"},
		{name: "private source, no push secret", cloneSecret: "git-creds", wantClone: "git-creds", wantNilPush: true},
		{name: "neither declared", wantNilClone: true, wantNilPush: true},
	}

	for provider, create := range buildSpecProviders() {
		for _, tt := range tests {
			t.Run(provider+"/"+tt.name, func(t *testing.T) {
				spec := newDomainSpec("ghcr.io/acme/app:main")
				spec.CloneSecret = tt.cloneSecret
				spec.ServiceAccountSecret = tt.pushSecret

				got, err := create(spec, newResolvedBuild(nil))
				if err != nil {
					t.Fatalf("CreateBuildSpec: %v", err)
				}

				clone := got.Spec.Source.Credentials
				if tt.wantNilClone != (clone == nil) || (clone != nil && clone.Name != tt.wantClone) {
					t.Errorf("source credentials = %+v, want nil=%v name=%q", clone, tt.wantNilClone, tt.wantClone)
				}
				push := got.Spec.Output.Credentials
				if tt.wantNilPush != (push == nil) || (push != nil && push.Name != tt.wantPush) {
					t.Errorf("output credentials = %+v, want nil=%v name=%q", push, tt.wantNilPush, tt.wantPush)
				}
			})
		}
	}
}

func TestRun_BuildRunUsesTheDeclaredServiceAccount(t *testing.T) {
	tests := []struct {
		name           string
		serviceAccount *buildResolution.ResolvedServiceAccount
		want           string
	}{
		{name: "declared", serviceAccount: &buildResolution.ResolvedServiceAccount{Name: "build-bot", Secret: "registry-creds"}, want: "build-bot"},
		{name: "declared without a name", serviceAccount: &buildResolution.ResolvedServiceAccount{Secret: "registry-creds"}},
		{name: "not declared"},
	}

	scheme := newRunScheme(t)
	for provider := range runProviders(nil, scheme) {
		for _, tt := range tests {
			t.Run(provider+"/"+tt.name, func(t *testing.T) {
				c := fake.NewClientBuilder().WithScheme(scheme).Build()
				p := runProviders(c, scheme)[provider]
				rb := newRetryBuild()
				rb.Spec.ServiceAccount = tt.serviceAccount

				mustRun(t, p, rb)

				runs := listRuns(t, c, rb)
				if len(runs) != 1 {
					t.Fatalf("%d BuildRuns, want 1", len(runs))
				}
				sa := runs[0].Spec.ServiceAccount
				switch {
				case tt.want == "" && sa != nil:
					t.Errorf("service account = %+v, want none", sa)
				case tt.want != "" && (sa == nil || sa.Name == nil || *sa.Name != tt.want):
					t.Errorf("service account = %+v, want %q", sa, tt.want)
				case sa != nil && sa.Generate != nil:
					t.Errorf("service account generation was requested: %+v", sa)
				}
			})
		}
	}
}

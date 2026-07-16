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

package cache

import (
	"context"
	"errors"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	ctrlcache "sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

func TestNoopExternalCache(t *testing.T) {
	var c NoopExternalCache

	if err := c.Set(context.Background(), "k", "v", time.Second); err != nil {
		t.Fatalf("Set: expected nil error, got %v", err)
	}

	found, err := c.Get(context.Background(), "k", nil)
	if found || err != nil {
		t.Fatalf("Get: expected (false, nil), got (%v, %v)", found, err)
	}

	if err := c.Del(context.Background(), "k"); err != nil {
		t.Fatalf("Del: expected nil error, got %v", err)
	}

	if err := c.DelPrefix(context.Background(), "prefix:"); err != nil {
		t.Fatalf("DelPrefix: expected nil error, got %v", err)
	}
}

// fakeInformerCache satisfies ctrlcache.Cache by embedding a nil interface —
// every method not explicitly overridden panics if called. Only
// WaitForCacheSync needs real behavior for these tests.
type fakeInformerCache struct {
	ctrlcache.Cache
	syncResult bool
}

func (f fakeInformerCache) WaitForCacheSync(ctx context.Context) bool { return f.syncResult }

// fakeManager satisfies manager.Manager by embedding a nil interface value —
// every method not explicitly overridden panics if called. NewCache only
// calls GetCache and GetFieldIndexer, so only those need real behavior.
type fakeManager struct {
	manager.Manager
	cache   fakeInformerCache
	indexer client.FieldIndexer
}

func (f *fakeManager) GetCache() ctrlcache.Cache            { return f.cache }
func (f *fakeManager) GetFieldIndexer() client.FieldIndexer { return f.indexer }

func TestNewCache_NilExternalFallsBackToNoop(t *testing.T) {
	mgr := &fakeManager{}
	c := NewCache(mgr, nil)

	if _, ok := c.External.(NoopExternalCache); !ok {
		t.Fatalf("expected nil external to be replaced with NoopExternalCache, got %T", c.External)
	}
}

func TestNewCache_PreservesProvidedExternal(t *testing.T) {
	mgr := &fakeManager{}
	external := &fakeExternalCache{}
	c := NewCache(mgr, external)

	if c.External != external {
		t.Fatalf("expected provided external cache to be preserved, got %T", c.External)
	}
}

type fakeExternalCache struct{}

func (*fakeExternalCache) Set(context.Context, string, any, time.Duration) error { return nil }
func (*fakeExternalCache) Get(context.Context, string, any) (bool, error)        { return false, nil }
func (*fakeExternalCache) Del(context.Context, string) error                     { return nil }
func (*fakeExternalCache) DelPrefix(context.Context, string) error               { return nil }

// fakeFieldIndexer captures the arguments passed to IndexField.
type fakeFieldIndexer struct {
	called bool
	field  string
}

func (f *fakeFieldIndexer) IndexField(ctx context.Context, obj client.Object, field string, extractValue client.IndexerFunc) error {
	f.called = true
	f.field = field
	return nil
}

func TestCache_IndexField_DelegatesToIndexer(t *testing.T) {
	indexer := &fakeFieldIndexer{}
	c := &Cache{Indexer: indexer}

	err := c.IndexField(context.Background(), &corev1.Pod{}, ".spec.name", func(client.Object) []string { return nil })
	if err != nil {
		t.Fatalf("IndexField: %v", err)
	}
	if !indexer.called {
		t.Fatal("expected underlying Indexer.IndexField to be called")
	}
	if indexer.field != ".spec.name" {
		t.Fatalf("expected field %q, got %q", ".spec.name", indexer.field)
	}
}

// fakeReader captures the ListOptions passed to List.
type fakeReader struct {
	gotOpts []client.ListOption
	err     error
}

func (f *fakeReader) Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
	return errors.New("not implemented")
}

func (f *fakeReader) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	f.gotOpts = opts
	return f.err
}

func TestListByField_DelegatesToReaderWithMatchingFields(t *testing.T) {
	r := &fakeReader{}
	var list corev1.PodList

	if err := ListByField(context.Background(), r, &list, ".spec.repo", "github.com/foo/bar"); err != nil {
		t.Fatalf("ListByField: %v", err)
	}

	if len(r.gotOpts) != 1 {
		t.Fatalf("expected exactly one ListOption, got %d", len(r.gotOpts))
	}
	mf, ok := r.gotOpts[0].(client.MatchingFields)
	if !ok {
		t.Fatalf("expected client.MatchingFields, got %T", r.gotOpts[0])
	}
	if mf[".spec.repo"] != "github.com/foo/bar" {
		t.Fatalf("unexpected MatchingFields: %+v", mf)
	}
}

func TestListByField_PropagatesReaderError(t *testing.T) {
	wantErr := errors.New("list failed")
	r := &fakeReader{err: wantErr}
	var list corev1.PodList

	if err := ListByField(context.Background(), r, &list, "field", "value"); !errors.Is(err, wantErr) {
		t.Fatalf("expected error to propagate, got %v", err)
	}
}

func TestWaitForSync(t *testing.T) {
	if got := WaitForSync(context.Background(), fakeInformerCache{syncResult: true}); !got {
		t.Fatal("expected WaitForSync to return true when the underlying cache reports synced")
	}
	if got := WaitForSync(context.Background(), fakeInformerCache{syncResult: false}); got {
		t.Fatal("expected WaitForSync to return false when the underlying cache reports not synced")
	}
}

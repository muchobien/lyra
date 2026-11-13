package e2e

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/random"
	"github.com/google/go-containerregistry/pkg/v1/remote"

	"github.com/muchobien/lyra/internal/store"
)

func TestRegistry(test *testing.T) {
	srv := startServer(test, "127.0.0.1")
	token := adminToken(test, srv)
	reader := createToken(test, srv, `{"name":"reader","scope":"read"}`)
	otherWriter := createToken(test, srv, `{"name":"other","scope":"write","repos":["other"]}`)
	createRepo(test, srv, token, "demo")
	createRepo(test, srv, token, "other")
	host := srv.address
	as := func(secret string) []crane.Option {
		return []crane.Option{crane.Insecure, crane.WithAuth(&authn.Basic{Username: "lyra", Password: secret})}
	}
	images := func() map[string]store.Image {
		test.Helper()
		var list []store.Image
		lyraJSON(test, srv, token, &list, "image", "list", "demo")
		found := map[string]store.Image{}
		for _, image := range list {
			found[image.Name+":"+image.Tag] = image
		}
		return found
	}
	expectError := func(err error, want string) {
		test.Helper()
		if err == nil || !strings.Contains(err.Error(), want) {
			test.Fatalf("expected an error with %q, got %v", want, err)
		}
	}

	first, err := random.Image(2048, 2)
	if err != nil {
		test.Fatal(err)
	}
	firstDigest, _ := first.Digest()
	if err := crane.Push(first, host+"/demo:v1", as(token)...); err != nil {
		test.Fatalf("push demo:v1: %v", err)
	}
	pulled, err := crane.Pull(host+"/demo:v1", as(reader)...)
	if err != nil {
		test.Fatalf("pull demo:v1 as reader: %v", err)
	}
	if digest, _ := pulled.Digest(); digest != firstDigest {
		test.Fatalf("pulled %s, pushed %s", digest, firstDigest)
	}
	if _, err := crane.Pull(host+"/demo:v1", crane.Insecure); err == nil {
		test.Fatal("anonymous pull succeeded")
	}
	expectError(crane.Push(first, host+"/demo:reader", as(reader)...), "DENIED")
	expectError(crane.Push(first, host+"/demo:other", as(otherWriter)...), "403")
	expectError(crane.Push(first, host+"/missing:v1", as(token)...), "NAME_UNKNOWN")
	if err := crane.Push(first, host+"/other/tool:v1", as(otherWriter)...); err != nil {
		test.Fatalf("push other/tool:v1 with a token for other: %v", err)
	}
	layers, _ := first.Layers()
	layerDigest, _ := layers[0].Digest()
	status, body := call(test, http.MethodPost, srv.url+"/v2/other/mounted/blobs/uploads/?mount="+layerDigest.String()+"&from=demo", bearer(otherWriter), "")
	if status != http.StatusAccepted {
		test.Fatalf("mount from demo with a token for other: HTTP %d, expected 202 (a plain upload instead of the mount): %s", status, body)
	}
	status, body = call(test, http.MethodPost, srv.url+"/v2/other/mounted/blobs/uploads/?mount="+layerDigest.String()+"&from=missing", bearer(otherWriter), "")
	if status != http.StatusAccepted {
		test.Fatalf("mount from a missing repository: HTTP %d: %s", status, body)
	}
	status, _ = call(test, http.MethodPost, srv.url+"/v2/demo/mounted/blobs/uploads/?mount="+layerDigest.String()+"&from=demo", bearer(token), "")
	if status != http.StatusCreated {
		test.Fatalf("mount from demo with access: HTTP %d, expected 201", status)
	}

	index, err := random.Index(1024, 1, 2)
	if err != nil {
		test.Fatal(err)
	}
	indexDigest, _ := index.Digest()
	reference, err := name.ParseReference(host+"/demo/runner:latest", name.Insecure)
	if err != nil {
		test.Fatal(err)
	}
	if err := remote.WriteIndex(reference, index, remote.WithAuth(&authn.Basic{Username: "lyra", Password: token})); err != nil {
		test.Fatalf("push index demo/runner:latest: %v", err)
	}

	second, _ := random.Image(1024, 1)
	secondDigest, _ := second.Digest()
	if err := crane.Push(second, host+"/demo:v1", as(token)...); err != nil {
		test.Fatalf("retag demo:v1: %v", err)
	}
	found := images()
	if len(found) != 2 || found["demo:v1"].Digest != secondDigest.String() || found["demo/runner:latest"].Digest != indexDigest.String() {
		test.Fatalf("images: %+v", found)
	}
	if found["demo:v1"].Reference != host+"/demo:v1" || found["demo:v1"].Pusher.Agent != "" || found["demo:v1"].Pusher.Via != "token:admin" {
		test.Fatalf("demo:v1 record: %+v", found["demo:v1"])
	}
	if !strings.Contains(found["demo/runner:latest"].MediaType, "index") {
		test.Fatalf("index media type: %s", found["demo/runner:latest"].MediaType)
	}

	status, data := call(test, http.MethodGet, srv.url+"/api/events?repo=demo&limit=100", bearer(token), "")
	if status != http.StatusOK {
		test.Fatalf("events: HTTP %d", status)
	}
	var events []store.Event
	json.Unmarshal(data, &events)
	pushes := 0
	for _, event := range events {
		if event.Action == "image.pushed" {
			pushes++
		}
	}
	if pushes != 3 {
		test.Fatalf("%d image.pushed events, expected 3 (demo:v1 twice, demo/runner:latest)", pushes)
	}

	if result := runLyra(test, srv, reader, "image", "delete", "demo:v1", "--yes"); result.code != 1 || !strings.Contains(result.stderr, "DENIED") {
		test.Fatalf("reader deletes: %+v", result)
	}
	if result := runLyra(test, srv, token, "image", "delete", "demo:v1", "--yes"); result.code != 0 {
		test.Fatalf("delete demo:v1: %+v", result)
	}
	expectError(func() error { _, err := crane.Pull(host+"/demo:v1", as(token)...); return err }(), "MANIFEST_UNKNOWN")
	if result := runLyra(test, srv, token, "image", "delete", "demo/runner@"+indexDigest.String(), "--yes"); result.code != 0 {
		test.Fatalf("delete demo/runner by digest: %+v", result)
	}
	if found := images(); len(found) != 0 {
		test.Fatalf("images after deletes: %+v", found)
	}
	if result := runLyra(test, srv, token, "image", "delete", "demo", "--yes"); result.code != 2 {
		test.Fatalf("delete without a tag: %+v", result)
	}

	catalog, err := crane.Catalog(host, as(token)...)
	if err != nil || !strings.Contains(strings.Join(catalog, ","), "other/tool") {
		test.Fatalf("catalog: %v, %v", catalog, err)
	}
	if _, err := crane.Catalog(host, as(otherWriter)...); err == nil {
		test.Fatal("a token for one repository listed the catalog")
	}
	if result := runLyra(test, srv, token, "repo", "create", "v2"); result.code != 1 {
		test.Fatalf("repo named v2: %+v", result)
	}
}

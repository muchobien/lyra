package e2e

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/v1/random"
)

func TestImagesInReleases(test *testing.T) {
	srv := startServer(test, "127.0.0.1")
	token := adminToken(test, srv)
	createRepo(test, srv, token, "acme/demo")
	host := srv.address
	auth := []crane.Option{crane.Insecure, crane.WithAuth(&authn.Basic{Username: "lyra", Password: token})}
	src := newSource(test)
	commitFile(test, src, "README.md", "# Demo\n", "First commit")
	mustGit(test, src, "-c", "http.extraHeader=Authorization: Bearer "+token, "push", "-q", srv.url+"/acme/demo.git", "main")

	image, err := random.Image(4096, 3)
	if err != nil {
		test.Fatal(err)
	}
	digest, _ := image.Digest()
	for _, reference := range []string{"acme/demo:v1.0.0", "acme/demo/runner:1.0.0", "acme/demo:v2.0.0"} {
		if err := crane.Push(image, host+"/"+reference, auth...); err != nil {
			test.Fatalf("push %s: %v", reference, err)
		}
	}

	release := releaseJSON(test, srv, token, "create", "acme/demo", "v1.0.0")
	references := []string{}
	for _, linked := range release.Images {
		references = append(references, linked.Reference)
	}
	if strings.Join(references, " ") != host+"/acme/demo:v1.0.0 "+host+"/acme/demo/runner:1.0.0" {
		test.Fatalf("images of release v1.0.0: %v", references)
	}

	status, page := call(test, http.MethodGet, srv.url+"/api/repos/acme/demo/releases/tags/v1.0.0", tailnet(), "")
	if status != http.StatusOK || !strings.Contains(compactJSON(test, page), `"reference":"`+host+`/acme/demo/runner:1.0.0"`) || strings.Contains(compactJSON(test, page), "v2.0.0") {
		test.Fatalf("release page: HTTP %d:\n%s", status, page)
	}
	status, page = call(test, http.MethodGet, srv.url+"/api/repos/acme/demo/images", tailnet(), "")
	for _, want := range []string{`"reference":"` + host + `/acme/demo:v2.0.0"`, `"reference":"` + host + `/acme/demo/runner:1.0.0"`, `"digest":"` + digest.String() + `"`} {
		if status != http.StatusOK || !strings.Contains(compactJSON(test, page), want) {
			test.Fatalf("images page lacks %q: HTTP %d:\n%s", want, status, page)
		}
	}

	if result := runLyra(test, srv, token, "repo", "delete", "acme/demo", "--yes"); result.code != 0 {
		test.Fatalf("repo delete: %+v", result)
	}
	createRepo(test, srv, token, "acme/demo")
	if _, err := crane.Pull(host+"/acme/demo:v1.0.0", auth...); err == nil || !strings.Contains(err.Error(), "MANIFEST_UNKNOWN") {
		test.Fatalf("pull from a recreated repo: %v", err)
	}
	if result := runLyra(test, srv, token, "image", "list", "acme/demo"); strings.TrimSpace(result.stdout) != "[]" {
		test.Fatalf("images of a recreated repo: %+v", result)
	}
	if err := crane.Push(image, host+"/acme/demo:again", auth...); err != nil {
		test.Fatalf("push to the recreated repo: %v", err)
	}
	pulled, err := crane.Pull(host+"/acme/demo:again", auth...)
	if err != nil {
		test.Fatal(err)
	}
	layers, err := pulled.Layers()
	if err != nil {
		test.Fatal(err)
	}
	for _, layer := range layers {
		reader, err := layer.Compressed()
		if err != nil {
			test.Fatalf("layer of the re-pushed image: %v", err)
		}
		_, err = io.Copy(io.Discard, reader)
		reader.Close()
		if err != nil {
			test.Fatalf("read a layer of the re-pushed image: %v", err)
		}
	}
}

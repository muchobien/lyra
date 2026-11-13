package e2e

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/muchobien/lyra/internal/store"
)

type releaseResult struct {
	store.Release
	EventID int64 `json:"event_id"`
}

func TestReleases(test *testing.T) {
	srv := startServer(test, "127.0.0.1")
	token := adminToken(test, srv)
	reader := createToken(test, srv, `{"name":"reader","scope":"read"}`)
	createRepo(test, srv, token, "acme/demo")
	auth := "http.extraHeader=Authorization: Bearer " + token
	remote := srv.url + "/acme/demo.git"
	src := newSource(test)
	push := func(args ...string) {
		test.Helper()
		mustGit(test, src, append([]string{"-c", auth, "push", "-q", remote}, args...)...)
	}
	remoteTag := func(tag string) string {
		test.Helper()
		output := mustGit(test, src, "-c", auth, "ls-remote", remote, "refs/tags/"+tag)
		sha, _, _ := strings.Cut(output, "\t")
		return sha
	}
	release := func(args ...string) releaseResult {
		test.Helper()
		var result releaseResult
		lyraJSON(test, srv, token, &result, append([]string{"release"}, args...)...)
		return result
	}
	objects := func() []string {
		test.Helper()
		found := []string{}
		root := filepath.Join(srv.data, "blobs", "lyra", "releases", "acme/demo")
		filepath.WalkDir(root, func(file string, entry fs.DirEntry, err error) error {
			if err == nil && !entry.IsDir() {
				found = append(found, file)
			}
			return nil
		})
		return found
	}
	download := func(target string, headers map[string]string) (int, []byte) {
		test.Helper()
		return call(test, http.MethodGet, srv.url+target, headers, "")
	}

	commitFile(test, src, "README.md", "# Demo\n", "First commit")
	first := mustGit(test, src, "rev-parse", "HEAD")
	commitFile(test, src, "README.md", "# Demo\n\nSecond.\n", "Second commit")
	head := mustGit(test, src, "rev-parse", "HEAD")
	push("main")

	files := test.TempDir()
	archive := make([]byte, 300_000)
	rand.Read(archive)
	archivePath := filepath.Join(files, "app.tar.gz")
	checksumsPath := filepath.Join(files, "checksums.txt")
	sum := sha256.Sum256(archive)
	if err := os.WriteFile(archivePath, archive, 0o644); err != nil {
		test.Fatal(err)
	}
	if err := os.WriteFile(checksumsPath, []byte(hex.EncodeToString(sum[:])+"  app.tar.gz\n"), 0o644); err != nil {
		test.Fatal(err)
	}

	draft := release("create", "acme/demo", "v1.0.0", "--draft", "--notes", "First release", "--asset", archivePath)
	if !draft.Draft || draft.Commit != "" || draft.Target != "main" || len(draft.Assets) != 1 || draft.Assets[0].SHA256 != hex.EncodeToString(sum[:]) {
		test.Fatalf("draft: %+v", draft)
	}
	if sha := remoteTag("v1.0.0"); sha != "" {
		test.Fatalf("a draft created tag v1.0.0 at %s", sha)
	}
	if result := runLyra(test, srv, reader, "release", "view", "acme/demo", "v1.0.0"); result.code != 1 || !strings.Contains(result.stderr, "no release for tag v1.0.0") {
		test.Fatalf("reader sees the draft: %+v", result)
	}
	if result := runLyra(test, srv, reader, "release", "list", "acme/demo"); strings.TrimSpace(result.stdout) != "[]" {
		test.Fatalf("reader lists drafts: %+v", result)
	}
	if status, _ := download("/acme/demo/releases/download/v1.0.0/app.tar.gz", bearer(reader)); status != http.StatusNotFound {
		test.Fatalf("reader downloads a draft asset: HTTP %d", status)
	}

	commitFile(test, src, "README.md", "# Demo\n\nThird.\n", "Third commit")
	push("main")
	published := release("publish", "acme/demo", "v1.0.0")
	third := mustGit(test, src, "rev-parse", "HEAD")
	if published.Draft || published.PublishedAt == nil || published.Commit != third || remoteTag("v1.0.0") != third {
		test.Fatalf("published at %s, tag at %s, expected %s: %+v", published.Commit, remoteTag("v1.0.0"), third, published)
	}

	status, data := call(test, http.MethodGet, srv.url+"/api/events?repo=acme/demo&limit=100", bearer(token), "")
	if status != http.StatusOK {
		test.Fatalf("events: HTTP %d: %s", status, data)
	}
	var events []store.Event
	if err := json.Unmarshal(data, &events); err != nil {
		test.Fatal(err)
	}
	actions := []string{}
	for _, event := range events {
		if strings.HasPrefix(event.Action, "release") {
			actions = append(actions, event.Action)
		}
		var payload struct {
			Assets []string `json:"assets"`
		}
		json.Unmarshal(event.Data, &payload)
		if event.Action == "release.published" && !slices.Equal(payload.Assets, []string{"app.tar.gz"}) {
			test.Fatalf("release.published without its asset: %s", event.Data)
		}
	}
	slices.Sort(actions)
	if !slices.Equal(actions, []string{"release.asset", "release.created", "release.published"}) {
		test.Fatalf("release events: %v", actions)
	}

	for _, target := range []string{"/acme/demo/releases/download/v1.0.0/app.tar.gz", "/acme/demo/releases/latest/download/app.tar.gz", published.Assets[0].DownloadURL[len(srv.url):]} {
		request, err := http.NewRequest(http.MethodGet, srv.url+target, nil)
		if err != nil {
			test.Fatal(err)
		}
		request.Header.Set("Authorization", "Bearer "+reader)
		response, err := http.DefaultClient.Do(request)
		if err != nil {
			test.Fatal(err)
		}
		var body bytes.Buffer
		body.ReadFrom(response.Body)
		response.Body.Close()
		if response.StatusCode != http.StatusOK || !bytes.Equal(body.Bytes(), archive) {
			test.Fatalf("%s: HTTP %d, %d bytes", target, response.StatusCode, body.Len())
		}
		if response.Header.Get("ETag") != `"`+hex.EncodeToString(sum[:])+`"` || response.Header.Get("Content-Disposition") != `attachment; filename=app.tar.gz` {
			test.Fatalf("%s headers: %v", target, response.Header)
		}
	}
	if status, _ := call(test, http.MethodGet, srv.url+"/acme/demo/releases/download/v1.0.0/app.tar.gz", map[string]string{"Authorization": "Bearer " + reader, "If-None-Match": `"` + hex.EncodeToString(sum[:]) + `"`}, ""); status != http.StatusNotModified {
		test.Fatalf("If-None-Match: HTTP %d", status)
	}
	for _, ranged := range []struct {
		header, contentRange string
		want                 []byte
	}{
		{"bytes=1000-1999", "bytes 1000-1999/300000", archive[1000:2000]},
		{"bytes=-10", "bytes 299990-299999/300000", archive[299990:]},
		{"bytes=299000-", "bytes 299000-299999/300000", archive[299000:]},
	} {
		request, _ := http.NewRequest(http.MethodGet, srv.url+"/acme/demo/releases/download/v1.0.0/app.tar.gz", nil)
		request.Header.Set("Authorization", "Bearer "+reader)
		request.Header.Set("Range", ranged.header)
		response, err := http.DefaultClient.Do(request)
		if err != nil {
			test.Fatal(err)
		}
		body, _ := io.ReadAll(response.Body)
		response.Body.Close()
		if response.StatusCode != http.StatusPartialContent || response.Header.Get("Content-Range") != ranged.contentRange || !bytes.Equal(body, ranged.want) {
			test.Fatalf("Range %s: HTTP %d, %s, %d bytes", ranged.header, response.StatusCode, response.Header.Get("Content-Range"), len(body))
		}
	}
	headRequest, _ := http.NewRequest(http.MethodHead, srv.url+"/acme/demo/releases/download/v1.0.0/app.tar.gz", nil)
	headRequest.Header.Set("Authorization", "Bearer "+reader)
	if response, err := http.DefaultClient.Do(headRequest); err != nil || response.StatusCode != http.StatusOK || response.ContentLength != 300000 || response.Header.Get("Accept-Ranges") != "bytes" {
		test.Fatalf("HEAD: %v %+v", err, response)
	}
	if status, _ := call(test, http.MethodPost, srv.url+"/api/repos/acme/demo/releases/"+fmt.Sprint(published.ID)+"/assets?name=x", bearer(reader), "x"); status != http.StatusForbidden {
		test.Fatalf("reader uploads: HTTP %d", status)
	}

	if result := runLyra(test, srv, token, "release", "upload", "acme/demo", "v1.0.0", archivePath); result.code != 1 || !strings.Contains(result.stderr, "already exists") {
		test.Fatalf("upload over an existing asset: %+v", result)
	}
	if count := len(objects()); count != 1 {
		test.Fatalf("a refused upload left %d objects", count)
	}
	if err := os.WriteFile(archivePath, []byte("replaced\n"), 0o644); err != nil {
		test.Fatal(err)
	}
	if result := runLyra(test, srv, token, "release", "upload", "acme/demo", "v1.0.0", archivePath, checksumsPath, "--clobber"); result.code != 0 {
		test.Fatalf("upload --clobber: %+v", result)
	}
	if status, data := download("/acme/demo/releases/download/v1.0.0/app.tar.gz", bearer(reader)); status != http.StatusOK || string(data) != "replaced\n" {
		test.Fatalf("after clobber: HTTP %d: %q", status, data)
	}
	if count := len(objects()); count != 2 {
		test.Fatalf("after clobber: %d objects, expected 2", count)
	}

	out := test.TempDir()
	var downloaded []struct {
		Name string `json:"name"`
		Path string `json:"path"`
	}
	lyraJSON(test, srv, reader, &downloaded, "release", "download", "acme/demo", "v1.0.0", "--dir", out, "--pattern", "*.txt")
	if len(downloaded) != 1 || downloaded[0].Name != "checksums.txt" {
		test.Fatalf("download --pattern: %+v", downloaded)
	}
	if data, err := os.ReadFile(filepath.Join(out, "checksums.txt")); err != nil || !strings.HasSuffix(string(data), "  app.tar.gz\n") {
		test.Fatalf("downloaded checksums.txt: %q, %v", data, err)
	}
	for _, object := range objects() {
		if data, _ := os.ReadFile(object); string(data) == "replaced\n" {
			os.WriteFile(object, []byte("corrupt\n!"), 0o644)
		}
	}
	if result := runLyra(test, srv, reader, "release", "download", "acme/demo", "v1.0.0", "--dir", out, "--pattern", "app.*"); result.code != 1 || !strings.Contains(result.stderr, "sha256 is") {
		test.Fatalf("download of a corrupt asset: %+v", result)
	}
	if _, err := os.Stat(filepath.Join(out, "app.tar.gz")); err == nil {
		test.Fatal("a corrupt download was kept")
	}

	mustGit(test, src, "tag", "v0.9.0", first)
	push("v0.9.0")
	if result := runLyra(test, srv, token, "release", "create", "acme/demo", "v0.9.0", "--target", "main"); result.code != 1 || !strings.Contains(result.stderr, "already points at "+first) {
		test.Fatalf("release of an existing tag at another target: %+v", result)
	}
	older := release("create", "acme/demo", "v0.9.0")
	if older.Commit != first || remoteTag("v0.9.0") != first {
		test.Fatalf("release of an existing tag: %+v", older)
	}
	rc := release("create", "acme/demo", "v2.0.0-rc.1", "--prerelease", "--target", head)
	if rc.Commit != head || remoteTag("v2.0.0-rc.1") != head {
		test.Fatalf("prerelease at %s: %+v", head, rc)
	}
	if latest := release("view", "acme/demo", "latest"); latest.Tag != "v0.9.0" {
		test.Fatalf("latest is %s, expected v0.9.0 (the prerelease must not count)", latest.Tag)
	}
	if result := runLyra(test, srv, token, "release", "create", "acme/demo", "v0.9.0"); result.code != 1 || !strings.Contains(result.stderr, "already exists") {
		test.Fatalf("second release for a tag: %+v", result)
	}
	if result := runLyra(test, srv, token, "release", "create", "acme/demo", "bad tag"); result.code != 1 || !strings.Contains(result.stderr, "invalid tag name") {
		test.Fatalf("invalid tag: %+v", result)
	}
	if result := runLyra(test, srv, token, "release", "create", "acme/demo", "v3.0.0", "--target", "nope"); result.code != 1 || !strings.Contains(result.stderr, "target nope not found") || remoteTag("v3.0.0") != "" {
		test.Fatalf("unknown target: %+v", result)
	}
	if result := runLyra(test, srv, token, "release", "edit", "acme/demo", "v0.9.0", "--target", "main"); result.code != 1 || !strings.Contains(result.stderr, "can't change") {
		test.Fatalf("target of a published release: %+v", result)
	}
	edited := release("edit", "acme/demo", "v0.9.0", "--title", "Beta", "--notes", "Notes")
	if edited.Title != "Beta" || edited.Notes != "Notes" || edited.Commit != first {
		test.Fatalf("edit: %+v", edited)
	}

	slashed := release("create", "acme/demo", "release/1.0", "--asset", checksumsPath)
	if !strings.HasSuffix(slashed.Assets[0].DownloadURL, "/acme/demo/releases/download/release/1.0/checksums.txt") {
		test.Fatalf("download URL of a tag with a slash: %s", slashed.Assets[0].DownloadURL)
	}
	if status, data := download("/acme/demo/releases/download/release/1.0/checksums.txt", bearer(reader)); status != http.StatusOK || !strings.HasSuffix(string(data), "  app.tar.gz\n") {
		test.Fatalf("download from a tag with a slash: HTTP %d: %s", status, data)
	}
	if viewed := release("view", "acme/demo", "release/1.0"); viewed.ID != slashed.ID {
		test.Fatalf("view a tag with a slash: %+v", viewed)
	}

	if result := runLyra(test, srv, token, "release", "delete-asset", "acme/demo", "release/1.0", "checksums.txt"); result.code != 0 {
		test.Fatalf("delete-asset: %+v", result)
	}
	if status, _ := download("/acme/demo/releases/download/release/1.0/checksums.txt", bearer(reader)); status != http.StatusNotFound {
		test.Fatalf("deleted asset: HTTP %d", status)
	}
	if result := runLyra(test, srv, token, "release", "delete", "acme/demo", "v1.0.0"); result.code != 2 {
		test.Fatalf("delete without --yes: %+v", result)
	}
	if result := runLyra(test, srv, token, "release", "delete", "acme/demo", "v1.0.0", "--cleanup-tag", "--yes"); result.code != 0 || !strings.Contains(result.stdout, `"tag_deleted": true`) {
		test.Fatalf("delete --cleanup-tag: %+v", result)
	}
	if remoteTag("v1.0.0") != "" || len(objects()) != 0 {
		test.Fatalf("after delete: tag %q, objects %v", remoteTag("v1.0.0"), objects())
	}

	if result := runLyra(test, srv, token, "release", "upload", "acme/demo", "v0.9.0", checksumsPath); result.code != 0 {
		test.Fatalf("upload: %+v", result)
	}
	if len(objects()) != 1 {
		test.Fatalf("objects before repo delete: %v", objects())
	}
	if result := runLyra(test, srv, token, "repo", "delete", "acme/demo", "--yes"); result.code != 0 {
		test.Fatalf("repo delete: %+v", result)
	}
	if _, err := os.Stat(filepath.Join(srv.data, "blobs", "lyra", "releases", "acme/demo")); err == nil {
		test.Fatal("repo delete left its release assets")
	}
}

package e2e

import (
	"archive/zip"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

const artifactsWorkflow = `on: workflow_dispatch
jobs:
  build:
    runs-on: test
    steps:
      - run: mkdir out && echo hello > out/greeting.txt
      - uses: https://code.forgejo.org/forgejo/upload-artifact@v4
        with:
          name: greeting
          path: out
  use:
    needs: build
    runs-on: test
    steps:
      - uses: https://code.forgejo.org/forgejo/download-artifact@v4
        with:
          name: greeting
          path: got
      - run: test "$(cat got/greeting.txt)" = hello
`

func TestArtifacts(test *testing.T) {
	srv := startServer(test, "127.0.0.1")
	token := adminToken(test, srv)
	createRepo(test, srv, token, "acme/demo")
	src := newSource(test)
	if err := os.MkdirAll(filepath.Join(src, ".forgejo", "workflows"), 0o755); err != nil {
		test.Fatal(err)
	}
	commitFile(test, src, ".forgejo/workflows/artifacts.yml", artifactsWorkflow, "Add artifacts workflow")
	mustGit(test, src, "-c", "http.extraHeader=Authorization: Bearer "+token, "push", "-q", srv.url+"/acme/demo.git", "main")
	runner := hostRunner(test, srv, token)

	runLyra(test, srv, token, "workflow", "run", "-R", "acme/demo", "artifacts.yml")
	result := runLyra(test, srv, token, "run", "watch", "-R", "acme/demo", "1", "--exit-status", "--timeout", "300s")
	if result.code != 0 {
		logs := runLyra(test, srv, token, "run", "logs", "-R", "acme/demo", "1")
		test.Fatalf("run: %+v\nlogs:\n%s\nrunner log:\n%s", result, logs.stdout, runner.logs.String())
	}

	var artifacts []struct {
		Name   string `json:"name"`
		Size   int64  `json:"size"`
		SHA256 string `json:"sha256"`
	}
	lyraJSON(test, srv, token, &artifacts, "run", "artifacts", "-R", "acme/demo", "1")
	if len(artifacts) != 1 || artifacts[0].Name != "greeting" || artifacts[0].Size == 0 {
		test.Fatalf("artifacts: %+v", artifacts)
	}

	dir := test.TempDir()
	var downloaded struct {
		Path string `json:"path"`
	}
	lyraJSON(test, srv, token, &downloaded, "run", "download", "-R", "acme/demo", "1", "greeting", "--dir", dir)
	archive, err := zip.OpenReader(downloaded.Path)
	if err != nil {
		test.Fatal(err)
	}
	defer archive.Close()
	if len(archive.File) != 1 || archive.File[0].Name != "greeting.txt" {
		test.Fatalf("zip entries: %+v", archive.File)
	}
	entry, err := archive.File[0].Open()
	if err != nil {
		test.Fatal(err)
	}
	content, _ := io.ReadAll(entry)
	entry.Close()
	if string(content) != "hello\n" {
		test.Fatalf("greeting.txt: %q", content)
	}

	if missing := runLyra(test, srv, token, "run", "download", "-R", "acme/demo", "1", "nothing"); missing.code == 0 {
		test.Fatalf("downloading a missing artifact: %+v", missing)
	}
	service := srv.url + "/twirp/github.actions.results.api.v1.ArtifactService/"
	status, body := call(test, http.MethodPost, service+"ListArtifacts", bearer("e30.e30.forged"), `{"workflow_run_backend_id":"1"}`)
	var failure struct {
		Code string `json:"code"`
	}
	if json.Unmarshal(body, &failure); status != http.StatusForbidden || failure.Code != "permission_denied" {
		test.Fatalf("ListArtifacts with a forged token: HTTP %d %s", status, body)
	}
	status, _ = call(test, http.MethodGet, service+"DownloadArtifact?artifact=1&task=1&expires=9999999999&sig=x", nil, "")
	if status != http.StatusForbidden {
		test.Fatalf("DownloadArtifact with a forged signature: HTTP %d", status)
	}
}

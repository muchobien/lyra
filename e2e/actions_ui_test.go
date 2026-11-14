package e2e

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/muchobien/lyra/internal/store"
)

const uiWorkflow = `name: CI
on: push
jobs:
  build:
    runs-on: test
    steps:
      - name: Say hello
        run: echo "hello from the job"
      - name: Wait
        run: echo "waiting" && sleep 4
      - name: Say bye
        run: echo "bye <b>tag</b>"
  gpu:
    runs-on: gpu
    steps:
      - run: echo never
`

func TestActionsUI(test *testing.T) {
	srv := startServer(test, "127.0.0.1")
	token := adminToken(test, srv)
	createRepo(test, srv, token, "acme/demo")
	auth := "http.extraHeader=Authorization: Bearer " + token
	remote := srv.url + "/acme/demo.git"
	src := newSource(test)
	read := func(target string) []byte {
		test.Helper()
		status, response := call(test, http.MethodGet, srv.url+target, tailnet(), "")
		if status != http.StatusOK {
			test.Fatalf("%s: HTTP %d:\n%s", target, status, response)
		}
		return response
	}
	run := func() store.Run {
		test.Helper()
		var result store.Run
		if err := json.Unmarshal(read("/api/repos/acme/demo/runs/1"), &result); err != nil {
			test.Fatal(err)
		}
		return result
	}
	hostRunner(test, srv, token)
	commitFile(test, src, "README.md", "# Demo\n", "First commit")
	if err := os.MkdirAll(filepath.Join(src, ".forgejo", "workflows"), 0o755); err != nil {
		test.Fatal(err)
	}
	commitFile(test, src, ".forgejo/workflows/ci.yml", uiWorkflow, "Add CI")
	mustGit(test, src, "-c", auth, "push", "-q", remote, "main")
	sha := mustGit(test, src, "rev-parse", "HEAD")
	eventually(test, "the live build log", func() bool {
		current := run()
		return len(current.Jobs) == 2 && current.Jobs[0].Status == "running" && strings.Contains(string(read("/api/repos/acme/demo/runs/1/jobs/1/logs")), "hello from the job")
	})
	current := run()
	if current.Status != "running" || current.Jobs[1].Status != "waiting" || strings.Join(current.Jobs[1].RunsOn, ",") != "gpu" {
		test.Fatalf("running workspace: %+v", current)
	}
	eventually(test, "the build job to finish", func() bool { return run().Jobs[0].Status == "success" })
	current = run()
	if len(current.Jobs[0].Steps) != 3 || current.Jobs[0].Steps[0].Name != "Say hello" || current.Jobs[0].Steps[0].Result != "success" || !strings.Contains(string(read("/api/repos/acme/demo/runs/1/jobs/1/logs")), "bye <b>tag</b>") {
		test.Fatalf("completed job workspace: %+v", current.Jobs[0])
	}
	var commit struct {
		Checks store.Checks `json:"checks"`
	}
	if err := json.Unmarshal(read("/api/repos/acme/demo/commits/"+sha+"/view"), &commit); err != nil || commit.Checks.State != "pending" || len(commit.Checks.Jobs) != 2 || commit.Checks.Jobs[0].Status != "success" {
		test.Fatalf("commit checks: %+v, %v", commit.Checks, err)
	}
	for _, target := range []string{"runs", "runs/1", "runs/1?job=2", "commits/main", "commit/" + sha} {
		if page := read("/acme/demo/" + target); !strings.Contains(string(page), `<div id="root"></div>`) {
			test.Fatalf("%s lacks the embedded application", target)
		}
	}
	runURL := srv.url + "/api/repos/acme/demo/runs/1"
	if status, _ := browserCall(test, http.MethodPost, runURL+"/cancel", "", map[string]string{"Sec-Fetch-Site": "cross-site"}); status != http.StatusForbidden {
		test.Fatalf("cross-site cancel: HTTP %d", status)
	}
	status, response := browserCall(test, http.MethodPost, runURL+"/cancel", "", nil)
	if status != http.StatusOK || run().Status != "cancelled" {
		test.Fatalf("cancel: HTTP %d: %s", status, response)
	}
	if err := json.Unmarshal(read("/api/repos/acme/demo/commits/"+sha+"/view"), &commit); err != nil || commit.Checks.State != "failure" {
		test.Fatalf("cancelled checks: %+v, %v", commit.Checks, err)
	}
	mustGit(test, src, "switch", "-q", "-c", "feature")
	commitFile(test, src, "feature.txt", "feature\n", "Add feature")
	mustGit(test, src, "-c", auth, "push", "-q", remote, "feature")
	lyraJSON(test, srv, token, &pullResult{}, "pr", "create", "acme/demo", "--head", "feature", "--title", "Feature")
	var pull struct {
		Checks store.Checks `json:"checks"`
	}
	if err := json.Unmarshal(read("/api/repos/acme/demo/pulls/1/view"), &pull); err != nil || len(pull.Checks.Jobs) != 2 || pull.Checks.Jobs[0].Run != 2 {
		test.Fatalf("pull workspace checks: %+v, %v", pull.Checks, err)
	}
	status, response = browserCall(test, http.MethodPost, runURL+"/rerun?failed=true", "", nil)
	if status != http.StatusOK || run().Status == "cancelled" || run().Jobs[0].Status != "success" || run().Jobs[1].Status != "waiting" {
		test.Fatalf("rerun failed jobs: HTTP %d %s, %+v", status, response, run())
	}
	var listed []store.Run
	if err := json.Unmarshal(read("/api/repos/acme/demo/runs"), &listed); err != nil || len(listed) != 2 || listed[1].Number != 1 {
		test.Fatalf("run list: %+v, %v", listed, err)
	}
	single := run()
	if len(listed[1].Jobs) != 2 || listed[1].Jobs[0].TaskID != single.Jobs[0].TaskID || listed[1].Jobs[0].TaskID == 0 || fmt.Sprint(listed[1].Jobs[0].Steps) != fmt.Sprint(single.Jobs[0].Steps) || len(single.Jobs[0].Steps) == 0 {
		test.Fatalf("listed jobs %+v, want %+v", listed[1].Jobs, single.Jobs)
	}
	if err := json.Unmarshal(read("/api/repos/acme/demo/runs?limit=1"), &listed); err != nil || len(listed) != 1 || listed[0].Number != 2 || len(listed[0].Jobs) != 2 {
		test.Fatalf("latest run: %+v, %v", listed, err)
	}
	reader := createToken(test, srv, `{"name":"reader","scope":"read"}`)
	if status, _ := call(test, http.MethodPost, runURL+"/cancel", bearer(reader), ""); status != http.StatusForbidden {
		test.Fatalf("cancel with a read token: HTTP %d", status)
	}
	identityStatus, identity := call(test, http.MethodGet, srv.url+"/api/me", bearer(reader), "")
	if identityStatus != http.StatusOK || !strings.Contains(compactJSON(test, identity), `"scope":"read"`) {
		test.Fatalf("reader controls identity: HTTP %d %s", identityStatus, identity)
	}
}

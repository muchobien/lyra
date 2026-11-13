package e2e

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/muchobien/lyra/internal/store"
)

const pullWorkflow = `name: PR
on:
  pull_request:
    branches: [main]
    paths-ignore: ['docs/**']
jobs:
  check:
    runs-on: test
    steps:
      - run: |
          echo "event=${{ github.event_name }} action=${{ github.event.action }} head=${{ github.head_ref }} base=${{ github.base_ref }} number=${{ github.event.pull_request.number }} ref=${{ github.ref }}"
          git -c "http.extraHeader=Authorization: Bearer ${{ github.token }}" clone -q "${{ github.server_url }}/${{ github.repository }}.git" checkout
          git -C checkout -c "http.extraHeader=Authorization: Bearer ${{ github.token }}" fetch -q origin "${{ github.ref }}"
          git -C checkout checkout -q FETCH_HEAD
          test "$(git -C checkout rev-parse HEAD)" = "${{ github.sha }}"
          test ! -f checkout/fail.txt
`

const slowPullWorkflow = `name: PR slow
on:
  pull_request:
    branches: [main]
    paths-ignore: ['docs/**']
jobs:
  slow:
    runs-on: test
    steps:
      - run: sleep 2
`

type pullChecks struct {
	Number int64        `json:"number"`
	State  string       `json:"state"`
	Checks store.Checks `json:"checks"`
	Merge  struct {
		Blocked []string `json:"blocked"`
	} `json:"merge"`
	Actions []string `json:"actions"`
}

func TestPullChecks(test *testing.T) {
	srv := startServer(test, "127.0.0.1")
	token := adminToken(test, srv)
	createRepo(test, srv, token, "acme/demo")
	auth := "http.extraHeader=Authorization: Bearer " + token
	remote := srv.url + "/acme/demo.git"
	src := newSource(test)
	push := func(args ...string) string {
		test.Helper()
		return mustGit(test, src, append([]string{"-c", auth, "push", remote}, args...)...)
	}
	pr := func(args ...string) pullChecks {
		test.Helper()
		var result pullChecks
		lyraJSON(test, srv, token, &result, append([]string{"pr"}, args...)...)
		return result
	}
	waitChecks := func(number string) string {
		test.Helper()
		result := runLyra(test, srv, token, "pr", "wait", "acme/demo", number, "--for", "checks", "--timeout", "90s", "--jq", ".data.state")
		if result.code != 0 {
			test.Fatalf("pr wait --for checks: %+v", result)
		}
		return strings.TrimSpace(result.stdout)
	}
	hostRunner(test, srv, token)

	commitFile(test, src, "README.md", "# Demo\n", "First commit")
	if err := os.MkdirAll(filepath.Join(src, ".forgejo", "workflows"), 0o755); err != nil {
		test.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, ".forgejo", "workflows", "slow.yml"), []byte(slowPullWorkflow), 0o644); err != nil {
		test.Fatal(err)
	}
	mustGit(test, src, "add", ".forgejo")
	commitFile(test, src, ".forgejo/workflows/pr.yml", pullWorkflow, "Add the pull request workflow")
	if output := push("main"); strings.Contains(output, "queued") {
		test.Fatalf("a pull_request workflow ran on push: %s", output)
	}
	if result := runLyra(test, srv, token, "rule", "set", "acme/demo", "main", "--require-checks"); result.code != 0 || !strings.Contains(result.stdout, `"require_checks": true`) {
		test.Fatalf("rule set --require-checks: %+v", result)
	}

	mustGit(test, src, "switch", "-q", "-c", "feature")
	commitFile(test, src, "feature.txt", "feature\n", "Add feature")
	commitFile(test, src, "fail.txt", "fail\n", "Break the check")
	push("feature")
	opened := pr("create", "acme/demo", "--head", "feature", "--title", "Feature")
	if strings.Join(opened.Actions, "|") != "queued run #1 (PR)|queued run #2 (PR slow)" {
		test.Fatalf("pr create actions: %v", opened.Actions)
	}
	if state := waitChecks("1"); state != "failure" {
		test.Fatalf("checks after opening: %s", state)
	}
	failed := pr("view", "acme/demo", "1")
	if failed.Checks.State != "failure" || len(failed.Checks.Jobs) != 2 || failed.Checks.Jobs[0].Event != "pull_request" || !strings.Contains(strings.Join(failed.Merge.Blocked, "|"), "checks failed for") {
		test.Fatalf("pull request with a failing check: %+v", failed)
	}
	if result := runLyra(test, srv, token, "pr", "merge", "acme/demo", "1", "--squash"); result.code != 1 || !strings.Contains(result.stderr, "checks failed for") {
		test.Fatalf("merge with failing checks: %+v", result)
	}
	logs := runLyra(test, srv, token, "run", "logs", "acme/demo", "1").stdout
	if !strings.Contains(logs, "event=pull_request action=opened head=feature base=main number=1 ref=refs/pull/1/head") {
		test.Fatalf("pull request run log:\n%s", logs)
	}

	mustGit(test, src, "rm", "-q", "fail.txt")
	mustGit(test, src, "commit", "-q", "-m", "Fix the check")
	if output := push("feature"); !strings.Contains(output, "lyra: queued run #3 (PR)") {
		test.Fatalf("push to the pull request: %s", output)
	}
	if state := waitChecks("1"); state != "success" {
		test.Fatalf("checks after the fix: %s", state)
	}
	if fixed := pr("view", "acme/demo", "1"); fixed.Checks.State != "success" || len(fixed.Checks.Jobs) != 2 || fixed.Checks.Jobs[0].Run != 3 || len(fixed.Merge.Blocked) != 0 {
		test.Fatalf("pull request after the fix: %+v", fixed)
	}
	if logs := runLyra(test, srv, token, "run", "logs", "acme/demo", "3").stdout; !strings.Contains(logs, "action=synchronize") {
		test.Fatalf("synchronize run log:\n%s", logs)
	}
	if result := runLyra(test, srv, token, "pr", "merge", "acme/demo", "1", "--squash"); result.code != 0 {
		test.Fatalf("merge with passing checks: %+v", result)
	}

	mustGit(test, src, "switch", "-q", "-c", "docs", "main")
	if err := os.MkdirAll(filepath.Join(src, "docs"), 0o755); err != nil {
		test.Fatal(err)
	}
	commitFile(test, src, "docs/guide.md", "# Guide\n", "Add a guide")
	push("docs")
	docs := pr("create", "acme/demo", "--head", "docs", "--title", "Docs")
	if len(docs.Actions) != 0 {
		test.Fatalf("a docs-only pull request queued: %v", docs.Actions)
	}
	if state := waitChecks("2"); state != "none" {
		test.Fatalf("pr wait --for checks on a pull request where nothing runs: %s", state)
	}
	if viewed := pr("view", "acme/demo", "2"); viewed.Checks.State != "none" || !strings.Contains(strings.Join(viewed.Merge.Blocked, "|"), "no checks ran for") {
		test.Fatalf("docs pull request: %+v", viewed)
	}

	mustGit(test, src, "switch", "-q", "-c", "again", "main")
	commitFile(test, src, "again.txt", "again\n", "Another change")
	push("again")
	pr("create", "acme/demo", "--head", "again", "--title", "Again")
	if state := waitChecks("3"); state != "success" {
		test.Fatalf("checks on #3: %s", state)
	}
	pr("close", "acme/demo", "3")
	if reopened := pr("reopen", "acme/demo", "3"); !strings.Contains(strings.Join(reopened.Actions, "|"), "queued run") {
		test.Fatalf("reopen actions: %v", reopened.Actions)
	}
	if state := waitChecks("3"); state != "success" {
		test.Fatalf("checks after reopening: %s", state)
	}

	status, data := call(test, http.MethodGet, srv.url+"/api/events?repo=acme/demo&limit=200", bearer(token), "")
	if status != http.StatusOK {
		test.Fatalf("events: HTTP %d", status)
	}
	var events []store.Event
	json.Unmarshal(data, &events)
	checks := 0
	for _, event := range events {
		if event.Action == "pr.checks" {
			checks++
			if event.Actor.User != "lyra" || event.Item == 0 {
				test.Fatalf("pr.checks event: %+v", event)
			}
		}
	}
	if checks != 4 {
		test.Fatalf("%d pr.checks events, want 4", checks)
	}
}

func TestChecksFromPushWorkflows(test *testing.T) {
	srv := startServer(test, "127.0.0.1")
	token := adminToken(test, srv)
	createRepo(test, srv, token, "acme/demo")
	auth := "http.extraHeader=Authorization: Bearer " + token
	remote := srv.url + "/acme/demo.git"
	src := newSource(test)
	hostRunner(test, srv, token)
	if err := os.MkdirAll(filepath.Join(src, ".forgejo", "workflows"), 0o755); err != nil {
		test.Fatal(err)
	}
	commitFile(test, src, ".forgejo/workflows/ci.yml", "on: push\njobs:\n  test:\n    runs-on: test\n    steps:\n      - run: echo ok\n", "Add CI")
	mustGit(test, src, "-c", auth, "push", "-q", remote, "main")
	mustGit(test, src, "switch", "-q", "-c", "feature")
	commitFile(test, src, "feature.txt", "one\n", "Feature")
	mustGit(test, src, "-c", auth, "push", "-q", remote, "feature")
	lyraJSON(test, srv, token, &pullResult{}, "pr", "create", "acme/demo", "--head", "feature", "--title", "Feature")
	cursor := strings.TrimSpace(runLyra(test, srv, token, "event", "list", "--repo", "acme/demo", "--limit", "1", "--jq", ".[0].id").stdout)
	commitFile(test, src, "feature.txt", "two\n", "More feature")
	mustGit(test, src, "-c", auth, "push", "-q", remote, "feature")
	result := runLyra(test, srv, token, "pr", "wait", "acme/demo", "1", "--for", "checks", "--since", cursor, "--timeout", "60s", "--jq", ".data.state")
	if result.code != 0 || strings.TrimSpace(result.stdout) != "success" {
		test.Fatalf("first pr.checks after a push to a pull request with push workflows: %+v", result)
	}
}

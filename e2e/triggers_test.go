package e2e

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

var triggerWorkflows = map[string]string{
	"dispatch.yml": `name: Deploy
on:
  workflow_dispatch:
    inputs:
      target:
        required: true
        type: choice
        options: [staging, production]
      dry:
        type: boolean
        default: "true"
      note:
        default: none
      reason:
        required: true
jobs:
  deploy:
    runs-on: test
    steps:
      - env:
          KEY: ${{ secrets.DEPLOY_KEY }}
        run: |
          echo "target=${{ inputs.target }} dry=${{ inputs.dry }} note=${{ inputs.note }} var=${{ vars.REGION }} event=${{ github.event_name }}"
          echo "secret=${{ secrets.DEPLOY_KEY }} length=${#KEY}"
`,
	"release.yml": `name: Release
on:
  release:
    types: [published]
jobs:
  announce:
    runs-on: test
    steps:
      - run: echo "release=${{ github.event.release.tag_name }} ref=${{ github.ref }} prerelease=${{ github.event.release.prerelease }}"
`,
	"tag.yml": `name: Tag
on:
  push:
    tags: ['v*']
jobs:
  tag:
    runs-on: test
    steps:
      - run: echo "tag sha=${{ github.sha }}"
`,
	"schedule.yml": `name: Nightly
on:
  schedule:
    - cron: '* * * * *'
jobs:
  nightly:
    runs-on: test
    steps:
      - run: echo "scheduled ${{ github.event.schedule }} by ${{ github.actor }}"
`,
}

func TestTriggers(test *testing.T) {
	srv := startServer(test, "127.0.0.1")
	token := adminToken(test, srv)
	reader := createToken(test, srv, `{"name":"reader","scope":"read"}`)
	createRepo(test, srv, token, "acme/demo")
	src := newSource(test)
	hostRunner(test, srv, token)
	runOf := func(name string) (int64, string) {
		test.Helper()
		var runs []struct {
			Number int64  `json:"number"`
			Name   string `json:"name"`
			Status string `json:"status"`
		}
		lyraJSON(test, srv, token, &runs, "run", "list", "-R", "acme/demo")
		for _, run := range runs {
			if run.Name == name {
				return run.Number, run.Status
			}
		}
		return 0, ""
	}
	logsOf := func(name string) string {
		test.Helper()
		number, _ := runOf(name)
		if result := runLyra(test, srv, token, "run", "watch", "-R", "acme/demo", itoa(number), "--exit-status", "--timeout", "60s"); result.code != 0 {
			test.Fatalf("run %s #%d: %+v", name, number, result)
		}
		return runLyra(test, srv, token, "run", "logs", "-R", "acme/demo", itoa(number)).stdout
	}

	commitFile(test, src, "README.md", "# Demo\n", "First commit")
	if err := os.MkdirAll(filepath.Join(src, ".forgejo", "workflows"), 0o755); err != nil {
		test.Fatal(err)
	}
	for name, content := range triggerWorkflows {
		if err := os.WriteFile(filepath.Join(src, ".forgejo", "workflows", name), []byte(content), 0o644); err != nil {
			test.Fatal(err)
		}
	}
	mustGit(test, src, "add", ".forgejo")
	mustGit(test, src, "commit", "-q", "-m", "Add workflows")
	if output := mustGit(test, src, "-c", "http.extraHeader=Authorization: Bearer "+token, "push", srv.url+"/acme/demo.git", "main"); strings.Contains(output, "queued") {
		test.Fatalf("push queued runs: %s", output)
	}

	secret := exec.Command(lyraBinary, "secret", "set", "-R", "acme/demo", "deploy_key")
	secret.Env = append(os.Environ(), "LYRA_URL="+srv.url, "LYRA_TOKEN="+token)
	secret.Stdin = strings.NewReader("s3cr3t-value\n")
	if output, err := secret.CombinedOutput(); err != nil || !strings.Contains(string(output), `"name": "DEPLOY_KEY"`) {
		test.Fatalf("secret set: %v: %s", err, output)
	}
	if result := runLyra(test, srv, token, "secret", "set", "-R", "acme/demo", "spare_key", "-b", "spare"); result.code != 0 {
		test.Fatalf("secret set --body: %+v", result)
	}
	if listed := runLyra(test, srv, token, "secret", "list", "-R", "acme/demo"); !strings.Contains(listed.stdout, `"name": "DEPLOY_KEY"`) || !strings.Contains(listed.stdout, `"name": "SPARE_KEY"`) || strings.Contains(listed.stdout, "s3cr3t") {
		test.Fatalf("secret list: %+v", listed)
	}
	if result := runLyra(test, srv, reader, "secret", "list", "-R", "acme/demo"); result.code != 1 || !strings.Contains(result.stderr, "lacks admin") {
		test.Fatalf("secret list with a read token: %+v", result)
	}
	if result := runLyra(test, srv, token, "variable", "set", "-R", "acme/demo", "GITHUB_REGION", "--body", "x"); result.code != 1 || !strings.Contains(result.stderr, "not starting with") {
		test.Fatalf("reserved variable name: %+v", result)
	}
	if result := runLyra(test, srv, token, "variable", "set", "-R", "acme/demo", "region", "-b", "eu-west"); result.code != 0 {
		test.Fatalf("variable set: %+v", result)
	}
	for _, file := range []string{"lyra.db", "lyra.db-wal"} {
		if data, err := os.ReadFile(filepath.Join(srv.data, file)); err == nil && bytes.Contains(data, []byte("s3cr3t-value")) {
			test.Fatalf("%s holds the secret in plain text", file)
		}
	}

	for _, refused := range []struct {
		args []string
		want string
	}{
		{[]string{"dispatch.yml", "-f", "reason=x"}, "input target is required"},
		{[]string{"dispatch.yml", "-f", "target=staging"}, "input reason is required"},
		{[]string{"dispatch.yml", "-f", "target=moon", "-f", "reason=x"}, "must be one of staging, production"},
		{[]string{"dispatch.yml", "-f", "target=staging", "-f", "reason=x", "-f", "dry=maybe"}, "must be true or false"},
		{[]string{"dispatch.yml", "-f", "target=staging", "-f", "reason=x", "-f", "extra=1"}, "has no input extra"},
		{[]string{"release.yml"}, "has no workflow_dispatch trigger"},
		{[]string{"missing.yml"}, "no workflow missing.yml"},
	} {
		if result := runLyra(test, srv, token, append([]string{"workflow", "run", "-R", "acme/demo"}, refused.args...)...); result.code != 1 || !strings.Contains(result.stderr, refused.want) {
			test.Fatalf("workflow run %v: %+v", refused.args, result)
		}
	}
	var dispatched struct {
		Number int64  `json:"number"`
		Event  string `json:"event"`
	}
	lyraJSON(test, srv, token, &dispatched, "workflow", "run", "-R", "acme/demo", "dispatch.yml", "-F", "target=production", "--raw-field", "reason=ship")
	if dispatched.Event != "workflow_dispatch" {
		test.Fatalf("dispatched run: %+v", dispatched)
	}
	logs := logsOf("Deploy")
	for _, want := range []string{"target=production dry=true note=none var=eu-west event=workflow_dispatch", "secret=*** length=12"} {
		if !strings.Contains(logs, want) {
			test.Fatalf("deploy log lacks %q:\n%s", want, logs)
		}
	}
	if strings.Contains(logs, "s3cr3t") {
		test.Fatalf("the secret leaked into the log:\n%s", logs)
	}

	mustGit(test, src, "tag", "-a", "-m", "v0.9.0", "v0.9.0")
	mustGit(test, src, "-c", "http.extraHeader=Authorization: Bearer "+token, "push", "-q", srv.url+"/acme/demo.git", "v0.9.0")
	tagCommit := strings.TrimSpace(mustGit(test, src, "rev-parse", "v0.9.0^{commit}"))
	if logs := logsOf("Tag"); !strings.Contains(logs, "tag sha="+tagCommit) {
		test.Fatalf("an annotated tag's run lacks its commit %s:\n%s", tagCommit, logs)
	}
	var oldDraft struct {
		Commit  string   `json:"commit"`
		Actions []string `json:"actions"`
	}
	lyraJSON(test, srv, token, &oldDraft, "release", "create", "-R", "acme/demo", "v0.9.0", "--draft")
	if oldDraft.Commit == "" || len(oldDraft.Actions) != 0 {
		test.Fatalf("a draft for an existing tag: %+v", oldDraft)
	}
	var draft struct {
		Actions []string `json:"actions"`
	}
	lyraJSON(test, srv, token, &draft, "release", "create", "-R", "acme/demo", "v1.0.0", "--draft", "--prerelease")
	if len(draft.Actions) != 0 {
		test.Fatalf("a draft release queued runs: %v", draft.Actions)
	}
	var published struct {
		Actions []string `json:"actions"`
	}
	lyraJSON(test, srv, token, &published, "release", "publish", "-R", "acme/demo", "v1.0.0")
	if len(published.Actions) != 1 || !strings.Contains(published.Actions[0], "(Release)") {
		test.Fatalf("publishing queued: %v", published.Actions)
	}
	if logs := logsOf("Release"); !strings.Contains(logs, "release=v1.0.0 ref=refs/tags/v1.0.0 prerelease=true") {
		test.Fatalf("release log:\n%s", logs)
	}

	deadline := time.Now().Add(80 * time.Second)
	for {
		if number, _ := runOf("Nightly"); number != 0 {
			break
		}
		if time.Now().After(deadline) {
			test.Fatal("no scheduled run within 80 s")
		}
		time.Sleep(time.Second)
	}
	if logs := logsOf("Nightly"); !strings.Contains(logs, "scheduled * * * * * by lyra") {
		test.Fatalf("scheduled run log:\n%s", logs)
	}

	status, data := call(test, http.MethodGet, srv.url+"/api/events?repo=acme/demo&limit=200", bearer(token), "")
	if status != http.StatusOK {
		test.Fatalf("events: HTTP %d", status)
	}
	if bytes.Contains(data, []byte("s3cr3t")) {
		test.Fatal("an event holds the secret")
	}
	var events []struct {
		Action string `json:"action"`
	}
	json.Unmarshal(data, &events)
	found := map[string]bool{}
	for _, event := range events {
		found[event.Action] = true
	}
	if !found["secret.set"] || !found["variable.set"] {
		test.Fatalf("events: %v", found)
	}
}

func itoa(number int64) string {
	return strconv.FormatInt(number, 10)
}

func TestSecretsKeyChange(test *testing.T) {
	address := "127.0.0.1:" + freePort(test)
	data := filepath.Join(test.TempDir(), "data")
	keys := test.TempDir()
	for name, key := range map[string]string{"old": strings.Repeat("ab", 32), "new": strings.Repeat("cd", 32)} {
		if err := os.WriteFile(filepath.Join(keys, name), []byte(key+"\n"), 0o600); err != nil {
			test.Fatal(err)
		}
	}
	srv := startServerAt(test, address, data, "127.0.0.1", "--secrets-key-file", filepath.Join(keys, "old"))
	token := adminToken(test, srv)
	createRepo(test, srv, token, "acme/demo")
	src := newSource(test)
	if err := os.MkdirAll(filepath.Join(src, ".forgejo", "workflows"), 0o755); err != nil {
		test.Fatal(err)
	}
	commitFile(test, src, ".forgejo/workflows/manual.yml", "on: workflow_dispatch\njobs:\n  echo:\n    runs-on: test\n    steps:\n      - run: echo ${{ secrets.KEY }}\n", "Add a manual workflow")
	mustGit(test, src, "-c", "http.extraHeader=Authorization: Bearer "+token, "push", "-q", srv.url+"/acme/demo.git", "main")
	status, body := call(test, http.MethodPut, srv.url+"/api/repos/acme/demo/secrets/KEY", bearer(token), `{"value":"value"}`)
	if status != http.StatusOK {
		test.Fatalf("set secret: HTTP %d: %s", status, body)
	}
	srv.stop()

	srv = startServerAt(test, address, data, "127.0.0.1", "--secrets-key-file", filepath.Join(keys, "new"))
	hostRunner(test, srv, token)
	if result := runLyra(test, srv, token, "workflow", "run", "-R", "acme/demo", "manual.yml"); result.code != 0 {
		test.Fatalf("workflow run: %+v", result)
	}
	if result := runLyra(test, srv, token, "run", "watch", "-R", "acme/demo", "1", "--exit-status", "--timeout", "30s"); result.code != 1 || !strings.Contains(result.stderr, "failure") {
		test.Fatalf("run with an undecryptable secret: %+v", result)
	}
	if result := runLyra(test, srv, token, "run", "watch", "-R", "acme/demo", "1", "--jq", ".status"); result.code != 0 || result.stdout != "failure\n" {
		test.Fatalf("run watch without --exit-status: %+v", result)
	}
	if logs := runLyra(test, srv, token, "run", "logs", "-R", "acme/demo", "1").stdout; !strings.Contains(logs, "lyra: the job couldn't start: secret KEY:") {
		test.Fatalf("log of a job that couldn't start:\n%s", logs)
	}
}

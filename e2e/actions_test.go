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

const ciWorkflow = `name: CI
on:
  push:
    branches: [main]
jobs:
  build:
    runs-on: test
    outputs:
      version: ${{ steps.version.outputs.value }}
    steps:
      - name: Clone with the job token
        run: |
          git -c "http.extraHeader=Authorization: Bearer ${{ github.token }}" clone -q "${{ github.server_url }}/${{ github.repository }}.git" checkout
          cat checkout/README.md
          echo "repository=${{ github.repository }} ref=${{ github.ref_name }} event=${{ github.event_name }} pusher=${{ github.event.pusher.name }}"
      - name: Use the API
        run: |
          echo "other-repo=$(curl -s -o /dev/null -w '%{http_code}' -H "Authorization: Bearer ${{ github.token }}" "${{ github.server_url }}/api/repos/acme/other")"
          echo "issue=$(curl -s -o /dev/null -w '%{http_code}' -H "Authorization: Bearer ${{ github.token }}" -d '{"title":"Opened by a job"}' "${{ github.server_url }}/api/repos/${{ github.repository }}/issues")"
      - id: version
        run: echo "value=1.2.3" >> "$GITHUB_OUTPUT"
  test:
    runs-on: test
    needs: build
    strategy:
      matrix:
        part: [one, two]
    steps:
      - run: echo "testing ${{ matrix.part }} of ${{ needs.build.outputs.version }}"
  broken:
    runs-on: test
    steps:
      - run: exit 3
  after-broken:
    runs-on: test
    needs: broken
    steps:
      - run: echo never
  cleanup:
    runs-on: test
    needs: broken
    if: always()
    steps:
      - run: echo "cleanup after ${{ needs.broken.result }}"
`

const otherWorkflows = `name: GPU
on:
  push:
    branches: [main]
jobs:
  train:
    runs-on: gpu
    steps:
      - run: echo never
---
name: Tags
on:
  push:
    tags: ['v*']
jobs:
  release:
    runs-on: test
    steps:
      - run: echo "releasing ${{ github.ref_name }} (${{ github.ref_type }})"
---
name: Slow
on:
  push:
    branches: [slow]
jobs:
  sleep:
    runs-on: test
    steps:
      - run: echo started && sleep 120
---
name: Pull requests
on: pull_request
jobs:
  never:
    runs-on: test
    steps:
      - run: echo never
`

type runResult struct {
	store.Run
	EventID int64 `json:"event_id"`
}

func TestActions(test *testing.T) {
	srv := startServer(test, "127.0.0.1")
	token := adminToken(test, srv)
	createRepo(test, srv, token, "acme/demo")
	createRepo(test, srv, token, "acme/other")
	auth := "http.extraHeader=Authorization: Bearer " + token
	remote := srv.url + "/acme/demo.git"
	src := newSource(test)
	push := func(args ...string) string {
		test.Helper()
		return mustGit(test, src, append([]string{"-c", auth, "push", remote}, args...)...)
	}
	view := func(number string) runResult {
		test.Helper()
		var result runResult
		lyraJSON(test, srv, token, &result, "run", "view", "acme/demo", number)
		return result
	}
	statuses := func(run runResult) map[string]string {
		found := map[string]string{}
		for _, job := range run.Jobs {
			found[job.Name] += job.Status
		}
		return found
	}

	runner := hostRunner(test, srv, token)

	commitFile(test, src, "README.md", "# Demo\n", "First commit")
	if err := os.MkdirAll(filepath.Join(src, ".forgejo", "workflows"), 0o755); err != nil {
		test.Fatal(err)
	}
	for index, workflow := range strings.Split(otherWorkflows, "---\n") {
		if err := os.WriteFile(filepath.Join(src, ".forgejo", "workflows", []string{"gpu.yml", "tags.yml", "slow.yml", "pulls.yaml"}[index]), []byte(workflow), 0o644); err != nil {
			test.Fatal(err)
		}
	}
	mustGit(test, src, "add", ".forgejo")
	commitFile(test, src, ".forgejo/workflows/ci.yml", ciWorkflow, "Add workflows")
	if output := push("main"); !strings.Contains(output, "lyra: queued run #1 (CI)") || !strings.Contains(output, "lyra: queued run #2 (GPU)") || strings.Contains(output, "#3") {
		test.Fatalf("push output: %s", output)
	}

	result := runLyra(test, srv, token, "run", "wait", "acme/demo", "1", "--timeout", "120s")
	if result.code != 1 || !strings.Contains(result.stderr, "run #1 finished: failure") {
		test.Fatalf("run wait: %+v\nrunner log:\n%s", result, runner.logs.String())
	}
	var first runResult
	if err := json.Unmarshal([]byte(result.stdout), &first); err != nil {
		test.Fatal(err)
	}
	want := map[string]string{"build": "success", "test (one)": "success", "test (two)": "success", "broken": "failure", "after-broken": "skipped", "cleanup": "success"}
	got := statuses(first)
	for name, status := range want {
		if got[name] != status {
			test.Fatalf("job statuses: %v, want %v", got, want)
		}
	}
	for _, job := range first.Jobs {
		if job.Name == "after-broken" && job.Attempt != 0 {
			test.Fatalf("after-broken was sent to a runner: %+v", job)
		}
	}
	if first.Status != "failure" || first.Event != "push" || first.Title != "Add workflows" || first.Workflow != ".forgejo/workflows/ci.yml" {
		test.Fatalf("run #1: %+v", first.Run)
	}
	if steps := first.Jobs[0].Steps; len(steps) != 3 || steps[0].Name != "Clone with the job token" || steps[0].Result != "success" || steps[2].Name != "Run echo \"value=1.2.3\" >> \"$GITHUB_OUTPUT\"" {
		test.Fatalf("build steps: %+v", steps)
	}

	logs := runLyra(test, srv, token, "run", "logs", "acme/demo", "1", "--job", "build").stdout
	for _, want := range []string{"# Demo", "repository=acme/demo ref=main event=push pusher=" + login, "other-repo=403", "issue=201"} {
		if !strings.Contains(logs, want) {
			test.Fatalf("build log lacks %q:\n%s", want, logs)
		}
	}
	all := runLyra(test, srv, token, "run", "logs", "acme/demo", "1").stdout
	for _, want := range []string{"== test", "testing one of 1.2.3", "testing two of 1.2.3", "cleanup after failure", "== broken"} {
		if !strings.Contains(all, want) {
			test.Fatalf("run logs lack %q:\n%s", want, all)
		}
	}
	var issue struct {
		Author store.Actor `json:"author"`
	}
	lyraJSON(test, srv, token, &issue, "issue", "view", "acme/demo", "1")
	if issue.Author.Via != "job:acme/demo#1" {
		test.Fatalf("issue opened by the job: %+v", issue.Author)
	}

	mustGit(test, src, "tag", "nightly")
	if output := push("nightly"); strings.Contains(output, "queued") {
		test.Fatalf("a tag outside v* queued a run: %s", output)
	}
	mustGit(test, src, "tag", "v1.0.0")
	if output := push("v1.0.0"); !strings.Contains(output, "lyra: queued run #3 (Tags)") || strings.Contains(output, "(CI)") {
		test.Fatalf("tag push output: %s", output)
	}
	if result := runLyra(test, srv, token, "run", "wait", "acme/demo", "3", "--timeout", "60s"); result.code != 0 {
		test.Fatalf("run #3: %+v", result)
	}
	if logs := runLyra(test, srv, token, "run", "logs", "acme/demo", "3").stdout; !strings.Contains(logs, "releasing v1.0.0 (tag)") {
		test.Fatalf("tag run log:\n%s", logs)
	}

	mustGit(test, src, "switch", "-q", "-c", "slow")
	if output := push("slow"); !strings.Contains(output, "lyra: queued run #4 (Slow)") {
		test.Fatalf("slow push output: %s", output)
	}
	eventually(test, "the slow job to start", func() bool {
		return strings.Contains(runLyra(test, srv, token, "run", "logs", "acme/demo", "4").stdout, "started")
	})
	if result := runLyra(test, srv, token, "run", "cancel", "acme/demo", "4"); result.code != 0 {
		test.Fatalf("cancel: %+v", result)
	}
	if result := runLyra(test, srv, token, "run", "wait", "acme/demo", "4", "--timeout", "30s"); result.code != 1 || !strings.Contains(result.stderr, "cancelled") {
		test.Fatalf("wait for the cancelled run: %+v", result)
	}
	eventually(test, "the runner to stop the cancelled job", func() bool {
		job := view("4").Jobs[0]
		return job.StoppedAt != nil && job.Status == "cancelled" && strings.Contains(runLyra(test, srv, token, "run", "logs", "acme/demo", "4").stdout, "this step has been cancelled")
	})
	if result := runLyra(test, srv, token, "run", "cancel", "acme/demo", "4"); result.code != 1 || !strings.Contains(result.stderr, "already cancelled") {
		test.Fatalf("cancel a finished run: %+v", result)
	}

	var rerun runResult
	lyraJSON(test, srv, token, &rerun, "run", "rerun", "acme/demo", "1", "--failed")
	attempts := map[string]int64{}
	for _, job := range rerun.Jobs {
		attempts[job.Name] = job.Attempt
	}
	if result := runLyra(test, srv, token, "run", "wait", "acme/demo", "1", "--timeout", "120s"); result.code != 1 {
		test.Fatalf("wait for the rerun: %+v", result)
	}
	again := view("1")
	for _, job := range again.Jobs {
		rerunJob := job.Name == "broken" || job.Name == "after-broken" || job.Name == "cleanup"
		if rerunJob && job.Name != "after-broken" && job.Attempt != 2 || !rerunJob && job.Attempt != 1 {
			test.Fatalf("attempts after rerun --failed: %v -> %+v", attempts, again.Jobs)
		}
	}

	if gpu := view("2"); gpu.Status != "waiting" || gpu.Jobs[0].Status != "waiting" || gpu.Jobs[0].Attempt != 0 {
		test.Fatalf("a job for a label no runner has: %+v", gpu.Jobs[0])
	}
	status, data := call(test, http.MethodGet, srv.url+"/api/events?repo=acme/demo&limit=200", bearer(token), "")
	if status != http.StatusOK {
		test.Fatalf("events: HTTP %d", status)
	}
	counts := map[string]int{}
	var events []store.Event
	json.Unmarshal(data, &events)
	for _, event := range events {
		counts[event.Action]++
	}
	if counts["run.queued"] != 5 || counts["run.completed"] != 4 || counts["run.cancelled"] != 1 || counts["job.completed"] < 9 {
		test.Fatalf("events: %v", counts)
	}
}

func hostRunner(test *testing.T, srv server, token string) runnerProcess {
	test.Helper()
	var created struct {
		UUID  string `json:"uuid"`
		Token string `json:"token"`
		URL   string `json:"url"`
	}
	lyraJSON(test, srv, token, &created, "runner", "create", "--name", "host")
	runner := newRunner(test, "host")
	work := filepath.Join(filepath.Dir(runner.config), "work")
	config := "log:\n  level: debug\nrunner:\n  fetch_interval: 1s\n  capacity: 1\nhost:\n  workdir_parent: " + work + "\nserver:\n  connections:\n    lyra:\n      url: " + created.URL + "\n      uuid: " + created.UUID + "\n      token: " + created.Token + "\n      labels: ['test:host']\n"
	if err := os.WriteFile(runner.config, []byte(config), 0o644); err != nil {
		test.Fatal(err)
	}
	runner.kill = runner.start(test, "daemon")
	return runner
}

func TestJobMaintenance(test *testing.T) {
	srv := startServerAt(test, "127.0.0.1:"+freePort(test), filepath.Join(test.TempDir(), "data"), "127.0.0.1", "--job-stale-after", "3s", "--actions-retention", "10s")
	token := adminToken(test, srv)
	createRepo(test, srv, token, "acme/demo")
	src := newSource(test)
	if err := os.MkdirAll(filepath.Join(src, ".forgejo", "workflows"), 0o755); err != nil {
		test.Fatal(err)
	}
	commitFile(test, src, ".forgejo/workflows/quick.yml", "on: workflow_dispatch\njobs:\n  quick:\n    runs-on: test\n    steps:\n      - run: echo \"retention=${{ github.retention_days }}\"\n", "Quick")
	commitFile(test, src, ".forgejo/workflows/steady.yml", "on: workflow_dispatch\njobs:\n  steady:\n    runs-on: test\n    steps:\n      - run: sleep 7 && echo done\n", "Steady")
	commitFile(test, src, ".forgejo/workflows/slow.yml", "on: workflow_dispatch\njobs:\n  slow:\n    runs-on: test\n    steps:\n      - run: echo started && sleep 60\n", "Slow")
	mustGit(test, src, "-c", "http.extraHeader=Authorization: Bearer "+token, "push", "-q", srv.url+"/acme/demo.git", "main")
	runner := hostRunner(test, srv, token)

	runLyra(test, srv, token, "workflow", "run", "acme/demo", "quick.yml")
	if result := runLyra(test, srv, token, "run", "wait", "acme/demo", "1", "--timeout", "60s"); result.code != 0 {
		test.Fatalf("quick run: %+v", result)
	}
	if logs := runLyra(test, srv, token, "run", "logs", "acme/demo", "1").stdout; !strings.Contains(logs, "retention=0") {
		test.Fatalf("quick run log:\n%s", logs)
	}
	logs := filepath.Join(srv.data, "blobs", "lyra", "actions", "acme", "demo")
	if entries, _ := os.ReadDir(logs); len(entries) != 1 {
		test.Fatalf("archived logs: %v", entries)
	}

	runLyra(test, srv, token, "workflow", "run", "acme/demo", "steady.yml")
	if result := runLyra(test, srv, token, "run", "wait", "acme/demo", "2", "--timeout", "60s"); result.code != 0 {
		test.Fatalf("a job longer than --job-stale-after with a live runner: %+v", result)
	}
	runLyra(test, srv, token, "workflow", "run", "acme/demo", "slow.yml")
	eventually(test, "the slow job to start", func() bool {
		return strings.Contains(runLyra(test, srv, token, "run", "logs", "acme/demo", "3").stdout, "started")
	})
	runner.kill()
	result := runLyra(test, srv, token, "run", "wait", "acme/demo", "3", "--timeout", "30s")
	if result.code != 1 || !strings.Contains(result.stderr, "failure") {
		test.Fatalf("run whose runner died: %+v", result)
	}
	if logs := runLyra(test, srv, token, "run", "logs", "acme/demo", "3").stdout; !strings.Contains(logs, "lyra: the runner stopped reporting for 3s; the job failed") {
		test.Fatalf("log of a job whose runner died:\n%s", logs)
	}

	eventually(test, "runs past retention to be deleted", func() bool {
		return strings.TrimSpace(runLyra(test, srv, token, "run", "list", "acme/demo").stdout) == "[]"
	})
	if entries, _ := os.ReadDir(logs); len(entries) != 0 {
		test.Fatalf("archived logs after retention: %v", entries)
	}
}

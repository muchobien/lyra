package e2e

import (
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
	page := func(target string) string {
		test.Helper()
		status, body := call(test, http.MethodGet, srv.url+target, tailnet(), "")
		if status != http.StatusOK {
			test.Fatalf("%s: HTTP %d:\n%s", target, status, body)
		}
		return string(body)
	}
	expect := func(target string, wants ...string) {
		test.Helper()
		body := page(target)
		for _, want := range wants {
			if !strings.Contains(body, want) {
				test.Fatalf("%s lacks %q:\n%s", target, want, body)
			}
		}
	}
	hostRunner(test, srv, token)

	commitFile(test, src, "README.md", "# Demo\n", "First commit")
	if err := os.MkdirAll(filepath.Join(src, ".forgejo", "workflows"), 0o755); err != nil {
		test.Fatal(err)
	}
	commitFile(test, src, ".forgejo/workflows/ci.yml", uiWorkflow, "Add CI")
	mustGit(test, src, "-c", auth, "push", "-q", remote, "main")
	sha := mustGit(test, src, "rev-parse", "HEAD")

	var build string
	eventually(test, "the build job to run", func() bool {
		build = page("/acme/demo/runs/1")
		return strings.Contains(build, `id="live-log"`) && strings.Contains(build, "hello from the job")
	})
	for _, want := range []string{
		`data-url="/api/repos/acme/demo/runs/1/jobs/1/logs"`, `class="badge run-running"`, "Cancel run",
		`<a href="/acme/demo/runs/1?job=2"><span class="badge run-waiting">waiting</span> gpu</a>`,
	} {
		if !strings.Contains(build, want) {
			test.Fatalf("running run page lacks %q:\n%s", want, build)
		}
	}
	expect("/acme/demo/runs/1?job=2", "Waiting for a runner with the labels <span class=\"mono\">gpu</span>")
	eventually(test, "the build job to finish", func() bool {
		return strings.Contains(page("/acme/demo/runs/1"), `<details class="step"`)
	})
	expect("/acme/demo/runs/1",
		`<span class="badge run-success">success</span> Say hello`, "hello from the job", "bye &lt;b&gt;tag&lt;/b&gt;",
		`<details class="step"><summary><span class="badge">log</span> Set up`, `>Actions</a>`,
	)
	if strings.Contains(page("/acme/demo/runs/1"), `id="live-log"`) {
		test.Fatal("a finished job still shows the live log")
	}
	expect("/acme/demo/runs", `<strong>CI</strong> #1</a>`, `class="badge run-running"`)
	expect("/acme/demo/commits/main", `<span class="badge run-pending">pending</span>`)
	expect("/acme/demo/commit/"+sha, `<th>Checks</th>`, `<a href="/acme/demo/runs/1?job=1">build</a> <span class="badge run-success">success</span>`)

	if status, _, _ := postForm(test, srv.url+"/acme/demo/runs/1/cancel", url.Values{}, map[string]string{"Sec-Fetch-Site": "cross-site"}); status != http.StatusForbidden {
		test.Fatalf("cross-site cancel: HTTP %d", status)
	}
	status, location, body := postForm(test, srv.url+"/acme/demo/runs/1/cancel", url.Values{}, nil)
	if status != http.StatusSeeOther || location != "/acme/demo/runs/1" {
		test.Fatalf("cancel form: HTTP %d %s: %s", status, location, body)
	}
	expect("/acme/demo/runs/1", `<span class="badge run-cancelled">cancelled</span> CI`, "Rerun all jobs", "Rerun failed jobs")
	expect("/acme/demo/commits/main", `<span class="badge run-failure">failure</span>`)

	mustGit(test, src, "switch", "-q", "-c", "feature")
	commitFile(test, src, "feature.txt", "feature\n", "Add feature")
	mustGit(test, src, "-c", auth, "push", "-q", remote, "feature")
	lyraJSON(test, srv, token, &pullResult{}, "pr", "create", "acme/demo", "--head", "feature", "--title", "Feature")
	expect("/acme/demo/pulls/1", "<strong>Checks</strong>", `<a href="/acme/demo/runs/2?job=3">build</a>`)

	status, location, _ = postForm(test, srv.url+"/acme/demo/runs/1/rerun", url.Values{"failed": {"true"}}, nil)
	if status != http.StatusSeeOther || location != "/acme/demo/runs/1" {
		test.Fatalf("rerun form: HTTP %d %s", status, location)
	}
	if rerun := page("/acme/demo/runs/1"); !strings.Contains(rerun, "Cancel run") || strings.Contains(rerun, `class="badge run-cancelled">cancelled</span> CI`) {
		test.Fatalf("run after rerun --failed:\n%s", rerun)
	}
	reader := createToken(test, srv, `{"name":"reader","scope":"read"}`)
	if status, _ := call(test, http.MethodPost, srv.url+"/acme/demo/runs/1/cancel", bearer(reader), ""); status != http.StatusForbidden {
		test.Fatalf("cancel with a read token: HTTP %d", status)
	}
	if strings.Contains(func() string {
		_, body := call(test, http.MethodGet, srv.url+"/acme/demo/runs/1", bearer(reader), "")
		return string(body)
	}(), "Cancel run") {
		test.Fatal("a read token sees the cancel button")
	}
}

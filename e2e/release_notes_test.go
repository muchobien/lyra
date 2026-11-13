package e2e

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReleaseNotes(test *testing.T) {
	srv := startServer(test, "127.0.0.1")
	token := adminToken(test, srv)
	reader := createToken(test, srv, `{"name":"reader","scope":"read"}`)
	createRepo(test, srv, token, "demo")
	auth := "http.extraHeader=Authorization: Bearer " + token
	remote := srv.url + "/demo.git"
	src := newSource(test)
	push := func(args ...string) {
		test.Helper()
		mustGit(test, src, append([]string{"-c", auth, "push", "-q", remote}, args...)...)
	}
	merged := func(branch, title string) {
		test.Helper()
		mustGit(test, src, "switch", "-q", "-c", branch, "main")
		commitFile(test, src, branch+".txt", branch+"\n", title)
		push(branch)
		var created pullResult
		lyraJSON(test, srv, token, &created, "pr", "create", "demo", "--head", branch, "--title", title)
		if result := runLyra(test, srv, token, "pr", "merge", "demo", fmt.Sprint(created.Number), "--squash"); result.code != 0 {
			test.Fatalf("merge #%d: %+v", created.Number, result)
		}
		mustGit(test, src, "switch", "-q", "main")
		mustGit(test, src, "-c", auth, "pull", "-q", "--ff-only", remote, "main")
	}
	notes := func(args ...string) string {
		test.Helper()
		return runLyra(test, srv, token, append([]string{"release", "notes", "demo"}, args...)...).stdout
	}

	commitFile(test, src, "README.md", "# Demo\n", "First commit")
	push("main")
	if text := notes("v1.0.0", "--jq", ".notes"); strings.TrimSpace(text) != "No pull requests merged." {
		test.Fatalf("notes without pull requests: %q", text)
	}
	merged("one", "Add one")
	v1 := releaseJSON(test, srv, token, "create", "demo", "v1.0.0", "--generate-notes", "--notes", "Intro.")
	if v1.Notes != "Intro.\n\n## Pull requests merged\n\n- Add one (#1) by e2e-agent" {
		test.Fatalf("v1.0.0 notes: %q", v1.Notes)
	}

	merged("two", "Add two")
	releaseJSON(test, srv, token, "create", "demo", "v1.1.0-rc.1", "--prerelease")

	mustGit(test, src, "switch", "-q", "-c", "three", "main")
	commitFile(test, src, "three.txt", "three\n", "Add three")
	push("three")
	lyraJSON(test, srv, token, &pullResult{}, "pr", "create", "demo", "--head", "three", "--title", "Add three")
	mustGit(test, src, "switch", "-q", "main")
	mustGit(test, src, "merge", "-q", "--ff-only", "three")
	push("main")
	if state := strings.TrimSpace(runLyra(test, srv, token, "pr", "view", "demo", "3", "--jq", ".state").stdout); state != "merged" {
		test.Fatalf("#3 after a push to main: %s", state)
	}

	mustGit(test, src, "switch", "-q", "-c", "unmerged", "main")
	commitFile(test, src, "four.txt", "four\n", "Add four")
	push("unmerged")
	lyraJSON(test, srv, token, &pullResult{}, "pr", "create", "demo", "--head", "unmerged", "--title", "Not merged")

	if text := notes("v2.0.0", "--jq", ".notes"); text != "## Pull requests merged since v1.0.0\n\n- Add two (#2) by e2e-agent\n- Add three (#3) by e2e-agent\n" {
		test.Fatalf("v2.0.0 notes: %q", text)
	}
	if text := notes("v1.1.0-rc.2", "--prerelease", "--jq", ".previous_tag"); strings.TrimSpace(text) != "v1.1.0-rc.1" {
		test.Fatalf("previous release of a prerelease: %q", text)
	}
	if text := notes("v0.1.0", "--target", "nope"); text != "" {
		test.Fatalf("notes for an unknown target: %q", text)
	}
	if text := notes("v1.0.0", "--jq", ".pulls | map(.number)"); strings.TrimSpace(text) != "[1]" {
		test.Fatalf("notes for the existing tag v1.0.0: %q", text)
	}

	asset := filepath.Join(test.TempDir(), "demo.txt")
	if err := os.WriteFile(asset, []byte(strings.Repeat("x", 1500)), 0o644); err != nil {
		test.Fatal(err)
	}
	releaseJSON(test, srv, token, "create", "demo", "v2.0.0", "--generate-notes", "--asset", asset)
	releaseJSON(test, srv, token, "create", "demo", "v3.0.0", "--draft", "--title", "Secret draft")

	status, page := call(test, http.MethodGet, srv.url+"/demo/releases", tailnet(), "")
	text := string(page)
	if status != http.StatusOK {
		test.Fatalf("releases page: HTTP %d", status)
	}
	for _, want := range []string{
		"Secret draft", ">Draft<", ">Latest<", ">Prerelease<", "<li>Add two (#2) by e2e-agent</li>",
		`href="` + srv.url + `/demo/releases/download/v2.0.0/demo.txt">demo.txt</a>`, "1.5 KB", `href="/demo/releases/tag/v1.1.0-rc.1"`,
	} {
		if !strings.Contains(text, want) {
			test.Fatalf("releases page lacks %q:\n%s", want, text)
		}
	}
	if strings.Index(text, "Secret draft") > strings.Index(text, "v2.0.0") {
		test.Fatal("drafts aren't listed first")
	}
	if status, page := call(test, http.MethodGet, srv.url+"/demo/releases", bearer(reader), ""); status != http.StatusOK || strings.Contains(string(page), "Secret draft") {
		test.Fatalf("reader's releases page: HTTP %d, shows the draft: %v", status, strings.Contains(string(page), "Secret draft"))
	}
	if status, page := call(test, http.MethodGet, srv.url+"/demo/releases/tag/v1.0.0", bearer(reader), ""); status != http.StatusOK || !strings.Contains(string(page), "<p>Intro.</p>") || strings.Contains(string(page), "v2.0.0") {
		test.Fatalf("release page: HTTP %d:\n%s", status, page)
	}
	if status, _ := call(test, http.MethodGet, srv.url+"/demo/releases/tag/v3.0.0", bearer(reader), ""); status != http.StatusNotFound {
		test.Fatalf("reader's page for a draft: HTTP %d", status)
	}
	if status, page := call(test, http.MethodGet, srv.url+"/demo", tailnet(), ""); status != http.StatusOK || !strings.Contains(string(page), `href="/demo/releases">Releases</a>`) {
		test.Fatalf("repo page lacks the Releases tab: HTTP %d", status)
	}
}

func releaseJSON(test *testing.T, srv server, token string, args ...string) releaseResult {
	test.Helper()
	var result releaseResult
	lyraJSON(test, srv, token, &result, append([]string{"release"}, args...)...)
	return result
}

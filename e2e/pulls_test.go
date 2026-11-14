package e2e

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/muchobien/lyra/internal/repos"
	"github.com/muchobien/lyra/internal/store"
)

type pullResult struct {
	store.Pull
	EventID  int64           `json:"event_id"`
	Comments []store.Comment `json:"comments"`
}

func TestPulls(test *testing.T) {
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
	pr := func(args ...string) pullResult {
		test.Helper()
		var result pullResult
		lyraJSON(test, srv, token, &result, append([]string{"pr"}, args...)...)
		return result
	}
	issueState := func(number string) string {
		test.Helper()
		return runLyra(test, srv, token, "issue", "view", "acme/demo", number, "--jq", ".state").stdout
	}

	commitFile(test, src, "README.md", "# Demo\n", "First commit")
	commitFile(test, src, "notes.txt", "one\ntwo\nthree\nfour\nfive\n", "Add notes")
	push("main")
	lyraJSON(test, srv, token, &issueResult{}, "issue", "create", "acme/demo", "--title", "Feature request")
	mustGit(test, src, "branch", "stale")
	push("stale")

	mustGit(test, src, "switch", "-q", "-c", "feature")
	commitFile(test, src, "feature.txt", "feature\n", "Add feature")
	if err := os.MkdirAll(filepath.Join(src, "docs"), 0o755); err != nil {
		test.Fatal(err)
	}
	mustGit(test, src, "mv", "notes.txt", "docs/notes.txt")
	commitFile(test, src, "README.md", "# Demo\n\nNow with a feature.\n", "Move notes and extend README")
	push("feature")
	mainSHA := mustGit(test, src, "rev-parse", "main")
	featureSHA := mustGit(test, src, "rev-parse", "feature")

	created := pr("create", "acme/demo", "--head", "feature", "--title", "Add feature", "--body", "Fixes #1", "--assignee", "@me")
	if created.Number != 2 || created.Kind != "pr" || created.Base != "main" || created.Head != "feature" || created.HeadSHA != featureSHA || created.BaseSHA != mainSHA || created.EventID == 0 || fmt.Sprint(created.Assignees) != "[e2e-agent]" {
		test.Errorf("pr create = %+v", created)
	}
	for _, args := range [][]string{
		{"pr", "create", "acme/demo", "--head", "missing", "--title", "Missing branch"},
		{"pr", "create", "acme/demo", "--head", "main", "--title", "Same branch"},
		{"pr", "create", "acme/demo", "--head", "feature", "--title", "Duplicate"},
		{"pr", "create", "acme/demo", "--head", "stale", "--title", "Nothing to merge"},
		{"pr", "view", "acme/demo", "1"},
		{"issue", "view", "acme/demo", "2"},
	} {
		if result := runLyra(test, srv, token, args...); result.code != 1 {
			test.Errorf("lyra %s: exit %d, want 1: %s", strings.Join(args, " "), result.code, result.stderr)
		}
	}
	if result := runLyra(test, srv, token, "pr", "create", "acme/demo", "--title", "No head"); result.code != 2 {
		test.Errorf("pr create without --head: exit %d, want 2", result.code)
	}
	if got := runLyra(test, srv, token, "pr", "list", "acme/demo", "--jq", `[.[].number] | map(tostring) | join(",")`).stdout; got != "2\n" {
		test.Errorf("pr list %q", got)
	}
	if got := runLyra(test, srv, token, "issue", "list", "acme/demo", "--jq", `[.[].number] | map(tostring) | join(",")`).stdout; got != "1\n" {
		test.Errorf("issue list %q", got)
	}
	if got := mustGit(test, src, "-c", auth, "ls-remote", remote, "refs/pull/2/head"); !strings.HasPrefix(got, featureSHA) {
		test.Errorf("refs/pull/2/head = %q, want %s", got, featureSHA)
	}

	var commits []repos.Commit
	lyraJSON(test, srv, token, &commits, "pr", "commits", "acme/demo", "2")
	if len(commits) != 2 || commits[0].Subject != "Move notes and extend README" || commits[1].Subject != "Add feature" || commits[0].Signature.Status != "unsigned" {
		test.Errorf("pr commits = %+v", commits)
	}
	var files []repos.FileChange
	lyraJSON(test, srv, token, &files, "pr", "diff", "acme/demo", "2", "--files")
	want := []repos.FileChange{
		{Path: "README.md", Status: "modified", Additions: 2},
		{Path: "docs/notes.txt", OldPath: "notes.txt", Status: "renamed"},
		{Path: "feature.txt", Status: "added", Additions: 1},
	}
	if fmt.Sprint(files) != fmt.Sprint(want) {
		test.Errorf("pr files = %+v, want %+v", files, want)
	}
	if diff := runLyra(test, srv, token, "pr", "diff", "acme/demo", "2").stdout; !strings.Contains(diff, "+feature") || !strings.Contains(diff, "rename to docs/notes.txt") {
		test.Errorf("pr diff:\n%s", diff)
	}

	commitFile(test, src, "feature.txt", "feature\nmore\n", "Extend feature")
	output := push("feature")
	newHead := mustGit(test, src, "rev-parse", "feature")
	if !strings.Contains(output, "lyra: updated pull request #2") {
		test.Errorf("push to the head branch printed:\n%s", output)
	}
	var synced store.Event
	lyraJSON(test, srv, token, &synced, "event", "wait", "--since", fmt.Sprint(created.EventID), "--action", "pr.synchronized", "--timeout", "5s")
	if synced.Item != 2 || !strings.Contains(string(synced.Data), newHead) || !strings.Contains(string(synced.Data), featureSHA) {
		test.Errorf("pr.synchronized = %+v %s", synced, synced.Data)
	}
	if got := mustGit(test, src, "-c", auth, "ls-remote", remote, "refs/pull/2/head"); !strings.HasPrefix(got, newHead) {
		test.Errorf("refs/pull/2/head after push = %q, want %s", got, newHead)
	}
	if viewed := pr("view", "acme/demo", "2"); viewed.HeadSHA != newHead {
		test.Errorf("head_sha after push = %s, want %s", viewed.HeadSHA, newHead)
	}
	lyraJSON(test, srv, token, &struct{}{}, "pr", "comment", "acme/demo", "2", "--body", "Ready to merge")

	mustGit(test, src, "switch", "-q", "main")
	mustGit(test, src, "merge", "-q", "--ff-only", "feature")
	output = push("main")
	if !strings.Contains(output, "lyra: merged pull request #2") || !strings.Contains(output, "lyra: closed issue #1") {
		test.Errorf("push of the merge printed:\n%s", output)
	}
	merged := pr("view", "acme/demo", "2")
	if merged.State != "merged" || merged.MergeSHA != newHead || merged.MergedAt == nil || merged.ClosedAt == nil || merged.BaseSHA != mainSHA || len(merged.Comments) != 1 {
		test.Errorf("merged pr = %+v", merged)
	}
	if state := issueState("1"); state != "closed\n" {
		test.Errorf("issue 1 after merging a PR with Fixes #1: %q", state)
	}
	lyraJSON(test, srv, token, &files, "pr", "diff", "acme/demo", "2", "--files")
	if len(files) != 3 {
		test.Errorf("files of the merged pr = %+v", files)
	}
	if result := runLyra(test, srv, token, "pr", "close", "acme/demo", "2"); result.code != 1 || !strings.Contains(result.stderr, "is merged") {
		test.Errorf("closing a merged pr: exit %d %s", result.code, result.stderr)
	}

	lyraJSON(test, srv, token, &issueResult{}, "issue", "create", "acme/demo", "--title", "Crash")
	commitFile(test, src, "fix.txt", "fixed\n", "Fix the crash\n\nCloses #3, fixes #99 and resolves #2")
	if output := push("main"); !strings.Contains(output, "lyra: closed issue #3") {
		test.Errorf("push with Closes #3 printed:\n%s", output)
	}
	var closedBy store.Event
	lyraJSON(test, srv, token, &closedBy, "event", "list", "--repo", "acme/demo", "--jq", `[.[] | select(.action == "issue.closed" and .item == 3)][0]`)
	if issueState("3") != "closed\n" || !strings.Contains(string(closedBy.Data), mustGit(test, src, "rev-parse", "HEAD")) {
		test.Errorf("issue 3 after Closes #3: %q, event %s", issueState("3"), closedBy.Data)
	}

	mustGit(test, src, "switch", "-q", "-c", "acme/other")
	commitFile(test, src, "other.txt", "other\n", "Other change")
	push("acme/other")
	other := pr("create", "acme/demo", "--head", "acme/other", "--title", "Other change")
	if other.Number != 4 {
		test.Errorf("other pr number %d, want 4", other.Number)
	}
	if closed := pr("close", "acme/demo", "4"); closed.State != "closed" || closed.Head != "acme/other" {
		test.Errorf("pr close = %+v", closed)
	}
	commitFile(test, src, "other.txt", "other\nagain\n", "Change while closed")
	if output := push("acme/other"); strings.Contains(output, "#4") {
		test.Errorf("push to a closed pr's branch printed:\n%s", output)
	}
	if reopened := pr("reopen", "acme/demo", "4"); reopened.State != "open" || reopened.HeadSHA != mustGit(test, src, "rev-parse", "HEAD") || reopened.HeadSHA == other.HeadSHA {
		test.Errorf("pr reopen did not resync the head: %+v", reopened)
	}
	if ref := mustGit(test, src, "-c", "http.extraHeader=Authorization: Bearer "+token, "ls-remote", remote, "refs/pull/4/head"); !strings.HasPrefix(ref, mustGit(test, src, "rev-parse", "HEAD")) {
		test.Errorf("refs/pull/4/head after reopen: %s", ref)
	}
	mustGit(test, src, "switch", "-q", "-c", "gone", "main")
	commitFile(test, src, "gone.txt", "gone\n", "Short-lived")
	push("gone")
	pr("create", "acme/demo", "--head", "gone", "--title", "Gone")
	pr("close", "acme/demo", "5")
	push("--delete", "gone")
	if result := runLyra(test, srv, token, "pr", "reopen", "acme/demo", "5"); result.code != 1 || !strings.Contains(result.stderr, "branch gone no longer exists") {
		test.Errorf("reopen without its branch: %+v", result)
	}
	mustGit(test, src, "switch", "-q", "acme/other")

	push("--delete", "feature")
	lyraJSON(test, srv, token, &commits, "pr", "commits", "acme/demo", "2")
	if len(commits) != 3 {
		test.Errorf("commits of the merged pr after deleting its branch: %d", len(commits))
	}

	for target, wants := range map[string][]string{
		"/api/repos/acme/demo/pulls":               {"Other change", `"head":"acme/other"`, `"base":"main"`},
		"/api/repos/acme/demo/pulls?state=merged":  {"Add feature", `"state":"merged"`},
		"/api/repos/acme/demo/pulls/2/view":        {`"state":"merged"`, "feature", "Fixes #1"},
		"/api/repos/acme/demo/pulls/2/commits":     {"Extend feature", `"status":"unsigned"`},
		"/api/repos/acme/demo/pulls/2/patch":       {`"head_sha":"`, `"old_path":"notes.txt"`, `"path":"docs/notes.txt"`, `"kind":"add"`},
		"/api/repos/acme/demo/issues?state=closed": {"Feature request", "Crash"},
	} {
		status, page := call(test, http.MethodGet, srv.url+target, tailnet(), "")
		for _, text := range wants {
			if status != http.StatusOK || !strings.Contains(compactJSON(test, page), text) {
				test.Errorf("GET %s: HTTP %d, missing %q", target, status, text)
			}
		}
	}
	if status, body := browserCall(test, http.MethodPost, srv.url+"/api/repos/acme/demo/pulls/4/comments", `{"body":"From the UI"}`, map[string]string{"Sec-Fetch-Site": "same-origin"}); status != http.StatusCreated {
		test.Errorf("UI pr comment: HTTP %d %s", status, body)
	}
	if status, _ := call(test, http.MethodGet, srv.url+"/acme/demo/pulls/1", tailnet(), ""); status != http.StatusNotFound {
		test.Errorf("UI pr page for an issue: HTTP %d, want 404", status)
	}

	reader := createToken(test, srv, `{"name":"reader","scope":"read","repos":["acme/demo"]}`)
	if result := runLyra(test, srv, reader, "pr", "create", "acme/demo", "--head", "acme/other", "--title", "Nope"); result.code != 1 || !strings.Contains(result.stderr, "lacks write") {
		test.Errorf("read token pr create: exit %d %s", result.code, result.stderr)
	}
}

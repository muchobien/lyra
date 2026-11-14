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

type mergeView struct {
	store.Pull
	EventID int64 `json:"event_id"`
	Merge   *struct {
		HeadSHA     string   `json:"head_sha"`
		FastForward bool     `json:"fast_forward"`
		Conflicts   []string `json:"conflicts"`
	} `json:"merge"`
}

type signingKey struct {
	PublicKey      string `json:"public_key"`
	Fingerprint    string `json:"fingerprint"`
	AllowedSigners string `json:"allowed_signers"`
}

func TestMerge(test *testing.T) {
	srv := startServer(test, "127.0.0.1")
	token := adminToken(test, srv)
	createRepo(test, srv, token, "acme/demo")
	auth := "http.extraHeader=Authorization: Bearer " + token
	remote := srv.url + "/acme/demo.git"
	src := newSource(test)
	push := func(args ...string) {
		test.Helper()
		mustGit(test, src, append([]string{"-c", auth, "push", "-q", remote}, args...)...)
	}
	commitAs := func(author, name, content, message string) string {
		test.Helper()
		if err := os.WriteFile(filepath.Join(src, name), []byte(content), 0o644); err != nil {
			test.Fatal(err)
		}
		mustGit(test, src, "add", name)
		mustGit(test, src, "commit", "-q", "--author", author, "-m", message)
		return mustGit(test, src, "rev-parse", "HEAD")
	}
	view := func(number string) mergeView {
		test.Helper()
		var result mergeView
		lyraJSON(test, srv, token, &result, "pr", "view", "acme/demo", number)
		return result
	}
	merge := func(args ...string) cliResult {
		test.Helper()
		return runLyra(test, srv, token, append([]string{"pr", "merge", "acme/demo"}, args...)...)
	}
	remoteSHA := func(ref string) string {
		test.Helper()
		return strings.Fields(mustGit(test, src, "-c", auth, "ls-remote", remote, ref) + " none")[0]
	}

	var key signingKey
	lyraJSON(test, srv, token, &key, "signing-key")
	keyFile := filepath.Join(srv.data, "lyra_signing_key")
	if info, err := os.Stat(keyFile); err != nil || info.Mode().Perm() != 0o600 {
		test.Fatalf("generated key file: %v %v", info, err)
	}
	if fingerprint(test, keyFile) != key.Fingerprint || !strings.HasPrefix(key.AllowedSigners, `lyra namespaces="git" ssh-ed25519 `) {
		test.Errorf("signing-key = %+v", key)
	}

	commitFile(test, src, "README.md", "# Demo\n", "First commit")
	commitFile(test, src, "shared.txt", "shared\n", "Add shared file")
	push("main")
	lyraJSON(test, srv, token, &issueResult{}, "issue", "create", "acme/demo", "--title", "Squash me")

	mustGit(test, src, "switch", "-q", "-c", "squashme")
	commitAs("Ana <ana@example.com>", "ana.txt", "ana\n", "Ana's part")
	squashHead := commitAs("Bob <bob@example.com>", "bob.txt", "bob\n", "Bob's part")
	push("squashme")
	lyraJSON(test, srv, token, &pullResult{}, "pr", "create", "acme/demo", "--head", "squashme", "--title", "Squash two authors", "--body", "Fixes #1")
	mustGit(test, src, "switch", "-q", "main")
	mainTip := commitAs("Test Agent <agent@example.com>", "main.txt", "main\n", "Main moves on")
	push("main")

	if plan := view("2").Merge; plan == nil || plan.FastForward || len(plan.Conflicts) != 0 || plan.HeadSHA != squashHead {
		test.Errorf("merge plan for #2 = %+v", plan)
	}
	for _, args := range [][]string{{"2"}, {"2", "--squash", "--ff"}} {
		if result := merge(args...); result.code != 2 {
			test.Errorf("pr merge %v: exit %d, want 2", args, result.code)
		}
	}
	if result := merge("2", "--ff"); result.code != 1 || !strings.Contains(result.stderr, "HTTP 409") {
		test.Errorf("ff merge after main moved: exit %d %s", result.code, result.stderr)
	}
	if result := merge("2", "--squash", "--head", mainTip); result.code != 1 || !strings.Contains(result.stderr, "HTTP 409") {
		test.Errorf("squash with the wrong --head: exit %d %s", result.code, result.stderr)
	}
	reader := createToken(test, srv, `{"name":"reader","scope":"read","repos":["acme/demo"]}`)
	if result := runLyra(test, srv, reader, "pr", "merge", "acme/demo", "2", "--squash"); result.code != 1 || !strings.Contains(result.stderr, "lacks write") {
		test.Errorf("read token merge: exit %d %s", result.code, result.stderr)
	}

	var merged mergeView
	lyraJSON(test, srv, token, &merged, "pr", "merge", "acme/demo", "2", "--squash", "--head", squashHead, "--delete-branch")
	squashSHA := remoteSHA("refs/heads/main")
	if merged.State != "merged" || merged.MergeSHA != squashSHA || merged.EventID == 0 || merged.Merge != nil {
		test.Errorf("squash merge = %+v, main at %s", merged, squashSHA)
	}
	mustGit(test, src, "-c", auth, "fetch", "-q", remote, "main")
	if parent := mustGit(test, src, "rev-parse", "FETCH_HEAD^"); parent != mainTip {
		test.Errorf("squash parent %s, want %s", parent, mainTip)
	}
	if files := mustGit(test, src, "ls-tree", "--name-only", "FETCH_HEAD"); files != "README.md\nana.txt\nbob.txt\nmain.txt\nshared.txt" {
		test.Errorf("squash tree:\n%s", files)
	}
	if identities := mustGit(test, src, "log", "-1", "--format=%an <%ae>|%cn <%ce>", "FETCH_HEAD"); identities != "Ana <ana@example.com>|Lyra <lyra@127.0.0.1>" {
		test.Errorf("squash author|committer = %s", identities)
	}
	if message := mustGit(test, src, "log", "-1", "--format=%B", "FETCH_HEAD"); message != "Squash two authors (#2)\n\nFixes #1\n\nCo-authored-by: Bob <bob@example.com>" {
		test.Errorf("squash message:\n%s", message)
	}
	var commits []repos.Commit
	lyraJSON(test, srv, token, &commits, "commit", "list", "acme/demo", "--limit", "1")
	if signature := commits[0].Signature; signature.Status != "verified" || signature.KeyUser != "lyra" || signature.Fingerprint != key.Fingerprint {
		test.Errorf("squash commit signature = %+v", signature)
	}
	allowedSigners := filepath.Join(test.TempDir(), "allowed_signers")
	if err := os.WriteFile(allowedSigners, []byte(key.AllowedSigners+"\n"), 0o644); err != nil {
		test.Fatal(err)
	}
	if output, err := gitCommand(src, nil, "-c", "gpg.ssh.allowedSignersFile="+allowedSigners, "verify-commit", "FETCH_HEAD"); err != nil {
		test.Errorf("git verify-commit of the squash commit: %v\n%s", err, output)
	}
	if state := runLyra(test, srv, token, "issue", "view", "acme/demo", "1", "--jq", ".state").stdout; state != "closed\n" {
		test.Errorf("issue 1 after squash with Fixes #1: %q", state)
	}
	if remoteSHA("refs/heads/squashme") != "none" || remoteSHA("refs/pull/2/head") != squashHead {
		test.Errorf("after --delete-branch: squashme %s, refs/pull/2/head %s", remoteSHA("refs/heads/squashme"), remoteSHA("refs/pull/2/head"))
	}
	if actions := runLyra(test, srv, token, "event", "list", "--repo", "acme/demo", "--limit", "3", "--jq", `[.[] | .action + ":" + (.data.method // "")] | join(",")`).stdout; actions != "issue.closed:,push:squash,pr.merged:squash\n" {
		test.Errorf("events after the squash merge %q", actions)
	}
	if result := merge("2", "--squash"); result.code != 1 || !strings.Contains(result.stderr, "is merged") {
		test.Errorf("merging twice: exit %d %s", result.code, result.stderr)
	}

	mustGit(test, src, "switch", "-q", "--detach", squashSHA)
	mustGit(test, src, "switch", "-q", "-c", "fast")
	fastHead := commitAs("Test Agent <agent@example.com>", "fast.txt", "fast\n", "Fast change")
	push("fast")
	lyraJSON(test, srv, token, &pullResult{}, "pr", "create", "acme/demo", "--head", "fast", "--title", "Fast-forward")
	if plan := view("3").Merge; plan == nil || !plan.FastForward {
		test.Errorf("merge plan for #3 = %+v", plan)
	}
	lyraJSON(test, srv, token, &merged, "pr", "merge", "acme/demo", "3", "--ff")
	if merged.State != "merged" || merged.MergeSHA != fastHead || remoteSHA("refs/heads/main") != fastHead || remoteSHA("refs/heads/fast") != fastHead {
		test.Errorf("ff merge = %+v, main at %s", merged, remoteSHA("refs/heads/main"))
	}

	mustGit(test, src, "switch", "-q", "-c", "clash")
	commitAs("Test Agent <agent@example.com>", "shared.txt", "clash\n", "Change shared on clash")
	push("clash")
	mustGit(test, src, "switch", "-q", "-c", "ui")
	mustGit(test, src, "reset", "-q", "--hard", fastHead)
	commitAs("Test Agent <agent@example.com>", "ui.txt", "ui\n", "Change from the UI")
	push("ui")
	mustGit(test, src, "switch", "-q", "--detach", fastHead)
	push("HEAD:refs/heads/main")
	commitAs("Test Agent <agent@example.com>", "shared.txt", "main\n", "Change shared on main")
	push("HEAD:refs/heads/main")
	lyraJSON(test, srv, token, &pullResult{}, "pr", "create", "acme/demo", "--head", "clash", "--title", "Clashing change")
	if plan := view("4").Merge; plan == nil || fmt.Sprint(plan.Conflicts) != "[shared.txt]" {
		test.Errorf("merge plan for #4 = %+v", plan)
	}
	if result := merge("4", "--squash"); result.code != 1 || !strings.Contains(result.stderr, "conflicts in shared.txt") {
		test.Errorf("squash with conflicts: exit %d %s", result.code, result.stderr)
	}
	status, page := call(test, http.MethodGet, srv.url+"/api/repos/acme/demo/pulls/4/view", tailnet(), "")
	if status != http.StatusOK || !strings.Contains(compactJSON(test, page), `"conflicts":["shared.txt"]`) {
		test.Errorf("conflicting pr page: HTTP %d", status)
	}

	lyraJSON(test, srv, token, &pullResult{}, "pr", "create", "acme/demo", "--head", "ui", "--title", "Merge from the UI")
	uiHead := remoteSHA("refs/heads/ui")
	status, page = call(test, http.MethodGet, srv.url+"/api/repos/acme/demo/pulls/5/view", tailnet(), "")
	if status != http.StatusOK || !strings.Contains(compactJSON(test, page), `"head_sha":"`+uiHead+`"`) {
		test.Errorf("mergeable pr page: HTTP %d", status)
	}
	if status, body := browserCall(test, http.MethodPost, srv.url+"/api/repos/acme/demo/pulls/5/merge", `{"method":"squash","head_sha":"`+uiHead+`"}`, map[string]string{"Sec-Fetch-Site": "same-origin"}); status != http.StatusOK {
		test.Errorf("UI merge: HTTP %d %s", status, body)
	}
	if uiMerged := view("5"); uiMerged.State != "merged" || uiMerged.MergeSHA != remoteSHA("refs/heads/main") {
		test.Errorf("pr merged from the UI = %+v", uiMerged)
	}

	srv.stop()
	restarted := startServerAt(test, srv.address, srv.data, "127.0.0.1")
	var again signingKey
	lyraJSON(test, restarted, token, &again, "signing-key")
	if again.Fingerprint != key.Fingerprint {
		test.Errorf("fingerprint after restart %s, want %s", again.Fingerprint, key.Fingerprint)
	}
	restarted.stop()
	rotated := newKey(test, "rotated")
	restarted = startServerAt(test, srv.address, srv.data, "127.0.0.1", "--signing-key-file", rotated)
	lyraJSON(test, restarted, token, &again, "signing-key")
	if again.Fingerprint != fingerprint(test, rotated) {
		test.Errorf("fingerprint with --signing-key-file %s, want %s", again.Fingerprint, fingerprint(test, rotated))
	}
	lyraJSON(test, restarted, token, &commits, "commit", "list", "acme/demo", "--ref", squashSHA, "--limit", "1")
	if commits[0].Signature.Status != "verified" {
		test.Errorf("squash commit after rotating the key: %+v", commits[0].Signature)
	}
}

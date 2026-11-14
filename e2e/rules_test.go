package e2e

import (
	"net/http"
	"strings"
	"testing"

	"github.com/muchobien/lyra/internal/repos"
	"github.com/muchobien/lyra/internal/store"
)

func TestRules(test *testing.T) {
	srv := startServer(test, "127.0.0.1")
	token := adminToken(test, srv)
	createRepo(test, srv, token, "acme/demo")
	auth := "http.extraHeader=Authorization: Bearer " + token
	remote := srv.url + "/acme/demo.git"
	src := newSource(test)
	agentKey := newKey(test, "agent")
	strangerKey := newKey(test, "stranger")
	lyraJSON(test, srv, token, &store.Key{}, "key", "add", "--signing", "--agent", "claude", agentKey+".pub")
	signed := func(key string) []string {
		return append(signedWith(key), "-c", "commit.gpgsign=true")
	}
	push := func(args ...string) (string, error) {
		test.Helper()
		return gitCommand(src, nil, append([]string{"-c", auth, "push", remote}, args...)...)
	}
	remoteSHA := func(ref string) string {
		test.Helper()
		return strings.Fields(mustGit(test, src, "-c", auth, "ls-remote", remote, ref) + " none")[0]
	}
	type planView struct {
		Merge *struct {
			Methods []string `json:"methods"`
			Blocked []string `json:"blocked"`
		} `json:"merge"`
	}
	plan := func(number string) planView {
		test.Helper()
		var view planView
		lyraJSON(test, srv, token, &view, "pr", "view", "acme/demo", number)
		return view
	}

	commitFile(test, src, "README.md", "# Demo\n", "Unsigned start")
	if output, err := push("main"); err != nil {
		test.Fatalf("push before any rule: %v\n%s", err, output)
	}

	var rule store.Rule
	lyraJSON(test, srv, token, &rule, "rule", "set", "acme/demo", "main", "--require-signed", "--merge-methods", "squash")
	if rule.Pattern != "main" || !rule.RequireSigned || strings.Join(rule.MergeMethods, ",") != "squash" {
		test.Errorf("rule set = %+v", rule)
	}
	lyraJSON(test, srv, token, &rule, "rule", "set", "acme/demo", "release/*", "--require-signed")
	if strings.Join(rule.MergeMethods, ",") != "ff,squash" {
		test.Errorf("default merge methods = %v", rule.MergeMethods)
	}
	writer := createToken(test, srv, `{"name":"writer","scope":"write","repos":["acme/demo"]}`)
	for _, check := range []struct {
		token string
		args  []string
		want  string
	}{
		{token, []string{"rule", "set", "acme/demo", "["}, "pattern must be"},
		{token, []string{"rule", "set", "acme/demo", "dev", "--merge-methods", "rebase"}, `one of \"ff, squash\"`},
		{token, []string{"rule", "delete", "acme/demo", "missing"}, "no rule for missing"},
		{writer, []string{"rule", "set", "acme/demo", "dev"}, "lacks admin"},
	} {
		if result := runLyra(test, srv, check.token, check.args...); result.code != 1 || !strings.Contains(result.stderr, check.want) {
			test.Errorf("lyra %s: exit %d %s", strings.Join(check.args, " "), result.code, result.stderr)
		}
	}
	if got := runLyra(test, srv, writer, "rule", "list", "acme/demo", "--jq", `[.[].pattern] | join(",")`).stdout; got != "main,release/*\n" {
		test.Errorf("rule list %q", got)
	}

	start := remoteSHA("refs/heads/main")
	commitFile(test, src, "unsigned.txt", "x\n", "Unsigned change")
	output, err := push("main")
	if err == nil || !strings.Contains(output, "lyra: branch rule main requires verified signatures on main") || !strings.Contains(output, "(Unsigned change) is unsigned") {
		test.Errorf("unsigned push to main: %v\n%s", err, output)
	}
	mustGit(test, src, "reset", "-q", "--hard", start)
	commitFile(test, src, "stranger.txt", "x\n", "Signed by a stranger", signed(strangerKey)...)
	if output, err := push("main"); err == nil || !strings.Contains(output, "is unverified") {
		test.Errorf("push signed by an unregistered key: %v\n%s", err, output)
	}
	mustGit(test, src, "reset", "-q", "--hard", start)
	if output, err := gitCommand(src, nil, "-c", "http.extraHeader=Tailscale-User-Login: "+login, "push", remote, "HEAD~0:refs/heads/main", "--force"); err != nil && !strings.Contains(output, "Everything up-to-date") {
		test.Fatalf("no-op push: %v\n%s", err, output)
	}
	commitFile(test, src, "tailnet.txt", "x\n", "Unsigned over the tailnet")
	if output, err := gitCommand(src, nil, "-c", "http.extraHeader=Tailscale-User-Login: "+login, "push", remote, "main"); err == nil || !strings.Contains(output, "requires verified signatures") {
		test.Errorf("unsigned push with the admin tailnet identity: %v\n%s", err, output)
	}
	mustGit(test, src, "reset", "-q", "--hard", start)
	commitFile(test, src, "agent.txt", "x\n", "Signed by the agent", signed(agentKey)...)
	if output, err := push("main"); err != nil {
		test.Errorf("push signed by the agent: %v\n%s", err, output)
	}
	if remoteSHA("refs/heads/main") != mustGit(test, src, "rev-parse", "HEAD") {
		test.Errorf("main did not move to the signed commit")
	}
	if output, err := push("main:refs/heads/release/1"); err == nil || !strings.Contains(output, "(Unsigned start) is unsigned") {
		test.Errorf("creating release/1 on a history with unsigned commits: %v\n%s", err, output)
	}

	mustGit(test, src, "switch", "-q", "-c", "feature")
	commitFile(test, src, "feature.txt", "x\n", "Unsigned feature")
	if output, err := push("feature"); err != nil {
		test.Fatalf("push to a branch without rules: %v\n%s", err, output)
	}
	lyraJSON(test, srv, token, &pullResult{}, "pr", "create", "acme/demo", "--head", "feature", "--title", "Feature")
	if view := plan("1"); view.Merge == nil || strings.Join(view.Merge.Methods, ",") != "squash" || len(view.Merge.Blocked) != 1 || !strings.Contains(view.Merge.Blocked[0], "(Unsigned feature) is unsigned") {
		test.Errorf("plan with an unsigned commit = %+v", view.Merge)
	}
	if result := runLyra(test, srv, token, "pr", "merge", "acme/demo", "1", "--squash"); result.code != 1 || !strings.Contains(result.stderr, "HTTP 409") || !strings.Contains(result.stderr, "requires verified signatures") {
		test.Errorf("squash of an unsigned commit: exit %d %s", result.code, result.stderr)
	}
	status, page := call(test, http.MethodGet, srv.url+"/api/repos/acme/demo/pulls/1/view", tailnet(), "")
	if status != http.StatusOK || !strings.Contains(string(page), "is unsigned") || !strings.Contains(compactJSON(test, page), `"methods":["squash"]`) {
		test.Errorf("blocked pr page: HTTP %d", status)
	}

	mustGit(test, src, "reset", "-q", "--hard", "main")
	commitFile(test, src, "feature.txt", "signed\n", "Signed feature", signed(agentKey)...)
	if output, err := push("--force", "feature"); err != nil {
		test.Fatalf("force push of the signed feature: %v\n%s", err, output)
	}
	if view := plan("1"); view.Merge == nil || len(view.Merge.Blocked) != 0 {
		test.Errorf("plan with signed commits = %+v", view.Merge)
	}
	if result := runLyra(test, srv, token, "pr", "merge", "acme/demo", "1", "--ff"); result.code != 1 || !strings.Contains(result.stderr, "allow only squash") {
		test.Errorf("ff merge where only squash is allowed: exit %d %s", result.code, result.stderr)
	}
	var merged pullResult
	lyraJSON(test, srv, token, &merged, "pr", "merge", "acme/demo", "1", "--squash")
	var commits []repos.Commit
	lyraJSON(test, srv, token, &commits, "commit", "list", "acme/demo", "--limit", "1")
	if merged.State != "merged" || commits[0].SHA != merged.MergeSHA || commits[0].Signature.Status != "verified" || commits[0].Signature.KeyUser != "lyra" {
		test.Errorf("squash under the rule: %+v, commit %+v", merged.State, commits[0].Signature)
	}

	if result := runLyra(test, srv, token, "rule", "delete", "acme/demo", "release/*"); result.code != 0 {
		test.Fatalf("rule delete: %s", result.stderr)
	}
	if output, err := push("main:refs/heads/release/1"); err != nil {
		test.Errorf("creating release/1 after deleting its rule: %v\n%s", err, output)
	}
	if actions := runLyra(test, srv, token, "event", "list", "--repo", "acme/demo", "--jq", `[.[] | select(.action | startswith("rule.")) | .action] | join(",")`).stdout; actions != "rule.delete,rule.set,rule.set\n" {
		test.Errorf("rule events %q", actions)
	}

	mustGit(test, src, "switch", "-q", "-c", "guarded", "main")
	if output, err := push("guarded"); err != nil {
		test.Fatalf("push guarded: %v: %s", err, output)
	}
	if result := runLyra(test, srv, token, "rule", "set", "acme/demo", "guarded"); result.code != 0 || !strings.Contains(result.stdout, `"allow_force_push": false`) {
		test.Fatalf("rule set guarded: %+v", result)
	}
	commitFile(test, src, "guarded.txt", "one\n", "Fast-forward")
	if output, err := push("guarded"); err != nil {
		test.Errorf("fast-forward push to a protected branch: %v: %s", err, output)
	}
	mustGit(test, src, "reset", "-q", "--hard", "HEAD~1")
	commitFile(test, src, "guarded.txt", "rewritten\n", "Rewrite")
	if output, err := push("+guarded"); err == nil || !strings.Contains(output, "refuses force pushes to guarded") {
		test.Errorf("force push to a protected branch: %v: %s", err, output)
	}
	if output, err := push("--delete", "guarded"); err == nil || !strings.Contains(output, "protects guarded from deletion") {
		test.Errorf("deleting a protected branch: %v: %s", err, output)
	}
	if result := runLyra(test, srv, token, "rule", "set", "acme/demo", "guarded", "--allow-force-push", "--allow-deletion"); result.code != 0 {
		test.Fatalf("rule set with allows: %+v", result)
	}
	if output, err := push("+guarded"); err != nil {
		test.Errorf("force push once allowed: %v: %s", err, output)
	}
	if output, err := push("--delete", "guarded"); err != nil {
		test.Errorf("delete once allowed: %v: %s", err, output)
	}

	mustGit(test, src, "switch", "-q", "-c", "target", "main")
	push("target")
	mustGit(test, src, "switch", "-q", "-c", "keep", "target")
	commitFile(test, src, "keep.txt", "keep\n", "Keep me")
	push("keep")
	lyraJSON(test, srv, token, &pullResult{}, "pr", "create", "acme/demo", "--head", "keep", "--base", "target", "--title", "Keep")
	runLyra(test, srv, token, "rule", "set", "acme/demo", "keep")
	number := strings.TrimSpace(runLyra(test, srv, token, "pr", "list", "acme/demo", "--jq", `.[] | select(.head == "keep") | .number`).stdout)
	if result := runLyra(test, srv, token, "pr", "merge", "acme/demo", number, "--squash", "--delete-branch"); result.code != 1 || !strings.Contains(result.stderr, "protects keep from deletion") {
		test.Errorf("merge deleting a protected head: %+v", result)
	}
	if result := runLyra(test, srv, token, "pr", "merge", "acme/demo", number, "--squash"); result.code != 0 {
		test.Errorf("merge keeping the protected head: %+v", result)
	}
}

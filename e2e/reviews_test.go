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

type reviewView struct {
	store.Pull
	Merge *struct {
		Blocked []string `json:"blocked"`
	} `json:"merge"`
	Reviews  []store.Review  `json:"reviews"`
	Comments []store.Comment `json:"comments"`
}

func TestReviews(test *testing.T) {
	srv := startServer(test, "127.0.0.1")
	token := adminToken(test, srv)
	createRepo(test, srv, token, "acme/demo")
	auth := "http.extraHeader=Authorization: Bearer " + token
	remote := srv.url + "/acme/demo.git"
	src := newSource(test)
	writeLines := func(changes map[int]string, message string) {
		test.Helper()
		lines := []string{}
		for number := 1; number <= 30; number++ {
			lines = append(lines, fmt.Sprintf("line %d", number))
			if changed, ok := changes[number]; ok {
				lines[number-1] = changed
			}
		}
		if err := os.WriteFile(filepath.Join(src, "app.txt"), []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
			test.Fatal(err)
		}
		mustGit(test, src, "add", "app.txt")
		mustGit(test, src, "commit", "-q", "-m", message)
	}
	push := func(ref string) {
		test.Helper()
		mustGit(test, src, "-c", auth, "push", "-q", remote, ref)
	}
	view := func() reviewView {
		test.Helper()
		var result reviewView
		lyraJSON(test, srv, token, &result, "pr", "view", "acme/demo", "1")
		return result
	}
	humanReview := func(body string) (int, []byte) {
		test.Helper()
		headers := tailnet()
		return call(test, http.MethodPost, srv.url+"/api/repos/acme/demo/pulls/1/reviews", headers, body)
	}
	blocked := func() string {
		test.Helper()
		if merge := view().Merge; merge != nil {
			return strings.Join(merge.Blocked, "; ")
		}
		return "no merge plan"
	}

	writeLines(nil, "Thirty lines")
	push("main")
	mustGit(test, src, "switch", "-q", "-c", "feature")
	writeLines(map[int]string{3: "line three", 28: "line twenty-eight"}, "Change lines 3 and 28")
	push("feature")
	lyraJSON(test, srv, token, &pullResult{}, "pr", "create", "acme/demo", "--head", "feature", "--title", "Change two lines")
	firstHead := mustGit(test, src, "rev-parse", "HEAD")
	lyraJSON(test, srv, token, &store.Rule{}, "rule", "set", "acme/demo", "main", "--require-approval")
	if got := blocked(); !strings.Contains(got, "needs an approving review of "+firstHead[:7]) {
		test.Errorf("blocked before any review: %q", got)
	}

	for _, check := range []struct {
		args []string
		code int
		want string
	}{
		{[]string{"pr", "review", "acme/demo", "1", "--approve"}, 1, "can't approve it"},
		{[]string{"pr", "review", "acme/demo", "1", "--comment"}, 1, "needs a body or line comments"},
		{[]string{"pr", "review", "acme/demo", "1", "--request-changes"}, 1, "needs a body"},
		{[]string{"pr", "review", "acme/demo", "1"}, 2, "exactly one"},
		{[]string{"pr", "review", "acme/demo", "1", "--approve", "--comment"}, 2, "exactly one"},
		{[]string{"pr", "comment", "acme/demo", "1", "--path", "app.txt", "--line", "15", "--body", "x"}, 1, "not in the diff"},
		{[]string{"pr", "comment", "acme/demo", "1", "--path", "other.txt", "--line", "3", "--body", "x"}, 1, "not in the diff"},
		{[]string{"pr", "comment", "acme/demo", "1", "--path", "app.txt", "--line", "3", "--side", "middle", "--body", "x"}, 1, `one of \"new, old\"`},
		{[]string{"issue", "comment", "acme/demo", "1", "--path", "app.txt", "--body", "x"}, 2, "unknown flag"},
	} {
		if result := runLyra(test, srv, token, check.args...); result.code != check.code || !strings.Contains(result.stderr, check.want) {
			test.Errorf("lyra %s: exit %d %s", strings.Join(check.args, " "), result.code, result.stderr)
		}
	}

	var lineComment struct {
		store.Comment
		EventID int64 `json:"event_id"`
	}
	lyraJSON(test, srv, token, &lineComment, "pr", "comment", "acme/demo", "1", "--path", "app.txt", "--line", "3", "--body", "Renamed on purpose")
	if lineComment.Path != "app.txt" || lineComment.Line != 3 || lineComment.Side != "new" || lineComment.CommitSHA != firstHead {
		test.Errorf("line comment = %+v", lineComment.Comment)
	}
	lyraJSON(test, srv, token, &lineComment, "pr", "comment", "acme/demo", "1", "--path", "app.txt", "--line", "3", "--side", "old", "--body", "The old wording")

	status, data := humanReview(`{"state":"request_changes","body":"Line 28 needs a test","comments":[{"path":"app.txt","line":28,"body":"Here"}]}`)
	var requested struct {
		store.Review
		EventID int64 `json:"event_id"`
	}
	if err := json.Unmarshal(data, &requested); err != nil || status != http.StatusCreated || requested.State != "changes_requested" || len(requested.Comments) != 1 || requested.Comments[0].ReviewID != requested.ID || requested.Author.Via != "tailscale" {
		test.Fatalf("request changes: HTTP %d %s", status, data)
	}
	if status, data := humanReview(`{"state":"comment","comments":[{"path":"app.txt","line":15,"body":"Out of the diff"}]}`); status != http.StatusBadRequest {
		test.Errorf("review comment outside the diff: HTTP %d %s", status, data)
	}
	lyraJSON(test, srv, token, &issueResult{}, "issue", "create", "acme/demo", "--title", "An issue")
	if status, data := call(test, http.MethodPost, srv.url+"/api/repos/acme/demo/issues/2/comments", tailnet(), `{"body":"x","path":"app.txt","line":3}`); status != http.StatusBadRequest || !strings.Contains(string(data), "only pull request comments") {
		test.Errorf("line comment on an issue: HTTP %d %s", status, data)
	}
	var event store.Event
	lyraJSON(test, srv, token, &event, "event", "wait", "--since", fmt.Sprint(lineComment.EventID), "--action", "pr.review", "--timeout", "5s")
	var eventData struct {
		State string `json:"state"`
	}
	if err := json.Unmarshal(event.Data, &eventData); err != nil || event.ID != requested.EventID || eventData.State != "changes_requested" {
		test.Errorf("pr.review event = %+v %s", event, event.Data)
	}
	if got := blocked(); !strings.Contains(got, "tester@example.com requested changes") || !strings.Contains(got, "needs an approving review") {
		test.Errorf("blocked after changes were requested: %q", got)
	}

	writeLines(map[int]string{3: "line 3, renamed back", 28: "line twenty-eight"}, "Address review")
	push("feature")
	secondHead := mustGit(test, src, "rev-parse", "HEAD")
	viewed := view()
	outdated := map[string]bool{}
	for _, comment := range viewed.Comments {
		outdated[fmt.Sprintf("%s:%d:%s", comment.Path, comment.Line, comment.Side)] = comment.Outdated
	}
	if !outdated["app.txt:3:new"] || outdated["app.txt:3:old"] || len(viewed.Reviews) != 1 || !viewed.Reviews[0].Comments[0].Outdated {
		test.Errorf("outdated after a new push = %v, review comments %+v", outdated, viewed.Reviews)
	}

	if status, data := humanReview(`{"state":"approve","body":"Looks good"}`); status != http.StatusCreated {
		test.Fatalf("approve: HTTP %d %s", status, data)
	}
	if got := blocked(); got != "" {
		test.Errorf("blocked after approval of the head: %q", got)
	}
	writeLines(map[int]string{3: "line 3, renamed back", 28: "line twenty-eight", 30: "one more"}, "Sneak in a change")
	push("feature")
	thirdHead := mustGit(test, src, "rev-parse", "HEAD")
	if got := blocked(); !strings.Contains(got, "needs an approving review of "+thirdHead[:7]) {
		test.Errorf("blocked after a push following approval: %q", got)
	}
	if result := runLyra(test, srv, token, "pr", "merge", "acme/demo", "1", "--squash"); result.code != 1 || !strings.Contains(result.stderr, "needs an approving review") {
		test.Errorf("merge without approval of the head: exit %d %s", result.code, result.stderr)
	}

	sameOrigin := map[string]string{"Sec-Fetch-Site": "same-origin"}
	if status, body := browserCall(test, http.MethodPost, srv.url+"/api/repos/acme/demo/pulls/1/comments", `{"body":"From the diff","path":"app.txt","line":30,"side":"new"}`, sameOrigin); status != http.StatusCreated {
		test.Errorf("UI line comment: HTTP %d %s", status, body)
	}
	if status, body := browserCall(test, http.MethodPost, srv.url+"/api/repos/acme/demo/pulls/1/reviews", `{"state":"approve","body":"Approved from the UI"}`, sameOrigin); status != http.StatusCreated {
		test.Errorf("UI approve: HTTP %d %s", status, body)
	}
	if got := blocked(); got != "" {
		test.Errorf("blocked after approving from the UI: %q", got)
	}
	for target, wants := range map[string][]string{
		"/api/repos/acme/demo/pulls/1/view":  {`"state":"changes_requested"`, `"state":"approved"`, "Line 28 needs a test", "Approved from the UI", `"outdated":true`, secondHead, thirdHead},
		"/api/repos/acme/demo/pulls/1/patch": {"From the diff", `"side":"new"`, `"line":30`, `"html":`},
	} {
		status, page := call(test, http.MethodGet, srv.url+target, tailnet(), "")
		for _, text := range wants {
			if status != http.StatusOK || !strings.Contains(compactJSON(test, page), text) {
				test.Errorf("GET %s: HTTP %d, missing %q", target, status, text)
			}
		}
	}
	for target, payload := range map[string]string{
		"reviews":  `{"state":"comment","body":"Stale workspace","head_sha":"` + firstHead + `"}`,
		"comments": `{"body":"Stale workspace","path":"app.txt","line":30,"head_sha":"` + firstHead + `"}`,
	} {
		status, response := call(test, http.MethodPost, srv.url+"/api/repos/acme/demo/pulls/1/"+target, tailnet(), payload)
		if status != http.StatusConflict || !strings.Contains(string(response), "changed") {
			test.Errorf("stale %s: HTTP %d %s", target, status, response)
		}
	}
	var merged pullResult
	lyraJSON(test, srv, token, &merged, "pr", "merge", "acme/demo", "1", "--squash")
	if merged.State != "merged" {
		test.Errorf("merge after approval = %s", merged.State)
	}
	if result := runLyra(test, srv, token, "rule", "list", "acme/demo", "--jq", ".[0].require_approval"); result.stdout != "true\n" {
		test.Errorf("rule list require_approval %q", result.stdout)
	}
}

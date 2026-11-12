package e2e

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/muchobien/lyra/internal/store"
)

type waitedEvent struct {
	store.Event
	Fields struct {
		Body     string `json:"body"`
		State    string `json:"state"`
		Assignee string `json:"assignee"`
		Label    string `json:"label"`
	}
}

func parseWaited(test *testing.T, result cliResult) waitedEvent {
	test.Helper()
	var waited waitedEvent
	if result.code != 0 {
		test.Fatalf("wait: exit %d: %s %s", result.code, result.stdout, result.stderr)
	}
	if err := json.Unmarshal([]byte(result.stdout), &waited.Event); err != nil {
		test.Fatalf("wait output: %v: %s", err, result.stdout)
	}
	if err := json.Unmarshal(waited.Data, &waited.Fields); err != nil {
		test.Fatalf("wait data: %v: %s", err, waited.Data)
	}
	return waited
}

func TestWait(test *testing.T) {
	srv := startServer(test, "127.0.0.1")
	token := adminToken(test, srv)
	createRepo(test, srv, token, "demo")
	auth := "http.extraHeader=Authorization: Bearer " + token
	src := newSource(test)
	commitFile(test, src, "README.md", "# Demo\n", "First commit")
	mustGit(test, src, "-c", auth, "push", "-q", srv.url+"/demo.git", "main")
	mustGit(test, src, "switch", "-q", "-c", "feature")
	commitFile(test, src, "feature.txt", "feature\n", "Add feature")
	mustGit(test, src, "-c", auth, "push", "-q", srv.url+"/demo.git", "feature")
	lyraJSON(test, srv, token, &pullResult{}, "pr", "create", "demo", "--head", "feature", "--title", "Feature")
	human := func(method, path, body string) {
		test.Helper()
		if status, data := call(test, method, srv.url+path, tailnet(), body); status >= 300 {
			test.Fatalf("%s %s: HTTP %d %s", method, path, status, data)
		}
	}
	settle := func() {
		time.Sleep(700 * time.Millisecond)
	}

	human(http.MethodPost, "/api/repos/demo/pulls/1/comments", `{"body":"Arrived before the wait"}`)
	early := parseWaited(test, runLyra(test, srv, token, "pr", "wait", "demo", "1", "--timeout", "5s"))
	if early.Action != "pr.comment" || early.Fields.Body != "Arrived before the wait" || early.Actor.Via != "tailscale" {
		test.Errorf("wait after an earlier reply = %+v %s", early.Event, early.Data)
	}

	lyraJSON(test, srv, token, &struct{}{}, "pr", "comment", "demo", "1", "--body", "Thanks, fixing")
	waiting := startLyra(test, srv, token, "pr", "wait", "demo", "1", "--timeout", "20s")
	settle()
	lyraJSON(test, srv, token, &struct{}{}, "pr", "comment", "demo", "1", "--body", "My own note")
	human(http.MethodPost, "/api/repos/demo/pulls/1/reviews", `{"state":"request_changes","body":"Add a test"}`)
	review := parseWaited(test, waiting())
	if review.Action != "pr.review" || review.Fields.State != "changes_requested" || review.Fields.Body != "Add a test" {
		test.Errorf("wait for a review = %+v %s", review.Event, review.Data)
	}

	waiting = startLyra(test, srv, token, "pr", "wait", "demo", "1", "--timeout", "20s")
	settle()
	lyraJSON(test, srv, token, &struct{}{}, "pr", "comment", "demo", "1", "--body", "From the same session")
	startLyraWith(test, srv, token, []string{"LYRA_SESSION=session-456"}, "pr", "comment", "demo", "1", "--body", "From another session")()
	other := parseWaited(test, waiting())
	if other.Fields.Body != "From another session" || other.Actor.Session != "session-456" {
		test.Errorf("wait with another session commenting = %+v %s", other.Event, other.Data)
	}

	if result := runLyra(test, srv, token, "pr", "wait", "demo", "1", "--since", fmt.Sprint(review.ID-1), "--for", "review", "--timeout", "5s"); parseWaited(test, result).ID != review.ID {
		test.Errorf("wait --since before the review: %s", result.stdout)
	}

	waiting = startLyra(test, srv, token, "pr", "wait", "demo", "1", "--for", "merged", "--timeout", "20s")
	settle()
	human(http.MethodPost, "/api/repos/demo/pulls/1/comments", `{"body":"Merging now"}`)
	human(http.MethodPost, "/api/repos/demo/pulls/1/merge", `{"method":"squash"}`)
	merged := parseWaited(test, waiting())
	if merged.Action != "pr.merged" || merged.Item != 1 {
		test.Errorf("wait --for merged = %+v", merged.Event)
	}

	for _, check := range []struct {
		args []string
		code int
	}{
		{[]string{"pr", "wait", "demo", "1", "--timeout", "1s"}, 3},
		{[]string{"pr", "wait", "demo", "1", "--for", "bogus"}, 2},
		{[]string{"issue", "wait", "demo", "1", "--for", "review"}, 2},
		{[]string{"pr", "wait", "demo"}, 2},
	} {
		if result := runLyra(test, srv, token, check.args...); result.code != check.code {
			test.Errorf("lyra %s: exit %d, want %d: %s", strings.Join(check.args, " "), result.code, check.code, result.stderr)
		}
	}
	if status, _ := call(test, http.MethodGet, srv.url+"/api/events/stream?since=mine", bearer(token), ""); status != http.StatusBadRequest {
		test.Errorf("since=mine without an item: HTTP %d, want 400", status)
	}

	lyraJSON(test, srv, token, &issueResult{}, "issue", "create", "demo", "--title", "Needs triage")
	lyraJSON(test, srv, token, &store.Label{}, "label", "create", "demo", "bug")
	waiting = startLyra(test, srv, token, "issue", "wait", "demo", "2", "--for", "labeled", "--timeout", "20s")
	settle()
	human(http.MethodPost, "/api/repos/demo/issues/2/comments", `{"body":"Looking"}`)
	human(http.MethodPatch, "/api/repos/demo/issues/2", `{"add_labels":["bug"]}`)
	if labeled := parseWaited(test, waiting()); labeled.Action != "issue.labeled" || labeled.Fields.Label != "bug" {
		test.Errorf("issue wait --for labeled = %+v %s", labeled.Event, labeled.Data)
	}

	waiting = startLyra(test, srv, token, "event", "wait", "--assignee", "me", "--timeout", "20s")
	settle()
	human(http.MethodPatch, "/api/repos/demo/issues/2", `{"add_assignees":["someone-else"]}`)
	human(http.MethodPatch, "/api/repos/demo/issues/2", `{"add_assignees":["e2e-agent"]}`)
	if assigned := parseWaited(test, waiting()); assigned.Action != "issue.assigned" || assigned.Fields.Assignee != "e2e-agent" || assigned.Item != 2 {
		test.Errorf("work queue wait = %+v %s", assigned.Event, assigned.Data)
	}
}

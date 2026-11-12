package e2e

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"testing"

	"github.com/muchobien/lyra/internal/store"
)

type issueResult struct {
	store.Item
	EventID  int64           `json:"event_id"`
	Comments []store.Comment `json:"comments"`
}

func postForm(test *testing.T, target string, form url.Values, headers map[string]string) (int, string, string) {
	test.Helper()
	request, err := http.NewRequest(http.MethodPost, target, strings.NewReader(form.Encode()))
	if err != nil {
		test.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("Tailscale-User-Login", login)
	for name, value := range headers {
		request.Header.Set(name, value)
	}
	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	response, err := client.Do(request)
	if err != nil {
		test.Fatal(err)
	}
	defer response.Body.Close()
	body, _ := io.ReadAll(response.Body)
	return response.StatusCode, response.Header.Get("Location"), string(body)
}

func TestIssues(test *testing.T) {
	srv := startServer(test, "127.0.0.1")
	token := adminToken(test, srv)
	createRepo(test, srv, token, "demo")
	issue := func(args ...string) issueResult {
		test.Helper()
		var result issueResult
		lyraJSON(test, srv, token, &result, append([]string{"issue"}, args...)...)
		return result
	}
	numbers := func(args ...string) string {
		test.Helper()
		return runLyra(test, srv, token, append([]string{"issue", "list", "demo", "--jq", `[.[].number] | map(tostring) | join(",")`}, args...)...).stdout
	}

	var label store.Label
	lyraJSON(test, srv, token, &label, "label", "create", "demo", "bug", "--color", "#D73A4A", "--description", "Something is broken")
	if label.Name != "bug" || label.Color != "d73a4a" {
		test.Errorf("label create = %+v", label)
	}
	for _, args := range [][]string{
		{"label", "create", "demo", "bug"},
		{"label", "create", "demo", "a,b"},
		{"label", "create", "demo", "docs", "--color", "red"},
		{"issue", "create", "demo", "--title", "Unknown label", "--label", "nope"},
		{"issue", "view", "demo", "99"},
	} {
		if result := runLyra(test, srv, token, args...); result.code != 1 {
			test.Errorf("lyra %s: exit %d, want 1: %s", strings.Join(args, " "), result.code, result.stderr)
		}
	}
	for _, args := range [][]string{
		{"issue", "create", "demo"},
		{"issue", "edit", "demo", "1"},
		{"issue", "comment", "demo", "1"},
		{"issue", "comment", "demo", "1", "--body", "x", "--body-file", "-"},
	} {
		if result := runLyra(test, srv, token, args...); result.code != 2 {
			test.Errorf("lyra %s: exit %d, want 2: %s", strings.Join(args, " "), result.code, result.stderr)
		}
	}

	crash := issue("create", "demo", "--title", "  Crash on start  ", "--body", "Steps:\n\n1. run", "--label", "bug", "--assignee", "@me")
	if crash.Number != 1 || crash.Title != "Crash on start" || crash.State != "open" || crash.EventID == 0 ||
		fmt.Sprint(crash.Labels) != "[bug]" || fmt.Sprint(crash.Assignees) != "[e2e-agent]" ||
		crash.Author != (store.Actor{User: login, Via: "token:admin", Agent: "e2e-agent", Session: "session-123"}) {
		test.Errorf("issue create = %+v", crash)
	}
	docs := issue("create", "demo", "--title", "Write docs")
	if docs.Number != 2 {
		test.Errorf("second issue number %d, want 2 (a failed create must not use a number)", docs.Number)
	}
	if got := numbers(); got != "2,1\n" {
		test.Errorf("open issues %q", got)
	}
	if got := numbers("--label", "bug"); got != "1\n" {
		test.Errorf("issues labeled bug %q", got)
	}
	if got := numbers("--assignee", "@me"); got != "1\n" {
		test.Errorf("issues assigned to @me %q", got)
	}

	var comment struct {
		store.Comment
		EventID int64 `json:"event_id"`
	}
	lyraJSON(test, srv, token, &comment, "issue", "comment", "demo", "1", "--body", "Looking into it")
	var commented store.Event
	lyraJSON(test, srv, token, &commented, "event", "wait", "--repo", "demo", "--action", "issue.comment", "--since", fmt.Sprint(crash.EventID), "--timeout", "5s")
	if commented.ID != comment.EventID || commented.Item != 1 || !strings.Contains(string(commented.Data), "Looking into it") {
		test.Errorf("comment event = %+v %s, want id %d", commented, commented.Data, comment.EventID)
	}
	viewed := issue("view", "demo", "1")
	if len(viewed.Comments) != 1 || viewed.Comments[0].Body != "Looking into it" || viewed.CommentCount != 1 || viewed.Comments[0].Author.Agent != "e2e-agent" {
		test.Errorf("issue view = %+v", viewed)
	}

	edited := issue("edit", "demo", "1", "--title", "Crash when starting", "--remove-label", "bug", "--add-assignee", "KROSF@github")
	if edited.Title != "Crash when starting" || len(edited.Labels) != 0 || fmt.Sprint(edited.Assignees) != "[KROSF@github e2e-agent]" {
		test.Errorf("issue edit = %+v", edited)
	}
	actions := runLyra(test, srv, token, "event", "list", "--repo", "demo", "--limit", "3", "--jq", `[.[].action] | join(",")`).stdout
	if actions != "issue.assigned,issue.unlabeled,issue.edited\n" {
		test.Errorf("edit events %q", actions)
	}

	closed := issue("close", "demo", "1")
	if closed.State != "closed" || closed.ClosedAt == nil || closed.EventID == 0 {
		test.Errorf("issue close = %+v", closed)
	}
	if again := issue("close", "demo", "1"); again.EventID != 0 {
		test.Errorf("closing a closed issue recorded event %d", again.EventID)
	}
	if got := numbers(); got != "2\n" {
		test.Errorf("open issues after close %q", got)
	}
	if got := numbers("--state", "closed"); got != "1\n" {
		test.Errorf("closed issues %q", got)
	}
	if reopened := issue("reopen", "demo", "1"); reopened.State != "open" || reopened.ClosedAt != nil {
		test.Errorf("issue reopen = %+v", reopened)
	}

	lyraJSON(test, srv, token, &label, "label", "create", "demo", "docs")
	issue("edit", "demo", "2", "--add-label", "docs")
	if result := runLyra(test, srv, token, "label", "delete", "demo", "docs"); result.code != 0 {
		test.Fatalf("label delete: %s", result.stderr)
	}
	if labels := issue("view", "demo", "2").Labels; len(labels) != 0 {
		test.Errorf("labels after deleting the label: %v", labels)
	}

	reader := createToken(test, srv, `{"name":"reader","scope":"read","repos":["demo"]}`)
	if result := runLyra(test, srv, reader, "issue", "create", "demo", "--title", "Nope"); result.code != 1 || !strings.Contains(result.stderr, "lacks write") {
		test.Errorf("read token issue create: exit %d stderr %s", result.code, result.stderr)
	}
	if result := runLyra(test, srv, reader, "issue", "list", "demo", "--jq", "length"); result.stdout != "2\n" {
		test.Errorf("read token issue list %q %s", result.stdout, result.stderr)
	}

	sameOrigin := map[string]string{"Sec-Fetch-Site": "same-origin"}
	status, location, _ := postForm(test, srv.url+"/demo/issues", url.Values{"title": {"From the browser"}, "body": {"**bold** <script>alert(1)</script> [x](javascript:alert(1))"}}, sameOrigin)
	if status != http.StatusSeeOther || location != "/demo/issues/3" {
		test.Fatalf("UI create: HTTP %d location %q", status, location)
	}
	for _, headers := range []map[string]string{{"Sec-Fetch-Site": "cross-site"}, {"Origin": "http://evil.example"}} {
		if status, _, _ := postForm(test, srv.url+"/demo/issues", url.Values{"title": {"Forged"}}, headers); status != http.StatusForbidden {
			test.Errorf("cross-origin form post with %v: HTTP %d, want 403", headers, status)
		}
	}
	if status, location, _ := postForm(test, srv.url+"/demo/issues/3/comments", url.Values{"body": {"From the *browser*"}}, sameOrigin); status != http.StatusSeeOther || location != "/demo/issues/3" {
		test.Errorf("UI comment: HTTP %d location %q", status, location)
	}
	lyraJSON(test, srv, token, &label, "label", "create", "demo", "ui")
	if status, _, body := postForm(test, srv.url+"/demo/issues/3/edit", url.Values{"labels": {"ui"}, "assignees": {"@me, claude"}}, sameOrigin); status != http.StatusSeeOther {
		test.Errorf("UI edit: HTTP %d %s", status, body)
	}
	if status, _, _ := postForm(test, srv.url+"/demo/issues/3/edit", url.Values{"labels": {"missing"}}, sameOrigin); status != http.StatusBadRequest {
		test.Errorf("UI edit with unknown label: HTTP %d, want 400", status)
	}
	if status, _, _ := postForm(test, srv.url+"/demo/issues/3/edit", url.Values{"state": {"closed"}}, sameOrigin); status != http.StatusSeeOther {
		test.Errorf("UI close: HTTP %d", status)
	}
	fromUI := issue("view", "demo", "3")
	if fromUI.State != "closed" || fmt.Sprint(fromUI.Labels) != "[ui]" || fmt.Sprint(fromUI.Assignees) != "[claude tester@example.com]" || fromUI.Author.Via != "tailscale" || len(fromUI.Comments) != 1 {
		test.Errorf("issue changed from the UI = %+v", fromUI)
	}
	status, page := call(test, http.MethodGet, srv.url+"/demo/issues/3", tailnet(), "")
	if status != http.StatusOK || !strings.Contains(string(page), "<strong>bold</strong>") || !strings.Contains(string(page), "<em>browser</em>") ||
		strings.Contains(string(page), "<script>alert") || strings.Contains(string(page), "javascript:") {
		test.Errorf("issue page: HTTP %d\n%s", status, page)
	}
	status, page = call(test, http.MethodGet, srv.url+"/demo/issues?state=closed", tailnet(), "")
	if status != http.StatusOK || !strings.Contains(string(page), "From the browser") {
		test.Errorf("closed issues page: HTTP %d", status)
	}

	if result := runLyra(test, srv, token, "repo", "delete", "demo", "--yes"); result.code != 0 {
		test.Fatalf("repo delete: %s", result.stderr)
	}
	createRepo(test, srv, token, "demo")
	if got := numbers("--state", "all"); got != "\n" {
		test.Errorf("issues after recreating the repo %q", got)
	}
	if recreated := issue("create", "demo", "--title", "Fresh start"); recreated.Number != 1 || len(recreated.Labels) != 0 {
		test.Errorf("first issue in recreated repo = %+v", recreated)
	}
	var labels []store.Label
	lyraJSON(test, srv, token, &labels, "label", "list", "demo")
	if len(labels) != 0 || slices.ContainsFunc(labels, func(label store.Label) bool { return label.Name == "bug" }) {
		test.Errorf("labels after recreating the repo: %+v", labels)
	}
}

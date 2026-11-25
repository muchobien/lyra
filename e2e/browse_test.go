package e2e

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"testing"
)

func pushDemo(test *testing.T, srv server, token string) string {
	test.Helper()
	createRepo(test, srv, token, "acme/demo")
	src := newSource(test)
	commitFile(test, src, "README.md", "# Demo\n", "Document the project")
	mustGit(test, src, "checkout", "-q", "-b", "feature/nested")
	commitFile(test, src, "branch.txt", "branch content\n", "Branch commit")
	mustGit(test, src, "checkout", "-q", "main")
	mustGit(test, src, "-c", "http.extraHeader=Authorization: Bearer "+token, "push", "-q", srv.url+"/acme/demo.git", "main", "feature/nested")
	return src
}

func TestBranches(test *testing.T) {
	srv := startServer(test, "127.0.0.1")
	token := adminToken(test, srv)
	src := pushDemo(test, srv, token)
	mustGit(test, src, "-c", "http.extraHeader=Authorization: Bearer "+token, "push", "-q", srv.url+"/acme/demo.git", "main:release/one")
	if status, body := call(test, http.MethodPost, srv.url+"/api/repos/acme/demo/rules", bearer(token), `{"pattern":"release/*"}`); status != http.StatusOK {
		test.Fatalf("set rule: HTTP %d, %s", status, body)
	}
	type branch struct {
		Name      string `json:"name"`
		Default   bool   `json:"default"`
		Protected bool   `json:"protected"`
		Ahead     int    `json:"ahead"`
		Behind    int    `json:"behind"`
		Commit    struct {
			Subject string `json:"subject"`
		} `json:"commit"`
	}
	list := func() []branch {
		status, body := call(test, http.MethodGet, srv.url+"/api/repos/acme/demo/branches", bearer(token), "")
		var branches []branch
		if err := json.Unmarshal(body, &branches); err != nil || status != http.StatusOK {
			test.Fatalf("list branches: HTTP %d, %s, %v", status, body, err)
		}
		return branches
	}
	branches := list()
	if len(branches) != 3 || branches[0].Name != "main" || !branches[0].Default || branches[0].Protected {
		test.Fatalf("branches: %+v", branches)
	}
	for _, entry := range branches[1:] {
		switch entry.Name {
		case "feature/nested":
			if entry.Ahead != 1 || entry.Behind != 0 || entry.Protected || entry.Commit.Subject != "Branch commit" {
				test.Fatalf("feature branch: %+v", entry)
			}
		case "release/one":
			if entry.Ahead != 0 || entry.Behind != 0 || !entry.Protected {
				test.Fatalf("protected branch: %+v", entry)
			}
		default:
			test.Fatalf("unexpected branch: %+v", entry)
		}
	}

	reader := createToken(test, srv, `{"name":"reader","scope":"read"}`)
	for _, attempt := range []struct {
		name  string
		token string
		want  int
	}{
		{"main", token, http.StatusConflict},
		{"release/one", token, http.StatusConflict},
		{"missing", token, http.StatusNotFound},
		{"feature/nested", reader, http.StatusForbidden},
	} {
		if status, body := call(test, http.MethodDelete, srv.url+"/api/repos/acme/demo/branches?name="+url.QueryEscape(attempt.name), bearer(attempt.token), ""); status != attempt.want {
			test.Fatalf("delete %s: HTTP %d, %s", attempt.name, status, body)
		}
	}
	status, body := call(test, http.MethodDelete, srv.url+"/api/repos/acme/demo/branches?name=feature%2Fnested", bearer(token), "")
	if status != http.StatusOK || !strings.Contains(compactJSON(test, body), `"deleted":"feature/nested"`) {
		test.Fatalf("delete branch: HTTP %d, %s", status, body)
	}
	if branches := list(); len(branches) != 2 || slices.ContainsFunc(branches, func(entry branch) bool { return entry.Name == "feature/nested" }) {
		test.Fatalf("branches after delete: %+v", branches)
	}
	if output := mustGit(test, src, "-c", "http.extraHeader=Authorization: Bearer "+token, "ls-remote", "--heads", srv.url+"/acme/demo.git"); strings.Contains(output, "feature/nested") {
		test.Fatalf("deleted branch still on the server: %s", output)
	}
	status, body = call(test, http.MethodGet, srv.url+"/api/events?repo=acme/demo&action=push", bearer(token), "")
	if status != http.StatusOK || !strings.Contains(compactJSON(test, body), `"ref":"refs/heads/feature/nested"`) {
		test.Fatalf("push event for deletion: HTTP %d, %s", status, body)
	}
	if status, body := call(test, http.MethodGet, srv.url+"/acme/demo/branches", tailnet(), ""); status != http.StatusOK || !strings.Contains(string(body), `<div id="root"></div>`) {
		test.Fatalf("branches page: HTTP %d", status)
	}
}

func TestCompare(test *testing.T) {
	srv := startServer(test, "127.0.0.1")
	token := adminToken(test, srv)
	src := pushDemo(test, srv, token)
	mainSHA := strings.TrimSpace(mustGit(test, src, "rev-parse", "main"))
	var comparison struct {
		MergeBase string `json:"merge_base"`
		Commits   []struct {
			Subject string `json:"subject"`
		} `json:"commits"`
		Files []struct {
			Path string `json:"path"`
		} `json:"files"`
	}
	compare := func(base, head string) int {
		query := url.Values{"base": {base}, "head": {head}}.Encode()
		status, body := call(test, http.MethodGet, srv.url+"/api/repos/acme/demo/compare/view?"+query, bearer(token), "")
		if status == http.StatusOK {
			comparison.Commits, comparison.Files = nil, nil
			if err := json.Unmarshal(body, &comparison); err != nil {
				test.Fatal(err)
			}
		}
		return status
	}
	if status := compare("main", "feature/nested"); status != http.StatusOK || comparison.MergeBase != mainSHA ||
		len(comparison.Commits) != 1 || comparison.Commits[0].Subject != "Branch commit" ||
		len(comparison.Files) != 1 || comparison.Files[0].Path != "branch.txt" {
		test.Fatalf("compare main...feature/nested: HTTP %d, %+v", status, comparison)
	}
	if status := compare("feature/nested", "main"); status != http.StatusOK || len(comparison.Commits) != 0 || len(comparison.Files) != 0 {
		test.Fatalf("compare feature/nested...main: HTTP %d, %+v", status, comparison)
	}
	if status := compare("main", "missing"); status != http.StatusNotFound {
		test.Fatalf("compare with an unknown ref: HTTP %d", status)
	}
	for target, want := range map[string]int{
		"/acme/demo/compare":                       http.StatusOK,
		"/acme/demo/compare/feature/nested":        http.StatusOK,
		"/acme/demo/compare/main...feature/nested": http.StatusOK,
		"/acme/demo/compare/main...missing":        http.StatusNotFound,
	} {
		if status, _ := call(test, http.MethodGet, srv.url+target, tailnet(), ""); status != want {
			test.Fatalf("%s: HTTP %d", target, status)
		}
	}
}

func TestBranchAndCompareCLI(test *testing.T) {
	srv := startServer(test, "127.0.0.1")
	token := adminToken(test, srv)
	pushDemo(test, srv, token)
	if result := runLyra(test, srv, token, "branch", "list", "-R", "acme/demo", "--jq", `[.[].name] | join(",")`); result.code != 0 || result.stdout != "main,feature/nested\n" {
		test.Fatalf("branch list: %+v", result)
	}
	var comparison struct {
		Commits []struct {
			Subject string `json:"subject"`
		} `json:"commits"`
		Files []struct {
			Path      string `json:"path"`
			Status    string `json:"status"`
			Additions int    `json:"additions"`
		} `json:"files"`
	}
	for _, spec := range []string{"main...feature/nested", "feature/nested"} {
		lyraJSON(test, srv, token, &comparison, "compare", "-R", "acme/demo", spec)
		if len(comparison.Commits) != 1 || comparison.Commits[0].Subject != "Branch commit" || len(comparison.Files) != 1 ||
			comparison.Files[0].Path != "branch.txt" || comparison.Files[0].Status != "added" || comparison.Files[0].Additions != 1 {
			test.Fatalf("compare %s: %+v", spec, comparison)
		}
	}
	if result := runLyra(test, srv, token, "compare", "-R", "acme/demo", "feature/nested", "--diff"); result.code != 0 || !strings.Contains(result.stdout, "+branch content") {
		test.Fatalf("compare --diff: %+v", result)
	}
	if result := runLyra(test, srv, token, "compare", "-R", "acme/demo", "main...missing"); result.code == 0 || !strings.Contains(result.stderr, "unknown ref missing") {
		test.Fatalf("compare with an unknown ref: %+v", result)
	}
	if result := runLyra(test, srv, token, "branch", "delete", "-R", "acme/demo", "main"); result.code != 1 || !strings.Contains(result.stderr, "default branch") {
		test.Fatalf("delete the default branch: %+v", result)
	}
	if result := runLyra(test, srv, token, "branch", "delete", "-R", "acme/demo", "feature/nested"); result.code != 0 {
		test.Fatalf("branch delete: %+v", result)
	}
	if result := runLyra(test, srv, token, "branch", "list", "-R", "acme/demo", "--jq", `[.[].name] | join(",")`); result.stdout != "main\n" {
		test.Fatalf("branch list after delete: %+v", result)
	}
}

func TestBlame(test *testing.T) {
	srv := startServer(test, "127.0.0.1")
	token := adminToken(test, srv)
	src := pushDemo(test, srv, token)
	commitFile(test, src, "README.md", "# Demo\n\nMore <b>text</b>\n", "Expand README")
	commitFile(test, src, "data.bin", "\x00\xff", "Add binary")
	mustGit(test, src, "-c", "http.extraHeader=Authorization: Bearer "+token, "push", "-q", srv.url+"/acme/demo.git", "main")
	var blame struct {
		Binary bool `json:"binary"`
		Ranges []struct {
			Start  int      `json:"start"`
			Lines  []string `json:"lines"`
			Commit struct {
				Author  string `json:"author"`
				Date    string `json:"date"`
				Summary string `json:"summary"`
			} `json:"commit"`
		} `json:"ranges"`
	}
	status, body := call(test, http.MethodGet, srv.url+"/api/repos/acme/demo/blame?path=README.md", bearer(token), "")
	if err := json.Unmarshal(body, &blame); err != nil || status != http.StatusOK || blame.Binary || len(blame.Ranges) != 2 {
		test.Fatalf("blame: HTTP %d, %s, %v", status, body, err)
	}
	first, second := blame.Ranges[0], blame.Ranges[1]
	if first.Start != 1 || len(first.Lines) != 1 || first.Commit.Summary != "Document the project" ||
		second.Start != 2 || len(second.Lines) != 2 || second.Commit.Summary != "Expand README" ||
		first.Commit.Author == "" || first.Commit.Date == "" {
		test.Fatalf("blame ranges: %+v", blame.Ranges)
	}
	if !strings.Contains(second.Lines[1], "More") || strings.Contains(second.Lines[1], "<b>") {
		test.Fatalf("highlighted line: %s", second.Lines[1])
	}
	status, body = call(test, http.MethodGet, srv.url+"/api/repos/acme/demo/blame?ref=feature/nested&path=README.md", bearer(token), "")
	if err := json.Unmarshal(body, &blame); err != nil || status != http.StatusOK || len(blame.Ranges) != 1 {
		test.Fatalf("blame at a branch: HTTP %d, %s, %v", status, body, err)
	}
	status, body = call(test, http.MethodGet, srv.url+"/api/repos/acme/demo/blame?path=data.bin", bearer(token), "")
	if err := json.Unmarshal(body, &blame); err != nil || status != http.StatusOK || !blame.Binary || len(blame.Ranges) != 0 {
		test.Fatalf("binary blame: HTTP %d, %s, %v", status, body, err)
	}
	if status, _ := call(test, http.MethodGet, srv.url+"/api/repos/acme/demo/blame?path=missing", bearer(token), ""); status != http.StatusNotFound {
		test.Fatalf("missing file blame: HTTP %d", status)
	}
	for target, want := range map[string]int{
		"/acme/demo/blame/main/README.md":            http.StatusOK,
		"/acme/demo/blame/feature/nested/branch.txt": http.StatusOK,
		"/acme/demo/blame/main/missing":              http.StatusNotFound,
		"/acme/demo/blame/main/README.md?raw":        http.StatusOK,
	} {
		status, body := call(test, http.MethodGet, srv.url+target, tailnet(), "")
		if status != want || (want == http.StatusOK && !strings.Contains(string(body), `<div id="root"></div>`)) {
			test.Fatalf("%s: HTTP %d", target, status)
		}
	}
}

func TestGoImport(test *testing.T) {
	srv := startServer(test, "127.0.0.1")
	token := adminToken(test, srv)
	pushDemo(test, srv, token)
	host := strings.TrimPrefix(srv.url, "http://")
	meta := `<meta name="go-import" content="` + host + `/acme/demo git ` + srv.url + `/acme/demo.git">`
	for _, target := range []string{"/acme/demo", "/acme/demo.git", "/acme/demo/internal/pkg", "/acme/demo/tree/main"} {
		status, body := call(test, http.MethodGet, srv.url+target+"?go-get=1", bearer(token), "")
		if status != http.StatusOK || !strings.Contains(string(body), meta) {
			test.Fatalf("go-import for %s: HTTP %d, %s", target, status, body)
		}
	}
	if status, body := call(test, http.MethodGet, srv.url+"/acme/demo?go-get=1", nil, ""); status != http.StatusUnauthorized {
		test.Fatalf("go-import without credentials: HTTP %d, %s", status, body)
	}
	if status, body := call(test, http.MethodGet, srv.url+"/acme/demo/internal/pkg", bearer(token), ""); status != http.StatusNotFound || strings.Contains(string(body), "go-import") {
		test.Fatalf("package path without go-get: HTTP %d", status)
	}
}

func TestArchives(test *testing.T) {
	srv := startServer(test, "127.0.0.1")
	token := adminToken(test, srv)
	pushDemo(test, srv, token)

	status, body := call(test, http.MethodGet, srv.url+"/acme/demo/archive/feature/nested.tar.gz", bearer(token), "")
	if status != http.StatusOK {
		test.Fatalf("tar.gz archive: HTTP %d, %s", status, body)
	}
	unzipped, err := gzip.NewReader(bytes.NewReader(body))
	if err != nil {
		test.Fatal(err)
	}
	names := []string{}
	reader := tar.NewReader(unzipped)
	for {
		header, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			test.Fatal(err)
		}
		names = append(names, header.Name)
	}
	if !slices.Contains(names, "demo-feature-nested/branch.txt") || !slices.Contains(names, "demo-feature-nested/README.md") {
		test.Fatalf("tar.gz entries: %v", names)
	}

	request, _ := http.NewRequest(http.MethodGet, srv.url+"/acme/demo/archive/main.zip", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		test.Fatal(err)
	}
	body, _ = io.ReadAll(response.Body)
	response.Body.Close()
	if response.StatusCode != http.StatusOK || response.Header.Get("Content-Disposition") != `attachment; filename=demo-main.zip` {
		test.Fatalf("zip archive: HTTP %d, headers %v", response.StatusCode, response.Header)
	}
	archive, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		test.Fatal(err)
	}
	names = []string{}
	for _, entry := range archive.File {
		names = append(names, entry.Name)
	}
	if !slices.Contains(names, "demo-main/README.md") || slices.Contains(names, "demo-main/branch.txt") {
		test.Fatalf("zip entries: %v", names)
	}

	for _, target := range []string{"/acme/demo/archive/missing.zip", "/acme/demo/archive/main.rar", "/acme/missing/archive/main.zip"} {
		if status, body := call(test, http.MethodGet, srv.url+target, bearer(token), ""); status != http.StatusNotFound {
			test.Fatalf("%s: HTTP %d, %s", target, status, body)
		}
	}
	outsider := createToken(test, srv, `{"name":"other","scope":"read","repos":["other/*"]}`)
	if status, _ := call(test, http.MethodGet, srv.url+"/acme/demo/archive/main.zip", bearer(outsider), ""); status != http.StatusForbidden {
		test.Fatalf("archive without access: HTTP %d", status)
	}
}

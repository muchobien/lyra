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

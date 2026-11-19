package e2e

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAttachments(test *testing.T) {
	srv := startServer(test, "127.0.0.1")
	token := adminToken(test, srv)
	createRepo(test, srv, token, "acme/demo")
	createRepo(test, srv, token, "acme/other")
	dir := test.TempDir()
	png := "\x89PNG\r\n\x1a\n" + strings.Repeat("x", 64)
	for name, content := range map[string]string{
		"login.png":  png,
		"before.png": png + "before",
		"demo.mp4":   "\x00\x00\x00\x18ftypmp42",
		"notes.txt":  "notes",
		"body.md":    "Steps:\n\n![The error](./login.png)\n",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
			test.Fatal(err)
		}
	}
	attachment := func(id, name string) string {
		return srv.url + "/acme/demo/attachments/" + id + "/" + name
	}

	var created struct {
		Number int64  `json:"number"`
		Body   string `json:"body"`
	}
	lyraJSON(test, srv, token, &created, "issue", "create", "-R", "acme/demo", "--title", "Login fails",
		"--body-file", filepath.Join(dir, "body.md"),
		"--attach", filepath.Join(dir, "login.png"),
		"--attach", filepath.Join(dir, "demo.mp4#ignored"),
		"--attach", filepath.Join(dir, "before.png")+"#Before [state]",
	)
	want := "Steps:\n\n![The error](" + attachment("1", "login.png") + ")\n\n" +
		"![](" + attachment("2", "demo.mp4") + ")\n\n" +
		"![Before \\[state\\]](" + attachment("3", "before.png") + ")"
	if created.Body != want {
		test.Errorf("issue create body = %q, want %q", created.Body, want)
	}

	request, err := http.NewRequest(http.MethodGet, attachment("1", "login.png"), nil)
	if err != nil {
		test.Fatal(err)
	}
	request.Header.Set("Authorization", "Bearer "+token)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		test.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusOK || response.Header.Get("Content-Type") != "image/png" ||
		response.Header.Get("Content-Security-Policy") != "default-src 'none'; sandbox" ||
		response.ContentLength != int64(len(png)) {
		test.Errorf("download = %d %v", response.StatusCode, response.Header)
	}
	for _, target := range []string{"/acme/other/attachments/1/login.png", "/acme/demo/attachments/9/login.png"} {
		if status, _ := call(test, http.MethodGet, srv.url+target, bearer(token), ""); status != http.StatusNotFound {
			test.Errorf("GET %s = %d, want 404", target, status)
		}
	}
	markdown, err := json.Marshal(map[string]string{"body": created.Body})
	if err != nil {
		test.Fatal(err)
	}
	status, rendered := browserCall(test, http.MethodPost, srv.url+"/api/markdown", string(markdown), nil)
	if status != http.StatusOK || !strings.Contains(rendered, `<img src=\"`+attachment("1", "login.png")+`\" alt=\"The error\">`) ||
		!strings.Contains(rendered, `<video controls src=\"`+attachment("2", "demo.mp4")+`\"></video>`) {
		test.Errorf("rendered markdown = %d %s", status, rendered)
	}

	var comment struct {
		Body string `json:"body"`
	}
	lyraJSON(test, srv, token, &comment, "issue", "comment", "-R", "acme/demo", "1",
		"--attach", filepath.Join(dir, "before.png"))
	if comment.Body != "![before.png]("+attachment("4", "before.png")+")" {
		test.Errorf("issue comment body = %q", comment.Body)
	}
	var edited struct {
		Body string `json:"body"`
	}
	lyraJSON(test, srv, token, &edited, "issue", "edit", "-R", "acme/demo", "1",
		"--attach", filepath.Join(dir, "login.png#After"))
	if edited.Body != want+"\n\n![After]("+attachment("5", "login.png")+")" {
		test.Errorf("issue edit body = %q", edited.Body)
	}

	partial := runLyra(test, srv, token, "issue", "comment", "-R", "acme/demo", "1", "--body", "Logs",
		"--attach", filepath.Join(dir, "notes.txt"), "--attach", filepath.Join(dir, "before.png"))
	comment.Body = ""
	if err := json.Unmarshal([]byte(partial.stdout), &comment); err != nil {
		test.Errorf("partial attach stdout: %v", err)
	}
	if partial.code != 1 || !strings.Contains(partial.stderr, "attach "+filepath.Join(dir, "notes.txt")) ||
		comment.Body != "Logs\n\n![before.png]("+attachment("6", "before.png")+")" {
		test.Errorf("partial attach: exit %d\nstdout: %s\nstderr: %s", partial.code, partial.stdout, partial.stderr)
	}

	tooMany := []string{"issue", "comment", "-R", "acme/demo", "1"}
	for range 51 {
		tooMany = append(tooMany, "--attach", filepath.Join(dir, "login.png"))
	}
	if result := runLyra(test, srv, token, tooMany...); result.code != 2 {
		test.Errorf("51 attachments: exit %d, want 2: %s", result.code, result.stderr)
	}
}

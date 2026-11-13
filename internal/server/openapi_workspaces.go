package server

import (
	"net/http"

	"github.com/danielgtaylor/huma/v2"
)

func (s *Server) registerWorkspaces(api huma.API) {
	tags := []string{"Workspaces"}
	repository := "/api/repos/{owner}/{repo}"
	register(api, huma.Operation{
		OperationID: "getRepositoryTree", Method: http.MethodGet, Path: repository + "/tree", Tags: tags,
		Summary: "Browse repository files and rendered README", Description: "Scope: read.",
	}, s.handleTree)
	register(api, huma.Operation{
		OperationID: "getRepositoryBlob", Method: http.MethodGet, Path: repository + "/blob", Tags: tags,
		Summary: "Read highlighted source or binary/large-file metadata", Description: "Scope: read.",
	}, s.handleBlob)
	register(api, huma.Operation{
		OperationID: "getRepositoryRawFile", Method: http.MethodGet, Path: repository + "/raw", Tags: tags,
		Summary: "Download the original file bytes", Description: "Scope: read. Served with nosniff and a sandbox policy.",
		Responses: map[string]*huma.Response{"200": {
			Description: "Original repository file",
			Content: map[string]*huma.MediaType{
				"application/octet-stream": {Schema: &huma.Schema{Type: "string", Format: "binary"}},
				"text/plain":               {Schema: &huma.Schema{Type: "string"}},
			},
		}},
	}, stream[sourceInput](s.handleRawBlob))
	register(api, huma.Operation{
		OperationID: "listRepositoryFiles", Method: http.MethodGet, Path: repository + "/files", Tags: tags,
		Summary: "List recursive file paths for the file finder", Description: "Scope: read.",
	}, s.handleFiles)
	register(api, huma.Operation{
		OperationID: "getIssueWorkspace", Method: http.MethodGet, Path: repository + "/issues/{number}/view", Tags: tags,
		Summary: "Read an issue with rendered discussion and permissions", Description: "Scope: read.",
	}, s.handleItemWorkspace(kindIssue))
	register(api, huma.Operation{
		OperationID: "getPullWorkspace", Method: http.MethodGet, Path: repository + "/pulls/{number}/view", Tags: tags,
		Summary: "Read a pull request with rendered discussion, merge requirements, and checks", Description: "Scope: read.",
	}, s.handleItemWorkspace(kindPull))
	register(api, huma.Operation{
		OperationID: "getPullPatch", Method: http.MethodGet, Path: repository + "/pulls/{number}/patch", Tags: tags,
		Summary: "Read structured, highlighted diffs and anchored comments", Description: "Scope: read.",
	}, s.handlePullPatch)
	register(api, huma.Operation{
		OperationID: "getCommitWorkspace", Method: http.MethodGet, Path: repository + "/commits/{rev}/view", Tags: tags,
		Summary: "Read a commit with checks and structured diffs", Description: "Scope: read.",
	}, s.handleCommitWorkspace)
	register(api, huma.Operation{
		OperationID: "listWorkflows", Method: http.MethodGet, Path: repository + "/workflows", Tags: tags,
		Summary: "Discover workflows and declared manual inputs at a ref", Description: "Scope: read.",
	}, s.handleWorkflows)
	register(api, huma.Operation{
		OperationID: "renderMarkdown", Method: http.MethodPost, Path: "/api/markdown", Tags: tags,
		Summary: "Preview sanitized Markdown", Description: "Scope: any. This read-only operation remains available during maintenance.",
	}, s.handleMarkdown)
	register(api, huma.Operation{
		OperationID: "getCodeStyles", Method: http.MethodGet, Path: "/api/code.css", Tags: tags,
		Summary: "Read syntax highlighting styles for light and dark appearance", Description: "Scope: any.",
		Responses: map[string]*huma.Response{"200": text("text/css", "Chroma styles")},
	}, stream[struct{}](s.handleCodeCSS))
}

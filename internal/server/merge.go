package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/hiddeco/sshsig"

	"github.com/muchobien/lyra/internal/repos"
	"github.com/muchobien/lyra/internal/store"
)

const (
	methodFastForward = "ff"
	methodSquash      = "squash"

	pullCommitLimit    = 250
	squashCommitLimit  = 1000
	closingCommitLimit = 1000
	signedCommitLimit  = 10000
)

type mergePlan struct {
	HeadSHA     string   `json:"head_sha"`
	BaseSHA     string   `json:"base_sha"`
	FastForward bool     `json:"fast_forward"`
	Conflicts   []string `json:"conflicts"`
	Methods     []string `json:"methods"`
	Blocked     []string `json:"blocked"`
	tree        string
}

type mergeRequest struct {
	Method       string `json:"method" enum:"squash,ff"`
	HeadSHA      string `json:"head_sha,omitzero" doc:"Refuse the merge unless the head branch is at this commit."`
	Title        string `json:"title,omitzero" doc:"Squash commit title; defaults to the title with (#number)."`
	Message      string `json:"message,omitzero" doc:"Squash commit body."`
	DeleteBranch bool   `json:"delete_branch,omitzero"`
}

type mergeInput struct {
	ItemPath
	Body mergeRequest
}

type PushEvent struct {
	Method string      `json:"method,omitempty"`
	Pull   int64       `json:"pull,omitzero"`
	Refs   []RefUpdate `json:"refs"`
}

func (s *Server) approvalBlocks(ctx context.Context, pull store.Pull, head string) ([]string, error) {
	reviews, err := s.store.Reviews(ctx, pull.Item)
	if err != nil {
		return nil, err
	}
	return approvalProblems(pull, reviews, head), nil
}

func (s *Server) checkBlocks(ctx context.Context, repo, head string) ([]string, error) {
	checks, err := s.store.Checks(ctx, repo, head)
	if err != nil {
		return nil, err
	}
	failing := []string{}
	for _, check := range checks.Jobs {
		if check.Status == statusFailure || check.Status == statusCancelled {
			failing = append(failing, check.Name)
		}
	}
	switch checks.State {
	case checksNone:
		return []string{"no checks ran for " + head[:7]}, nil
	case checksPending:
		return []string{"checks for " + head[:7] + " are still running"}, nil
	case checksFailure:
		return []string{"checks failed for " + head[:7] + ": " + strings.Join(failing, ", ")}, nil
	}
	return nil, nil
}

func (s *Server) signatureBlocks(ctx context.Context, dir string, plan mergePlan) ([]string, error) {
	commits, err := s.git.Log(ctx, dir, plan.HeadSHA, signedCommitLimit, plan.BaseSHA)
	if err != nil {
		return nil, err
	}
	keys, err := s.store.Keys(ctx, "")
	if err != nil {
		return nil, err
	}
	return unverifiedCommits(keys, commits)
}

func (s *Server) planBlocks(
	ctx context.Context,
	repo string,
	pull store.Pull,
	plan mergePlan,
	policy branchPolicy,
) ([]string, error) {
	blocked := []string{}
	rule := "branch rule " + strings.Join(policy.rules, ", ")
	if policy.requireApproval {
		problems, err := s.approvalBlocks(ctx, pull, plan.HeadSHA)
		if err != nil {
			return nil, err
		}
		for _, problem := range problems {
			blocked = append(blocked, rule+": "+problem)
		}
	}
	if policy.requireChecks {
		problems, err := s.checkBlocks(ctx, repo, plan.HeadSHA)
		if err != nil {
			return nil, err
		}
		for _, problem := range problems {
			blocked = append(blocked, rule+": "+problem)
		}
	}
	if len(policy.methods) == 0 {
		blocked = append(blocked, "branch rules "+strings.Join(policy.rules, ", ")+" allow no merge method")
	}
	if policy.requireSigned {
		problems, err := s.signatureBlocks(ctx, s.repoDir(repo), plan)
		if err != nil {
			return nil, err
		}
		for _, problem := range problems {
			blocked = append(blocked, rule+" requires verified signatures: "+problem)
		}
	}
	return blocked, nil
}

func (s *Server) planMerge(ctx context.Context, repo string, pull store.Pull) (mergePlan, error) {
	dir := s.repoDir(repo)
	plan := mergePlan{Blocked: []string{}}
	var err error
	if plan.HeadSHA, err = s.git.ResolveCommit(ctx, dir, "refs/heads/"+pull.Head); err != nil {
		return plan, store.InvalidError("branch " + pull.Head + " no longer exists")
	}
	if plan.BaseSHA, err = s.git.ResolveCommit(ctx, dir, "refs/heads/"+pull.Base); err != nil {
		return plan, store.InvalidError("branch " + pull.Base + " no longer exists")
	}
	if plan.FastForward, err = s.git.IsAncestor(ctx, dir, plan.BaseSHA, plan.HeadSHA); err != nil {
		return plan, err
	}
	if plan.tree, plan.Conflicts, err = s.git.MergeTree(ctx, dir, plan.BaseSHA, plan.HeadSHA); err != nil {
		return plan, err
	}
	rules, err := s.store.Rules(ctx, repo)
	if err != nil {
		return plan, err
	}
	policy := policyFor(rules, pull.Base)
	plan.Methods = policy.methods
	plan.Blocked, err = s.planBlocks(ctx, repo, pull, plan, policy)
	return plan, err
}

func gitIdentity(name, email string, when time.Time) string {
	return fmt.Sprintf("%s <%s> %d +0000", name, email, when.Unix())
}

func (s *Server) checkMerge(
	ctx context.Context,
	repo string,
	pull store.Pull,
	request mergeRequest,
) (mergePlan, error) {
	if pull.State != stateOpen {
		return mergePlan{}, store.InvalidError(fmt.Sprintf("pull request #%d is %s", pull.Number, pull.State))
	}
	if !slices.Contains(mergeMethods, request.Method) {
		return mergePlan{}, store.InvalidError("method must be ff or squash")
	}
	plan, err := s.planMerge(ctx, repo, pull)
	if err != nil {
		return plan, err
	}
	if len(plan.Blocked) > 0 {
		return plan, conflict(strings.Join(plan.Blocked, "; "))
	}
	if request.DeleteBranch {
		rules, err := s.store.Rules(ctx, repo)
		if err != nil {
			return plan, err
		}
		if policy := policyFor(rules, pull.Head); policy.forbidDeletion {
			return plan, conflict(
				"branch rule " + strings.Join(policy.rules, ", ") + " protects " + pull.Head +
					" from deletion; merge without deleting it",
			)
		}
	}
	if !slices.Contains(plan.Methods, request.Method) {
		return plan, conflict("branch rules for " + pull.Base + " allow only " + strings.Join(plan.Methods, ", "))
	}
	if request.HeadSHA != "" && request.HeadSHA != plan.HeadSHA {
		return plan, conflict(fmt.Sprintf("%s is at %s, not %s", pull.Head, plan.HeadSHA, request.HeadSHA))
	}
	return plan, nil
}

func squashMessage(pull store.Pull, commits []repos.Commit, request mergeRequest) string {
	author := commits[len(commits)-1].Author
	title := strings.TrimSpace(request.Title)
	if title == "" {
		title = fmt.Sprintf("%s (#%d)", pull.Title, pull.Number)
	}
	message := strings.TrimSpace(request.Message)
	if request.Message == "" {
		message = strings.TrimSpace(pull.Body)
	}
	coauthors := []string{}
	for _, commit := range slices.Backward(commits) {
		trailer := fmt.Sprintf("Co-authored-by: %s <%s>", commit.Author.Name, commit.Author.Email)
		if commit.Author.Email != author.Email && !slices.Contains(coauthors, trailer) {
			coauthors = append(coauthors, trailer)
		}
	}
	text := title + "\n"
	if message != "" {
		text += "\n" + message + "\n"
	}
	if len(coauthors) > 0 {
		text += "\n" + strings.Join(coauthors, "\n") + "\n"
	}
	return text
}

func (s *Server) squashCommit(
	r *http.Request,
	dir string,
	pull store.Pull,
	plan mergePlan,
	request mergeRequest,
) (string, error) {
	ctx := r.Context()
	commits, err := s.git.Log(ctx, dir, plan.HeadSHA, squashCommitLimit, plan.BaseSHA)
	if err != nil {
		return "", err
	}
	if len(commits) == 0 {
		return "", store.InvalidError(pull.Base + " already contains every commit on " + pull.Head)
	}
	author := commits[len(commits)-1].Author
	text := squashMessage(pull, commits, request)
	host, _, err := net.SplitHostPort(r.Host)
	if err != nil {
		host = r.Host
	}
	now := time.Now()
	headers := "tree " + plan.tree + "\nparent " + plan.BaseSHA +
		"\nauthor " + gitIdentity(author.Name, author.Email, now) +
		"\ncommitter " + gitIdentity("Lyra", "lyra@"+host, now) + "\n"
	signature, err := sshsig.Sign(strings.NewReader(headers+"\n"+text), s.signer, sshsig.HashSHA512, "git")
	if err != nil {
		return "", err
	}
	armored := strings.TrimSuffix(string(sshsig.Armor(signature)), "\n")
	raw := headers + "gpgsig " + strings.ReplaceAll(armored, "\n", "\n ") + "\n\n" + text
	return s.git.WriteCommit(ctx, dir, []byte(raw))
}

func (s *Server) mergeCommit(
	r *http.Request,
	dir string,
	pull store.Pull,
	plan mergePlan,
	request mergeRequest,
) (string, error) {
	switch {
	case request.Method == methodFastForward && !plan.FastForward:
		return "", conflict(pull.Base + " has commits that " + pull.Head + " doesn't; rebase it or merge with squash")
	case request.Method == methodSquash && len(plan.Conflicts) > 0:
		return "", conflict(
			"merging " + pull.Head + " into " + pull.Base + " conflicts in " + strings.Join(plan.Conflicts, ", "),
		)
	case request.Method == methodSquash:
		return s.squashCommit(r, dir, pull, plan, request)
	}
	return plan.HeadSHA, nil
}

func (s *Server) finishMerge(
	r *http.Request,
	pull store.Pull,
	plan mergePlan,
	request mergeRequest,
	mergeSHA string,
) (int64, error) {
	ctx := r.Context()
	actor := identityFrom(r).actor
	repo := pull.Repo
	dir := s.repoDir(repo)
	if err := s.git.UpdateRef(ctx, dir, "refs/heads/"+pull.Base, mergeSHA, plan.BaseSHA); err != nil {
		return 0, conflict(pull.Base + " moved while merging; try again")
	}
	mergeBase, err := s.git.MergeBase(ctx, dir, plan.BaseSHA, plan.HeadSHA)
	if err != nil {
		mergeBase = pull.BaseSHA
	}
	eventID, err := s.store.MarkPullMerged(ctx, repo, pull.Number, actor, mergeSHA, mergeBase, request.Method)
	if err != nil {
		return 0, err
	}
	updates := []RefUpdate{{Ref: "refs/heads/" + pull.Base, Old: plan.BaseSHA, New: mergeSHA}}
	if request.DeleteBranch {
		if err := s.git.UpdateRef(ctx, dir, "refs/heads/"+pull.Head, "", plan.HeadSHA); err == nil {
			updates = append(updates, RefUpdate{
				Ref: "refs/heads/" + pull.Head,
				Old: plan.HeadSHA,
				New: strings.Repeat("0", len(plan.HeadSHA)),
			})
		}
	}
	pushed := PushEvent{Method: request.Method, Pull: pull.Number, Refs: updates}
	if _, err := s.store.AddEvent(ctx, "push", repo, actor, pushed); err != nil {
		return eventID, err
	}
	if pull.Base == s.git.DefaultBranch(ctx, dir) {
		if _, err := closeReferenced(ctx, s.store, repo, actor, pull.Title+"\n"+pull.Body, mergeSHA); err != nil {
			return eventID, err
		}
	}
	_, err = ProcessPush(ctx, s.store, s.git, dir, repo, actor, updates)
	return eventID, err
}

func (s *Server) mergePull(
	r *http.Request,
	repo string,
	number int64,
	request mergeRequest,
) (store.Pull, int64, error) {
	ctx := r.Context()
	dir := s.repoDir(repo)
	pull, err := s.store.Pull(ctx, repo, number)
	if err != nil {
		return pull, 0, err
	}
	plan, err := s.checkMerge(ctx, repo, pull, request)
	if err != nil {
		return pull, 0, err
	}
	if plan.HeadSHA != pull.HeadSHA {
		if err := s.git.UpdateRef(ctx, dir, pullRef(number), plan.HeadSHA, ""); err != nil {
			return pull, 0, err
		}
		actor := identityFrom(r).actor
		if _, err := s.store.SyncPull(ctx, repo, number, actor, plan.HeadSHA, pull.BaseSHA); err != nil {
			return pull, 0, err
		}
	}
	mergeSHA, err := s.mergeCommit(r, dir, pull, plan, request)
	if err != nil {
		return pull, 0, err
	}
	eventID, err := s.finishMerge(r, pull, plan, request, mergeSHA)
	if err != nil {
		return pull, eventID, err
	}
	pull, err = s.store.Pull(ctx, repo, number)
	return pull, eventID, err
}

func (s *Server) handleMergePull(r *http.Request, input *mergeInput) (*body[pullWrite], error) {
	repo, err := s.lookupRepo(r, scopeWrite)
	if err != nil {
		return nil, err
	}
	request := input.Body
	if err := checkText(nil, &request.Message); err != nil {
		return nil, err
	}
	pull, eventID, err := s.mergePull(r, repo.Name, input.Number, request)
	if err != nil {
		return nil, missing(err, "pull request not found")
	}
	return reply(pullWrite{Pull: pull, EventID: eventID})
}

func (s *Server) uiMergePull(w http.ResponseWriter, r *http.Request) error {
	repo, item, err := s.uiItem(r, kindPull, true)
	if err != nil {
		return err
	}
	request := mergeRequest{
		Method:       r.FormValue("method"),
		HeadSHA:      r.FormValue("head_sha"),
		DeleteBranch: r.FormValue("delete_branch") != "",
	}
	if _, _, err := s.mergePull(r, repo.Name, item.Number, request); err != nil {
		return missing(err, "Pull request not found.")
	}
	http.Redirect(w, r, fmt.Sprintf("/%s/pulls/%d", repo.Name, item.Number), http.StatusSeeOther)
	return nil
}

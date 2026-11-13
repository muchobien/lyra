package server

import (
	"context"
	"fmt"
	"net/http"
	"path"
	"slices"
	"strings"

	"github.com/muchobien/lyra/internal/repos"
	"github.com/muchobien/lyra/internal/store"
)

var mergeMethods = []string{methodFastForward, methodSquash}

type branchPolicy struct {
	rules           []string
	requireSigned   bool
	requireApproval bool
	requireChecks   bool
	forbidForcePush bool
	forbidDeletion  bool
	methods         []string
}

type setRuleRequest struct {
	Pattern         string   `json:"pattern"`
	RequireSigned   bool     `json:"require_signed"`
	RequireApproval bool     `json:"require_approval"`
	RequireChecks   bool     `json:"require_checks"`
	AllowForcePush  bool     `json:"allow_force_push"`
	AllowDeletion   bool     `json:"allow_deletion"`
	MergeMethods    []string `json:"merge_methods"`
}

type ruleWrite struct {
	store.Rule
	EventID int64 `json:"event_id"`
}

func policyFor(rules []store.Rule, branch string) branchPolicy {
	policy := branchPolicy{methods: slices.Clone(mergeMethods)}
	for _, rule := range rules {
		if matched, _ := path.Match(rule.Pattern, branch); !matched {
			continue
		}
		policy.rules = append(policy.rules, rule.Pattern)
		policy.requireSigned = policy.requireSigned || rule.RequireSigned
		policy.requireApproval = policy.requireApproval || rule.RequireApproval
		policy.requireChecks = policy.requireChecks || rule.RequireChecks
		policy.forbidForcePush = policy.forbidForcePush || !rule.AllowForcePush
		policy.forbidDeletion = policy.forbidDeletion || !rule.AllowDeletion
		policy.methods = slices.DeleteFunc(policy.methods, func(method string) bool {
			return !slices.Contains(rule.MergeMethods, method)
		})
	}
	return policy
}

func unverifiedCommits(keys []store.Key, commits []repos.Commit) ([]string, error) {
	signatures := []*repos.Signature{}
	for index := range commits {
		signatures = append(signatures, &commits[index].Signature)
	}
	if err := verifyWithKeys(keys, signatures...); err != nil {
		return nil, err
	}
	problems := []string{}
	for _, commit := range commits {
		if commit.Signature.Status != repos.Verified {
			problems = append(problems, fmt.Sprintf(
				"commit %s (%s) is %s",
				commit.SHA[:7], commit.Subject, commit.Signature.Status,
			))
		}
	}
	return problems, nil
}

func checkBranchPush(
	ctx context.Context,
	git repos.Git,
	dir string,
	keys []store.Key,
	policy branchPolicy,
	branch string,
	update RefUpdate,
) error {
	rule := "branch rule " + strings.Join(policy.rules, ", ")
	if zeroSHA(update.New) {
		if policy.forbidDeletion {
			return fmt.Errorf("%s protects %s from deletion", rule, branch)
		}
		return nil
	}
	if policy.forbidForcePush && !zeroSHA(update.Old) {
		forward, err := git.IsAncestor(ctx, dir, update.Old, update.New)
		if err != nil {
			return err
		}
		if !forward {
			return fmt.Errorf(
				"%s refuses force pushes to %s; push commits on top of %s",
				rule, branch, update.Old[:7],
			)
		}
	}
	if !policy.requireSigned {
		return nil
	}
	exclude := []string{}
	if !zeroSHA(update.Old) {
		exclude = append(exclude, update.Old)
	}
	commits, err := git.Log(ctx, dir, update.New, signedCommitLimit, exclude...)
	if err != nil {
		return err
	}
	problems, err := unverifiedCommits(keys, commits)
	if err != nil {
		return err
	}
	if len(problems) > 0 {
		return fmt.Errorf("%s requires verified signatures on %s: %s", rule, branch, strings.Join(problems, "; "))
	}
	return nil
}

func CheckPush(ctx context.Context, db *store.Store, git repos.Git, dir, repo string, updates []RefUpdate) error {
	rules, err := db.Rules(ctx, repo)
	if err != nil || len(rules) == 0 {
		return err
	}
	keys, err := db.Keys(ctx, "")
	if err != nil {
		return err
	}
	for _, update := range updates {
		branch, isBranch := strings.CutPrefix(update.Ref, "refs/heads/")
		if !isBranch {
			continue
		}
		if err := checkBranchPush(ctx, git, dir, keys, policyFor(rules, branch), branch, update); err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) handleListRules(w http.ResponseWriter, r *http.Request) error {
	repo, err := s.lookupRepo(r, scopeRead)
	if err != nil {
		return err
	}
	rules, err := s.store.Rules(r.Context(), repo.Name)
	if err != nil {
		return err
	}
	return writeJSON(w, http.StatusOK, rules)
}

func (s *Server) handleSetRule(w http.ResponseWriter, r *http.Request) error {
	repo, err := s.lookupRepo(r, scopeAdmin)
	if err != nil {
		return err
	}
	body, err := decodeBody[setRuleRequest](w, r)
	if err != nil {
		return err
	}
	if _, err := path.Match(body.Pattern, ""); err != nil || body.Pattern == "" {
		return badRequest("pattern must be a branch name or glob, such as main or release/*")
	}
	if len(body.MergeMethods) == 0 {
		body.MergeMethods = mergeMethods
	}
	for _, method := range body.MergeMethods {
		if !slices.Contains(mergeMethods, method) {
			return badRequest("merge methods must be ff or squash")
		}
	}
	rule, eventID, err := s.store.SetRule(r.Context(), repo.Name, identityFrom(r).actor, store.Rule{
		Pattern:         body.Pattern,
		RequireSigned:   body.RequireSigned,
		RequireApproval: body.RequireApproval,
		RequireChecks:   body.RequireChecks,
		AllowForcePush:  body.AllowForcePush,
		AllowDeletion:   body.AllowDeletion,
		MergeMethods:    body.MergeMethods,
	})
	if err != nil {
		return missing(err, "repository not found")
	}
	return writeJSON(w, http.StatusOK, ruleWrite{Rule: rule, EventID: eventID})
}

func (s *Server) handleDeleteRule(w http.ResponseWriter, r *http.Request) error {
	repo, err := s.lookupRepo(r, scopeAdmin)
	if err != nil {
		return err
	}
	pattern := r.URL.Query().Get("pattern")
	eventID, err := s.store.DeleteRule(r.Context(), repo.Name, identityFrom(r).actor, pattern)
	if err != nil {
		return missing(err, "no rule for "+pattern)
	}
	return writeJSON(w, http.StatusOK, deletedWithEvent{Deleted: pattern, EventID: eventID})
}

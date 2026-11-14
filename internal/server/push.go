package server

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/muchobien/lyra/internal/repos"
	"github.com/muchobien/lyra/internal/store"
)

type RefUpdate struct {
	Ref string `json:"ref"`
	Old string `json:"old"`
	New string `json:"new"`
}

type push struct {
	db        *store.Store
	git       repos.Git
	dir       string
	repo      string
	actor     store.Actor
	updates   []RefUpdate
	queueRuns bool
}

type branchUpdate struct {
	RefUpdate
	branch string
}

var closingKeyword = regexp.MustCompile(`(?i)\b(?:close[sd]?|fix(?:e[sd])?|resolve[sd]?):?\s+#(\d+)\b`)

func zeroSHA(sha string) bool {
	return strings.Trim(sha, "0") == ""
}

func closeReferenced(
	ctx context.Context,
	db *store.Store,
	repo string,
	actor store.Actor,
	text, commit string,
) ([]string, error) {
	messages := []string{}
	for _, match := range closingKeyword.FindAllStringSubmatch(text, -1) {
		number, err := strconv.ParseInt(match[1], 10, 64)
		if err != nil {
			continue
		}
		issue, err := db.Item(ctx, kindIssue, repo, number)
		if err != nil || issue.State != stateOpen {
			continue
		}
		closed := stateClosed
		edit := store.ItemEdit{State: &closed, Commit: commit}
		if _, _, err := db.EditItem(ctx, kindIssue, repo, number, actor, edit); err != nil {
			return messages, err
		}
		messages = append(messages, fmt.Sprintf("closed issue #%d", number))
	}
	return messages, nil
}

func (work push) branchTips() []branchUpdate {
	tips := []branchUpdate{}
	for _, update := range work.updates {
		branch, isBranch := strings.CutPrefix(update.Ref, "refs/heads/")
		if isBranch && !zeroSHA(update.New) {
			tips = append(tips, branchUpdate{RefUpdate: update, branch: branch})
		}
	}
	return tips
}

func (work push) syncPull(ctx context.Context, pull store.Pull, head string) ([]string, error) {
	baseSHA, err := work.git.MergeBase(ctx, work.dir, "refs/heads/"+pull.Base, head)
	if err != nil {
		baseSHA = pull.BaseSHA
	}
	if err := work.git.UpdateRef(ctx, work.dir, pullRef(pull.Number), head, ""); err != nil {
		return nil, err
	}
	if _, err := work.db.SyncPull(ctx, work.repo, pull.Number, work.actor, head, baseSHA); err != nil {
		return nil, err
	}
	messages := []string{fmt.Sprintf("updated pull request #%d", pull.Number)}
	if !work.queueRuns {
		return messages, nil
	}
	pull.HeadSHA, pull.BaseSHA = head, baseSHA
	queued := QueuePullRuns(ctx, work.db, work.git, work.dir, work.repo, work.actor, pull, "synchronize")
	return append(messages, queued...), nil
}

func (work push) syncPulls(ctx context.Context) ([]string, []string, error) {
	messages, heads := []string{}, []string{}
	open, err := work.db.Pulls(ctx, work.repo, stateOpen)
	if err != nil {
		return messages, heads, err
	}
	for _, tip := range work.branchTips() {
		for _, pull := range open {
			if pull.Head != tip.branch || pull.HeadSHA == tip.New {
				continue
			}
			synced, err := work.syncPull(ctx, pull, tip.New)
			messages = append(messages, synced...)
			if err != nil {
				return messages, heads, err
			}
			heads = append(heads, tip.New)
		}
	}
	return messages, heads, nil
}

func (work push) markMerged(
	ctx context.Context,
	pull store.Pull,
	tip branchUpdate,
	defaultBranch string,
) ([]string, error) {
	merged, err := work.git.IsAncestor(ctx, work.dir, pull.HeadSHA, tip.New)
	if err != nil || !merged {
		return nil, err
	}
	baseSHA := pull.BaseSHA
	if !zeroSHA(tip.Old) {
		if mergeBase, err := work.git.MergeBase(ctx, work.dir, tip.Old, pull.HeadSHA); err == nil {
			baseSHA = mergeBase
		}
	}
	_, err = work.db.MarkPullMerged(ctx, work.repo, pull.Number, work.actor, pull.HeadSHA, baseSHA, "push")
	if err != nil {
		return nil, err
	}
	messages := []string{fmt.Sprintf("merged pull request #%d", pull.Number)}
	if tip.branch != defaultBranch {
		return messages, nil
	}
	closed, err := closeReferenced(ctx, work.db, work.repo, work.actor, pull.Title+"\n"+pull.Body, pull.HeadSHA)
	return append(messages, closed...), err
}

func (work push) mergePulls(ctx context.Context, defaultBranch string) ([]string, error) {
	messages := []string{}
	open, err := work.db.Pulls(ctx, work.repo, stateOpen)
	if err != nil {
		return messages, err
	}
	for _, tip := range work.branchTips() {
		for _, pull := range open {
			if pull.Base != tip.branch {
				continue
			}
			merged, err := work.markMerged(ctx, pull, tip, defaultBranch)
			messages = append(messages, merged...)
			if err != nil {
				return messages, err
			}
		}
	}
	return messages, nil
}

func (work push) closeFromCommits(ctx context.Context, defaultBranch string) ([]string, error) {
	messages := []string{}
	for _, update := range work.updates {
		if update.Ref != "refs/heads/"+defaultBranch || zeroSHA(update.New) {
			continue
		}
		exclude := []string{}
		if !zeroSHA(update.Old) {
			exclude = append(exclude, update.Old)
		}
		commits, err := work.git.Log(ctx, work.dir, update.New, closingCommitLimit, exclude...)
		if err != nil {
			return messages, err
		}
		for _, commit := range slices.Backward(commits) {
			text := commit.Subject + "\n" + commit.Body
			closed, err := closeReferenced(ctx, work.db, work.repo, work.actor, text, commit.SHA)
			messages = append(messages, closed...)
			if err != nil {
				return messages, err
			}
		}
	}
	return messages, nil
}

func ProcessPush(
	ctx context.Context,
	db *store.Store,
	git repos.Git,
	dir, repo string,
	actor store.Actor,
	updates []RefUpdate,
	queueRuns bool,
) ([]string, error) {
	work := push{db: db, git: git, dir: dir, repo: repo, actor: actor, updates: updates, queueRuns: queueRuns}
	defaultBranch := git.DefaultBranch(ctx, dir)
	messages, syncedHeads, err := work.syncPulls(ctx)
	if err != nil {
		return messages, err
	}
	merged, err := work.mergePulls(ctx, defaultBranch)
	messages = append(messages, merged...)
	if err != nil {
		return messages, err
	}
	closed, err := work.closeFromCommits(ctx, defaultBranch)
	messages = append(messages, closed...)
	if err != nil {
		return messages, err
	}
	if queueRuns {
		messages = append(messages, QueueRuns(ctx, db, git, dir, repo, actor, updates)...)
	}
	for _, sha := range syncedHeads {
		if err := db.AnnounceChecks(ctx, repo, sha); err != nil {
			return messages, err
		}
	}
	return messages, nil
}

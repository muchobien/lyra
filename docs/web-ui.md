# Web UI

The web interface targets current stable Chrome. It follows your system's light/dark
appearance initially; the header's appearance control saves your selection for this
browser. Dialogs, select menus and popovers use the browser's native controls.

## Navigation

`/` lists owners and repositories. The header switches repositories and opens Code,
Pull requests, Issues, Actions, Activity and repository settings. On narrow screens,
navigation scrolls horizontally and keeps the selected section visible.

Code contains files, commit history and tags. Directories open at
`/<owner>/<repo>/tree/<ref>/<path>` and files at `/<owner>/<repo>/blob/<ref>/<path>`; the
ref is the longest branch or tag name the path starts with, or else a commit. Add `?raw`
to a file URL to download it. Markdown files open rendered; Code, or `?plain=1`, shows
their highlighted source. Press `T` to open the file finder. The repository sidebar shows the latest
stable release and containers associated with its tag, or the latest container when
there is no match. Releases and containers have their own list and detail views.

## Collaboration

Issues and pull requests support creation, edits, comments, labels and assignees. Lists
filter by state, label, assignee and title or issue number. Pull requests include commits,
checks, structured diffs, line comments, reviews and merging.

Go renders Markdown without raw HTML, highlights source with Chroma and parses diffs.
Diffs offer unified and split layouts; narrow screens use unified layout. Large diffs
are expanded on demand. Binary files have a separate state.

Review, line-comment and merge submissions identify the head that was displayed. Go
validates the head and enforces signatures, approvals, checks and allowed merge methods.
Authors can comment on their work; approving it requires a different author identity.

Comment drafts and viewed-file progress survive navigation inside the application and
reset on reload. Drafts are held in memory, not browser storage.

## On-device AI

Where Chrome provides its built-in AI models, the UI offers features that run on your
device. Their controls appear only when the browser reports the model as available; the
first use downloads the model and shows its progress. Results are suggestions: they
appear in a panel with Stop and Discard, and nothing is saved until you use them.

- **Summarize** on an issue or pull request lists the key points of its description,
  comments and reviews, using the Summarizer API.
- **Explain failure** on a failed job sends the end of the failed step's log to the Prompt
  API and shows the likely cause, the log lines behind it and a suggested fix.
- **Draft description from commits** in the new pull request dialog writes a summary and
  a list of changes from the commits on the compare branch that aren't on the base branch.
  Insert appends it to the description.
- **Draft commit message** in the merge dialog writes a Conventional Commit subject and
  body for a squash merge from the pull request and its commits. Use fills in the commit
  title and message.
- **Suggest labels** in the new issue and pull request dialogs picks up to three of the
  repository's labels from the title and description. Apply adds them to the labels field.
- **Pre-review** on a pull request's Files changed tab reviews each changed file's added
  lines for likely bugs, unhandled errors and leftover debug code. Comment opens the line
  comment dialog with the suggestion filled in, to edit or post.
- **Describe images** in Markdown editors writes alt text for this server's images whose
  alt text is empty or a file name, such as attachments uploaded with the CLI. Use as alt
  text writes the descriptions into the Markdown.
- **Proofread** in Markdown editors fixes spelling, grammar and punctuation without
  changing code, links or structure. Replace swaps the text in the editor for the result.

## Delivery and configuration

Actions discovers workflows at a branch or tag. Workflows with declared manual triggers
provide typed inputs. Runs show jobs, steps and incremental logs, with log filtering,
download, pause/follow, cancellation and reruns. Waiting jobs show their runner labels
or dependencies.

Releases support drafts, publishing, generated notes, asset upload/download, replacement,
deletion and checksums. Matching container tags link releases and container images.
Deleting a release can also remove its Git tag; its container images remain.

Activity refreshes from the repository event stream. Its connection indicator shows when
the stream is reconnecting.

Repository readers can view variables, labels and branch rules. Writers can edit variables
and labels. Repository administrators manage secrets and branch rules; unrestricted
administrators also manage instance tokens, signing keys and runners.

Secret values cannot be read back and are cleared from their editor after saving or
closing. New credentials appear once in a masked field with a copy control and are
cleared when their dialog closes.

# Web UI

The web interface targets current stable Chrome. It follows your system's light/dark
appearance initially; the header's appearance control saves your selection for this
browser. Dialogs, select menus and popovers use the browser's native controls.

## Navigation

`/` lists owners and repositories. The header switches repositories and opens Code,
Pull requests, Issues, Actions, Activity and repository settings. On narrow screens,
navigation scrolls horizontally and keeps the selected section visible.

Code contains files, commit history and tags. Branch, directory and file selection stay
in the URL. Press `T` to open the file finder. The repository sidebar shows the latest
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

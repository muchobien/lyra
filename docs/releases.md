# Releases

A release belongs to a tag.

- **Tags:** if the tag doesn't exist, publishing creates it as a lightweight tag at
  `--target` (default: the default branch). A draft doesn't create its tag until it's
  published. An existing tag is never moved: `--target` must match it.
- **Drafts** are visible only with `write` access.
- **Assets:** `--asset` creates the release as a draft, uploads the files, then publishes it,
  so `release.published` fires once with every asset present. Uploading a name that exists
  needs `--clobber`. `download` checks each file's sha256 before keeping it.
- **Latest** is the most recently published release that isn't a prerelease.
- **Generated notes:** `--generate-notes` appends the pull requests whose merge commit is
  reachable from the release's commit but not from the previous release: the most recently
  published release whose commit is an ancestor. Prereleases count as the previous release
  only for another prerelease. `lyra release notes` previews them.
- **Images:** a release lists the container images tagged with its tag, with or without a
  leading `v` (`v1.2.0` matches `1.2.0`).
- **Events:** `release.created` (drafts), `release.published`, `release.updated`,
  `release.deleted`, and `release.asset` for each upload or deletion. Publishing also runs
  `on: release` [workflows](actions.md).

## Stable download URLs

```
/{owner}/{repo}/releases/download/{tag}/{name}
/{owner}/{repo}/releases/latest/download/{name}
```

They serve assets with their sha256 as the `ETag` and support range requests (`Range`,
`If-Range`), so interrupted downloads can resume.

# Release Checklist

Use this checklist before publishing any `switchbot-cli` release.

## Release inputs

Fill in these values before starting the release process.

- Release version: `v<major>.<minor>.<patch>`
- Release branch: `<branch-name>`
- Release commit: `<commit-sha>`

## Scope and release candidate

- [ ] Confirm the release commit is final and ready to tag.
- [ ] Confirm the working tree is clean before creating the release tag.
- [ ] Confirm the release will be created from the intended branch and commit.
- [ ] Confirm the tag name follows the expected format: `v<major>.<minor>.<patch>`.

## Documentation and versioning

- [ ] Review `README.md` and confirm all user-facing changes are documented.
- [ ] Confirm installation instructions are still accurate for Homebrew and Linux packages.
- [ ] Decide whether `version/version.go` should be updated for non-release local builds.
- [ ] Confirm any release-related documentation is up to date.

## Validation and tests

- [ ] Run `go test ./...`.
- [ ] Run `go vet ./...`.
- [ ] Run `make release-check`.
- [ ] Run `make release-snapshot`.
- [ ] Confirm the snapshot build completes successfully.

## Artifact verification

- [ ] Confirm archive artifacts are generated for the supported platforms.
- [ ] Confirm Linux `.deb` packages are generated.
- [ ] Confirm Linux `.rpm` packages are generated.
- [ ] Confirm release archives include `README*`.
- [ ] Confirm release archives include `LICENSE*`.
- [ ] Verify the generated binaries report the expected release version where applicable.

## GitHub and automation readiness

- [ ] Confirm the GitHub Actions release workflow still triggers on tags matching `v*`.
- [ ] Confirm the GitHub release is configured to be created as a draft.
- [ ] Confirm `HOMEBREW_TAP_GITHUB_TOKEN` is configured in GitHub Actions secrets.
- [ ] Confirm changelog labels are in good shape for the generated release notes.

## Publish steps

- [ ] Create the tag: `git tag v<major>.<minor>.<patch>`
- [ ] Push the tag: `git push origin v<major>.<minor>.<patch>`
- [ ] Monitor the GitHub Actions release workflow until it finishes.
- [ ] Review the generated draft release notes and attached artifacts.
- [ ] Publish the draft release after verification is complete.

## Post-release verification

- [ ] Confirm the Homebrew tap update was published successfully.
- [ ] Confirm the GitHub release contains all expected artifacts.
- [ ] Test a Linux package install path if practical.
- [ ] Record any follow-up fixes or release process issues discovered during publication.

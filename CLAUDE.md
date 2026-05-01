# CLAUDE.md

## Project overview

This is a Spectrocloud fork of [sigstore/cosign](https://github.com/sigstore/cosign). It carries custom patches on top of upstream cosign releases (OCI layout support, remote-load/save, file locking, GC fixes, annotation preservation, etc.).

## Releasing

Releases follow the format `v{upstream_version}_spectro` (e.g., `v2.6.2_spectro`).

### Steps

1. Ensure all CI checks pass on the PR/branch.
2. Create the release targeting the branch with the rebased patches:
   ```
   gh release create v<upstream_version>_spectro \
     --target <branch-name> \
     --title "v<upstream_version>_spectro" \
     --notes "..."
   ```
3. Include in the release notes:
   - Summary of the upstream version jump (e.g., v2.4.x → v2.6.2)
   - Key upstream changes
   - Spectrocloud patches carried forward
   - What changed since the last `_spectro` release
   - Full changelog link: `https://github.com/spectrocloud/cosign/compare/<previous_tag>...v<new_tag>`

### Versioning rules

- The version tracks the upstream cosign version this fork is based on.
- The `_spectro` suffix indicates it is a Spectrocloud fork release.
- When rebasing onto a new upstream version, the tag jumps to match (e.g., `v2.4.18_spectro` → `v2.6.2_spectro`).
- Do **not** increment patch versions beyond the upstream version. If multiple spectro releases are needed on the same upstream base, append a numeric suffix (e.g., `v2.6.2_spectro.1`, `v2.6.2_spectro.2`).

## CI / Security scans

- **gosec**: G115 (int64 → uint64) false positives in upstream code are suppressed with `// #nosec G115` inline annotations. The `.golangci.yml` also excludes G115.
- **gitleaks**: Known false positives (JWT test tokens) are listed in `.gitleaksignore`.

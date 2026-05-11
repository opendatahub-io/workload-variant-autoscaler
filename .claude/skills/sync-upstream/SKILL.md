---
name: sync-upstream
description: Sync code from upstream llm-d/llm-d-workload-variant-autoscaler into opendatahub-io fork. Fetches upstream main, merges into an ODH-based branch, resolves conflicts, scans for leftover conflict markers, pushes, and opens a PR.
disable-model-invocation: true
user-invocable: true
allowed-tools: Bash(git *) Bash(gh *)
argument-hint: "[commit-sha]"
arguments: [sha]
---

# Sync Upstream

Sync code from the upstream llm-d org to a user's fork and open a PR to opendatahub-io.

## Input

- If the user provides a **commit SHA** via `$sha`, use that as the target commit
- If no SHA is given, default to `upstream/main` HEAD

## Workflow

1. **Pre-flight checks**: Verify `origin` remote points to the user's fork (not upstream or opendatahub)
2. **Fetch remotes**: Add/update `upstream` and `opendatahub` remotes, fetch `upstream/main` and `opendatahub/${ODH_BRANCH}`
3. **Resolve target commit**: Use the user-provided SHA, or default to `upstream/main` HEAD. Verify it exists on `upstream/main`
4. **Show what will be synced**: Display a formatted summary to the user before proceeding. Use this exact format:
   ```
   Target: upstream/main HEAD (<short_sha>)
   Into: opendatahub-io/<repo-name> ${ODH_BRANCH} branch

   <N> commits including:
   - <sha> — <commit message>
   - ...

   <N> files changed, +<additions> / -<deletions> lines

   Creating the sync branch and merging now.
   ```
5. **Check for duplicates**: If a local/remote sync branch exists, or an open PR for this SHA already exists on opendatahub-io (from any fork), inform the user and ask whether to proceed or stop
6. **Create sync branch**: Create `sync/upstream-<short_sha>` based on `opendatahub/${ODH_BRANCH}`
7. **Merge upstream**: Merge the target upstream commit into the sync branch. Resolve conflicts if needed
8. **Restore ODH-protected files and verify**: After merge, restore any ODH-protected files to their opendatahub version. Verify merge completed successfully
9. **Scan for conflict markers**: Scan all tracked files for leftover conflict markers (`<<<<`, `>>>>`, `====`) — git can sometimes report a clean merge when markers remain. If any are found, treat them as unresolved conflicts
10. **Push branch**: Push to `origin`
11. **Verify push**: Confirm branch was pushed successfully
12. **Show PR summary**: Display what will be included in the PR
13. **Confirm PR creation**: Ask the user whether to open a PR or stop with the branch pushed only
14. **Open PR** (if confirmed): Open a PR to `opendatahub-io/workload-variant-autoscaler` targeting `${ODH_BRANCH}`
15. **Return to original branch**: Check out the branch the user was on before the sync

## Commands

**IMPORTANT**: Use `git -C <repo_path>` for all git commands to avoid working directory issues. Detect the repository root dynamically.

```bash
# 0. Detect repository path, save current branch, and set config
REPO_PATH=$(git rev-parse --show-toplevel)
ORIGINAL_BRANCH=$(git -C "${REPO_PATH}" rev-parse --abbrev-ref HEAD)
ODH_BRANCH="main"

# 1. Pre-flight: verify origin is the user's fork
ORIGIN_URL=$(git -C "${REPO_PATH}" remote get-url origin)
if echo "${ORIGIN_URL}" | grep -qE '(llm-d/llm-d-workload-variant-autoscaler|opendatahub-io/workload-variant-autoscaler)'; then
  echo "Error: origin remote points to upstream or opendatahub, not your fork"
  exit 1
fi

# 2. Set up remotes and fetch
git -C "${REPO_PATH}" remote add upstream https://github.com/llm-d/llm-d-workload-variant-autoscaler.git 2>/dev/null || \
  git -C "${REPO_PATH}" remote set-url upstream https://github.com/llm-d/llm-d-workload-variant-autoscaler.git
git -C "${REPO_PATH}" remote add opendatahub https://github.com/opendatahub-io/workload-variant-autoscaler.git 2>/dev/null || \
  git -C "${REPO_PATH}" remote set-url opendatahub https://github.com/opendatahub-io/workload-variant-autoscaler.git
git -C "${REPO_PATH}" fetch upstream main
git -C "${REPO_PATH}" fetch opendatahub "${ODH_BRANCH}"

# 3. Resolve target commit
TARGET_COMMIT="${sha:-upstream/main}"
FULL_SHA=$(git -C "${REPO_PATH}" rev-parse "${TARGET_COMMIT}")
SHORT_SHA=$(git -C "${REPO_PATH}" rev-parse --short "${TARGET_COMMIT}")
git -C "${REPO_PATH}" merge-base --is-ancestor "${FULL_SHA}" upstream/main || {
  echo "Error: commit not on upstream/main"; exit 1;
}

# 4. Show what will be synced
echo "=== Commits to be synced ==="
git -C "${REPO_PATH}" log opendatahub/${ODH_BRANCH}..${FULL_SHA} --oneline
echo ""
echo "=== File changes summary ==="
git -C "${REPO_PATH}" diff --stat opendatahub/${ODH_BRANCH}...${FULL_SHA}

# 5. Check for duplicates (local branch, remote branch, and existing PRs from any fork)
BRANCH="sync/upstream-${SHORT_SHA}"

# 5a. Check for existing open PRs from any fork
EXISTING_PR=$(gh pr list --repo opendatahub-io/workload-variant-autoscaler \
  --state open --search "${FULL_SHA}" --json number,title,url,author,createdAt \
  --jq '.[0] | "PR #\(.number): \(.title)\n  URL: \(.url)\n  Author: @\(.author.login)\n  Opened: \(.createdAt)"' 2>/dev/null)
if [ -n "${EXISTING_PR}" ]; then
  echo "BLOCKED: An open PR already exists for this SHA:"
  echo "${EXISTING_PR}"
  echo ""
  echo "Stop here. Show the above to the user and ask if they want to proceed anyway or abort."
  exit 1
fi

# 5b. Check for existing local or remote branch
if git -C "${REPO_PATH}" show-ref --verify --quiet "refs/heads/${BRANCH}"; then
  echo "BLOCKED: Local branch ${BRANCH} already exists."
  echo "Stop here. Ask the user: delete it and continue, or abort."
  exit 1
fi
if git -C "${REPO_PATH}" show-ref --verify --quiet "refs/remotes/origin/${BRANCH}"; then
  echo "BLOCKED: Remote branch origin/${BRANCH} already exists."
  echo "Stop here. Ask the user: delete it and continue, or abort."
  exit 1
fi

# 6. Create branch from opendatahub/${ODH_BRANCH}
git -C "${REPO_PATH}" checkout -b "${BRANCH}" opendatahub/${ODH_BRANCH}

# 7. Merge upstream commit into the sync branch
git -C "${REPO_PATH}" merge --no-ff "${FULL_SHA}" --no-edit \
  -m "Sync upstream llm-d/llm-d-workload-variant-autoscaler ${SHORT_SHA}"
```

If the merge command succeeds, continue to step 8 below.

If the merge command fails or reports conflicts, go to the **Conflict Resolution** section below before continuing.

After the merge (whether clean or after conflict resolution), restore ODH-protected files (`OWNERS*`, `Dockerfile*konflux`, `.tekton/*.yaml`) to their opendatahub version. These files are ODH-only and must never take upstream changes.

```bash
# 8. Restore ODH-protected files
RESTORED=false
for file in $(git -C "${REPO_PATH}" diff --name-only opendatahub/${ODH_BRANCH} HEAD 2>/dev/null); do
  case "${file}" in
    OWNERS*|Dockerfile*konflux|.tekton/*.yaml)
      echo "Restoring ODH-protected file: ${file}"
      git -C "${REPO_PATH}" checkout opendatahub/${ODH_BRANCH} -- "${file}"
      RESTORED=true
      ;;
  esac
done
if [ "${RESTORED}" = true ]; then
  git -C "${REPO_PATH}" add -u
  git -C "${REPO_PATH}" commit --amend --no-edit
fi
```

If the restore and amend succeed, verify with `git -C "${REPO_PATH}" log -1 --stat` and continue to step 9.

```bash
# 9. Scan for leftover conflict markers
# Git can sometimes report a clean merge when conflict markers remain.
CONFLICT_FILES=$(git -C "${REPO_PATH}" grep -rlE '^(<{4,}|={4,}|>{4,})' || true)
if [ -n "${CONFLICT_FILES}" ]; then
  echo "Conflict markers found in the following files after merge:"
  echo "${CONFLICT_FILES}"
  echo ""
  for f in ${CONFLICT_FILES}; do
    echo "--- ${f} ---"
    git -C "${REPO_PATH}" grep -nE '^(<{4,}|={4,}|>{4,})' "${f}"
  done
  # Treat as a merge conflict — show the markers to the user and ask how to resolve
fi
```

## Conflict Resolution

If step 7 produces merge conflicts:

1. List conflicted files with `git -C "${REPO_PATH}" diff --name-only --diff-filter=U`
2. **Auto-resolve ODH-protected files**: For any conflicted file matching ODH-protected patterns (`OWNERS*`, `Dockerfile*konflux`, `.tekton/*.yaml`), immediately resolve by keeping the opendatahub version:
   ```bash
   git -C "${REPO_PATH}" checkout opendatahub/${ODH_BRANCH} -- "${file}"
   git -C "${REPO_PATH}" add "${file}"
   ```
3. Show remaining conflicts to the user
4. Attempt to resolve trivial conflicts automatically (whitespace, import order)
5. For non-trivial conflicts, show the diff and ask the user how to resolve each file
6. After all conflicts are resolved:
   ```bash
   git -C "${REPO_PATH}" add -u
   git -C "${REPO_PATH}" commit --no-edit
   ```

## Push and Open PR

```bash
# 10. Push to origin (user's fork)
git -C "${REPO_PATH}" push -u origin "${BRANCH}"

# 11. Verify push succeeded
if git -C "${REPO_PATH}" ls-remote --heads origin "${BRANCH}" | grep -q "${BRANCH}"; then
  echo "Branch pushed successfully to origin/${BRANCH}"
else
  echo "Push verification failed"
  exit 1
fi
```

**Before creating the PR, show what will be included and ask the user whether to open a PR:**

```bash
# 12. Show PR summary
echo "=== PR Summary ==="
echo "From: upstream/main (${FULL_SHA})"
echo "To: opendatahub-io/workload-variant-autoscaler ${ODH_BRANCH}"
echo ""
echo "Commits to be included:"
git -C "${REPO_PATH}" log opendatahub/${ODH_BRANCH}..HEAD --oneline
echo ""
echo "Files changed:"
git -C "${REPO_PATH}" diff --stat opendatahub/${ODH_BRANCH}..HEAD
```

**Ask the user: "Do you want to open a PR to opendatahub-io or skip?"**

If the user chooses to open the PR:

```bash
# 13. Get fork owner from origin remote URL
# NOTE: Do not use `gh repo view --json owner` — on forks it resolves to the parent repo owner.
FORK_OWNER=$(git -C "${REPO_PATH}" remote get-url origin | sed -E 's|.*[:/]([^/]+)/[^/]+(.git)?$|\1|')

# 14. Open PR to opendatahub-io
gh pr create \
  --repo opendatahub-io/workload-variant-autoscaler \
  --base "${ODH_BRANCH}" \
  --head "${FORK_OWNER}:${BRANCH}" \
  --title "[sync] upstream llm-d main branch ${SHORT_SHA} [$(date -u +%Y-%m-%d)]" \
  --body "$(cat <<EOF
Syncs llm-d/llm-d-workload-variant-autoscaler main branch into ODH ${ODH_BRANCH} branch.

Upstream commit: https://github.com/llm-d/llm-d-workload-variant-autoscaler/commit/${FULL_SHA}
EOF
)"
```

After PR creation (or skip), immediately return to the original branch without asking:

```bash
# 15. Return to the original branch
git -C "${REPO_PATH}" checkout "${ORIGINAL_BRANCH}"
```

If `gh pr create` fails, inform the user that the branch has been pushed to `origin/${BRANCH}` and ask them to create the PR manually at `https://github.com/opendatahub-io/workload-variant-autoscaler/compare/${ODH_BRANCH}...${FORK_OWNER}:${BRANCH}`.

## Completion Summary

After all steps are done, display a formatted completion summary to the user. Use this exact format:

```
Sync completed successfully

- Branch: sync/upstream-<short_sha> pushed to origin
- PR: opendatahub-io/workload-variant-autoscaler#<pr_number>
- URL: <pr_url>
- Syncs: <N> upstream commits from llm-d/llm-d-workload-variant-autoscaler main (<short_sha>) into opendatahub-io ${ODH_BRANCH}
- Conflict resolved: <file> (<resolution details>)   <- only if conflicts were resolved
- Returned to: <original_branch> branch
```

If the user chose to skip the PR, omit the PR and URL lines.

## Error Handling

- If the user-provided SHA does not exist on `upstream/main`, report the error and ask for a valid SHA
- If conflicts cannot be resolved, abort the merge and clean up:
  ```bash
  git -C "${REPO_PATH}" merge --abort
  git -C "${REPO_PATH}" checkout "${ORIGINAL_BRANCH}"
  git -C "${REPO_PATH}" branch -D "${BRANCH}"
  ```
- If the branch already exists, ask the user whether to force-update or skip
- On any failure after branch creation, clean up with:
  ```bash
  git -C "${REPO_PATH}" checkout "${ORIGINAL_BRANCH}"
  git -C "${REPO_PATH}" branch -D "${BRANCH}"
  ```
- Always return the PR URL on success

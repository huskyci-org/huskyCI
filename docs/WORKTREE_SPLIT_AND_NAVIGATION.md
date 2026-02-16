# Git worktrees: feature split and navigation

This document explains how to split the current `feat/setup-wizard-command` changes into **separate git worktrees** (one per feature) and how to work with multiple worktrees in this repo.

---

## 1. Proposed feature split (worktrees)

Current modifications are grouped into **three features**. Each gets its own branch and worktree.

| Feature | Branch name | Purpose |
|--------|-------------|--------|
| **Setup wizard & test-connection** | `feat/setup-wizard-command` | CLI setup wizard, `test-connection` command, local deployment docs |
| **Zip upload for analysis** | `feat/zip-upload-analysis` | API: upload zip, store, extract for local/repo analysis |
| **Multi-platform Docker builds** | `feat/multi-platform-docker-builds` | Scripts and Dockerfiles for building/pushing amd64+arm64 images |

### 1.1 Files per worktree

**Worktree 1 – Setup wizard & test-connection** (`feat/setup-wizard-command`)

- **CLI:** `cli/cmd/setup.go`, `cli/cmd/testConnection.go`, `cli/analysis/analysis.go`, `cli/config/config.go`, `cli/types/types.go`, `cli/util/util.go`
- **Client:** `client/analysis/analysis.go`
- **API (for local dev / healthcheck / version / token):** `api/server.go`, `api/routes/token.go`, `api/routes/analysis.go` (only changes for healthcheck/version if split; otherwise keep analysis.go in zip worktree and accept overlap), `api/token/generator.go`, `api/token/token.go`, `api/token/token_test.go`, `api/config.yaml`, `api/dockers/api.go`, `api/dockers/huskydocker.go`, `api/kubernetes/api.go`, `api/kubernetes/huskykube.go`, `api/types/types.go`, `api/util/api/api.go`, `api/util/util.go`, `api/log/messagecodes.go`
- **Deploy / docs:** `deployments/docker-compose.yml`, `LOCAL_DEPLOYMENT.md`, `README.md`, `tests/e2e/README.md`

**Worktree 2 – Zip upload for analysis** (`feat/zip-upload-analysis`)

- **New:** `api/util/zip.go`
- **Modified:** `api/routes/analysis.go` (UploadZip + zip flow in StartAnalysis), `api/analysis/analysis.go`, `api/dockers/api.go`, `api/dockers/huskydocker.go`, `api/server.go` (POST `/analysis/upload`), `api/log/messagecodes.go` (zip-related codes)

**Worktree 3 – Multi-platform Docker builds** (`feat/multi-platform-docker-builds`)

- **New:** `deployments/scripts/README-multi-platform-build.md`, `deployments/scripts/build-and-push-enry.sh`, `deployments/scripts/build-and-push-multi-platform.sh`
- **Modified:** `deployments/dockerfiles/enry/Dockerfile`, `deployments/dockerfiles/spotbugs/Dockerfile`

**Overlap note:** `api/server.go`, `api/routes/analysis.go`, `api/dockers/*`, `api/log/messagecodes.go` are used by both setup-wizard (local API) and zip-upload. Options: (a) keep these only in **setup-wizard** worktree and add zip changes there so zip is implemented on top of setup-wizard, or (b) keep zip worktree based on `main` and add only zip-specific changes there, then merge zip into setup-wizard later. The commands below use (a): **setup-wizard** worktree keeps the full current API (including zip); **zip-upload** worktree is optional for a “zip-only” branch from `main` if you want to merge that separately.

---

## 2. Creating the worktrees (step-by-step)

Assume repo root: `/Users/guilherme.ferreira/Gits/huskyCI` and default branch `main`. Adjust paths if your clone lives elsewhere.

### 2.1 Save current state

```bash
cd /Users/guilherme.ferreira/Gits/huskyCI

# Option A: commit everything on current branch (simplest for later reference)
git add -A
git status   # review
git commit -m "WIP: setup wizard, zip upload, multi-platform scripts (to be split)"

# Option B: or create a patch of all changes and stash
# git add -A && git diff --cached > /tmp/huskyci-all-changes.patch && git reset HEAD
```

### 2.2 Create worktree for setup wizard (keep as main feature tree)

This worktree keeps the **full** current feature set (setup wizard + zip + docker-compose, etc.). You can later strip zip or multi-platform from this branch if you want a “setup only” branch.

```bash
cd /Users/guilherme.ferreira/Gits/huskyCI

# Already on feat/setup-wizard-command; if you committed in 2.1, this worktree is “current”
# To create a dedicated worktree for this branch elsewhere (e.g. ../huskyCI-setup-wizard):
git worktree add ../huskyCI-setup-wizard feat/setup-wizard-command
```

So:

- **Main repo** `huskyCI` can stay on `feat/setup-wizard-command` (or `main`).
- **../huskyCI-setup-wizard** = worktree for `feat/setup-wizard-command` with all current changes.

### 2.3 Create worktree for zip-upload (zip-only branch from main)

Use a new branch from `main` and bring only zip-related changes into this worktree.

```bash
cd /Users/guilherme.ferreira/Gits/huskyCI

git worktree add -b feat/zip-upload-analysis ../huskyCI-zip-upload main
cd ../huskyCI-zip-upload

# Copy zip-related files from the commit you created in 2.1
COMMIT=$(git -C /Users/guilherme.ferreira/Gits/huskyCI rev-parse feat/setup-wizard-command)
git checkout "$COMMIT" -- api/util/zip.go api/routes/analysis.go api/analysis/analysis.go api/dockers/api.go api/dockers/huskydocker.go api/server.go api/log/messagecodes.go
git add -A && git commit -m "feat(api): zip upload for analysis"
```

So:

- **../huskyCI-zip-upload** = worktree for `feat/zip-upload-analysis` (zip-only API changes).

### 2.4 Create worktree for multi-platform Docker builds

```bash
cd /Users/guilherme.ferreira/Gits/huskyCI

git worktree add -b feat/multi-platform-docker-builds ../huskyCI-multi-platform main
cd ../huskyCI-multi-platform

# Copy script and Dockerfile changes from your WIP commit
COMMIT=$(git -C /Users/guilherme.ferreira/Gits/huskyCI rev-parse feat/setup-wizard-command)
git checkout "$COMMIT" -- deployments/scripts/ deployments/dockerfiles/enry/Dockerfile deployments/dockerfiles/spotbugs/Dockerfile
git add -A && git commit -m "feat(deploy): multi-platform Docker build scripts"
```

So:

- **../huskyCI-multi-platform** = worktree for `feat/multi-platform-docker-builds`.

### 2.5 Summary of worktree layout

| Worktree path | Branch | Purpose |
|---------------|--------|--------|
| `.../huskyCI` | `feat/setup-wizard-command` or `main` | Main repo (or “setup” branch) |
| `.../huskyCI-setup-wizard` | `feat/setup-wizard-command` | Setup wizard + test-connection + local deploy |
| `.../huskyCI-zip-upload` | `feat/zip-upload-analysis` | Zip upload for analysis |
| `.../huskyCI-multi-platform` | `feat/multi-platform-docker-builds` | Multi-platform build scripts |

---

## 3. Navigating and managing worktrees

### 3.1 List worktrees

```bash
# From anywhere (repo or worktree)
git worktree list
```

Example output:

```
/Users/guilherme.ferreira/Gits/huskyCI              abc1234 [feat/setup-wizard-command]
/Users/guilherme.ferreira/Gits/huskyCI-setup-wizard def5678 [feat/setup-wizard-command]
/Users/guilherme.ferreira/Gits/huskyCI-zip-upload   a1b2c3d [feat/zip-upload-analysis]
/Users/guilherme.ferreira/Gits/huskyCI-multi-platform 9e8d7c6 [feat/multi-platform-docker-builds]
```

### 3.2 Switch “where you work” (change directory)

Each worktree is a separate directory; you “switch” by changing directory and using that tree’s branch.

```bash
# Work on setup wizard
cd /Users/guilherme.ferreira/Gits/huskyCI-setup-wizard
git status   # branch: feat/setup-wizard-command

# Work on zip upload
cd /Users/guilherme.ferreira/Gits/huskyCI-zip-upload
git status   # branch: feat/zip-upload-analysis

# Work on multi-platform builds
cd /Users/guilherme.ferreira/Gits/huskyCI-multi-platform
git status   # branch: feat/multi-platform-docker-builds
```

No “checkout” of branch is needed when moving between worktrees: the directory is already bound to its branch.

### 3.3 Create a new worktree (generic)

```bash
# New worktree for an existing branch
git worktree add <path> <branch>

# Example: work on main in a separate folder
git worktree add ../huskyCI-main main

# New worktree with a new branch (branch is created from current HEAD of <start-branch>)
git worktree add -b <new-branch> <path> <start-branch>

# Example: new feature branch from main
git worktree add -b feat/my-feature ../huskyCI-my-feature main
```

### 3.4 Remove a worktree

```bash
# From main repo (or any worktree)
cd /Users/guilherme.ferreira/Gits/huskyCI
git worktree remove ../huskyCI-zip-upload

# If the worktree has uncommitted changes or is “dirty”, force remove:
git worktree remove --force ../huskyCI-zip-upload
```

You can also delete the directory yourself and then run `git worktree prune` in the main repo to clean the worktree list.

### 3.5 Prune stale worktree references

After deleting a worktree directory manually:

```bash
cd /Users/guilherme.ferreira/Gits/huskyCI
git worktree prune
```

### 3.6 See which branch is checked out in each worktree

```bash
git worktree list
# or
git branch -a
# and compare with:
for wt in $(git worktree list --porcelain | awk '/^worktree/ {print $2}'); do
  echo "=== $wt ===" && (cd "$wt" && git branch --show-current && git status -sb)
done
```

### 3.7 Build or run in a specific worktree

```bash
# Build CLI in setup-wizard worktree
cd /Users/guilherme.ferreira/Gits/huskyCI-setup-wizard
make build   # or go build ./cli/...

# Run API in zip-upload worktree
cd /Users/guilherme.ferreira/Gits/huskyCI-zip-upload
make run-api # or whatever your Makefile uses
```

### 3.8 Merge order (when features are ready)

1. Merge **feat/zip-upload-analysis** into **feat/setup-wizard-command** (if you kept zip in setup-wizard, you may already have it).
2. Merge **feat/multi-platform-docker-builds** into **feat/setup-wizard-command** (or into **main**).
3. Merge **feat/setup-wizard-command** into **main**.

Example:

```bash
cd /Users/guilherme.ferreira/Gits/huskyCI-setup-wizard
git fetch origin
git merge feat/zip-upload-analysis
git merge feat/multi-platform-docker-builds
# resolve conflicts if any, then push
```

---

## 4. Quick reference: create worktrees (copy-paste)

```bash
# 1) From huskyCI repo root
cd /Users/guilherme.ferreira/Gits/huskyCI

# 2) Ensure you have a commit with your current work (optional but recommended)
git add -A && git status
git commit -m "WIP: all changes (to split into worktrees)" || true

# 3) Setup wizard worktree (current branch)
git worktree add ../huskyCI-setup-wizard feat/setup-wizard-command

# 4) Zip-upload worktree (new branch from main)
git worktree add -b feat/zip-upload-analysis ../huskyCI-zip-upload main

# 5) Multi-platform worktree (new branch from main)
git worktree add -b feat/multi-platform-docker-builds ../huskyCI-multi-platform main

# 6) List
git worktree list
```

After that, copy only the files that belong to each feature into **huskyCI-zip-upload** and **huskyCI-multi-platform** from your WIP commit (e.g. `git checkout <commit> -- <files>`), then commit in each worktree. The **huskyCI-setup-wizard** worktree already has the full state if you committed in step 2.

---

## 5. Summary

- **Split:** One worktree per feature (setup wizard, zip upload, multi-platform builds); each has its own branch and directory.
- **Navigate:** `cd` to the worktree directory; that directory is tied to one branch.
- **Create:** `git worktree add <path> <branch>` or `git worktree add -b <new-branch> <path> <start-branch>`.
- **List/remove:** `git worktree list`, `git worktree remove <path>`, `git worktree prune`.

This keeps features isolated while sharing the same repo history and makes it easier to open different worktrees in different editor windows or to run different branches side by side.

#!/usr/bin/env bash
# Merge request checks, run by the mr-check job in .gitlab-ci.yml (docs/06-ci-cd.md §4):
#   - title: Conventional Commits, becomes the squash commit message (STD-GIT-02);
#   - description of a feat or fix MR: names a WP- and a PRD ID (STD-GIT-04);
#   - migrations: ones already on the target branch never change, and new ones are numbered after the
#     target branch's latest, so goose applies them in order on staging and production (STD-DB-01).
# Input: GitLab's predefined CI_MERGE_REQUEST_* variables (set them by hand to run it locally) and a
# clone that contains the merge base (GIT_DEPTH 0). GitLab reads the title and description when the
# pipeline is created: after editing them, run a new pipeline.
set -euo pipefail

failed=0
fail() { printf 'mr-check: %s\n' "$*" >&2; failed=1; }

title=${CI_MERGE_REQUEST_TITLE:?}
title=${title#Draft: }
re='^(feat|fix|test|docs|refactor|chore)(\([^()]+\))?!?: [^ ]'
[[ $title =~ $re ]] ||
  fail "title must be <type>(<scope>): <summary>, type feat|fix|test|docs|refactor|chore, '!' for breaking, e.g. 'feat(WP-14): ...' (STD-GIT-02)"

# feat and fix change behaviour, so they trace to a WP and the PRD; docs, chore, refactor and test may not.
desc=${CI_MERGE_REQUEST_DESCRIPTION:-}
if [[ $title =~ ^(feat|fix) ]]; then
  grep -Eq '\bWP-[0-9]{2}\b' <<<"$desc" ||
    fail "description must name the work package, e.g. WP-14 (STD-GIT-04, template .gitlab/merge_request_templates/Default.md)"
  grep -Eq '\b((FR|BR)-[A-Z]+-[0-9]{2}|NFR-[0-9]{2})\b' <<<"$desc" ||
    fail "description must name the PRD IDs it implements, e.g. FR-SRCH-01, BR-EST-02, NFR-03 (STD-GIT-04)"
fi

migrations=apps/api/db/migrations
base=${CI_MERGE_REQUEST_DIFF_BASE_SHA:?}
git fetch --quiet origin "${CI_MERGE_REQUEST_TARGET_BRANCH_NAME:?}"
changed=$(git diff --name-only --diff-filter=DMRT "$base" HEAD -- "$migrations/*.sql")
[ -z "$changed" ] ||
  fail "migrations already on ${CI_MERGE_REQUEST_TARGET_BRANCH_NAME} never change; write a new migration instead (STD-DB-01): $(echo "$changed" | tr '\n' ' ')"

version() { local v=${1##*/}; v=${v%%_*}; echo $((10#$v)); }
latest=0
for f in $(git ls-tree --name-only FETCH_HEAD "$migrations/" | grep '\.sql$'); do
  v=$(version "$f"); [ "$v" -le "$latest" ] || latest=$v
done
for f in $(git diff --name-only --diff-filter=A "$base" HEAD -- "$migrations/*.sql"); do
  [ "$(version "$f")" -gt "$latest" ] ||
    fail "$f: number must be above $latest, the latest migration on ${CI_MERGE_REQUEST_TARGET_BRANCH_NAME}; rebase and renumber (STD-DB-01)"
done

[ "$failed" -eq 0 ] && echo "mr-check: ok"
exit "$failed"

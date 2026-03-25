#!/usr/bin/env bash
# Fetch GitHub releases and write docs/releases.json (for CI or local use).
# Requires: jq. One of: GITHUB_TOKEN + curl, or gh (logged in) for gh api.
set -euo pipefail

main() {
  local script_dir
  script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

  local github_repo="${GITHUB_REPOSITORY:-}"
  if [[ -z "$github_repo" ]] && command -v gh &>/dev/null; then
    github_repo="$(gh repo view --json nameWithOwner -q .nameWithOwner 2>/dev/null || true)"
  fi
  if [[ -z "$github_repo" ]]; then
    echo "error: set GITHUB_REPOSITORY=owner/repo or run from a repo with gh" >&2
    return 1
  fi

  local github_api_url="${GITHUB_API_URL:-https://api.github.com}"

  local output_path="${RELEASES_JSON_PATH:-}"
  if [[ -z "$output_path" ]]; then
    local repo_root
    repo_root="$(git -C "${script_dir}/.." rev-parse --show-toplevel 2>/dev/null || true)"
    if [[ -z "$repo_root" ]]; then
      echo "error: run from a git checkout or set RELEASES_JSON_PATH" >&2
      return 1
    fi
    output_path="${repo_root}/docs/releases.json"
  fi

  local use_gh=false
  local -a curl_headers=()
  if [[ -n "${GITHUB_TOKEN:-}" ]]; then
    curl_headers=(
      -fsS
      -H "Authorization: Bearer ${GITHUB_TOKEN}"
      -H "Accept: application/vnd.github+json"
      -H "X-GitHub-Api-Version: 2022-11-28"
    )
  elif command -v gh &>/dev/null && gh auth status &>/dev/null; then
    use_gh=true
  else
    echo "error: set GITHUB_TOKEN or run: gh auth login" >&2
    return 1
  fi

  local page=1
  local all_releases='[]'

  local part page_count
  while true; do
    if [[ "$use_gh" == true ]]; then
      part="$(gh api "repos/${github_repo}/releases" -f per_page=100 -f page="${page}")"
    else
      part="$(curl "${curl_headers[@]}" "${github_api_url}/repos/${github_repo}/releases?per_page=100&page=${page}")"
    fi
    page_count="$(jq 'length' <<<"$part")"

    if ((page_count == 0)); then
      break
    fi

    all_releases="$(jq -n --argjson a "$all_releases" --argjson b "$part" '$a + $b')"
    ((page++)) || true

    if ((page_count < 100)); then
      break
    fi
  done

  mkdir -p "$(dirname "$output_path")"
  echo "$all_releases" | jq -f "${script_dir}/update-releases-json.jq" >"$output_path"
  echo "wrote ${output_path}"
}

main "$@"

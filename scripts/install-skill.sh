#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLI_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
SOURCE_ROOT="${CLI_DIR}/skills"
TARGET_ROOT="${LAPS_SKILLS_DIR:-${HOME}/.agents/skills}"
ALL_SKILLS=(
  laps-cli-auth
  laps-orders
  laps-material-master
  laps-material-readiness
  production-scheduling
  laps-capacity
  laps-master-data
  laps-scheduling-policy
  laps-workbuddy-mcp
)

skills=("${ALL_SKILLS[@]}")
if (( $# > 0 )); then
  skills=("$@")
fi

is_known_skill() {
  local candidate="$1"
  local known
  for known in "${ALL_SKILLS[@]}"; do
    [[ "${candidate}" == "${known}" ]] && return 0
  done
  return 1
}

mkdir -p "${TARGET_ROOT}"

for skill in "${skills[@]}"; do
  if ! is_known_skill "${skill}"; then
    echo "unknown skill: ${skill}" >&2
    exit 2
  fi

  source_dir="${SOURCE_ROOT}/${skill}"
  target_dir="${TARGET_ROOT}/${skill}"
  if [[ ! -f "${source_dir}/SKILL.md" || ! -f "${source_dir}/agents/openai.yaml" ]]; then
    echo "invalid source skill: ${source_dir}" >&2
    exit 2
  fi

  staging_root="$(mktemp -d "${TARGET_ROOT}/.laps-skill-install.XXXXXX")"
  staging_dir="${staging_root}/${skill}"
  cp -R "${source_dir}" "${staging_dir}"
  if [[ ! -f "${staging_dir}/SKILL.md" || ! -f "${staging_dir}/agents/openai.yaml" ]]; then
    rm -rf "${staging_root}"
    echo "staged skill validation failed: ${skill}" >&2
    exit 2
  fi

  backup_dir="${staging_root}/previous"
  if [[ -e "${target_dir}" ]]; then
    mv "${target_dir}" "${backup_dir}"
  fi
  mv "${staging_dir}" "${target_dir}"
  rm -rf "${staging_root}"
  echo "installed ${skill} to ${target_dir}"
done

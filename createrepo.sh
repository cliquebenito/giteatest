#!/usr/bin/env bash

# ══ Настройки для проектов и репозиториев ══
PROJECT_START=1       # первый номер проекта (project_key1)
PROJECT_END=800       # последний номер проекта (project_key800)
REPOS_PER_PROJECT=3   # сколько репо создать в каждом проекте
REPO_COUNTER=1     # с какого номера начинать repo_key

# ══ Параметры API ══
API_URL="http://localhost:3000/api/v2/projects/repos"
TENANT_KEY="tenant"
DEFAULT_BRANCH="main"
DESCRIPTION="Описание"
VISIBILITY=true      # true — приватный, false — публичный

# ══ Basic Auth ══
AUTH_USER="cliquebenito"
AUTH_PASS="cliquebenito"  # или ваш персональный токен, если требуется

for (( proj=PROJECT_START; proj<=PROJECT_END; proj++ )); do
  project_key="project_key${proj}"

  for (( i=1; i<=REPOS_PER_PROJECT; i++ )); do
    repo_key="repo_key${REPO_COUNTER}"

    echo "Создаём репозиторий '${repo_key}' в проекте '${project_key}'..."

    curl --silent --show-error --location --request POST "${API_URL}" \
      --user "${AUTH_USER}:${AUTH_PASS}" \
      --header "Content-Type: application/json" \
      --data "{
        \"tenant_key\":     \"${TENANT_KEY}\",
        \"project_key\":    \"${project_key}\",
        \"repository_key\": \"${repo_key}\",
        \"default_branch\": \"${DEFAULT_BRANCH}\",
        \"description\":    \"${DESCRIPTION}\",
        \"name\":           \"${repo_key}\",
        \"private\":        ${VISIBILITY}
      }" \
    && echo "   → OK" || echo "   → FAIL"

    REPO_COUNTER=$(( REPO_COUNTER + 1 ))
  done

  echo
done

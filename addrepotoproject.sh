#!/usr/bin/env bash

# ══ Настройки для одного проекта ══
PROJECT_KEY="project_key10"  # фиксированный ключ проекта
REPOS_TO_CREATE=2400           # сколько репозиториев создать
REPO_COUNTER=2300               # с какого номера начинать repo_key

# ══ Параметры API ══
API_URL="http://localhost:3000/api/v2/projects/repos"
TENANT_KEY="tenant"
DEFAULT_BRANCH="main"
DESCRIPTION="Описание"
VISIBILITY=true  # true — приватный, false — публичный

# ══ Basic Auth ══
AUTH_USER="cliquebenito"
AUTH_PASS="cliquebenito"  # или токен

for (( i=1; i<=REPOS_TO_CREATE; i++ )); do
  repo_key="repo_key${REPO_COUNTER}"

  echo "Создаём репозиторий '${repo_key}' в проекте '${PROJECT_KEY}'..."

  curl --silent --show-error --location --request POST "${API_URL}" \
    --user "${AUTH_USER}:${AUTH_PASS}" \
    --header "Content-Type: application/json" \
    --data "{
      \"tenant_key\":     \"${TENANT_KEY}\",
      \"project_key\":    \"${PROJECT_KEY}\",
      \"repository_key\": \"${repo_key}\",
      \"default_branch\": \"${DEFAULT_BRANCH}\",
      \"description\":    \"${DESCRIPTION}\",
      \"name\":           \"${repo_key}\",
      \"private\":        ${VISIBILITY}
    }" \
  && echo "   → OK" || echo "   → FAIL"

  REPO_COUNTER=$(( REPO_COUNTER + 1 ))
done

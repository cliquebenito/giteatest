#!/bin/bash

# Проверка количества аргументов
if [ "$#" -ne 5 ]; then
  echo "Usage: $0 <org_id> <team_id> <user_id> <tenant_id> <team_name>"
  echo "Example: $0 1 18 4 42 test_for_vlad"
  exit 1
fi

# Подключение к БД (хардкод)
DB_TYPE="postgres"
PGHOST="localhost"
PGPORT="5432"
PGUSER="postgres"
PGDATABASE="postgres"
PGPASSWORD="postgres"
SCHEMA="public"

# Проверка наличия psql
if ! command -v psql >/dev/null 2>&1; then
  echo "Error: psql not found. Please install PostgreSQL client."
  exit 1
fi

# Аргументы
ORG_ID=$1
TEAM_ID=$2
USER_ID=$3
TENANT_ID=$4
TEAM_NAME=$5

export PGPASSWORD=$PGPASSWORD

# Выполнение SQL
psql -h "$PGHOST" -p "$PGPORT" -U "$PGUSER" -d "$PGDATABASE" <<EOSQL
DO \$\$
DECLARE
  v_org_id      INT := $ORG_ID;
  v_team_id     BIGINT := $TEAM_ID;
  v_user_id     INT := $USER_ID;
  v_tenant_id   TEXT := '$TENANT_ID';
  v_team_name   TEXT := '$TEAM_NAME';
BEGIN
  RAISE NOTICE '▶ Добавление пользователя % в команду % (org_id=%)', v_user_id, v_team_id, v_org_id;

  INSERT INTO "${SCHEMA}"."team_user" ("org_id", "team_id", "uid")
  VALUES (v_org_id, v_team_id, v_user_id);

  UPDATE "${SCHEMA}"."team"
  SET "num_members" = "num_members" + 1
  WHERE "id" = v_team_id;

  UPDATE "${SCHEMA}"."access"
  SET "mode" = 4
  WHERE "user_id" = v_user_id
    AND "repo_id" IN (
      SELECT repo_id FROM "${SCHEMA}"."team_repo" WHERE team_id = v_team_id
    )
    AND (mode < 4);

  RAISE NOTICE '▶ Добавление casbin-правила p4 для пользователя %', v_user_id;

  INSERT INTO "${SCHEMA}"."_casbin_rule" (ptype, v0, v1, v2, v3)
  VALUES ('p4', v_user_id::text, v_tenant_id, v_org_id::text, v_team_name);

  RAISE NOTICE '✔ Выполнено успешно.';
END
\$\$;
EOSQL

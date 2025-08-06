#!/bin/bash

# Проверка количества аргументов
if [ "$#" -ne 5 ]; then
  echo "Usage: $0 <org_id> <team_id> <repo_ids_csv> <team_name> <custom_privs>"
  echo "Example: $0 2 18 \"1,2,3\" test_for_vlad '1,3,4'"
  exit 1
fi

# Аргументы
ORG_ID=$1
TEAM_ID=$2
REPO_IDS_CSV=$3
TEAM_NAME=$4
CUSTOM_PRIVS=$5

# Подключение к БД
DB_TYPE="postgres"
PGHOST="localhost"
PGPORT="5432"
PGUSER="postgres"
PGDATABASE="postgres"
PGPASSWORD="postgres"
SCHEMA="public"

export PGPASSWORD=$PGPASSWORD

# Преобразуем CSV в SQL-массив
REPO_IDS_ARRAY="ARRAY[$REPO_IDS_CSV]"

# Выполнение SQL
psql -h "$PGHOST" -p "$PGPORT" -U "$PGUSER" -d "$PGDATABASE" <<EOSQL
DO \$\$
DECLARE
  v_org_id         INT := $ORG_ID;
  v_team_id        BIGINT := $TEAM_ID;
  v_team_name      TEXT := '$TEAM_NAME';
  v_repo_ids       INT[] := $REPO_IDS_ARRAY;
  v_repo_id        INT;
  v_custom_privs   TEXT := '$CUSTOM_PRIVS';
  privileges       TEXT := 'vB_cPR_aPr';
BEGIN
  RAISE NOTICE '▶ Очистка team_repo и счётчиков';
  DELETE FROM ${SCHEMA}.team_repo WHERE team_id = v_team_id;
  UPDATE ${SCHEMA}.team SET num_repos = 0 WHERE id = v_team_id;

  DELETE FROM ${SCHEMA}.sc_team_custom_privilege WHERE team_name = v_team_name;
  DELETE FROM ${SCHEMA}._casbin_rule WHERE ptype = 'p5' AND v0 = v_team_name;

  FOREACH v_repo_id IN ARRAY v_repo_ids LOOP
    INSERT INTO ${SCHEMA}.team_repo (org_id, team_id, repo_id)
    VALUES (v_org_id, v_team_id, v_repo_id);

    UPDATE ${SCHEMA}.team
    SET num_repos = num_repos + 1
    WHERE id = v_team_id;

    INSERT INTO ${SCHEMA}.sc_team_custom_privilege (team_name, repository_id, all_repositories, custom_privileges)
    VALUES (v_team_name, v_repo_id, false, v_custom_privs);

    IF length(v_custom_privs) > 3 THEN
      privileges := 'vB_chB_cPR_aPr_mPr';
    END IF;

    INSERT INTO ${SCHEMA}._casbin_rule (ptype, v0, v1, v2, v3)
    VALUES ('p5', v_team_name, v_org_id::text, v_repo_id::text, privileges);
  END LOOP;

  RAISE NOTICE '✔ Репозитории и привилегии успешно обновлены.';
END
\$\$;
EOSQL

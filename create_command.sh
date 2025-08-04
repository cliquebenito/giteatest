#!/bin/bash


if [ "$#" -ne 3 ]; then
  echo "Usage: $0 <project_id> <team_name> <description>"
  echo "Example: $0 2 my_team \"Team for backend services\""
  exit 1
fi

ORG_ID=$1
TEAM_NAME=$2
DESCRIPTION=$3

# DB setup...
DB_TYPE="postgres"
PGHOST="localhost"
PGPORT="5432"
PGUSER="postgres"
PGDATABASE="postgres"
PGPASSWORD="postgres"
SCHEMA="public"

PGPASSWORD=$PGPASSWORD psql -h "$PGHOST" -p "$PGPORT" -U "$PGUSER" -d "$PGDATABASE" <<EOSQL
DO \$\$
DECLARE
  v_org_id int := $ORG_ID;
  v_team_name text := '$TEAM_NAME';
  v_description text := '$DESCRIPTION';
  v_team_id bigint;
BEGIN
  INSERT INTO ${SCHEMA}.team
    (org_id, lower_name, name, description, authorize, num_repos, num_members, includes_all_repositories, can_create_org_repo)
  VALUES
    (v_org_id, v_team_name, v_team_name, v_description, 0, 0, 0, false, false)
  RETURNING id INTO v_team_id;

  RAISE NOTICE '------ Команда создана: ID: % -----------', v_team_id;

  INSERT INTO ${SCHEMA}.team_unit (org_id, team_id, type, access_mode)
  SELECT v_org_id, v_team_id, generate_series(1, 10), 0;
END
\$\$;
EOSQL

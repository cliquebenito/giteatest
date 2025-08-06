#!/usr/bin/env bash
# ══ Жёстко зашитое стартовое значение ══
START=6     # первый стартовый номер
END=800      # последний стартовый номер (не включая его)

# Цикл от START до END-1
for (( s=START; s<END; s++ )); do
  # Всегда +1
  count=$(( s + 1 ))

  # Формируем имена
  project_name="project${count}"
  project_key="project_key${count}"

  echo "Создаём проект #${count}: ${project_name} / ${project_key}"

  # Отправляем запрос с Basic Auth
  curl --silent --show-error --location --request POST 'http://localhost:3000/api/v2/projects/create' \
    --user 'cliquebenito:cliquebenito' \
    --header 'Content-Type: application/json' \
    --data "{
      \"tenant_key\":   \"tenant\",
      \"name\":         \"${project_name}\",
      \"project_key\":  \"${project_key}\",
      \"description\":  \"eblo\",
      \"visibility\":   1
    }" \
  && echo " → OK" || echo " → FAIL"

  echo
done

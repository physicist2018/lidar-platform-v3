#!/bin/bash
set -euo pipefail

# Проверяем, что ветка master существует и мы на ней
git rev-parse --verify HEAD >/dev/null || { echo "Нет коммитов в репозитории"; exit 1; }
current_branch="$(git branch --show-current)"
if [[ "$current_branch" != "master" ]]; then
  echo "Вы не на ветке master, текущая ветка: $current_branch"
  exit 1
fi

# Список удалённых репозиториев, куда нужно пушить
for remote in origin nas gitlab; do
  # Проверяем, существует ли remote
  if git remote get-url "$remote" >/dev/null 2>&1; then
    echo "Пушим в $remote..."
    git push -u "$remote" master
  else
    echo "Remote '$remote' не найден, пропускаем."
  fi
done

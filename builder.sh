#!/bin/bash
set -e

# Настройка версии: берём git-тег, если нет — короткий хеш
VERSION=$(git describe --tags --abbrev=0 2>/dev/null || git rev-parse --short HEAD)
echo "Сборка версии: $VERSION"

# Экспорт переменной для docker-compose
export GIT_VERSION=$VERSION

# Имя сервиса
SERVICE_NAME=mail2tg

cd docker || exit 1

# Останавливаем и удаляем старый контейнер
if [ "$(docker ps -q -f name=$SERVICE_NAME)" ]; then
  echo "Останавливаем контейнер $SERVICE_NAME..."
  docker stop $SERVICE_NAME
fi
if [ "$(docker ps -aq -f name=$SERVICE_NAME)" ]; then
  echo "Удаляем контейнер $SERVICE_NAME..."
  docker rm $SERVICE_NAME
fi

# Удаляем старый образ mail2tg:latest, если он есть
OLD_IMAGE=$(docker images -q mail2tg:latest)
if [ -n "$OLD_IMAGE" ]; then
  echo "Удаляем старый образ mail2tg:latest..."
  docker rmi "$OLD_IMAGE"
else
  echo "Образ mail2tg:latest не найден, пропускаем удаление."
fi


# Сборка нового образа и запуск
docker-compose build --build-arg APP_VERSION=$GIT_VERSION
docker-compose up -d

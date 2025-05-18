#!/bin/sh

# Ждем, пока Kafka будет доступна
echo "Waiting for Kafka to be ready..."
while ! nc -z kafka 9092; do
  sleep 1
done
echo "Kafka is ready!"

# Запускаем приложение
exec "$@" 
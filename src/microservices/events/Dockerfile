# Используем официальный образ Go
FROM golang:1.21-alpine

# Устанавливаем необходимые пакеты
RUN apk add --no-cache netcat-openbsd

# Устанавливаем рабочую директорию
WORKDIR /app

# Копируем файлы зависимостей
COPY go.mod ./
COPY go.sum ./

# Загружаем зависимости
RUN go mod download

# Копируем исходный код
COPY . .

# Делаем скрипт ожидания исполняемым
RUN chmod +x wait-for-kafka.sh

# Собираем приложение
RUN go build -o main .

# Открываем порт
EXPOSE 8082

# Запускаем приложение через скрипт ожидания
CMD ["./wait-for-kafka.sh", "./main"] 
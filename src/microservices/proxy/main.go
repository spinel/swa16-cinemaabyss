package main

import (
	"encoding/json"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// Конфигурация сервиса
type Config struct {
	Port                   string
	MonolithURL            string
	MoviesServiceURL       string
	EventsServiceURL       string
	GradualMigration       bool
	MoviesMigrationPercent int
}

// Загрузка конфигурации из переменных окружения
func loadConfig() *Config {
	gradualMigration, _ := strconv.ParseBool(getEnv("GRADUAL_MIGRATION", "false"))
	moviesMigrationPercent, _ := strconv.Atoi(getEnv("MOVIES_MIGRATION_PERCENT", "75"))

	return &Config{
		Port:                   getEnv("PORT", "8000"),
		MonolithURL:            getEnv("MONOLITH_URL", "http://localhost:8080"),
		MoviesServiceURL:       getEnv("MOVIES_SERVICE_URL", "http://localhost:8081"),
		EventsServiceURL:       getEnv("EVENTS_SERVICE_URL", "http://localhost:8082"),
		GradualMigration:       gradualMigration,
		MoviesMigrationPercent: moviesMigrationPercent,
	}
}

// Получение значения переменной окружения с дефолтным значением
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// Проксирование запроса к целевому сервису
func proxyRequest(targetURL string, w http.ResponseWriter, r *http.Request) {
	// Создаем новый запрос
	req, err := http.NewRequest(r.Method, targetURL+r.URL.Path, r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Копируем заголовки
	for name, values := range r.Header {
		for _, value := range values {
			req.Header.Add(name, value)
		}
	}

	// Добавляем query параметры
	req.URL.RawQuery = r.URL.RawQuery

	// Выполняем запрос
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// Копируем заголовки ответа
	for name, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(name, value)
		}
	}

	// Устанавливаем статус код
	w.WriteHeader(resp.StatusCode)

	// Копируем тело ответа
	io.Copy(w, resp.Body)
}

// Обработчик для эндпоинтов фильмов
func handleMovies(w http.ResponseWriter, r *http.Request, config *Config) {
	// Если постепенная миграция выключена, используем монолит
	if !config.GradualMigration {
		proxyRequest(config.MonolithURL, w, r)
		return
	}

	// Генерируем случайное число для определения целевого сервиса
	rand.Seed(time.Now().UnixNano())
	randomPercent := rand.Intn(100)

	// Выбираем целевой сервис на основе процента миграции
	targetURL := config.MonolithURL
	if randomPercent < config.MoviesMigrationPercent {
		targetURL = config.MoviesServiceURL
	}

	proxyRequest(targetURL, w, r)
}

// Обработчик для всех остальных эндпоинтов
func handleDefault(w http.ResponseWriter, r *http.Request, config *Config) {
	proxyRequest(config.MonolithURL, w, r)
}

// Проверка здоровья сервиса
func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"status": true})
}

func main() {
	// Загружаем конфигурацию
	config := loadConfig()

	// Инициализируем генератор случайных чисел
	rand.Seed(time.Now().UnixNano())

	// Настраиваем маршруты
	http.HandleFunc("/health", handleHealth)

	// Обработчик для всех запросов
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Определяем тип запроса
		path := strings.TrimPrefix(r.URL.Path, "/")
		parts := strings.Split(path, "/")

		if len(parts) > 1 && parts[0] == "api" {
			switch parts[1] {
			case "movies":
				handleMovies(w, r, config)
			default:
				handleDefault(w, r, config)
			}
		} else {
			handleDefault(w, r, config)
		}
	})

	// Запускаем сервер
	log.Printf("Starting proxy service on port %s", config.Port)
	log.Printf("Movies migration percent: %d%%", config.MoviesMigrationPercent)
	log.Fatal(http.ListenAndServe(":"+config.Port, nil))
}

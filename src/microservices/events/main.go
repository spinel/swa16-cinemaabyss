package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/Shopify/sarama"
)

// Конфигурация сервиса
type Config struct {
	Port        string
	KafkaBroker string
}

// Модели событий
type Event struct {
	Type      string          `json:"type"`
	Data      json.RawMessage `json:"data"`
	Timestamp time.Time       `json:"timestamp"`
}

type UserEvent struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

type PaymentEvent struct {
	ID     int     `json:"id"`
	UserID int     `json:"user_id"`
	Amount float64 `json:"amount"`
	Status string  `json:"status"`
}

type MovieEvent struct {
	ID          int      `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Genres      []string `json:"genres"`
	Rating      float64  `json:"rating"`
}

// Загрузка конфигурации
func loadConfig() *Config {
	return &Config{
		Port:        getEnv("PORT", "8082"),
		KafkaBroker: getEnv("KAFKA_BROKER", "kafka:9092"),
	}
}

// Получение значения переменной окружения
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// Создание Kafka producer
func createProducer(broker string) (sarama.SyncProducer, error) {
	config := sarama.NewConfig()
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Retry.Max = 5
	config.Producer.Return.Successes = true

	producer, err := sarama.NewSyncProducer([]string{broker}, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create producer: %v", err)
	}

	return producer, nil
}

// Создание Kafka consumer
func createConsumer(broker string) (sarama.Consumer, error) {
	config := sarama.NewConfig()
	config.Consumer.Return.Errors = true

	consumer, err := sarama.NewConsumer([]string{broker}, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create consumer: %v", err)
	}

	return consumer, nil
}

// Отправка события в Kafka
func sendEvent(producer sarama.SyncProducer, topic string, event Event) error {
	event.Timestamp = time.Now()
	eventBytes, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %v", err)
	}

	msg := &sarama.ProducerMessage{
		Topic: topic,
		Value: sarama.StringEncoder(eventBytes),
	}

	_, _, err = producer.SendMessage(msg)
	if err != nil {
		return fmt.Errorf("failed to send message: %v", err)
	}

	return nil
}

// Обработка событий из Kafka
func consumeEvents(consumer sarama.Consumer, topic string, wg *sync.WaitGroup) {
	defer wg.Done()

	partitionConsumer, err := consumer.ConsumePartition(topic, 0, sarama.OffsetNewest)
	if err != nil {
		log.Printf("Failed to create partition consumer: %v", err)
		return
	}
	defer partitionConsumer.Close()

	for {
		select {
		case msg := <-partitionConsumer.Messages():
			var event Event
			if err := json.Unmarshal(msg.Value, &event); err != nil {
				log.Printf("Failed to unmarshal event: %v", err)
				continue
			}

			log.Printf("Received event: Type=%s, Data=%s, Timestamp=%v",
				event.Type, string(event.Data), event.Timestamp)
		case err := <-partitionConsumer.Errors():
			log.Printf("Error consuming message: %v", err)
		}
	}
}

// Обработчики HTTP
func handleCreateEvent(w http.ResponseWriter, r *http.Request, producer sarama.SyncProducer) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Определяем тип события из URL
	path := r.URL.Path
	var eventType string
	switch {
	case path == "/api/events/user":
		eventType = "user"
	case path == "/api/events/payment":
		eventType = "payment"
	case path == "/api/events/movie":
		eventType = "movie"
	default:
		http.Error(w, "Unknown event type", http.StatusBadRequest)
		return
	}

	// Читаем тело запроса
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Создаем событие
	event := Event{
		Type:      eventType,
		Data:      body,
		Timestamp: time.Now(),
	}

	// Определяем топик на основе типа события
	topic := eventType + "_events"

	// Отправляем событие в Kafka
	if err := sendEvent(producer, topic, event); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"status": true})
}

func main() {
	// Загружаем конфигурацию
	config := loadConfig()

	const maxRetries int = 12
	const retryInterval = 5 * time.Second

	var producer sarama.SyncProducer
	var err error
	for i := 0; i < maxRetries; i++ {
		producer, err = createProducer(config.KafkaBroker)
		if err == nil {
			break
		}
		log.Printf("[Retry %d/%d] Failed to create producer: %v", i+1, maxRetries, err)
		time.Sleep(retryInterval)
	}
	if err != nil {
		log.Fatalf("Failed to create producer after retries: %v", err)
	}
	defer producer.Close()

	var consumer sarama.Consumer
	for i := 0; i < maxRetries; i++ {
		consumer, err = createConsumer(config.KafkaBroker)
		if err == nil {
			break
		}
		log.Printf("[Retry %d/%d] Failed to create consumer: %v", i+1, maxRetries, err)
		time.Sleep(retryInterval)
	}
	if err != nil {
		log.Fatalf("Failed to create consumer after retries: %v", err)
	}
	defer consumer.Close()

	// Запускаем обработчики событий
	var wg sync.WaitGroup
	topics := []string{"user_events", "payment_events", "movie_events"}
	for _, topic := range topics {
		wg.Add(1)
		go consumeEvents(consumer, topic, &wg)
	}

	// Настраиваем HTTP сервер
	http.HandleFunc("/api/events/health", handleHealth)
	http.HandleFunc("/api/events/user", func(w http.ResponseWriter, r *http.Request) {
		handleCreateEvent(w, r, producer)
	})
	http.HandleFunc("/api/events/payment", func(w http.ResponseWriter, r *http.Request) {
		handleCreateEvent(w, r, producer)
	})
	http.HandleFunc("/api/events/movie", func(w http.ResponseWriter, r *http.Request) {
		handleCreateEvent(w, r, producer)
	})

	// Запускаем HTTP сервер
	server := &http.Server{
		Addr: ":" + config.Port,
	}

	// Graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		log.Println("Shutting down server...")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			log.Fatalf("Server forced to shutdown: %v", err)
		}
	}()

	log.Printf("Starting events service on port %s", config.Port)
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("Server error: %v", err)
	}

	wg.Wait()
}

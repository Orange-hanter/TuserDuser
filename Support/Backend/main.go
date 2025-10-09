package main

import (
	"bufio"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// Структура фидбэка
type Feedback struct {
	Step    int    `json:"step"`
	Email   string `json:"email"`
	Message string `json:"text"`
	Time    string `json:"time"`
}

type Event struct {
	Type      string                 `json:"type"`
	Start     string                 `json:"start"`
	End       string                 `json:"end"`
	Duration  string                 `json:"duration"`
	Place     string                 `json:"place"`
	PriceType string                 `json:"priceType"`
	NeedReg   bool                   `json:"needReg"`
	Details   map[string]interface{} `json:"details"`
	Time      string                 `json:"time"`
}

// Глобальная очередь (канал)
var feedbackQueue = make(chan Feedback, 1000) // буфер на 1000 сообщений
var eventQueue = make(chan Event, 1000)       // буфер на 1000 сообщений

// Фоновый воркер для записи в файл
func startWorker[T any](filePath string, queue <-chan T, wg *sync.WaitGroup) {
	defer wg.Done()
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Fatal("Cannot open log file:", err)
	}
	defer file.Close()

	// Use a buffered writer and flush+sync periodically to avoid blocking on every write.
	writer := bufio.NewWriterSize(file, 64*1024) // 64KB buffer
	ticker := time.NewTicker(1 * time.Second)
	defer func() {
		ticker.Stop()
		writer.Flush()
		file.Sync()
	}()

	for {
		select {
		case item, ok := <-queue:
			if !ok {
				return
			}
			data, _ := json.Marshal(item)
			writer.Write(append(data, '\n'))
		case <-ticker.C:
			// Flush buffered data to OS and sync to disk periodically.
			writer.Flush()
			file.Sync()
		}
	}
}

func submitHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST allowed", http.StatusMethodNotAllowed)
		return
	}

	var fb Feedback
	if err := json.NewDecoder(r.Body).Decode(&fb); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	fb.Time = time.Now().Format(time.RFC3339)

	// Пытаемся отправить в очередь **неблокирующим способом**
	select {
	case feedbackQueue <- fb:
		// Успех — сообщение в очереди
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte(`{"status":"queued"}`))
	case <-time.After(100 * time.Millisecond):
		// Очередь переполнена — отказать
		http.Error(w, "Service busy, try again later", http.StatusTooManyRequests)
	}
}

func createEventHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST allowed", http.StatusMethodNotAllowed)
		return
	}

	var ev Event
	if err := json.NewDecoder(r.Body).Decode(&ev); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	ev.Time = time.Now().Format(time.RFC3339)

	// Валидация полей
	if ev.Type == "" || ev.Start == "" || (ev.End == "" && ev.Duration == "") || ev.Place == "" {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	select {
	case eventQueue <- ev:
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte(`{"status":"event queued"}`))
	case <-time.After(100 * time.Millisecond):
		http.Error(w, "Service busy, try again later", http.StatusTooManyRequests)
	}
}

func main() {
	// Start workers with a WaitGroup so they can be gracefully shut down.
	var wg sync.WaitGroup
	wg.Add(2)
	go startWorker("feedbacks.jsonl", feedbackQueue, &wg)
	go startWorker("events.jsonl", eventQueue, &wg)

	// HTTP server
	mux := http.NewServeMux()
	mux.HandleFunc("/feedback", submitHandler)
	mux.HandleFunc("/create", createEventHandler)

	// Serve the `Support` directory under /static/
	fs := http.FileServer(http.Dir("Support"))
	mux.Handle("/static/", http.StripPrefix("/static/", fs))

	// Serve the demo HTML at the root. The file in the repo is named `rpoto4.html`.
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.ServeFile(w, r, "Support/proto4.html")
			return
		}
		// Fall back to the static handler for other paths
		fs.ServeHTTP(w, r)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	addr := ":" + port

	srv := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	// Start server in a goroutine
	go func() {
		log.Println("Server starting on", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("ListenAndServe(): %v", err)
		}
	}()

	// Setup signal handling for graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Wait for a termination signal
	<-stop
	log.Println("Shutting down server...")

	// Create a context with timeout for shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Shutdown HTTP server
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Server Shutdown Failed:%+v", err)
	}

	// Close queues so workers can finish
	close(feedbackQueue)
	close(eventQueue)

	// Wait for workers to finish flushing
	wg.Wait()

	log.Println("Server exited properly")
}

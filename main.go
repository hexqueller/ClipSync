package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/atotto/clipboard"
)

var (
	clipboardContent string       // Текущее содержимое буфера обмена
	mu               sync.RWMutex // Мьютекс для безопасного доступа
	servers          []string     // Список серверов для синхронизации
	port             string       // Порт, на котором работает сервер
	lastClipboard    string       // Последнее известное содержимое буфера обмена
)

// Config структура для конфигурационного файла
type Config struct {
	Servers []string `json:"servers"`
	Port    string   `json:"port"`
}

// Загрузка конфигурации из файла
func loadConfig(filename string) (Config, error) {
	var config Config
	file, err := os.Open(filename)
	if err != nil {
		return config, err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	err = decoder.Decode(&config)
	return config, err
}

// Обработчик для получения текущего содержимого буфера обмена
func getClipboardHandler(w http.ResponseWriter, r *http.Request) {
	mu.RLock()
	defer mu.RUnlock()

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(clipboardContent))
}

// Обработчик для обновления содержимого буфера обмена
func updateClipboardHandler(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	newContent := string(body)
	clipboardContent = newContent
	clipboard.WriteAll(newContent) // Обновляем локальный буфер обмена

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Clipboard updated"))
}

// Функция для синхронизации с другими серверами
func syncClipboard() {
	for {
		mu.RLock()
		currentContent := clipboardContent
		mu.RUnlock()

		// Если содержимое буфера обмена изменилось, отправляем его на другие серверы
		if currentContent != lastClipboard {
			for _, server := range servers {
				url := fmt.Sprintf("%s/clipboard", server)
				resp, err := http.Post(url, "text/plain", ioutil.NopCloser([]byte(currentContent)))
				if err != nil {
					log.Printf("Failed to sync with %s: %v", server, err)
					continue
				}
				resp.Body.Close()
			}
			lastClipboard = currentContent
		}

		// Проверяем изменения на других серверах
		for _, server := range servers {
			url := fmt.Sprintf("%s/clipboard", server)
			resp, err := http.Get(url)
			if err != nil {
				log.Printf("Failed to fetch clipboard from %s: %v", server, err)
				continue
			}
			defer resp.Body.Close()

			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				log.Printf("Failed to read response from %s: %v", server, err)
				continue
			}

			remoteContent := string(body)
			if remoteContent != currentContent {
				mu.Lock()
				clipboardContent = remoteContent
				clipboard.WriteAll(remoteContent)
				mu.Unlock()
				lastClipboard = remoteContent
			}
		}

		time.Sleep(5 * time.Second) // Проверяем каждые 5 секунд
	}
}

func main() {
	// Загружаем конфигурацию
	config, err := loadConfig("config.json")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	servers = config.Servers
	port = config.Port

	// Инициализируем начальное содержимое буфера обмена
	initialContent, err := clipboard.ReadAll()
	if err != nil {
		log.Fatalf("Failed to read initial clipboard content: %v", err)
	}
	clipboardContent = initialContent
	lastClipboard = initialContent

	// Запускаем синхронизацию в отдельной горутине
	go syncClipboard()

	// Настраиваем HTTP-сервер
	http.HandleFunc("/clipboard", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			getClipboardHandler(w, r)
		case http.MethodPost:
			updateClipboardHandler(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	log.Printf("Clipboard server is running on %s...", port)
	log.Fatal(http.ListenAndServe(port, nil))
}

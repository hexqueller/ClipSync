package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sync"

	"github.com/atotto/clipboard"
)

var (
	clipboardContent string
	mu               sync.RWMutex
)

func getClipboardHandler(w http.ResponseWriter, r *http.Request) {
	mu.RLock()
	defer mu.RUnlock()

	content, err := clipboard.ReadAll()
	if err != nil {
		http.Error(w, "Failed to read clipboard", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(content))
}

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
	err = clipboard.WriteAll(newContent)
	if err != nil {
		http.Error(w, "Failed to write to clipboard", http.StatusInternalServerError)
		return
	}

	clipboardContent = newContent
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Clipboard updated"))
}

func main() {
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

	fmt.Println("Clipboard server is running on :8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

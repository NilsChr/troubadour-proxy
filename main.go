package main

import (
    "os"
    "fmt"
    "bytes"
    "io"
    "log"
    "net/http"
    "strconv"
    "strings"
    "sync"
)

var (
    cache      = make(map[string]int64)
    cacheMutex = sync.Mutex{}
)

func getPort() string {
    // Set a default port
    defaultPort := "8080"

    // Get the port from the environment variables
    port := os.Getenv("PORT")
    if port == "" {
        port = defaultPort
        fmt.Printf("No PORT environment variable detected, defaulting to %s\n", defaultPort)
    }
    return port
}

func main() {
    http.HandleFunc("/audio", audioHandler)
    port := getPort()
    fmt.Printf("Starting server on :%s\n", port)
    log.Fatal(http.ListenAndServe(":" + port, nil))
}

func audioHandler(w http.ResponseWriter, r *http.Request) {
    audioURL := r.URL.Query().Get("url")
    if audioURL == "" {
        log.Println("Missing 'url' query parameter")
        http.Error(w, "Missing 'url' query parameter", http.StatusBadRequest)
        return
    }

    var length int64
    var buf bytes.Buffer

    // Check cache
    cacheMutex.Lock()
    cachedLength, found := cache[audioURL]
    cacheMutex.Unlock()

    if found {
        log.Println("Cache hit for URL:", audioURL)
        length = cachedLength

        // Fetch the audio content without recalculating the length
        resp, err := http.Get(audioURL)
        if err != nil {
            log.Println("Failed to fetch the audio file:", err)
            http.Error(w, "Failed to fetch the audio file", http.StatusInternalServerError)
            return
        }
        defer resp.Body.Close()

        if resp.StatusCode != http.StatusOK {
            log.Println("Failed to fetch the audio file, status code:", resp.StatusCode)
            http.Error(w, "Failed to fetch the audio file", resp.StatusCode)
            return
        }

        if _, err := io.Copy(&buf, resp.Body); err != nil {
            log.Println("Error copying response body:", err)
            http.Error(w, "Failed to read the audio file", http.StatusInternalServerError)
            return
        }
    } else {
        log.Println("Cache miss for URL:", audioURL)
        resp, err := http.Get(audioURL)
        if err != nil {
            log.Println("Failed to fetch the audio file:", err)
            http.Error(w, "Failed to fetch the audio file", http.StatusInternalServerError)
            return
        }
        defer resp.Body.Close()

        if resp.StatusCode != http.StatusOK {
            log.Println("Failed to fetch the audio file, status code:", resp.StatusCode)
            http.Error(w, "Failed to fetch the audio file", resp.StatusCode)
            return
        }

        length, err = io.Copy(&buf, resp.Body)
        if err != nil {
            log.Println("Failed to read the audio file:", err)
            http.Error(w, "Failed to read the audio file", http.StatusInternalServerError)
            return
        }

        // Store length in cache
        cacheMutex.Lock()
        cache[audioURL] = length
        cacheMutex.Unlock()
    }

    log.Println("Serving audio file with Content-Length:", length)

    // Set headers to force download and range handling
    w.Header().Set("Content-Disposition", "attachment; filename="+getFileName(audioURL))
    w.Header().Set("Content-Type", "audio/mpeg")
    w.Header().Set("Accept-Ranges", "bytes")
    w.Header().Set("Content-Length", strconv.FormatInt(length, 10))
    w.Header().Set("Content-Range", "bytes 0-"+strconv.FormatInt(length-1, 10)+"/"+strconv.FormatInt(length, 10))

    // Copy the buffer to the response writer
    w.WriteHeader(http.StatusOK)
    if _, err := io.Copy(w, &buf); err != nil {
        log.Println("Error copying response body:", err)
    }
}

func getFileName(url string) string {
    parts := strings.Split(url, "/")
    return parts[len(parts)-1]
}


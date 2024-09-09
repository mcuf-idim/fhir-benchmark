package main

import (
    "bufio"
    "bytes"
    "flag"
    "fmt"
    "io"
    "net/http"
    "os"
    "path/filepath"
    "strings"
)

func isClientError(statusCode int) bool {
    return statusCode >= 400 && statusCode < 500
}

func uploadResource(serverURL, resourceType string, resource []byte) (int, error) {
    url := fmt.Sprintf("%s/%s", serverURL, resourceType)
    req, err := http.NewRequest("POST", url, bytes.NewBuffer(resource))
    if err != nil {
        return 0, err
    }
    req.Header.Set("Content-Type", "application/fhir+json")

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        return 0, err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(resp.Body)
        return resp.StatusCode, fmt.Errorf("upload failed: %s, response: %s", resp.Status, string(body))
    }

    return resp.StatusCode, nil
}

func processFile(serverURL, filePath string, errorCounts map[int]int) (int, int, error) {
    baseName := filepath.Base(filePath)
    resourceType := strings.TrimSuffix(baseName, filepath.Ext(baseName))

    file, err := os.Open(filePath)
    if err != nil {
        return 0, 0, err
    }
    defer file.Close()

    scanner := bufio.NewScanner(file)
    rejectedCount := 0
    resourceCount := 0

    for scanner.Scan() {
        resource := scanner.Bytes()
        resourceCount++
        statusCode, err := uploadResource(serverURL, resourceType, resource)
        if err != nil {
            if isClientError(statusCode) {
                errorCounts[statusCode]++
                rejectedCount++
            } else {
                return 0, 0, fmt.Errorf("unexpected error uploading resource from %s: %v", filePath, err)
            }
        }
    }

    if err := scanner.Err(); err != nil {
        return 0, 0, err
    }

    return rejectedCount, resourceCount, nil
}

func main() {
    serverURL := flag.String("server_url", "", "Base URL of the FHIR server")
    filesFlag := flag.String("files", "", "Comma-separated list of NDJSON files to process")
    flag.Parse()

    if *serverURL == "" {
        fmt.Println("No server URL specified. Use -server_url=[BASE URL OF THE SERVER]")
        return
    }

    if *filesFlag == "" {
        fmt.Println("No files specified. Use -files=invalid-data/Condition.ndjson,invalid-data/Encounter.ndjson")
        return
    }

    files := strings.Split(*filesFlag, ",")
    totalResources := 0
    totalRejected := 0
    errorCounts := make(map[int]int)

    for _, filePath := range files {
        filePath = strings.TrimSpace(filePath)
        if filePath == "" {
            continue
        }
        rejectedCount, resourceCount, err := processFile(*serverURL, filePath, errorCounts)
        if err != nil {
            fmt.Printf("Error processing file %s: %v\n", filePath, err)
            return
        }
        totalRejected += rejectedCount
        totalResources += resourceCount
    }

    fmt.Printf("The server rejected %d out of %d resources as expected.\n", totalRejected, totalResources)
    for code, count := range errorCounts {
        fmt.Printf("Error code %d: %d times\n", code, count)
    }
}

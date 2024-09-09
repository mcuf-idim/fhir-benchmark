package main

import (
    "encoding/json"
    "flag"
    "fmt"
    "io/ioutil"
    "net/http"
    "path/filepath"
    "strings"
    "sync"
    "time"
)

type Resource struct {
    ResourceType string
    ID           string `json:"id"`
}

func main() {
    bearerToken := flag.String("bearer_token", "", "Bearer token for authorization")
    serverURL := flag.String("server_url", "", "Base URL of the FHIR server")
    filesList := flag.String("files", "", "Comma-separated list of NDJSON files to upload")
    threads := flag.Int("threads", 2, "Number of concurrent threads within a single file")
    flag.Parse()

    if *serverURL == "" || *filesList == "" {
        fmt.Println("Server URL and files list are required.")
        return
    }

    files := strings.Split(*filesList, ",")
    totalResources := 0
    totalErrors := 0
    idsMap := make(map[string][]string)

    for _, file := range files {
        trimmedFile := strings.TrimSpace(file)
        data, err := ioutil.ReadFile(trimmedFile)
        if err != nil {
            fmt.Printf("Failed to read file %s: %v\n", trimmedFile, err)
            continue
        }

        lines := strings.Split(string(data), "\n")
        ids := make([]string, 0, len(lines))
        for _, line := range lines {
            if line == "" {
                continue
            }
            var resource Resource
            if err := json.Unmarshal([]byte(line), &resource); err != nil {
                fmt.Println("Error parsing JSON:", err)
                continue
            }
            ids = append(ids, resource.ID)
        }
        resourceType := filepath.Base(trimmedFile)
        resourceType = strings.Split(resourceType, ".")[0]
        idsMap[resourceType] = ids
    }

    start := time.Now()

    for _, file := range files {
        trimmedFile := strings.TrimSpace(file)
        resourceType := filepath.Base(trimmedFile)
        resourceType = strings.Split(resourceType, ".")[0]

        data, err := ioutil.ReadFile(trimmedFile)
        if err != nil {
            fmt.Printf("Failed to read file %s: %v\n", trimmedFile, err)
            continue
        }

        lines := strings.Split(string(data), "\n")
        ids := idsMap[resourceType]

        count, errors := uploadData(*bearerToken, *serverURL, resourceType, lines, ids, *threads)
        totalResources += count
        totalErrors += errors
    }

    fmt.Printf("Completed in %v\n", time.Since(start))
    fmt.Printf("Successfully uploaded %d resources with %d errors\n", totalResources, totalErrors)
}

func uploadData(bearerToken, serverURL, resourceType string, data, ids []string, threads int) (int, int) {
    var wg sync.WaitGroup
    semaphore := make(chan struct{}, threads)
    client := &http.Client{
        Timeout: time.Second * 30,
    }
    count := 0
    errors := 0

    for i, jsonData := range data {
        if jsonData == "" {
            continue
        }

        wg.Add(1)
        go func(jsonStr, id string) {
            defer wg.Done()
            semaphore <- struct{}{}

            // Construct the resource URL with the ID for the PUT request
            resourceURL := fmt.Sprintf("%s/%s/%s", serverURL, resourceType, id)

            req, err := http.NewRequest("PUT", resourceURL, strings.NewReader(jsonStr))
            if err != nil {
                fmt.Println("Error creating PUT request:", err)
                errors++
                <-semaphore
                return
            }

            req.Header.Set("Content-Type", "application/fhir+json")
            if bearerToken != "" {
                req.Header.Set("Authorization", "Bearer " + bearerToken)
            }

            response, err := client.Do(req)
            if err != nil {
                fmt.Println("Error sending PUT request:", err)
                errors++
                <-semaphore
                return
            }
            defer response.Body.Close()

            if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusCreated {
                fmt.Printf("Received non-successful status: %s\n", response.Status)
                errors++
            } else {
                count++
            }
            <-semaphore
        }(jsonData, ids[i])
    }

    wg.Wait()
    return count, errors
}

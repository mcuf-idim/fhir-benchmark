package main

import (
    "bufio"
    "flag"
    "fmt"
    "io/ioutil"
    "log"
    "math/rand"
    "net/http"
    "os"
    "strings"
    "sync"
    "time"

    "gopkg.in/yaml.v3"
)

type Query struct {
    Name       string `yaml:"name"`
    QueryCode  string `yaml:"query_code"`
    IDPoolFile string `yaml:"id_pool_file"`
}

type Task struct {
    Query Query
    ID    string
}

type Stats struct {
    sync.Mutex
    TotalRequests int
    Errors        int
    StartTime     time.Time
}

var (
    threads     int
    serverURL   string
    bearerToken string
)

func main() {
    var configFile string
    flag.StringVar(&configFile, "config", "queries.yaml", "Path to the configuration file")
    flag.IntVar(&threads, "threads", 10, "Number of concurrent threads")
    flag.StringVar(&serverURL, "server_url", "http://localhost:8080/fhir", "Base URL of the FHIR server")
    flag.StringVar(&bearerToken, "bearer_token", "", "Bearer token for authorization")
    flag.Parse()

    if err := os.MkdirAll("./query-output", 0755); err != nil {
        log.Fatalf("Failed to create output directory: %v", err)
    }

    stats := Stats{
        StartTime: time.Now(),
    }

    var queries []Query
    if err := readYAML(configFile, &queries); err != nil {
        log.Fatalf("Error reading YAML configuration: %v", err)
    }

    var tasks []Task
    for _, query := range queries {
        ids, err := loadIDs(query.IDPoolFile)
        if err != nil {
            log.Printf("Error loading IDs from file %s: %v", query.IDPoolFile, err)
            continue
        }
        for _, id := range ids {
            tasks = append(tasks, Task{Query: query, ID: id})
        }
    }

    shuffleTasks(tasks)

    jobs := make(chan Task, len(tasks))
    var wg sync.WaitGroup
    wg.Add(threads)

    client := &http.Client{
        Timeout: time.Second * 30,
        Transport: &http.Transport{
            MaxIdleConns:        256,
            MaxIdleConnsPerHost: 256,
            MaxConnsPerHost:     256,
        },
    }

    for i := 0; i < threads; i++ {
        go worker(jobs, &wg, &stats, i, client)
    }

    for _, task := range tasks {
        jobs <- task
    }
    close(jobs)

    wg.Wait()

    printFinalReport(&stats)
}

func shuffleTasks(tasks []Task) {
    rand.Seed(time.Now().UnixNano())
    rand.Shuffle(len(tasks), func(i, j int) {
        tasks[i], tasks[j] = tasks[j], tasks[i]
    })
}

func worker(jobs <-chan Task, wg *sync.WaitGroup, stats *Stats, workerID int, client *http.Client) {
    defer wg.Done()

    fileName := fmt.Sprintf("./query-output/worker_%d_output.ndjson", workerID)
    file, err := os.Create(fileName)
    if err != nil {
        log.Fatalf("Failed to create file for worker %d: %v", workerID, err)
    }
    writer := bufio.NewWriter(file)
    defer func() {
        writer.Flush()
        file.Close()
    }()

    for task := range jobs {
        requestURL := fmt.Sprintf("%s%s", serverURL, replacePlaceholder(task.Query.QueryCode, task.ID))

        req, err := http.NewRequest("GET", requestURL, nil)
        if err != nil {
            log.Printf("Error creating request to %s: %v", requestURL, err)
            stats.Lock()
            stats.Errors++
            stats.Unlock()
            continue
        }

        if bearerToken != "" {
            req.Header.Set("Authorization", "Bearer "+bearerToken)
        }

        resp, err := client.Do(req)
        if err != nil {
            log.Printf("Error sending request to %s: %v", requestURL, err)
            stats.Lock()
            stats.Errors++
            stats.Unlock()
            continue
        }

        body, err := ioutil.ReadAll(resp.Body)
        resp.Body.Close()
        if err != nil {
            log.Printf("Error reading response body: %v", err)
            stats.Lock()
            stats.Errors++
            stats.Unlock()
            continue
        }

        _, err = writer.Write(body)
        _, err = writer.WriteString("\n")
        if err != nil {
            log.Printf("Error writing to file %s: %v", fileName, err)
        }

        stats.Lock()
        stats.TotalRequests++
        stats.Unlock()
    }
}

func replacePlaceholder(queryCode, id string) string {
    return strings.ReplaceAll(queryCode, "{id}", id)
}

func readYAML(path string, out interface{}) error {
    bytes, err := ioutil.ReadFile(path)
    if err != nil {
        return err
    }
    return yaml.Unmarshal(bytes, out)
}

func loadIDs(filename string) ([]string, error) {
    file, err := os.Open(filename)
    if err != nil {
        return nil, err
    }
    defer file.Close()

    var ids []string
    scanner := bufio.NewScanner(file)
    for scanner.Scan() {
        ids = append(ids, scanner.Text())
    }
    if err := scanner.Err(); err != nil {
        return nil, err
    }
    return ids, nil
}

func printFinalReport(stats *Stats) {
    duration := time.Since(stats.StartTime)
    fmt.Printf("Total time taken: %v, total requests sent: %d, total errors: %d\n", duration, stats.TotalRequests, stats.Errors)
}

package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"testing"
	"time"

	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	tcredis "github.com/testcontainers/testcontainers-go/modules/redis"
)

var (
	baseURL    string
	envDBConn  string
	envRedis   string
	httpClient = http.DefaultClient
)

func TestMain(m *testing.M) {
	flag.Parse()
	os.Exit(runTests(m))
}

func runTests(m *testing.M) int {
	if testing.Short() {
		return m.Run()
	}

	ctx := context.Background()

	container, err := tcpostgres.Run(ctx, "postgres:17-alpine",
		tcpostgres.WithDatabase("kitchen_test"),
		tcpostgres.WithUsername("kitchen"),
		tcpostgres.WithPassword("kitchen"),
		tcpostgres.BasicWaitStrategies(),
	)
	if err != nil {
		fmt.Fprintln(os.Stderr, "start postgres container:", err)
		return 1
	}
	defer func() { _ = container.Terminate(ctx) }()

	dbConnString, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		fmt.Fprintln(os.Stderr, "connection string:", err)
		return 1
	}

	redisContainer, err := tcredis.Run(ctx, "redis:7-alpine")
	if err != nil {
		fmt.Fprintln(os.Stderr, "start redis container:", err)
		return 1
	}
	defer func() { _ = redisContainer.Terminate(ctx) }()

	redisAddr, err := redisContainer.PortEndpoint(ctx, "6379/tcp", "")
	if err != nil {
		fmt.Fprintln(os.Stderr, "redis endpoint:", err)
		return 1
	}

	url, stop, err := startKitchenServer(dbConnString, redisAddr)
	if err != nil {
		fmt.Fprintln(os.Stderr, "start kitchen server:", err)
		return 1
	}
	defer stop()

	baseURL = url
	envDBConn = dbConnString
	envRedis = redisAddr

	return m.Run()
}

func startKitchenServer(dbConnString, redisAddr string) (string, func(), error) {
	port, err := freePort()
	if err != nil {
		return "", nil, err
	}
	return startKitchenServerOnPort(dbConnString, redisAddr, port)
}

func startKitchenServerOnPort(dbConnString, redisAddr string, port int) (string, func(), error) {
	repoRoot, err := filepath.Abs("..")
	if err != nil {
		return "", nil, fmt.Errorf("resolve repo root: %w", err)
	}

	binDir, err := os.MkdirTemp("", "kitchen-test-bin")
	if err != nil {
		return "", nil, fmt.Errorf("create temp dir: %w", err)
	}
	cleanupBin := func() { _ = os.RemoveAll(binDir) }

	binPath := filepath.Join(binDir, "kitchen")
	build := exec.Command("go", "build", "-o", binPath, "./services/kitchen/cmd/app")
	build.Dir = repoRoot
	if out, err := build.CombinedOutput(); err != nil {
		cleanupBin()
		return "", nil, fmt.Errorf("build kitchen: %w\n%s", err, out)
	}

	addr := fmt.Sprintf("127.0.0.1:%d", port)

	var output syncBuffer
	cmd := exec.Command(binPath)
	cmd.Dir = repoRoot
	cmd.Env = append(os.Environ(),
		"DB_CONN_STRING="+dbConnString,
		"REDIS_ADDR="+redisAddr,
		"LISTEN_ADDRESS=127.0.0.1",
		fmt.Sprintf("LISTEN_PORT=%d", port),
	)
	cmd.Stdout = &output
	cmd.Stderr = &output
	if err := cmd.Start(); err != nil {
		cleanupBin()
		return "", nil, fmt.Errorf("start kitchen: %w", err)
	}

	stop := func() {
		_ = cmd.Process.Signal(syscall.SIGTERM)
		_ = cmd.Wait()
		cleanupBin()
	}

	url := "http://" + addr
	if err := waitForHTTP(url+"/readyz", 30*time.Second); err != nil {
		stop()
		return "", nil, fmt.Errorf("kitchen did not become ready: %w\noutput:\n%s", err, output.String())
	}

	return url, stop, nil
}

func freePort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

func waitForHTTP(url string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		resp, err := http.Get(url)
		if err == nil {
			resp.Body.Close()
			return nil
		}
		lastErr = err
		time.Sleep(200 * time.Millisecond)
	}
	return lastErr
}

type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func doJSON(t *testing.T, client *http.Client, method, url, apiKey string, body any, wantStatus int, out any) {
	t.Helper()

	headers := map[string]string{}
	if apiKey != "" {
		headers["X-API-Key"] = apiKey
	}
	doRequest(t, method, url, headers, body, wantStatus, out)
}

func doRequest(t *testing.T, method, url string, headers map[string]string, body any, wantStatus int, out any) {
	t.Helper()

	var bodyReader *bytes.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal request body: %v", err)
		}
		bodyReader = bytes.NewReader(b)
	} else {
		bodyReader = bytes.NewReader(nil)
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != wantStatus {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("%s %s: expected status %d, got %d, body: %s", method, url, wantStatus, resp.StatusCode, respBody)
	}

	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			t.Fatalf("%s %s: decode response: %v", method, url, err)
		}
	}
}

type placeRegistration struct {
	PlaceID string `json:"place_id"`
	APIKey  string `json:"api_key"`
}

func registerTestPlace(t *testing.T, overrides map[string]any) placeRegistration {
	t.Helper()

	body := map[string]any{
		"name":        "Test Place",
		"city":        "Moscow",
		"address":     "Lenina 1",
		"timezone":    "Europe/Moscow",
		"opens_at":    "09:00:00",
		"closes_at":   "22:00:00",
		"type":        "restaurant",
		"webhook_url": "https://example.com/webhook",
	}
	for k, v := range overrides {
		body[k] = v
	}

	var resp placeRegistration
	doJSON(t, httpClient, http.MethodPost, baseURL+"/api/v1/b2b/places", "", body, http.StatusCreated, &resp)
	return resp
}

type menuItemResp struct {
	ExternalID string `json:"external_id"`
}

func addTestMenuItems(t *testing.T, placeID, apiKey string, items []map[string]any) []menuItemResp {
	t.Helper()

	var resp []menuItemResp
	doJSON(t, httpClient, http.MethodPost, baseURL+"/api/v1/b2b/places/"+placeID+"/menu", apiKey, items, http.StatusCreated, &resp)
	return resp
}

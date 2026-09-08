package tests

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

const testapiPlaceName = "Тестовая Пиццерия"

var (
	testapiBinOnce sync.Once
	testapiBinPath string
	testapiBinErr  error
)

func buildTestapi(t *testing.T) string {
	t.Helper()

	testapiBinOnce.Do(func() {
		repoRoot, err := filepath.Abs("..")
		if err != nil {
			testapiBinErr = err
			return
		}

		binDir, err := os.MkdirTemp("", "testapi-bin")
		if err != nil {
			testapiBinErr = err
			return
		}

		testapiBinPath = filepath.Join(binDir, "testapi")
		build := exec.Command("go", "build", "-o", testapiBinPath, "./services/testapi/cmd/app")
		build.Dir = repoRoot
		if out, err := build.CombinedOutput(); err != nil {
			testapiBinErr = fmt.Errorf("build testapi: %w\n%s", err, out)
		}
	})

	if testapiBinErr != nil {
		t.Fatalf("build testapi: %v", testapiBinErr)
	}
	return testapiBinPath
}

func startTestapi(t *testing.T, env map[string]string) {
	t.Helper()

	binPath := buildTestapi(t)

	port, err := freePort()
	if err != nil {
		t.Fatalf("find free port: %v", err)
	}

	var output syncBuffer
	cmd := exec.Command(binPath)
	cmd.Env = append(os.Environ(),
		"KITCHEN_BASE_URL="+baseURL,
		fmt.Sprintf("PUBLIC_URL=http://127.0.0.1:%d", port),
		"LISTEN_ADDRESS=127.0.0.1",
		fmt.Sprintf("LISTEN_PORT=%d", port),
	)
	for key, value := range env {
		cmd.Env = append(cmd.Env, key+"="+value)
	}
	cmd.Stdout = &output
	cmd.Stderr = &output

	if err := cmd.Start(); err != nil {
		t.Fatalf("start testapi: %v", err)
	}
	t.Cleanup(func() {
		_ = cmd.Process.Signal(syscall.SIGTERM)
		_ = cmd.Wait()
	})

	if err := waitForHTTP(fmt.Sprintf("http://127.0.0.1:%d/healthz", port), 30*time.Second); err != nil {
		t.Fatalf("testapi did not become ready: %v\noutput:\n%s", err, output.String())
	}
}

type placeListItem struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func waitForTestapiPlace(t *testing.T) placeListItem {
	t.Helper()

	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		var places []placeListItem
		doJSON(t, httpClient, http.MethodGet,
			baseURL+"/api/v1/places?search="+urlQueryEscape(testapiPlaceName), "", nil, http.StatusOK, &places)

		if len(places) > 0 {
			return places[0]
		}
		time.Sleep(100 * time.Millisecond)
	}

	t.Fatal("testapi did not register itself on the platform")
	return placeListItem{}
}

func urlQueryEscape(value string) string {
	return strings.ReplaceAll(value, " ", "%20")
}

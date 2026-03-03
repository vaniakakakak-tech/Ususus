package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type LogEntry struct {
	Time    string `json:"time"`
	Message string `json:"message"`
	Type    string `json:"type"`
}

var (
	logs   []LogEntry
	logsMu sync.Mutex
)

func addLog(msg, t string) {
	logsMu.Lock()
	defer logsMu.Unlock()
	logs = append(logs, LogEntry{
		Time:    time.Now().Format("15:04:05"),
		Message: msg,
		Type:    t,
	})
	if len(logs) > 200 {
		logs = logs[len(logs)-200:]
	}
}

func cors(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")
}

func main() {
	http.HandleFunc("/api/steal", handleSteal)
	http.HandleFunc("/api/logs", handleLogs)
	http.HandleFunc("/api/logs/clear", handleClearLogs)
	http.HandleFunc("/api/dns", handleDNS)
	http.HandleFunc("/api/history", handleHistory)
	http.HandleFunc("/api/auth", handleAuth)
	http.Handle("/", http.FileServer(http.Dir("./web")))

	fmt.Println("PackSteal запущен: http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func handleSteal(w http.ResponseWriter, r *http.Request) {
	cors(w)
	if r.Method == "OPTIONS" {
		return
	}

	var req struct {
		Server string `json:"server"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	if req.Server == "" {
		json.NewEncoder(w).Encode(map[string]string{"error": "Укажи IP"})
		return
	}

	addLog("Подключаюсь к "+req.Server+"...", "info")

	go func() {
		execPath, _ := os.Executable()
		execDir := filepath.Dir(execPath)
		packstealPath := filepath.Join(execDir, "packsteal")

		cmd := exec.Command(packstealPath, "steal", req.Server)
		cmd.Dir = execDir
		out, _ := cmd.CombinedOutput()

		for _, line := range strings.Split(string(out), "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			t := "info"
			if strings.Contains(line, "✓") || strings.Contains(line, "Готово") {
				t = "success"
			} else if strings.Contains(line, "✗") || strings.Contains(line, "Ошибка") {
				t = "error"
			}
			addLog(line, t)
		}
	}()

	json.NewEncoder(w).Encode(map[string]string{"status": "started"})
}

func handleLogs(w http.ResponseWriter, r *http.Request) {
	cors(w)
	logsMu.Lock()
	defer logsMu.Unlock()
	json.NewEncoder(w).Encode(logs)
}

func handleClearLogs(w http.ResponseWriter, r *http.Request) {
	cors(w)
	logsMu.Lock()
	logs = nil
	logsMu.Unlock()
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func handleDNS(w http.ResponseWriter, r *http.Request) {
	cors(w)
	var req struct {
		DNS string `json:"dns"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	prefix := os.Getenv("PREFIX")
	cmd := exec.Command("sh", "-c", fmt.Sprintf("echo 'nameserver %s' > %s/etc/resolv.conf", req.DNS, prefix))
	if err := cmd.Run(); err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	addLog("DNS → "+req.DNS, "success")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func handleHistory(w http.ResponseWriter, r *http.Request) {
	cors(w)
	packsDir := "/storage/emulated/0/packs"
	entries, err := os.ReadDir(packsDir)
	if err != nil {
		json.NewEncoder(w).Encode([]interface{}{})
		return
	}

	type Item struct {
		Server string `json:"server"`
		Packs  int    `json:"packs"`
	}

	var result []Item
	for _, e := range entries {
		if e.IsDir() {
			files, _ := filepath.Glob(filepath.Join(packsDir, e.Name(), "*.zip"))
			result = append(result, Item{Server: e.Name(), Packs: len(files)})
		}
	}
	json.NewEncoder(w).Encode(result)
}

func handleAuth(w http.ResponseWriter, r *http.Request) {
	cors(w)
	execPath, _ := os.Executable()
	tokenPath := filepath.Join(filepath.Dir(execPath), "token.json")
	_, err := os.Stat(tokenPath)
	json.NewEncoder(w).Encode(map[string]bool{"authenticated": err == nil})
}

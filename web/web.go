package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var (
	ngrokProcess   *exec.Cmd
	relayProcess    *exec.Cmd
	relayConn       net.Conn
	deviceID        string
	serverAddr      string
	ngrokURL        string
	ngrokAuthToken  string
	mu              sync.RWMutex
	registeredDevs  = make(map[string]bool)
	fileReceiveChan = make(chan string, 100)
	videoFrameChan  = make(chan string, 100)
	// Short code mapping: 6-digit code -> full ngrok URL
	shortCodeMap    = make(map[string]string)
)

// Response types
type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

type DeviceInfo struct {
	ID        string    `json:"id"`
	Connected bool      `json:"connected"`
	LastSeen  time.Time `json:"last_seen"`
}

type NgrokStatus struct {
	Running bool   `json:"running"`
	URL     string `json:"url,omitempty"`
}

type RelayStatus struct {
	Running  bool   `json:"running"`
	ServerID string `json:"server_id,omitempty"`
}

// StartNgrok starts ngrok for TCP port 9999
func StartNgrok(w http.ResponseWriter, r *http.Request) {
	if ngrokProcess != nil && ngrokProcess.Process != nil {
		writeJSON(w, APIResponse{Success: true, Message: "Ngrok already running", Data: NgrokStatus{Running: true, URL: ngrokURL}})
		return
	}

	cmdArgs := []string{"tcp", "9999"}
	if ngrokAuthToken != "" {
		cmdArgs = append(cmdArgs, "--authtoken", ngrokAuthToken)
	}
	ngrokProcess = exec.Command("./ngrok", cmdArgs...)
	ngrokProcess.Stdout = os.Stdout
	ngrokProcess.Stderr = os.Stderr

	err := ngrokProcess.Start()
	if err != nil {
		writeJSON(w, APIResponse{Success: false, Message: "Failed to start ngrok: " + err.Error()})
		return
	}

	// Wait for ngrok to start and get the URL
	go func() {
		time.Sleep(3 * time.Second)
		getNgrokURL()
	}()

	writeJSON(w, APIResponse{Success: true, Message: "Ngrok started", Data: NgrokStatus{Running: true}})
}

// StopNgrok stops the ngrok process
func StopNgrok(w http.ResponseWriter, r *http.Request) {
	if ngrokProcess == nil || ngrokProcess.Process == nil {
		writeJSON(w, APIResponse{Success: false, Message: "Ngrok not running"})
		return
	}

	err := ngrokProcess.Process.Kill()
	if err != nil {
		writeJSON(w, APIResponse{Success: false, Message: "Failed to stop ngrok: " + err.Error()})
		return
	}

	ngrokProcess = nil
	ngrokURL = ""
	writeJSON(w, APIResponse{Success: true, Message: "Ngrok stopped"})
}

// GetNgrokStatus returns the current ngrok status
func GetNgrokStatus(w http.ResponseWriter, r *http.Request) {
	status := NgrokStatus{
		Running: ngrokProcess != nil && ngrokProcess.Process != nil,
		URL:     strings.TrimPrefix(ngrokURL, "tcp://"),
	}
	writeJSON(w, APIResponse{Success: true, Data: status})
}

// SaveNgrokToken saves the ngrok authtoken
func SaveNgrokToken(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Token string `json:"token"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, APIResponse{Success: false, Message: "Invalid request"})
		return
	}

	ngrokAuthToken = req.Token
	writeJSON(w, APIResponse{Success: true, Message: "Ngrok token saved"})
}

// getNgrokURL fetches the ngrok URL from the API
func getNgrokURL() {
	client := &http.Client{}
	for i := 0; i < 10; i++ {
		resp, err := client.Get("http://localhost:4040/api/tunnels")
		if err == nil {
			defer resp.Body.Close()
			body, _ := io.ReadAll(resp.Body)
			var result map[string]interface{}
			json.Unmarshal(body, &result)
			if tunnels, ok := result["tunnels"].([]interface{}); ok && len(tunnels) > 0 {
				for _, t := range tunnels {
					tunnel := t.(map[string]interface{})
					if proto, ok := tunnel["proto"].(string); ok && proto == "tcp" {
						if url, ok := tunnel["public_url"].(string); ok {
							ngrokURL = url
							// Generate short code for the URL
							generateShortCode(url)
							fmt.Println("Ngrok URL:", ngrokURL)
							
							// Tell relay server about our public URL
							if relayConn != nil {
								urlClean := strings.TrimPrefix(url, "tcp://")
								fmt.Fprintf(relayConn, "MAP_URL|%s\n", urlClean)
								fmt.Println("🗺️ Sent URL to relay server")
							}
							return
						}
					}
				}
			}
		}
		time.Sleep(1 * time.Second)
	}
}

// generateShortCode creates a 6-digit code from ngrok URL
func generateShortCode(url string) string {
	mu.Lock()
	defer mu.Unlock()

	// Clean up old shortcode files first
	oldFiles, _ := filepath.Glob(".shortcode_*.txt")
	for _, f := range oldFiles {
		os.Remove(f)
	}
	oldFiles2, _ := filepath.Glob("Copycatv1/.shortcode_*.txt")
	for _, f := range oldFiles2 {
		os.Remove(f)
	}
	
	// Remove tcp:// prefix from URL before storing
	url = strings.TrimPrefix(url, "tcp://")
	
	// Generate random 6-digit code
	code := fmt.Sprintf("%06d", rand.Intn(1000000))
	shortCodeMap[code] = url
	
	// Also save to file for cross-process sharing - save in multiple locations
	codeFile := ".shortcode_" + code + ".txt"
	os.WriteFile(codeFile, []byte(url), 0644)
	
	// Also save in Copycatv1 folder
	os.WriteFile("Copycatv1/"+codeFile, []byte(url), 0644)
	
	fmt.Println("🔑 Short code:", code, "->", url)
	fmt.Println("⚠️ Old codes cleared! Use ONLY this new code.")
	return code
}

// getURLFromCode converts short code back to full URL
func getURLFromCode(code string) string {
	fmt.Println("🔍 getURLFromCode called with:", code)
	
	mu.RLock()
	defer mu.RUnlock()
	
	// Try to find in memory first
	if url, ok := shortCodeMap[code]; ok {
		fmt.Println("✅ Found in memory:", url)
		return url
	}
	
	// Try to load from file (for cross-process sharing)
	// Check multiple possible locations
	possiblePaths := []string{
		".shortcode_" + code + ".txt",
		"./Copycatv1/.shortcode_" + code + ".txt",
		"/data/data/com.termux/files/home/storage/shared/Download/Copycatv1/.shortcode_" + code + ".txt",
	}
	
	for _, codeFile := range possiblePaths {
		fmt.Println("🔍 Checking file:", codeFile)
		if data, err := os.ReadFile(codeFile); err == nil {
			url := strings.TrimSpace(string(data))
			fmt.Println("✅ Found in file:", url)
			return url
		}
	}
	
	// Also scan all .shortcode_*.txt files in current directory
	files, _ := filepath.Glob(".shortcode_*.txt")
	for _, f := range files {
		fmt.Println("🔍 Scanning file:", f)
		// Extract code from filename (e.g., .shortcode_123456.txt -> 123456)
		base := filepath.Base(f)
		fmt.Println("🔍 Base filename:", base)
		if len(base) >= 19 && strings.HasPrefix(base, ".shortcode_") {
			fileCode := base[12:18]
			fmt.Println("🔍 File code:", fileCode, "Requested code:", code)
			if fileCode == code {
				if data, err := os.ReadFile(f); err == nil {
					url := strings.TrimSpace(string(data))
					fmt.Println("✅ Found matching file! URL:", url)
					return url
				}
			}
		}
	}
	
	// Also scan Copycatv1 directory
	files2, _ := filepath.Glob("Copycatv1/.shortcode_*.txt")
	for _, f := range files2 {
		fmt.Println("🔍 Scanning Copycatv1 file:", f)
		base := filepath.Base(f)
		if len(base) >= 19 && strings.HasPrefix(base, ".shortcode_") {
			fileCode := base[12:18]
			fmt.Println("🔍 File code:", fileCode, "Requested code:", code)
			if fileCode == code {
				if data, err := os.ReadFile(f); err == nil {
					url := strings.TrimSpace(string(data))
					fmt.Println("✅ Found matching file! URL:", url)
					return url
				}
			}
		}
	}

	fmt.Println("❌ Code not found in any file!")
	return ""
}

// isAllDigits checks if a string contains only digits
func isAllDigits(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// GetShortCode returns the current short code for the ngrok URL
func GetShortCode(w http.ResponseWriter, r *http.Request) {
	mu.RLock()
	defer mu.RUnlock()

	var code string
	// ngrokURL still has tcp:// prefix, so we need to compare without it
	ngrokURLClean := strings.TrimPrefix(ngrokURL, "tcp://")
	for c, url := range shortCodeMap {
		if url == ngrokURLClean {
			code = c
			break
		}
	}

	writeJSON(w, APIResponse{Success: true, Data: map[string]string{
		"code":    code,
		"url":     ngrokURL,
		"tcp_url": strings.TrimPrefix(ngrokURL, "tcp://"),
	}})
}

// StartRelayServer starts the relay server
func StartRelayServer(w http.ResponseWriter, r *http.Request) {
	if relayProcess != nil && relayProcess.Process != nil {
		writeJSON(w, APIResponse{Success: true, Message: "Relay server already running", Data: RelayStatus{Running: true}})
		return
	}

	relayProcess = exec.Command("go", "run", ".")
	relayProcess.Dir = "app/copycat-v1/modes/internet/server"
	relayProcess.Stdout = os.Stdout
	relayProcess.Stderr = os.Stderr

	err := relayProcess.Start()
	if err != nil {
		writeJSON(w, APIResponse{Success: false, Message: "Failed to start relay server: " + err.Error()})
		return
	}

	// Wait for relay server to start, then register with it if we have ngrok URL
	go func() {
		time.Sleep(2 * time.Second)
		if ngrokURL != "" {
			// Connect to our own relay server and send the ngrok URL
			urlClean := strings.TrimPrefix(ngrokURL, "tcp://")
			if conn, err := net.Dial("tcp", "localhost:9999"); err == nil {
				fmt.Fprintf(conn, "MAP_URL|%s\n", urlClean)
				fmt.Println("🗺️ Registered our URL with relay server:", urlClean)
				conn.Close()
			}
		}
	}()

	writeJSON(w, APIResponse{Success: true, Message: "Relay server started", Data: RelayStatus{Running: true}})
}

// StopRelayServer stops the relay server
func StopRelayServer(w http.ResponseWriter, r *http.Request) {
	if relayProcess == nil || relayProcess.Process == nil {
		writeJSON(w, APIResponse{Success: false, Message: "Relay server not running"})
		return
	}

	err := relayProcess.Process.Kill()
	if err != nil {
		writeJSON(w, APIResponse{Success: false, Message: "Failed to stop relay server: " + err.Error()})
		return
	}

	relayProcess = nil
	writeJSON(w, APIResponse{Success: true, Message: "Relay server stopped"})
}

// GetRelayStatus returns the relay server status
func GetRelayStatus(w http.ResponseWriter, r *http.Request) {
	status := RelayStatus{
		Running:  relayProcess != nil && relayProcess.Process != nil,
		ServerID: deviceID,
	}
	writeJSON(w, APIResponse{Success: true, Data: status})
}

// RegisterDevice registers this device with the relay server
func RegisterDevice(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID         string `json:"id"`
		ServerAddr string `json:"server_addr"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, APIResponse{Success: false, Message: "Invalid request"})
		return
	}

	if req.ID == "" {
		writeJSON(w, APIResponse{Success: false, Message: "Device ID is required"})
		return
	}

	if req.ServerAddr == "" {
		req.ServerAddr = ngrokURL
	}

	// Check if it's a short code (6 digits) and convert to URL
	// Or if it's empty, ask relay server for the URL
	input := strings.TrimSpace(req.ServerAddr)
	fmt.Println("🔍 Registration input:", input, "length:", len(input))
	
	if input == "" {
		// No server address provided - ask relay server for it
		if relayConn != nil {
			relayConn.Write([]byte("GET_URL\n"))
			buf := make([]byte, 4096)
			n, err := relayConn.Read(buf)
			if err == nil {
				response := strings.TrimSpace(string(buf[:n]))
				if response != "" && response != "NO_URL" {
					req.ServerAddr = response
					fmt.Println("🗺️ Got URL from relay server:", req.ServerAddr)
				}
			}
		}
	} else if len(input) == 6 && isAllDigits(input) {
		// It's a short code - still try to convert but also ask server as fallback
		if url := getURLFromCode(input); url != "" {
			req.ServerAddr = url
			fmt.Println("🔑 Converted short code to URL:", req.ServerAddr)
		} else if relayConn != nil {
			// Fallback: ask relay server
			relayConn.Write([]byte("GET_URL\n"))
			buf := make([]byte, 4096)
			n, err := relayConn.Read(buf)
			if err == nil {
				response := strings.TrimSpace(string(buf[:n]))
				if response != "" && response != "NO_URL" {
					req.ServerAddr = response
					fmt.Println("🗺️ Got URL from relay server (fallback):", req.ServerAddr)
				}
			}
		}
	}

	// Strip tcp:// prefix if present
	serverAddr := strings.TrimPrefix(req.ServerAddr, "tcp://")
	deviceID = req.ID
	serverAddr = req.ServerAddr

	// Connect to relay server
	conn, err := net.Dial("tcp", serverAddr)
	if err != nil {
		writeJSON(w, APIResponse{Success: false, Message: "Failed to connect: " + err.Error()})
		return
	}

	relayConn = conn
	serverAddr = serverAddr // Save for reference

	// Register with server
	fmt.Fprintf(conn, "REGISTER|%s\n", deviceID)

	// Start receiving files in background
	go receiveFiles(conn)

	writeJSON(w, APIResponse{Success: true, Message: "Device registered", Data: map[string]string{
		"device_id":  deviceID,
		"server_addr": serverAddr,
	}})
}

// GetDevices gets the list of registered devices
func GetDevices(w http.ResponseWriter, r *http.Request) {
	// First, try to get devices from our own connected clients (if we're a client)
	if relayConn != nil {
		relayConn.Write([]byte("LIST_DEVICES\n"))

		buf := make([]byte, 4096)
		n, _ := relayConn.Read(buf)
		data := string(buf[:n])

		var devices []string
		if data != "" {
			devices = strings.Split(data, "|")
		}

		writeJSON(w, APIResponse{Success: true, Data: devices})
		return
	}

	// If we're the server (relay server running locally), get devices from relay process
	// We'll check if relayProcess is running
	if relayProcess != nil && relayProcess.Process != nil {
		// The relay server runs locally, we can't easily get the device list
		// For now, return the device ID itself as it's the server
		if deviceID != "" {
			writeJSON(w, APIResponse{Success: true, Data: []string{deviceID}})
			return
		}
	}

	writeJSON(w, APIResponse{Success: true, Data: []string{}})
}

// SendFile sends a file to a target device
func SendFile(w http.ResponseWriter, r *http.Request) {
	if relayConn == nil {
		writeJSON(w, APIResponse{Success: false, Message: "Not connected to server"})
		return
	}

	var req struct {
		TargetID string `json:"target_id"`
		FilePath string `json:"file_path"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, APIResponse{Success: false, Message: "Invalid request"})
		return
	}

	// Open file
	file, err := os.Open(req.FilePath)
	if err != nil {
		writeJSON(w, APIResponse{Success: false, Message: "Failed to open file: " + err.Error()})
		return
	}
	defer file.Close()

	filename := filepath.Base(req.FilePath)

	// Send file header
	fmt.Fprintf(relayConn, "SEND|%s|%s\n", req.TargetID, filename)

	// Send file content
	io.Copy(relayConn, file)

	writeJSON(w, APIResponse{Success: true, Message: "File sent: " + filename})
}

// UploadFile handles file upload from web UI
func UploadFile(w http.ResponseWriter, r *http.Request) {
	if relayConn == nil {
		writeJSON(w, APIResponse{Success: false, Message: "Not connected to server"})
		return
	}

	r.ParseMultipartForm(10 << 20) // 10MB max

	targetID := r.FormValue("target_id")
	file, header, err := r.FormFile("file")
	if err != nil {
		writeJSON(w, APIResponse{Success: false, Message: "Failed to get file: " + err.Error()})
		return
	}
	defer file.Close()

	filename := header.Filename

	// Send file header
	fmt.Fprintf(relayConn, "SEND|%s|%s\n", targetID, filename)

	// Send file content
	io.Copy(relayConn, file)

	// Send end-of-file marker
	relayConn.Write([]byte("EOF_MARKER\n"))

	writeJSON(w, APIResponse{Success: true, Message: "File sent: " + filename})
}

// StreamVideo handles video streaming from web UI
func StreamVideo(w http.ResponseWriter, r *http.Request) {
	// Check if we're in server mode (relay server running locally) or client mode (connected to external relay)
	
	// Client mode: connected to external relay server
	if relayConn != nil {
		streamVideoToRelay(w, r)
		return
	}
	
	// Server mode: relay server running locally - check if we can forward video
	if relayProcess != nil && relayProcess.Process != nil {
		// We're the server - need to find the target client connection
		// This requires the relay server to support video forwarding
		// For now, return an error with instructions
		writeJSON(w, APIResponse{Success: false, Message: "Server mode - video streaming requires target to be connected. Make sure the target device is registered and try again."})
		return
	}
	
	writeJSON(w, APIResponse{Success: false, Message: "Not connected. Please start relay server or connect to a relay server first."})
}

// streamVideoToRelay sends video to external relay server
func streamVideoToRelay(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TargetID string `json:"target_id"`
		Frame     string `json:"frame"` // base64 encoded frame
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, APIResponse{Success: false, Message: "Invalid request"})
		return
	}

	// Send video frame to target device via relay
	// Format: VIDEO_B64|targetID|frame_data (all in one line)
	fmt.Fprintf(relayConn, "VIDEO_B64|%s|%s\n", req.TargetID, req.Frame)

	writeJSON(w, APIResponse{Success: true, Message: "Video frame sent"})
}

// receiveFiles handles incoming file transfers
func receiveFiles(conn net.Conn) {
	reader := bufio.NewReader(conn)

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Disconnected from server")
			return
		}

		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "INCOMING|") {
			parts := strings.Split(line, "|")
			filename := parts[1]

			os.MkdirAll("ReceivedFiles", 0755)
			file, _ := os.Create("ReceivedFiles/" + filename)

			// Read file content until EOF_MARKER
			for {
				chunk, err := reader.ReadString('\n')
				if err != nil {
					break
				}
				if strings.TrimSpace(chunk) == "EOF_MARKER" {
					break
				}
				file.Write([]byte(chunk))
			}
			file.Close()

			fmt.Println("📥 Received:", filename)
			fileReceiveChan <- filename
		}

		// Handle incoming video frames
		if strings.HasPrefix(line, "VIDEO_B64|") {
			parts := strings.SplitN(line, "|", 3)
			if len(parts) >= 3 {
				b64Data := parts[2]
				fmt.Println("📹 Received video frame, size:", len(b64Data))
				
				// Send to web interface via channel
				videoFrameChan <- b64Data
				
				// Optionally save to file
				os.MkdirAll("ReceivedFiles/Video", 0755)
				frameFile := fmt.Sprintf("ReceivedFiles/Video/frame_%d.png", time.Now().UnixNano())
				// Decode and save (for now just save the base64 data)
				os.WriteFile(frameFile+".b64", []byte(b64Data), 0644)
			}
		}
	}
}

// GetReceivedFiles returns list of received files
func GetReceivedFiles(w http.ResponseWriter, r *http.Request) {
	files, err := filepath.Glob("ReceivedFiles/*")
	if err != nil {
		writeJSON(w, APIResponse{Success: false, Message: "Failed to list files"})
		return
	}

	var filenames []string
	for _, f := range files {
		filenames = append(filenames, filepath.Base(f))
	}

	writeJSON(w, APIResponse{Success: true, Data: filenames})
}

// DownloadFile serves a received file for download
func DownloadFile(w http.ResponseWriter, r *http.Request) {
	filename := r.URL.Query().Get("file")
	if filename == "" {
		http.Error(w, "Filename required", 400)
		return
	}

	path := "ReceivedFiles/" + filename
	http.ServeFile(w, r, path)
}

// GetStatus returns overall system status
func GetStatus(w http.ResponseWriter, r *http.Request) {
	status := map[string]interface{}{
		"ngrok": NgrokStatus{
			Running: ngrokProcess != nil && ngrokProcess.Process != nil,
			URL:     strings.TrimPrefix(ngrokURL, "tcp://"),
		},
		"relay": RelayStatus{
			Running:  relayProcess != nil && relayProcess.Process != nil,
			ServerID: deviceID,
		},
		"connected": relayConn != nil,
		"device_id": deviceID,
		"server_addr": serverAddr,
	}

	writeJSON(w, APIResponse{Success: true, Data: status})
}

// GetNotifications returns recent file receive notifications
func GetNotifications(w http.ResponseWriter, r *http.Request) {
	select {
	case file := <-fileReceiveChan:
		writeJSON(w, APIResponse{Success: true, Data: map[string]string{"file": file}})
	case videoFrame := <-videoFrameChan:
		writeJSON(w, APIResponse{Success: true, Data: map[string]string{"video": videoFrame}})
	case <-time.After(1 * time.Second):
		writeJSON(w, APIResponse{Success: true, Data: nil})
	}
}

func writeJSON(w http.ResponseWriter, resp APIResponse) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func main() {
	// Serve static files
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/", fs)

	// API endpoints
	http.HandleFunc("/api/ngrok/start", StartNgrok)
	http.HandleFunc("/api/ngrok/stop", StopNgrok)
	http.HandleFunc("/api/ngrok/status", GetNgrokStatus)
	http.HandleFunc("/api/ngrok/token", SaveNgrokToken)
	http.HandleFunc("/api/ngrok/code", GetShortCode)

	http.HandleFunc("/api/relay/start", StartRelayServer)
	http.HandleFunc("/api/relay/stop", StopRelayServer)
	http.HandleFunc("/api/relay/status", GetRelayStatus)

	http.HandleFunc("/api/device/register", RegisterDevice)
	http.HandleFunc("/api/devices", GetDevices)

	http.HandleFunc("/api/file/send", SendFile)
	http.HandleFunc("/api/file/upload", UploadFile)
	http.HandleFunc("/api/files", GetReceivedFiles)
	http.HandleFunc("/api/file/download", DownloadFile)

	http.HandleFunc("/api/video/stream", StreamVideo)

	http.HandleFunc("/api/status", GetStatus)
	http.HandleFunc("/api/notifications", GetNotifications)

	fmt.Println("🌐 Copycat Web Dashboard starting at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

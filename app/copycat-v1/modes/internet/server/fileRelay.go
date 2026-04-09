package main

import (
	"bufio"
	"fmt"
	"net"
	"strings"
)

// Server-side URL mapping
var serverURL string

func handleClient(conn net.Conn) {
	reader := bufio.NewReader(conn)

	// First message = REGISTER|deviceID
	msg, _ := reader.ReadString('\n')
	msg = strings.TrimSpace(msg)

	parts := strings.Split(msg, "|")
	if len(parts) < 2 {
		conn.Close()
		return
	}

	deviceID := parts[1]
	registerDevice(deviceID, conn)

	fmt.Println("✅ Registered:", deviceID)

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Disconnected:", deviceID)
			removeDevice(deviceID)
			return
		}

		line = strings.TrimSpace(line)

		// MAP_URL|url - Server registers its public URL
		if strings.HasPrefix(line, "MAP_URL|") {
			urlPart := strings.TrimPrefix(line, "MAP_URL|")
			serverURL = urlPart
			fmt.Println("🗺️ Server URL mapped:", serverURL)
			conn.Write([]byte("OK\n"))
			continue
		}

		// GET_URL - Client asks for the server's public URL
		if line == "GET_URL" {
			if serverURL != "" {
				conn.Write([]byte(serverURL + "\n"))
				fmt.Println("🗺️ Sent URL to client:", serverURL)
			} else {
				conn.Write([]byte("NO_URL\n"))
				fmt.Println("🗺️ No URL available")
			}
			continue
		}

		// LIST_DEVICES - return all registered device IDs
		if line == "LIST_DEVICES" {
			devices := listDevices()
			var ids []string
			for id := range devices {
				ids = append(ids, id)
			}
			conn.Write([]byte(strings.Join(ids, "|")))
			continue
		}

		// SEND|targetID|filename - read file until EOF_MARKER
		if strings.HasPrefix(line, "SEND|") {
			p := strings.Split(line, "|")
			targetID := p[1]
			filename := p[2]

			targetConn := getDevice(targetID)
			if targetConn == nil {
				fmt.Println("Target not found")
				continue
			}

			// Notify receiver
			fmt.Fprintf(targetConn, "INCOMING|%s\n", filename)

			// Read file content until EOF_MARKER
			for {
				chunk, err := reader.ReadString('\n')
				if err != nil {
					break
				}
				if strings.TrimSpace(chunk) == "EOF_MARKER" {
					break
				}
				targetConn.Write([]byte(chunk))
			}

			fmt.Println("📤 Relayed file to", targetID)
		}

		// VIDEO_START|targetID - forward video frames to target device
		if strings.HasPrefix(line, "VIDEO_START|") {
			parts := strings.Split(line, "|")
			if len(parts) < 2 {
				continue
			}
			targetID := parts[1]

			targetConn := getDevice(targetID)
			if targetConn == nil {
				fmt.Println("Video target not found:", targetID)
				continue
			}

			// Notify receiver about incoming video
			fmt.Fprintf(targetConn, "VIDEO_INCOMING|\n")

			// Read video frame data (binary)
			// For streaming, we'll read raw bytes until a special marker
			// This is a simplified version - could be enhanced with size headers
			buf := make([]byte, 1024*1024) // 1MB buffer for frame
			n, err := reader.Read(buf)
			if err != nil && err.Error() != "EOF" {
				fmt.Println("Video read error:", err)
				continue
			}
			if n > 0 {
				targetConn.Write(buf[:n])
				fmt.Println("📹 Relayed video frame to", targetID, "size:", n)
			}
		}

		// VIDEO_B64|targetID - forward base64 encoded video frame
		if strings.HasPrefix(line, "VIDEO_B64|") {
			parts := strings.SplitN(line, "|", 3)  // Split into max 3 parts
			if len(parts) < 3 {
				fmt.Println("Video B64: invalid format - not enough parts")
				continue
			}
			targetID := parts[1]
			b64Data := parts[2]  // The frame data is already in the third part

			targetConn := getDevice(targetID)
			if targetConn == nil {
				fmt.Println("Video B64: target not found:", targetID)
				continue
			}

			// Forward to target device
			fmt.Fprintf(targetConn, "VIDEO_B64|%s\n", b64Data)
			fmt.Println("📹 Relayed base64 video frame to", targetID, "size:", len(b64Data))
		}
	}
}

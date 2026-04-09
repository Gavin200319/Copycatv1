package internet

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
)

func receiveFiles(conn net.Conn) {
	reader := bufio.NewReader(conn)

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Disconnected from server")
			return
		}

		line = strings.TrimSpace(line)

		// Handle incoming file
		if strings.HasPrefix(line, "INCOMING|") {
			parts := strings.Split(line, "|")
			filename := parts[1]

			os.MkdirAll("ReceivedFiles", 0755)
			file, _ := os.Create("ReceivedFiles/" + filename)

			io.Copy(file, reader)

			fmt.Println("📥 Received:", filename)
			file.Close()
		}

		// Handle incoming video notification
		if strings.HasPrefix(line, "VIDEO_INCOMING|") {
			fmt.Println("📹 Receiving video stream...")
			receiveVideoStream(reader)
		}

		// Handle base64 encoded video frame
		if strings.HasPrefix(line, "VIDEO_B64|") {
			b64Data := strings.TrimPrefix(line, "VIDEO_B64|")
			// Decode and display/save frame
			frameData, err := base64.StdEncoding.DecodeString(b64Data)
			if err != nil {
				fmt.Println("Failed to decode video frame:", err)
				continue
			}
			saveVideoFrame(frameData)
		}
	}
}

// receiveVideoStream handles receiving video frames
func receiveVideoStream(reader *bufio.Reader) {
	os.MkdirAll("ReceivedFiles/Video", 0755)
	frameNum := 0

	for {
		// Check for end of stream or new command
		// In this simplified version, we'll read until disconnected
		// or until we get a specific end marker
		
		buf := make([]byte, 1024*1024) // 1MB buffer
		n, err := reader.Read(buf)
		if err != nil {
			if err.Error() != "EOF" {
				fmt.Println("Video stream error:", err)
			}
			break
		}

		if n > 0 {
			// Save frame to file for now (could be displayed in GUI)
			frameFile := fmt.Sprintf("ReceivedFiles/Video/frame_%04d.png", frameNum)
			os.WriteFile(frameFile, buf[:n], 0644)
			fmt.Printf("📹 Saved frame %d (size: %d bytes)\n", frameNum, n)
			frameNum++
		}
	}

	fmt.Println("📹 Video stream ended, saved", frameNum, "frames")
}

// saveVideoFrame saves a decoded video frame
func saveVideoFrame(frameData []byte) {
	os.MkdirAll("ReceivedFiles/Video", 0755)
	
	// Save as PNG
	frameFile := fmt.Sprintf("ReceivedFiles/Video/b64_frame_%d.png", 
		len([]byte{}))
	
	err := os.WriteFile(frameFile, frameData, 0644)
	if err != nil {
		fmt.Println("Failed to save video frame:", err)
		return
	}
	
	fmt.Printf("📹 Saved base64 frame (%d bytes)\n", len(frameData))
}

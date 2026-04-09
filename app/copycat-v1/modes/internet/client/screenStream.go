package internet

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"time"
)

// streamScreen captures and streams screen to target device
func streamScreen(conn net.Conn, targetID string) {
	fmt.Println("📺 Starting screen capture stream to:", targetID)
	
	// Different methods based on platform
	// Try native screenshot first, then fall back to external tools
	
	for {
		select {
		case <-time.After(time.Second / 5): // 5 FPS
			// Try to capture screen using available method
			frameData, err := captureScreen()
			if err != nil {
				fmt.Println("Capture error:", err)
				continue
			}
			
			if len(frameData) == 0 {
				continue
			}
			
			// Send video frame
			fmt.Fprintf(conn, "VIDEO_START|%d\n", targetID)
			io.Copy(conn, bytes.NewReader(frameData))
		}
	}
}

// captureScreen attempts to capture screen using various methods
func captureScreen() ([]byte, error) {
	// Method 1: Try using ffmpeg (works on Linux)
	// This captures the whole screen
	cmd := exec.Command("ffmpeg", "-f", "x11grab", "-s", "1920x1080", "-i", ":0", 
		"-vframes", "1", "-f", "image2pipe", "-vcodec", "png", "-")
	
	output, err := cmd.Output()
	if err == nil && len(output) > 0 {
		return output, nil
	}
	
	// Method 2: Try scrot (screenshot tool on Linux)
	cmd2 := exec.Command("scrot", "-")
	output2, err2 := cmd2.Output()
	if err2 == nil && len(output2) > 0 {
		return output2, nil
	}
	
	// Method 3: Check for Android screen capture
	if checkAndroid() {
		return captureAndroidScreen()
	}
	
	// Method 4: Try grim (sway/wayland)
	cmd3 := exec.Command("grim", "-")
	output3, err3 := cmd3.Output()
	if err3 == nil && len(output3) > 0 {
		return output3, nil
	}
	
	return nil, fmt.Errorf("no screen capture method available")
}

// checkAndroid checks if running on Android (Termux)
func checkAndroid() bool {
	_, err := os.Stat("/system/bin/screencap")
	return err == nil
}

// captureAndroidScreen captures screen on Android
func captureAndroidScreen() ([]byte, error) {
	// Save to temp file then read
	tmpFile := "/sdcard/Download/copycat_temp.png"
	cmd := exec.Command("screencap", "-p", tmpFile)
	err := cmd.Run()
	if err != nil {
		return nil, err
	}
	
	return os.ReadFile(tmpFile)
}

// sendBase64Screen sends screen as base64 (for web display)
func sendBase64Screen(conn net.Conn, targetID string) {
	for {
		frameData, err := captureScreen()
		if err != nil {
			time.Sleep(time.Second)
			continue
		}
		
		// Encode to base64
		encoded := base64.StdEncoding.EncodeToString(frameData)
		
		// Send with header
		fmt.Fprintf(conn, "VIDEO_B64|%s\n", encoded)
		
		time.Sleep(time.Second / 5) // 5 FPS
	}
}

// StreamToWeb streams screen to web interface via relay
func streamToWeb(conn net.Conn, frameChan chan []byte) {
	for frame := range frameChan {
		size := len(frame)
		header := fmt.Sprintf("VIDEO_WEB|%d\n", size)
		conn.Write([]byte(header))
		conn.Write(frame)
	}
}
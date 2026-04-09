// Copycat Web Dashboard - JavaScript

const API_BASE = '';

// Helper function for API calls
async function apiCall(endpoint, options = {}) {
    try {
        const response = await fetch(API_BASE + endpoint, {
            ...options,
            headers: {
                'Content-Type': 'application/json',
                ...options.headers,
            },
        });
        return await response.json();
    } catch (error) {
        console.error('API Error:', error);
        return { success: false, message: error.message };
    }
}

// Show notification
function showNotification(message, isError = false) {
    const container = document.getElementById('notifications');
    const notification = document.createElement('div');
    notification.className = `notification ${isError ? 'error' : ''}`;
    notification.innerHTML = message;
    container.appendChild(notification);
    
    setTimeout(() => {
        notification.remove();
    }, 5000);
}

// Update status display
function updateStatus(elementId, status, isOnline) {
    const element = document.getElementById(elementId);
    element.textContent = status;
    element.className = `value ${isOnline ? 'status-online' : 'status-offline'}`;
}

// Get status
async function getStatus() {
    const result = await apiCall('/api/status');
    if (result.success && result.data) {
        const data = result.data;
        
        // Ngrok status
        if (data.ngrok) {
            updateStatus('ngrok-status', data.ngrok.running ? 'Running' : 'Stopped', data.ngrok.running);
            const urlElement = document.getElementById('ngrok-url');
            urlElement.textContent = data.ngrok.url || '';
            urlElement.style.display = data.ngrok.url ? 'block' : 'none';
        }
        
        // Relay status
        if (data.relay) {
            updateStatus('relay-status', data.relay.running ? 'Running' : 'Stopped', data.relay.running);
        }
        
        // Connection status
        updateStatus('connection-status', data.connected ? 'Connected' : 'Disconnected', data.connected);
        
        // Device ID
        const deviceIdElement = document.getElementById('device-id');
        deviceIdElement.textContent = data.device_id || 'Not registered';
        
        // Pre-fill server address if available
        if (data.server_addr) {
            document.getElementById('server-addr-input').value = data.server_addr;
        }
    }
    
    // Also fetch the short code
    fetchShortCode();
}

// Get and display short code
async function fetchShortCode() {
    try {
        const result = await apiCall('/api/ngrok/code');
        const codeElement = document.getElementById('ngrok-code');
        if (result.success && result.data && result.data.code) {
            codeElement.textContent = '🔑 Code: ' + result.data.code;
            codeElement.style.display = 'block';
        } else {
            codeElement.style.display = 'none';
        }
    } catch (e) {
        // Ignore errors
    }
}

// Save ngrok token
async function saveNgrokToken() {
    const token = document.getElementById('ngrok-authtoken').value;
    if (!token) {
        showNotification('Please enter a ngrok authtoken', true);
        return;
    }
    
    const result = await apiCall('/api/ngrok/token', {
        method: 'POST',
        body: JSON.stringify({ token: token })
    });
    
    if (result.success) {
        showNotification('Ngrok token saved. You can now start ngrok.');
    } else {
        showNotification(result.message || 'Failed to save token', true);
    }
}

// Start ngrok
async function startNgrok() {
    const result = await apiCall('/api/ngrok/start', { method: 'POST' });
    if (result.success) {
        showNotification('Ngrok started successfully');
        setTimeout(getStatus, 2000);
    } else {
        showNotification(result.message || 'Failed to start ngrok', true);
    }
}

// Start relay server
async function startRelay() {
    const result = await apiCall('/api/relay/start', { method: 'POST' });
    if (result.success) {
        showNotification('Relay server started successfully');
        setTimeout(getStatus, 2000);
    } else {
        showNotification(result.message || 'Failed to start relay server', true);
    }
}

// Start everything
async function startAll() {
    showNotification('Starting all services...');
    
    // Start ngrok first
    await startNgrok();
    
    // Wait for ngrok to get URL
    await new Promise(resolve => setTimeout(resolve, 3000));
    
    // Get ngrok URL
    const statusResult = await apiCall('/api/ngrok/status');
    if (statusResult.success && statusResult.data && statusResult.data.url) {
        document.getElementById('server-addr-input').value = statusResult.data.url;
    }
    
    // Start relay server
    await startRelay();
    
    showNotification('All services started!');
    setTimeout(getStatus, 2000);
}

// Stop all
async function stopAll() {
    await apiCall('/api/ngrok/stop', { method: 'POST' });
    await apiCall('/api/relay/stop', { method: 'POST' });
    showNotification('All services stopped');
    setTimeout(getStatus, 1000);
}

// Register device
async function registerDevice() {
    const deviceId = document.getElementById('device-id-input').value.trim();
    const serverAddr = document.getElementById('server-addr-input').value.trim();
    
    if (!deviceId) {
        showNotification('Please enter a device ID', true);
        return;
    }
    
    if (!serverAddr) {
        showNotification('Please enter server address or start ngrok first', true);
        return;
    }
    
    const result = await apiCall('/api/device/register', {
        method: 'POST',
        body: JSON.stringify({ id: deviceId, server_addr: serverAddr }),
    });
    
    if (result.success) {
        showNotification('Device registered successfully');
        getStatus();
    } else {
        showNotification(result.message || 'Failed to register device', true);
    }
}

// Refresh devices list
async function refreshDevices() {
    const result = await apiCall('/api/devices');
    const container = document.getElementById('devices-list');
    
    if (result.success && result.data && result.data.length > 0) {
        container.innerHTML = result.data.map(id => `
            <div class="device-item">
                <span class="device-id">${id}</span>
                <button class="copy-btn" onclick="copyToClipboard('${id}')">Copy</button>
            </div>
        `).join('');
    } else {
        container.innerHTML = '<p class="empty-message">No devices connected</p>';
    }
}

// Refresh files list
async function refreshFiles() {
    const result = await apiCall('/api/files');
    const container = document.getElementById('files-list');
    
    if (result.success && result.data && result.data.length > 0) {
        container.innerHTML = result.data.map(filename => `
            <div class="file-item">
                <span class="file-name">${filename}</span>
                <a class="download-btn" href="/api/file/download?file=${encodeURIComponent(filename)}" target="_blank">Download</a>
            </div>
        `).join('');
    } else {
        container.innerHTML = '<p class="empty-message">No files received</p>';
    }
}

// Send file
async function sendFile() {
    const targetId = document.getElementById('target-id-input').value.trim();
    const fileInput = document.getElementById('file-input');
    
    if (!targetId) {
        showNotification('Please enter target device ID', true);
        return;
    }
    
    if (!fileInput.files || fileInput.files.length === 0) {
        showNotification('Please select a file', true);
        return;
    }
    
    const formData = new FormData();
    formData.append('target_id', targetId);
    formData.append('file', fileInput.files[0]);
    
    try {
        const response = await fetch(API_BASE + '/api/file/upload', {
            method: 'POST',
            body: formData,
        });
        
        const result = await response.json();
        
        if (result.success) {
            showNotification('File sent successfully!');
            fileInput.value = ''; // Clear file input
        } else {
            showNotification(result.message || 'Failed to send file', true);
        }
    } catch (error) {
        showNotification('Error sending file: ' + error.message, true);
    }
}

// Video streaming - start screen capture and stream to target
let videoStreamInterval = null;
let videoFrameCount = 0;

async function startVideoStream() {
    const targetId = document.getElementById('video-target-id').value.trim();
    
    if (!targetId) {
        showNotification('Please enter target device ID', true);
        return;
    }
    
    // Check if we're registered
    const statusResult = await apiCall('/api/status');
    if (!statusResult.success || !statusResult.data.connected) {
        showNotification('Please register device first', true);
        return;
    }
    
    showNotification('Starting screen stream to ' + targetId + '...');
    
    // Update UI
    document.getElementById('video-status').innerHTML = '<span class="status-online">Streaming...</span>';
    
    // Start capturing and sending frames
    videoStreamInterval = setInterval(async () => {
        try {
            // Capture screen using browser's getDisplayMedia or we can capture the desktop
            // For now, we'll capture the current viewport as an image
            const canvas = document.createElement('canvas');
            canvas.width = window.innerWidth;
            canvas.height = window.innerHeight;
            const ctx = canvas.getContext('2d');
            
            // Draw current page to canvas
            ctx.drawWindow(window, 0, 0, window.innerWidth, window.innerHeight);
            
            // Convert to base64
            const frameData = canvas.toDataURL('image/png');
            const base64 = frameData.split(',')[1]; // Remove data:image/png;base64, prefix
            
            // Send to server
            const response = await fetch(API_BASE + '/api/video/stream', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    target_id: targetId,
                    frame: base64
                })
            });
            
            const result = await response.json();
            if (result.success) {
                videoFrameCount++;
                document.getElementById('video-status').innerHTML = 
                    '<span class="status-online">Streaming - Frame ' + videoFrameCount + '</span>';
            }
        } catch (error) {
            console.error('Video stream error:', error);
        }
    }, 1000); // 1 FPS - adjust for more/less frames
}

// Stop video streaming
function stopVideoStream() {
    if (videoStreamInterval) {
        clearInterval(videoStreamInterval);
        videoStreamInterval = null;
    }
    
    document.getElementById('video-status').innerHTML = '<span class="status-offline">Not streaming</span>';
    showNotification('Video stream stopped. Sent ' + videoFrameCount + ' frames');
    videoFrameCount = 0;
}

// Display incoming video frame on web interface
function displayVideoFrame(base64Image) {
    const placeholder = document.querySelector('#video-preview .placeholder');
    const img = document.getElementById('video-frame');
    if (img) {
        img.src = 'data:image/png;base64,' + base64Image;
        img.style.display = 'block';
        
        // Hide placeholder
        if (placeholder) {
            placeholder.style.display = 'none';
        }
        
        console.log("Displaying video frame, size:", base64Image.length);
    }
}

// Copy to clipboard
function copyToClipboard(text) {
    navigator.clipboard.writeText(text).then(() => {
        showNotification('Copied to clipboard!');
    }).catch(() => {
        showNotification('Failed to copy', true);
    });
}

// Check for notifications (incoming files and video)
async function checkNotifications() {
    try {
        const result = await apiCall('/api/notifications');
        if (result.success && result.data) {
            // Check for incoming file
            if (result.data.file) {
                const filename = result.data.file;
                const downloadUrl = `/api/file/download?file=${encodeURIComponent(filename)}`;
                showNotification(`
                    <div style="display: flex; align-items: center; gap: 10px;">
                        <span style="font-size: 24px;">📥</span>
                        <div>
                            <div style="font-weight: bold;">File Received!</div>
                            <div style="font-size: 0.9em; color: #333;">${filename}</div>
                            <a href="${downloadUrl}" target="_blank" style="color: #0066cc; text-decoration: underline; font-size: 0.85em;">Download</a>
                        </div>
                    </div>
                `);
                refreshFiles();
            }
            
            // Check for incoming video frame
            if (result.data.video) {
                console.log("Received video frame");
                displayVideoFrame(result.data.video);
                showNotification(`
                    <div style="display: flex; align-items: center; gap: 10px;">
                        <span style="font-size: 24px;">📹</span>
                        <div>
                            <div style="font-weight: bold;">Video Frame Received!</div>
                            <div style="font-size: 0.9em; color: #333;">Size: ${result.data.video.length} bytes</div>
                        </div>
                    </div>
                `);
            }
        }
    } catch (e) {
        // Ignore errors
    }
}

// Initialize
document.addEventListener('DOMContentLoaded', () => {
    getStatus();
    refreshDevices();
    refreshFiles();
    
    // Poll for status and notifications
    setInterval(getStatus, 5000);
    setInterval(checkNotifications, 3000);
});

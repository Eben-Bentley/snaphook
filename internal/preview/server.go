package preview

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"sync"
	"time"
)

const (
	maxHistorySize = 50
	maxClients     = 5
)

var (
	server           *http.Server
	latestImage      string
	imageHistory     []string
	imageMutex       sync.RWMutex
	serverStarted    bool
	serverURL        = "http://localhost:8765"
	lastRequest      time.Time
	requestMutex     sync.RWMutex
	clients          []chan string
	clientsMutex     sync.Mutex
	hotkeyChangeChan = make(chan string, 10)
)

func Start() {
	if serverStarted {
		return
	}
	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		requestMutex.Lock()
		lastRequest = time.Now()
		requestMutex.Unlock()

		html := `<!DOCTYPE html>
<html>
<head>
    <title>SnapHook</title>
    <style>
        body {
            margin: 0;
            padding: 20px;
            background: #1e1e1e;
            display: flex;
            flex-direction: column;
            justify-content: center;
            align-items: center;
            min-height: 100vh;
            cursor: default;
        }
        .container {
            display: flex;
            flex-direction: column;
            align-items: center;
            gap: 20px;
        }
        img {
            max-width: 90vw;
            max-height: 85vh;
            box-shadow: 0 4px 20px rgba(0,0,0,0.5);
        }
        .waiting {
            color: #888;
            font-family: Arial;
            font-size: 24px;
            text-align: center;
        }
        .spinner {
            border: 4px solid #333;
            border-top: 4px solid #888;
            border-radius: 50%;
            width: 40px;
            height: 40px;
            animation: spin 1s linear infinite;
            margin: 20px auto;
        }
        @keyframes spin {
            0% { transform: rotate(0deg); }
            100% { transform: rotate(360deg); }
        }
        .save-btn {
            padding: 12px 24px;
            background: #4CAF50;
            color: white;
            border: none;
            border-radius: 4px;
            font-size: 16px;
            cursor: pointer;
            font-family: Arial;
            transition: background 0.3s;
        }
        .save-btn:hover {
            background: #45a049;
        }
        .save-btn:active {
            background: #3d8b40;
        }
        .history-btn {
            padding: 12px 24px;
            background: #2196F3;
            color: white;
            border: none;
            border-radius: 4px;
            font-size: 16px;
            cursor: pointer;
            font-family: Arial;
            transition: background 0.3s;
        }
        .history-btn:hover {
            background: #0b7dda;
        }
        .button-group {
            display: flex;
            gap: 10px;
        }
    </style>
</head>
<body>
    <div class="container">
        <img id="screenshot" src="/image?t=` + fmt.Sprintf("%d", time.Now().UnixNano()) + `" onerror="this.style.display='none';document.querySelector('.waiting').style.display='block';document.querySelector('.save-btn').style.display='none'" style="cursor: default;">
        <div class="button-group">
            <button class="save-btn" onclick="saveImage()">Save Screenshot</button>
            <button class="history-btn" onclick="window.location='/history'">View History</button>
        </div>
        <div class="waiting" style="display:none">
            <div class="spinner"></div>
            Waiting for screenshot...
        </div>
    </div>
    <script>
        const img = document.getElementById('screenshot');
        const waiting = document.querySelector('.waiting');
        const saveBtn = document.querySelector('.save-btn');

        const eventSource = new EventSource('/events');
        eventSource.onmessage = function(event) {
            img.src = '/image?t=' + Date.now();
            img.style.display = 'block';
            waiting.style.display = 'none';
            saveBtn.style.display = 'block';
        };

        function saveImage() {
            const timestamp = new Date().toISOString().replace(/[:.]/g, '-').slice(0, 19);
            const link = document.createElement('a');
            link.href = '/image?t=' + Date.now();
            link.download = 'screenshot_' + timestamp + '.png';
            link.click();
        }
    </script>
</body>
</html>`
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(html))
	})

	mux.HandleFunc("/image", func(w http.ResponseWriter, r *http.Request) {
		requestMutex.Lock()
		lastRequest = time.Now()
		requestMutex.Unlock()

		indexStr := r.URL.Query().Get("index")

		imageMutex.RLock()
		var imgPath string
		if indexStr != "" {
			var index int
			fmt.Sscanf(indexStr, "%d", &index)
			if index >= 0 && index < len(imageHistory) {
				imgPath = imageHistory[index]
			}
		} else {
			imgPath = latestImage
		}
		imageMutex.RUnlock()

		if imgPath == "" {
			http.Error(w, "No image yet", http.StatusNotFound)
			return
		}

		file, err := os.Open(imgPath)
		if err != nil {
			http.Error(w, "Image not found", http.StatusNotFound)
			return
		}
		defer file.Close()

		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		io.Copy(w, file)
	})

	mux.HandleFunc("/history", func(w http.ResponseWriter, r *http.Request) {
		requestMutex.Lock()
		lastRequest = time.Now()
		requestMutex.Unlock()

		imageMutex.RLock()
		history := make([]string, len(imageHistory))
		copy(history, imageHistory)
		imageMutex.RUnlock()

		html := `<!DOCTYPE html>
<html>
<head>
    <title>SnapHook - History</title>
    <style>
        body {
            margin: 0;
            padding: 20px;
            background: #1e1e1e;
            font-family: Arial;
            color: #fff;
        }
        h1 {
            text-align: center;
            color: #888;
        }
        .gallery {
            display: grid;
            grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
            gap: 20px;
            padding: 20px;
        }
        .thumbnail {
            background: #2a2a2a;
            border-radius: 8px;
            overflow: hidden;
            position: relative;
            transition: transform 0.2s;
        }
        .thumbnail:hover {
            transform: scale(1.05);
        }
        .thumbnail img {
            width: 100%;
            height: 150px;
            object-fit: cover;
            cursor: pointer;
        }
        .thumbnail .info {
            padding: 10px;
            text-align: center;
            font-size: 12px;
            color: #888;
        }
        .delete-text {
            position: absolute;
            top: 8px;
            right: 8px;
            background: #f44336;
            color: white;
            padding: 6px 12px;
            border-radius: 4px;
            font-size: 12px;
            font-weight: bold;
            cursor: pointer;
            opacity: 0;
            transition: opacity 0.2s;
            z-index: 10;
        }
        .thumbnail:hover .delete-text {
            opacity: 1;
        }
        .delete-text:hover {
            background: #d32f2f;
        }
        .back-btn, .clear-btn {
            padding: 12px 24px;
            color: white;
            border: none;
            border-radius: 4px;
            font-size: 16px;
            cursor: pointer;
            margin: 10px;
        }
        .back-btn {
            background: #4CAF50;
        }
        .back-btn:hover {
            background: #45a049;
        }
        .clear-btn {
            background: #f44336;
        }
        .clear-btn:hover {
            background: #d32f2f;
        }
        .button-group {
            text-align: center;
            margin: 20px 0;
        }
    </style>
</head>
<body>
    <h1>Screenshot History (` + fmt.Sprintf("%d/%d", len(history), maxHistorySize) + `)</h1>
    <div class="button-group">
        <button class="back-btn" onclick="window.location='/'">Back to Latest</button>
        <button class="clear-btn" onclick="clearAll()">Clear All History</button>
    </div>
    <div class="gallery">`

		for i := len(history) - 1; i >= 0; i-- {
			html += fmt.Sprintf(`
        <div class="thumbnail">
            <img src="/image?index=%d" alt="Screenshot %d" onclick="window.location='/?index=%d'">
            <div class="delete-text" onclick="deleteScreenshot(%d, event)">Delete</div>
            <div class="info">Screenshot #%d</div>
        </div>`, i, i+1, i, i, i+1)
		}

		html += `
    </div>
    <script>
        function deleteScreenshot(index, event) {
            event.stopPropagation();
            fetch('/delete', {
                method: 'POST',
                headers: {'Content-Type': 'application/x-www-form-urlencoded'},
                body: 'index=' + index
            })
            .then(r => r.json())
            .then(() => window.location.reload())
            .catch(err => alert('Failed to delete: ' + err));
        }

        function clearAll() {
            if (confirm('Clear all screenshot history? This cannot be undone.')) {
                fetch('/clear-all', {
                    method: 'POST'
                })
                .then(r => r.json())
                .then(() => window.location.reload())
                .catch(err => alert('Failed to clear history: ' + err));
            }
        }
    </script>
</body>
</html>`
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(html))
	})

	mux.HandleFunc("/settings", func(w http.ResponseWriter, r *http.Request) {
		requestMutex.Lock()
		lastRequest = time.Now()
		requestMutex.Unlock()

		if r.Method == "POST" {
			newHotkey := r.FormValue("hotkey")
			if newHotkey != "" {
				hotkeyChangeChan <- newHotkey
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`{"success": true}`))
				return
			}
		}

		html := `<!DOCTYPE html>
<html>
<head>
    <title>SnapHook Settings</title>
    <style>
        body {
            margin: 0;
            padding: 20px;
            background: #1e1e1e;
            color: #e0e0e0;
            font-family: Arial, sans-serif;
        }
        .container {
            max-width: 600px;
            margin: 0 auto;
            background: #2a2a2a;
            border-radius: 8px;
            padding: 30px;
            box-shadow: 0 4px 20px rgba(0,0,0,0.5);
        }
        h1 {
            color: #fff;
            margin-top: 0;
        }
        .section {
            margin: 20px 0;
            padding: 20px;
            background: #333;
            border-radius: 4px;
        }
        .section h2 {
            margin-top: 0;
            color: #4CAF50;
        }
        label {
            display: block;
            margin-bottom: 8px;
            color: #bbb;
        }
        input[type="text"] {
            width: 100%;
            padding: 12px;
            background: #1e1e1e;
            border: 2px solid #444;
            border-radius: 4px;
            color: #fff;
            font-size: 16px;
            box-sizing: border-box;
        }
        input[type="text"]:focus {
            outline: none;
            border-color: #4CAF50;
        }
        .hint {
            color: #888;
            font-size: 12px;
            margin-top: 5px;
        }
        .btn {
            padding: 12px 24px;
            background: #4CAF50;
            color: white;
            border: none;
            border-radius: 4px;
            font-size: 16px;
            cursor: pointer;
            margin-top: 15px;
        }
        .btn:hover {
            background: #45a049;
        }
        .status {
            margin-top: 15px;
            padding: 10px;
            border-radius: 4px;
            display: none;
        }
        .status.success {
            background: #4CAF50;
            color: white;
        }
        .status.error {
            background: #f44336;
            color: white;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>SnapHook Settings</h1>

        <div class="section">
            <h2>Hotkey Configuration</h2>
            <label>Press your desired hotkey combination:</label>
            <input type="text" id="hotkeyInput" readonly placeholder="Click here and press keys..." value="">
            <div class="hint">Examples: Ctrl+Shift+S, Ctrl+Alt+S, PrintScreen</div>
            <button class="btn" onclick="saveHotkey()">Save Hotkey</button>
            <div id="status" class="status"></div>
        </div>
    </div>

    <script>
        const input = document.getElementById('hotkeyInput');
        const status = document.getElementById('status');

        input.addEventListener('keydown', function(e) {
            e.preventDefault();

            const keys = [];
            if (e.ctrlKey) keys.push('Ctrl');
            if (e.altKey) keys.push('Alt');
            if (e.shiftKey) keys.push('Shift');

            const key = e.key;
            if (key === 'Control' || key === 'Alt' || key === 'Shift') {
                return;
            }

            if (key === 'PrintScreen') {
                input.value = 'PrintScreen';
            } else {
                keys.push(key.toUpperCase());
                input.value = keys.join('+');
            }
        });

        function saveHotkey() {
            const hotkey = input.value;
            if (!hotkey) {
                showStatus('Please press a hotkey combination first', false);
                return;
            }

            fetch('/settings', {
                method: 'POST',
                headers: {'Content-Type': 'application/x-www-form-urlencoded'},
                body: 'hotkey=' + encodeURIComponent(hotkey)
            })
            .then(r => r.json())
            .then(data => {
                if (data.success) {
                    showStatus('Hotkey changed successfully!', true);
                    setTimeout(() => window.close(), 2000);
                } else {
                    showStatus('Failed to change hotkey', false);
                }
            })
            .catch(() => showStatus('Error saving hotkey', false));
        }

        function showStatus(message, success) {
            status.textContent = message;
            status.className = 'status ' + (success ? 'success' : 'error');
            status.style.display = 'block';
        }
    </script>
</body>
</html>`
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(html))
	})

	mux.HandleFunc("/delete", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		indexStr := r.FormValue("index")
		if indexStr == "" {
			http.Error(w, "Missing index parameter", http.StatusBadRequest)
			return
		}

		var index int
		fmt.Sscanf(indexStr, "%d", &index)

		imageMutex.Lock()
		if index >= 0 && index < len(imageHistory) {
			imagePath := imageHistory[index]
			os.Remove(imagePath)
			imageHistory = append(imageHistory[:index], imageHistory[index+1:]...)
		}
		imageMutex.Unlock()

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"success": true}`))
	})

	mux.HandleFunc("/clear-all", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		imageMutex.Lock()
		for _, imagePath := range imageHistory {
			os.Remove(imagePath)
		}
		imageHistory = []string{}
		latestImage = ""
		imageMutex.Unlock()

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"success": true}`))
	})

	mux.HandleFunc("/events", func(w http.ResponseWriter, r *http.Request) {
		requestMutex.Lock()
		lastRequest = time.Now()
		requestMutex.Unlock()

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("X-Accel-Buffering", "no")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming not supported", http.StatusInternalServerError)
			return
		}

		clientChan := make(chan string, 10)
		clientsMutex.Lock()

		if len(clients) >= maxClients {
			safeClose(clients[0])
			clients = clients[1:]
		}
		clients = append(clients, clientChan)
		clientsMutex.Unlock()

		defer func() {
			clientsMutex.Lock()
			for i, ch := range clients {
				if ch == clientChan {
					clients = append(clients[:i], clients[i+1:]...)
					break
				}
			}
			clientsMutex.Unlock()
			safeClose(clientChan)
		}()

		ctx := r.Context()

		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-clientChan:
				if !ok {
					return
				}
				fmt.Fprintf(w, "data: %s\n\n", msg)
				flusher.Flush()
			}
		}
	})

	server = &http.Server{
		Addr:    ":8765",
		Handler: mux,
	}

	go func() {
		server.ListenAndServe()
	}()

	serverStarted = true
}

func ShowInBrowser(imagePath string) error {
	imageMutex.Lock()
	latestImage = imagePath
	imageHistory = append(imageHistory, imagePath)

	if len(imageHistory) > maxHistorySize {
		oldPath := imageHistory[0]
		os.Remove(oldPath)
		imageHistory = imageHistory[1:]
	}
	imageMutex.Unlock()

	if !serverStarted {
		return fmt.Errorf("preview server not started")
	}

	notifyClients("update")

	go tryOpenBrowser()

	return nil
}

func notifyClients(msg string) {
	clientsMutex.Lock()
	defer clientsMutex.Unlock()

	for _, clientChan := range clients {
		select {
		case clientChan <- msg:
		default:
		}
	}
}

// OpenBrowser opens the preview in the browser
func OpenBrowser() {
	// Always open when user explicitly clicks the menu button
	go func() {
		browserMutex.Lock()
		openBrowserWindow()
		browserOpened = true
		browserMutex.Unlock()
	}()
}

var (
	browserOpened bool
	browserMutex  sync.Mutex
)

func tryOpenBrowser() {
	browserMutex.Lock()
	defer browserMutex.Unlock()

	clientsMutex.Lock()
	hasActiveClients := len(clients) > 0
	clientsMutex.Unlock()

	if hasActiveClients {
		return
	}

	openBrowserWindow()
	browserOpened = true
}

func openBrowserWindow() {
	go func() {
		var cmd *exec.Cmd
		switch runtime.GOOS {
		case "windows":
			cmd = exec.Command("cmd", "/c", "start", serverURL)
		case "darwin":
			cmd = exec.Command("open", serverURL)
		default:
			cmd = exec.Command("xdg-open", serverURL)
		}

		if err := cmd.Start(); err != nil {
			fmt.Printf("Failed to open browser: %v\n", err)
		}
	}()
}

func safeClose(ch chan string) {
	defer func() {
		recover()
	}()
	close(ch)
}

func Shutdown() {
	if server != nil {
		server.Close()
		serverStarted = false
	}
}

func OpenSettings() {
	go openBrowserWindow()
	go func() {
		time.Sleep(100 * time.Millisecond)
		var cmd *exec.Cmd
		switch runtime.GOOS {
		case "windows":
			cmd = exec.Command("cmd", "/c", "start", serverURL+"/settings")
		case "darwin":
			cmd = exec.Command("open", serverURL+"/settings")
		default:
			cmd = exec.Command("xdg-open", serverURL+"/settings")
		}
		cmd.Start()
	}()
}

func GetHotkeyChangeChan() chan string {
	return hotkeyChangeChan
}


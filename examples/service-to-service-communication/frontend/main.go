package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

// BackendResponse represents the response from backend API
type BackendResponse struct {
	Service   string    `json:"service"`
	Message   string    `json:"message"`
	Hostname  string    `json:"hostname"`
	Timestamp time.Time `json:"timestamp"`
}

// PageData represents data for the HTML template
type PageData struct {
	Title           string
	FrontendHost    string
	BackendURL      string
	BackendResponse *BackendResponse
	Error           string
}

const htmlTemplate = `
<!DOCTYPE html>
<html>
<head>
    <title>{{.Title}}</title>
    <style>
        body {
            font-family: Arial, sans-serif;
            max-width: 800px;
            margin: 50px auto;
            padding: 20px;
            background-color: #f5f5f5;
        }
        .container {
            background-color: white;
            border-radius: 8px;
            padding: 30px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        h1 { color: #333; }
        .info { 
            background-color: #e7f3fe;
            border-left: 4px solid #2196F3;
            padding: 15px;
            margin: 20px 0;
        }
        .success {
            background-color: #d4edda;
            border-left: 4px solid #28a745;
            padding: 15px;
            margin: 20px 0;
        }
        .error {
            background-color: #f8d7da;
            border-left: 4px solid #dc3545;
            padding: 15px;
            margin: 20px 0;
        }
        button {
            background-color: #4CAF50;
            color: white;
            padding: 10px 20px;
            border: none;
            border-radius: 4px;
            cursor: pointer;
            font-size: 16px;
        }
        button:hover {
            background-color: #45a049;
        }
        pre {
            background-color: #f4f4f4;
            padding: 10px;
            border-radius: 4px;
            overflow-x: auto;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>{{.Title}}</h1>
        
        <div class="info">
            <strong>Frontend Host:</strong> {{.FrontendHost}}<br>
            <strong>Backend URL:</strong> {{.BackendURL}}
        </div>

        <h2>Test Backend Communication</h2>
        <form action="/call-backend" method="GET">
            <button type="submit">Call Backend Service</button>
        </form>

        {{if .BackendResponse}}
        <div class="success">
            <h3>✅ Backend Response:</h3>
            <pre>{{.BackendResponse | json}}</pre>
        </div>
        {{end}}

        {{if .Error}}
        <div class="error">
            <h3>❌ Error:</h3>
            <pre>{{.Error}}</pre>
        </div>
        {{end}}

        <h2>Service Discovery Test</h2>
        <p>The frontend service discovers and communicates with the backend service using:</p>
        <ul>
            <li>ECS Service Discovery (AWS Cloud Map compatible)</li>
            <li>DNS resolution via namespace: <code>backend-api.production.local</code></li>
            <li>Automatic health checking and instance registration</li>
        </ul>
    </div>
</body>
</html>
`

func jsonFilter(v interface{}) template.HTML {
	b, _ := json.MarshalIndent(v, "", "  ")
	return template.HTML(b)
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	// Backend service URL - using service discovery
	backendURL := os.Getenv("BACKEND_URL")
	if backendURL == "" {
		// Default to service discovery DNS name
		backendURL = "http://backend-api.production.local:8080"
	}

	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}

	tmpl := template.Must(template.New("index").Funcs(template.FuncMap{
		"json": jsonFilter,
	}).Parse(htmlTemplate))

	// Health check endpoint
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"healthy"}`)
	})

	// Main page
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		data := PageData{
			Title:        "Service-to-Service Communication Demo",
			FrontendHost: hostname,
			BackendURL:   backendURL,
		}
		tmpl.Execute(w, data)
	})

	// Call backend endpoint
	http.HandleFunc("/call-backend", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Calling backend service at %s", backendURL)
		
		data := PageData{
			Title:        "Service-to-Service Communication Demo",
			FrontendHost: hostname,
			BackendURL:   backendURL,
		}

		// Call backend service
		client := &http.Client{
			Timeout: 10 * time.Second,
		}

		resp, err := client.Get(backendURL + "/api/data")
		if err != nil {
			data.Error = fmt.Sprintf("Failed to call backend: %v", err)
			tmpl.Execute(w, data)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			data.Error = fmt.Sprintf("Backend returned status %d: %s", resp.StatusCode, string(body))
			tmpl.Execute(w, data)
			return
		}

		var backendResp BackendResponse
		if err := json.NewDecoder(resp.Body).Decode(&backendResp); err != nil {
			data.Error = fmt.Sprintf("Failed to decode backend response: %v", err)
			tmpl.Execute(w, data)
			return
		}

		data.BackendResponse = &backendResp
		tmpl.Execute(w, data)
	})

	log.Printf("Frontend service starting on port %s", port)
	log.Printf("Backend URL configured as: %s", backendURL)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
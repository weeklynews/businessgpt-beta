package main

import (
	"fmt"
	"net/http"
	"os"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		html := `<!DOCTYPE html>
<html lang="ja">
<head>
    <meta charset="UTF-8">
    <title>BusinessGPT Beta</title>
    <style>
        body { 
            font-family: Arial, sans-serif; 
            text-align: center; 
            margin-top: 50px;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            min-height: 100vh;
        }
        .container {
            background: white;
            color: #333;
            padding: 2rem;
            border-radius: 15px;
            display: inline-block;
            margin-top: 100px;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>🚀 BusinessGPT Beta</h1>
        <p>サーバーが正常に稼働しています！</p>
        <p>まもなく完全版をリリースします</p>
    </div>
</body>
</html>`
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, html)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("🚀 BusinessGPT Beta starting on port %s\n", port)
	http.ListenAndServe(":"+port, nil)
}

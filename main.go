package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

var (
	db                *sql.DB
	store             *sessions.CookieStore
	googleOauthConfig *oauth2.Config
	oauthStateString  = "businessgpt-random-state"
)

type APIRequest struct {
	Message string `json:"message"`
	Model   string `json:"model"`
}

type APIResponse struct {
	Response string `json:"response"`
	Model    string `json:"model"`
	Tokens   int    `json:"tokens"`
}

type OpenAIRequest struct {
	Model       string              `json:"model"`
	Messages    []map[string]string `json:"messages"`
	Temperature float64             `json:"temperature"`
	MaxTokens   int                 `json:"max_tokens"`
}

type OpenAIResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		TotalTokens int `json:"total_tokens"`
	} `json:"usage"`
}

func init() {
	godotenv.Load()

	sessionKey := os.Getenv("SESSION_SECRET")
	if sessionKey == "" {
		sessionKey = "businessgpt-super-secret-key-2024"
	}
	store = sessions.NewCookieStore([]byte(sessionKey))

	googleOauthConfig = &oauth2.Config{
		ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		RedirectURL:  os.Getenv("BASE_URL") + "/auth/callback",
		Scopes:       []string{"email", "profile"},
		Endpoint:     google.Endpoint,
	}
}

func main() {
	initDB()
	defer db.Close()

	router := mux.NewRouter()

	router.HandleFunc("/", homeHandler).Methods("GET")
	router.HandleFunc("/auth/google", googleAuthHandler).Methods("GET")
	router.HandleFunc("/auth/callback", googleCallbackHandler).Methods("GET")
	router.HandleFunc("/chat", chatHandler).Methods("GET")
	router.HandleFunc("/api/chat", apiChatHandler).Methods("POST")
	router.HandleFunc("/logout", logoutHandler).Methods("GET")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("🚀 BusinessGPT Beta Server starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, router))
}

func initDB() {
	var err error
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	db, err = sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	if err = db.Ping(); err != nil {
		log.Fatal("Failed to ping database:", err)
	}

	createTables()
	log.Println("✅ Database connected successfully")
}

func createTables() {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS users (
            id SERIAL PRIMARY KEY,
            google_id VARCHAR(255) UNIQUE NOT NULL,
            email VARCHAR(255) UNIQUE NOT NULL,
            name VARCHAR(255) NOT NULL,
            picture VARCHAR(255),
            plan VARCHAR(50) DEFAULT 'trial',
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        )`,
		`CREATE TABLE IF NOT EXISTS chat_sessions (
            id SERIAL PRIMARY KEY,
            user_id INTEGER REFERENCES users(id),
            title VARCHAR(255),
            model VARCHAR(50) NOT NULL,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        )`,
		`CREATE TABLE IF NOT EXISTS chat_messages (
            id SERIAL PRIMARY KEY,
            session_id INTEGER REFERENCES chat_sessions(id),
            role VARCHAR(20) NOT NULL,
            content TEXT NOT NULL,
            model VARCHAR(50),
            tokens_used INTEGER,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        )`,
		`CREATE TABLE IF NOT EXISTS user_usage (
            id SERIAL PRIMARY KEY,
            user_id INTEGER REFERENCES users(id),
            date DATE NOT NULL,
            chat_count INTEGER DEFAULT 0,
            token_count INTEGER DEFAULT 0,
            UNIQUE(user_id, date)
        )`,
	}

	for _, query := range queries {
		if _, err := db.Exec(query); err != nil {
			log.Printf("Error creating table: %v", err)
		}
	}
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	tmpl := `<!DOCTYPE html>
<html lang="ja">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>BusinessGPT Beta</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
        }
        .container {
            background: white;
            padding: 3rem;
            border-radius: 20px;
            box-shadow: 0 20px 40px rgba(0,0,0,0.1);
            text-align: center;
            max-width: 500px;
            width: 90%;
        }
        h1 { color: #333; margin-bottom: 1rem; font-size: 2.5rem; }
        .subtitle { color: #666; margin-bottom: 2rem; font-size: 1.1rem; }
        .login-btn {
            background: #4285f4;
            color: white;
            padding: 1rem 2rem;
            border: none;
            border-radius: 10px;
            font-size: 1.1rem;
            cursor: pointer;
            text-decoration: none;
            display: inline-block;
            transition: background 0.3s;
        }
        .login-btn:hover { background: #3367d6; }
        .features { margin-top: 2rem; text-align: left; }
        .feature {
            margin: 1rem 0;
            padding: 1rem;
            background: #f8f9fa;
            border-radius: 8px;
        }
        .beta-badge {
            background: #ff6b6b;
            color: white;
            padding: 0.3rem 0.8rem;
            border-radius: 20px;
            font-size: 0.8rem;
            margin-left: 0.5rem;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>🚀 BusinessGPT <span class="beta-badge">BETA</span></h1>
        <p class="subtitle">マルチLLM対応のビジネスAIアシスタント</p>
        <a href="/auth/google" class="login-btn">Googleでログイン</a>
        <div class="features">
            <div class="feature">
                <strong>🤖 マルチモデル対応</strong><br>
                GPT-4o、Claude、Geminiを切り替えて使用
            </div>
            <div class="feature">
                <strong>💼 ビジネス特化</strong><br>
                企画書作成、メール作成、会議準備などに最適化
            </div>
            <div class="feature">
                <strong>📱 シンプルUI</strong><br>
                誰でも簡単に使える直感的なインターフェース
            </div>
        </div>
    </div>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(tmpl))
}

func googleAuthHandler(w http.ResponseWriter, r *http.Request) {
	url := googleOauthConfig.AuthCodeURL(oauthStateString)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func googleCallbackHandler(w http.ResponseWriter, r *http.Request) {
	state := r.FormValue("state")
	if state != oauthStateString {
		log.Printf("Invalid OAuth state: %s", state)
		http.Error(w, "Invalid state", http.StatusBadRequest)
		return
	}

	code := r.FormValue("code")
	token, err := googleOauthConfig.Exchange(context.Background(), code)
	if err != nil {
		log.Printf("Failed to exchange token: %v", err)
		http.Error(w, "Failed to exchange token", http.StatusInternalServerError)
		return
	}

	client := googleOauthConfig.Client(context.Background(), token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		log.Printf("Failed to get user info: %v", err)
		http.Error(w, "Failed to get user info", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	var userInfo struct {
		ID      string `json:"id"`
		Email   string `json:"email"`
		Name    string `json:"name"`
		Picture string `json:"picture"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		log.Printf("Failed to decode user info: %v", err)
		http.Error(w, "Failed to decode user info", http.StatusInternalServerError)
		return
	}

	userID, err := saveUser(userInfo.ID, userInfo.Email, userInfo.Name, userInfo.Picture)
	if err != nil {
		log.Printf("Failed to save user: %v", err)
		http.Error(w, "Failed to save user", http.StatusInternalServerError)
		return
	}

	session, _ := store.Get(r, "businessgpt-session")
	session.Values["user_id"] = userID
	session.Values["email"] = userInfo.Email
	session.Values["name"] = userInfo.Name
	session.Save(r, w)

	log.Printf("User logged in: %s (%d)", userInfo.Email, userID)
	http.Redirect(w, r, "/chat", http.StatusTemporaryRedirect)
}

func chatHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "businessgpt-session")
	userID, ok := session.Values["user_id"]
	if !ok {
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	userName, _ := session.Values["name"].(string)

	tmpl := `<!DOCTYPE html>
<html lang="ja">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>BusinessGPT - Chat</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            background: #f5f5f5;
            height: 100vh;
            display: flex;
            flex-direction: column;
        }
        .header {
            background: white;
            padding: 1rem 2rem;
            border-bottom: 1px solid #eee;
            display: flex;
            justify-content: space-between;
            align-items: center;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        .logo { font-size: 1.5rem; font-weight: bold; color: #333; }
        .beta-badge {
            background: #ff6b6b;
            color: white;
            padding: 0.2rem 0.6rem;
            border-radius: 12px;
            font-size: 0.7rem;
            margin-left: 0.5rem;
        }
        .user-info { display: flex; align-items: center; gap: 1rem; }
        .model-selector {
            padding: 0.5rem 1rem;
            border: 1px solid #ddd;
            border-radius: 8px;
            background: white;
            cursor: pointer;
        }
        .logout-btn {
            color: #666;
            text-decoration: none;
            padding: 0.5rem 1rem;
            border-radius: 6px;
            transition: background 0.3s;
        }
        .logout-btn:hover { background: #f0f0f0; }
        .chat-container {
            flex: 1;
            display: flex;
            flex-direction: column;
            max-width: 1000px;
            margin: 0 auto;
            width: 100%;
            padding: 2rem;
            gap: 2rem;
        }
        .messages {
            flex: 1;
            overflow-y: auto;
            padding: 2rem;
            background: white;
            border-radius: 12px;
            box-shadow: 0 4px 6px rgba(0,0,0,0.1);
            min-height: 400px;
        }
        .message {
            margin-bottom: 1.5rem;
            padding: 1rem 1.5rem;
            border-radius: 12px;
            max-width: 80%;
            word-wrap: break-word;
        }
        .message.user {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            margin-left: auto;
            border-bottom-right-radius: 4px;
        }
        .message.assistant {
            background: #f8f9fa;
            color: #333;
            border: 1px solid #e9ecef;
            border-bottom-left-radius: 4px;
            white-space: pre-wrap;
        }
        .message.assistant .model-tag {
            display: inline-block;
            background: #007bff;
            color: white;
            padding: 0.2rem 0.5rem;
            border-radius: 4px;
            font-size: 0.8rem;
            margin-bottom: 0.5rem;
        }
        .input-area {
            background: white;
            padding: 1.5rem;
            border-radius: 12px;
            box-shadow: 0 4px 6px rgba(0,0,0,0.1);
        }
        .input-controls {
            display: flex;
            gap: 1rem;
            margin-bottom: 1rem;
            font-size: 0.9rem;
            color: #666;
        }
        .input-row { display: flex; gap: 1rem; }
        .message-input {
            flex: 1;
            padding: 1rem 1.5rem;
            border: 2px solid #e9ecef;
            border-radius: 12px;
            resize: none;
            font-size: 16px;
            font-family: inherit;
            transition: border-color 0.3s;
        }
        .message-input:focus {
            outline: none;
            border-color: #667eea;
        }
        .send-button {
            padding: 1rem 2rem;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            border: none;
            border-radius: 12px;
            cursor: pointer;
            font-size: 1rem;
            transition: transform 0.2s;
        }
        .send-button:hover { transform: translateY(-2px); }
        .send-button:disabled {
            background: #ccc;
            cursor: not-allowed;
            transform: none;
        }
        .welcome { text-align: center; color: #666; margin: 2rem 0; }
        @media (max-width: 768px) {
            .chat-container { padding: 1rem; }
            .message { max-width: 95%; }
            .input-row { flex-direction: column; }
            .user-info { flex-direction: column; gap: 0.5rem; }
        }
    </style>
</head>
<body>
    <div class="header">
        <div class="logo">🚀 BusinessGPT <span class="beta-badge">BETA</span></div>
        <div class="user-info">
            <select class="model-selector" id="modelSelect">
                <option value="gpt-4o">GPT-4o</option>
                <option value="claude-3">Claude 3</option>
                <option value="gemini">Gemini 1.5</option>
            </select>
            <span>{{.UserName}}</span>
            <a href="/logout" class="logout-btn">ログアウト</a>
        </div>
    </div>
    <div class="chat-container">
        <div class="messages" id="messages">
            <div class="welcome">
                <h2>✨ BusinessGPT Beta へようこそ！</h2>
                <p>AIがあなたのビジネスをサポートします。何でもお気軽にご質問ください。</p>
            </div>
        </div>
        <div class="input-area">
            <div class="input-controls">
                <span>💡 例: 「新規事業の企画書を作って」「効果的なメールの書き方を教えて」</span>
            </div>
            <div class="input-row">
                <textarea 
                    class="message-input" 
                    id="messageInput" 
                    placeholder="メッセージを入力してください..."
                    rows="3"
                ></textarea>
                <button class="send-button" id="sendButton">
                    <span id="buttonText">送信</span>
                </button>
            </div>
        </div>
    </div>
    <script>
        const messagesDiv = document.getElementById('messages');
        const messageInput = document.getElementById('messageInput');
        const sendButton = document.getElementById('sendButton');
        const buttonText = document.getElementById('buttonText');
        const modelSelect = document.getElementById('modelSelect');

        sendButton.addEventListener('click', sendMessage);
        messageInput.addEventListener('keydown', (e) => {
            if (e.key === 'Enter' && !e.shiftKey) {
                e.preventDefault();
                sendMessage();
            }
        });

        async function sendMessage() {
            const message = messageInput.value.trim();
            if (!message || sendButton.disabled) return;

            const welcome = document.querySelector('.welcome');
            if (welcome) welcome.remove();

            addMessage(message, 'user');
            messageInput.value = '';
            
            sendButton.disabled = true;
            buttonText.textContent = '送信中...';

            try {
                const response = await fetch('/api/chat', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({
                        message: message,
                        model: modelSelect.value
                    })
                });

                const data = await response.json();
                
                if (response.ok) {
                    addMessage(data.response, 'assistant', data.model);
                } else {
                    addMessage('エラーが発生しました: ' + (data.error || 'Unknown error'), 'assistant');
                }
            } catch (error) {
                console.error('Chat error:', error);
                addMessage('ネットワークエラーが発生しました。もう一度お試しください。', 'assistant');
            }

            sendButton.disabled = false;
            buttonText.textContent = '送信';
            messageInput.focus();
        }

        function addMessage(content, role, model = '') {
            const messageDiv = document.createElement('div');
            messageDiv.className = `message ${role}`;
            
            if (role === 'assistant' && model) {
                messageDiv.innerHTML = `<div class="model-tag">${model.toUpperCase()}</div>${content}`;
            } else {
                messageDiv.textContent = content;
            }
            
            messagesDiv.appendChild(messageDiv);
            messagesDiv.scrollTop = messagesDiv.scrollHeight;
        }

        messageInput.focus();
    </script>
</body>
</html>`

	t, _ := template.New("chat").Parse(tmpl)
	data := struct {
		UserID   int
		UserName string
	}{
		UserID:   userID.(int),
		UserName: userName,
	}
	t.Execute(w, data)
}

func apiChatHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "businessgpt-session")
	userID, ok := session.Values["user_id"]
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req APIRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Invalid JSON: %v", err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if !checkUsageLimit(userID.(int)) {
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "使用制限に達しました。本日の制限: 50回",
		})
		return
	}

	response, tokens, err := callLLMAPI(req.Message, req.Model)
	if err != nil {
		log.Printf("LLM API error: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "AI APIでエラーが発生しました: " + err.Error(),
		})
		return
	}

	if err := recordUsage(userID.(int), req.Model, tokens); err != nil {
		log.Printf("Failed to record usage: %v", err)
	}

	if err := saveChatHistory(userID.(int), req.Message, response, req.Model, tokens); err != nil {
		log.Printf("Failed to save chat history: %v", err)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(APIResponse{
		Response: response,
		Model:    req.Model,
		Tokens:   tokens,
	})
}

func callLLMAPI(message, model string) (string, int, error) {
	switch model {
	case "gpt-4o":
		return callOpenAI(message)
	case "claude-3":
		return "Claude APIは開発中です。現在はテスト応答を返しています:\n\n" +
			"あなたの質問「" + message + "」について、Claude 3からの応答をシミュレートしています。", 100, nil
	case "gemini":
		return "Gemini APIは開発中です。現在はテスト応答を返しています:\n\n" +
			"あなたの質問「" + message + "」について、Gemini 1.5からの応答をシミュレートしています。", 100, nil
	default:
		return callOpenAI(message)
	}
}

func callOpenAI(message string) (string, int, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return "", 0, fmt.Errorf("OpenAI API key not configured")
	}

	reqBody := OpenAIRequest{
		Model: "gpt-4o",
		Messages: []map[string]string{
			{"role": "user", "content": message},
		},
		Temperature: 0.7,
		MaxTokens:   2000,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", 0, err
	}

	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", 0, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", 0, err
	}

	if resp.StatusCode != 200 {
		return "", 0, fmt.Errorf("OpenAI API error: %s", string(body))
	}

	var apiResp OpenAIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return "", 0, err
	}

	if len(apiResp.Choices) == 0 {
		return "", 0, fmt.Errorf("no response from OpenAI API")
	}

	return apiResp.Choices[0].Message.Content, apiResp.Usage.TotalTokens, nil
}

func saveUser(googleID, email, name, picture string) (int, error) {
	var userID int
	err := db.QueryRow(`
        INSERT INTO users (google_id, email, name, picture, plan)
        VALUES ($1, $2, $3, $4, 'trial')
        ON CONFLICT (google_id) 
        DO UPDATE SET 
            name = EXCLUDED.name,
            picture = EXCLUDED.picture,
            updated_at = CURRENT_TIMESTAMP
        RETURNING id
    `, googleID, email, name, picture).Scan(&userID)

	return userID, err
}

func saveChatHistory(userID int, userMessage, aiResponse, model string, tokens int) error {
	var sessionID int
	title := userMessage
	if len(title) > 50 {
		title = title[:47] + "..."
	}

	err := db.QueryRow(`
        INSERT INTO chat_sessions (user_id, title, model)
        VALUES ($1, $2, $3)
        RETURNING id
    `, userID, title, model).Scan(&sessionID)

	if err != nil {
		return err
	}

	_, err = db.Exec(`
        INSERT INTO chat_messages (session_id, role, content, model)
        VALUES ($1, 'user', $2, $3)
    `, sessionID, userMessage, model)

	if err != nil {
		return err
	}

	_, err = db.Exec(`
        INSERT INTO chat_messages (session_id, role, content, model, tokens_used)
        VALUES ($1, 'assistant', $2, $3, $4)
    `, sessionID, aiResponse, model, tokens)

	return err
}

func recordUsage(userID int, model string, tokens int) error {
	_, err := db.Exec(`
        INSERT INTO user_usage (user_id, date, chat_count, token_count)
        VALUES ($1, CURRENT_DATE, 1, $2)
        ON CONFLICT (user_id, date)
        DO UPDATE SET 
            chat_count = user_usage.chat_count + 1,
            token_count = user_usage.token_count + $2
    `, userID, tokens)

	return err
}

func checkUsageLimit(userID int) bool {
	var chatCount int
	err := db.QueryRow(`
        SELECT COALESCE(chat_count, 0)
        FROM user_usage 
        WHERE user_id = $1 AND date = CURRENT_DATE
    `, userID).Scan(&chatCount)

	if err != nil {
		return true
	}

	return chatCount < 50
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "businessgpt-session")
	session.Values = make(map[interface{}]interface{})
	session.Save(r, w)
	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}

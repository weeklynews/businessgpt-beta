# 🚀 BusinessGPT Beta

マルチLLM対応のビジネスAIアシスタント

## ✨ 機能

- 🤖 **マルチモデル対応**: GPT-4o、Claude 3、Gemini 1.5 Pro
- 🔐 **Google OAuth認証**: 簡単ログイン
- 💾 **チャット履歴保存**: PostgreSQL
- 📱 **レスポンシブデザイン**: PC・スマホ対応
- 🎯 **ビジネス特化**: 企画書、メール、会議準備に最適

## 🛠 技術スタック

- **Backend**: Go 1.21
- **Database**: PostgreSQL
- **Hosting**: Railway
- **Authentication**: OAuth 2.0 (Google)
- **Frontend**: HTML + Vanilla JavaScript

## 🚀 デプロイ

1. Railway で PostgreSQL データベースを作成
2. 環境変数を設定:
   - `GOOGLE_CLIENT_ID`
   - `GOOGLE_CLIENT_SECRET`
   - `OPENAI_API_KEY`
   - `SESSION_SECRET`
   - `BASE_URL`
3. このリポジトリを Railway にデプロイ

## 📝 ライセンス

MIT License

## 👥 開発者

BusinessGPT Beta Team
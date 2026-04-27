# quiz-generator

テキスト・ファイル・PowerPointスライドからAIが自動でクイズを生成するWebアプリ。

<table>
  <tr>
    <td align="center"><b>① テキスト入力</b><br>学習素材を貼り付けてクイズを生成</td>
    <td align="center"><b>② クイズ回答</b><br>4択・穴埋め問題をブラウザでプレイ</td>
    <td align="center"><b>③ 結果確認</b><br>スコアと解説で振り返り</td>
  </tr>
  <tr>
    <td><img src="docs/screenshot-top.png" width="260"/></td>
    <td><img src="docs/screenshot-quiz.png" width="260"/></td>
    <td><img src="docs/screenshot-result.png" width="260"/></td>
  </tr>
</table>

## 機能

- テキスト貼り付け / ファイルアップロード（.txt, .md, .pptx）からクイズを生成
- 4択問題・穴埋め選択問題・ミックス形式に対応
- 生成したクイズをブラウザ上でプレイ
- クイズライブラリで過去問を管理・エクスポート
- 制限時間付きモード

## 注意事項

- クイズはAIが自動生成するため、穴埋め問題の空欄位置や選択肢の内容が意図通りにならない場合があります
- 生成結果が不自然な場合は再生成してください

## セットアップ

### 必要なもの

- Go 1.23+
- [OpenAI APIキー](https://platform.openai.com/)

### ローカル起動

```bash
git clone https://github.com/takeshun256/quiz-generator.git
cd quiz-generator

cp .env.example .env
# .env を編集して OPENAI_API_KEY を設定

source .env
go run .
```

ブラウザで http://localhost:8080 を開く。

### Docker で起動

```bash
cp .env.example .env
# .env を編集して OPENAI_API_KEY を設定

docker compose up --build
```

## 環境変数

| 変数名 | 必須 | 説明 |
|--------|------|------|
| `OPENAI_API_KEY` | ✅ | OpenAI APIキー |
| `PORT` | - | サーバーポート（デフォルト: `8080`） |
| `DB_PATH` | - | SQLiteファイルパス（デフォルト: `./quiz.db`） |

## 技術スタック

- **Backend**: Go + [chi](https://github.com/go-chi/chi)
- **AI**: [OpenAI API](https://platform.openai.com/docs) (gpt-4o-mini)
- **DB**: SQLite ([modernc.org/sqlite](https://gitlab.com/cznic/sqlite))
- **Frontend**: HTML テンプレート + [htmx](https://htmx.org/) + [Tailwind CSS](https://tailwindcss.com/)

## ライセンス

MIT

package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

type GeneratedQuiz struct {
	Title     string              `json:"title"`
	Questions []GeneratedQuestion `json:"questions"`
}

type GeneratedQuestion struct {
	Type        string   `json:"type"`
	Category    string   `json:"category"`
	Question    string   `json:"question"`
	Options     []string `json:"options"`
	Correct     string   `json:"correct"`
	Explanation string   `json:"explanation"`
}

type QuizFormat string

const (
	FormatMultiple  QuizFormat = "multiple"
	FormatFillblank QuizFormat = "fillblank"
	FormatMix       QuizFormat = "mix"
)

func GenerateQuiz(ctx context.Context, sourceText string, count int, format QuizFormat, apiKey string, extraInstruction string) (*GeneratedQuiz, error) {
	formatInstruction := ""
	switch format {
	case FormatMultiple:
		formatInstruction = `全問 type="multiple"（4択問題）にしてください。`
	case FormatFillblank:
		formatInstruction = `全問 type="fillblank"（穴埋め選択問題）にしてください。問題文中の空欄は___(アンダースコア3つ)で表してください。空欄は必ず1問につき1つだけにしてください。`
	default:
		formatInstruction = `type="multiple"（4択）と type="fillblank"（穴埋め選択）を半々程度でミックスしてください。穴埋め問題の空欄は___(アンダースコア3つ)で表してください。空欄は必ず1問につき1つだけにしてください。`
	}

	extraSection := ""
	if extraInstruction != "" {
		extraSection = fmt.Sprintf("\n追加指示:\n%s\n", extraInstruction)
	}

	prompt := fmt.Sprintf(`以下のソーステキストを読んで、学習用クイズを%d問生成してください。

%s
%s
出力は以下のJSONのみを返してください（説明文・マークダウン不要）:
{
  "title": "クイズのタイトル（ソースの内容を反映した短いタイトル）",
  "questions": [
    {
      "type": "multiple",
      "category": "トピック名（例: トランザクション・インデックス・パフォーマンス など、ソース内容に合わせた短いカテゴリ名）",
      "question": "問題文",
      "options": ["選択肢A", "選択肢B", "選択肢C", "選択肢D"],
      "correct": "選択肢A",
      "explanation": "なぜこれが正解かの解説（2〜3文）"
    }
  ]
}

重要なルール:
- options は必ず4つ
- correct は options のいずれかと完全一致する文字列
- explanation は必ず含める
- category は各問題のトピックを表す短いラベル（10文字以内）
- 問題は互いに重複しないようにする
- ソーステキスト全体のメインテーマに基づいた問題のみ生成する
- 自己紹介・目次・アジェンダ・謝辞など学習内容と無関係なスライドは無視する
- メインテーマから大きく外れるトピックは問題に含めない

ソーステキスト:
%s`, count, formatInstruction, extraSection, sourceText)

	opts := []option.RequestOption{}
	if apiKey != "" {
		opts = append(opts, option.WithAPIKey(apiKey))
	}
	client := openai.NewClient(opts...)

	resp, err := client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model: openai.ChatModelGPT4oMini,
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(prompt),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("AI API error: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("APIからの応答が空でした")
	}
	raw := resp.Choices[0].Message.Content

	// マークダウンコードブロックが混入した場合に対応
	if idx := strings.Index(raw, "{"); idx >= 0 {
		raw = raw[idx:]
	}
	if idx := strings.LastIndex(raw, "}"); idx >= 0 {
		raw = raw[:idx+1]
	}

	var quiz GeneratedQuiz
	if err := json.Unmarshal([]byte(raw), &quiz); err != nil {
		return nil, fmt.Errorf("クイズを生成できませんでした。\n\nAIからのメッセージ:\n%s", raw)
	}
	if len(quiz.Questions) == 0 {
		return nil, fmt.Errorf("問題が生成されませんでした。ソーステキストに十分な学習内容が含まれているか確認してください。")
	}
	return &quiz, nil
}

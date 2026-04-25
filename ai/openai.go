package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

type GeneratedQuiz struct {
	Title     string              `json:"title"`
	Questions []GeneratedQuestion `json:"questions"`
}

type GeneratedQuestion struct {
	Type        string   `json:"type"`
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

// claudeOutput は `claude --output-format json` の戻り値構造
type claudeOutput struct {
	Result string `json:"result"`
}

func GenerateQuiz(ctx context.Context, sourceText string, count int, format QuizFormat) (*GeneratedQuiz, error) {
	formatInstruction := ""
	switch format {
	case FormatMultiple:
		formatInstruction = `全問 type="multiple"（4択問題）にしてください。`
	case FormatFillblank:
		formatInstruction = `全問 type="fillblank"（穴埋め選択問題）にしてください。問題文中の空欄は___(アンダースコア3つ)で表してください。`
	default:
		formatInstruction = `type="multiple"（4択）と type="fillblank"（穴埋め選択）を半々程度でミックスしてください。穴埋め問題の空欄は___(アンダースコア3つ)で表してください。`
	}

	prompt := fmt.Sprintf(`以下のソーステキストを読んで、学習用クイズを%d問生成してください。

%s

出力は以下のJSONのみを返してください（説明文・マークダウン不要）:
{
  "title": "クイズのタイトル（ソースの内容を反映した短いタイトル）",
  "questions": [
    {
      "type": "multiple",
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
- 問題は互いに重複しないようにする
- ソーステキストの内容に基づいた問題のみ生成する

ソーステキスト:
%s`, count, formatInstruction, sourceText)

	cmd := exec.CommandContext(ctx, "claude", "--bare", "-p", prompt, "--output-format", "json")
	out, err := cmd.Output()
	if err != nil {
		stderr := ""
		if ee, ok := err.(*exec.ExitError); ok {
			stderr = string(ee.Stderr)
		}
		return nil, fmt.Errorf("claude command failed: %w\nstderr: %s", err, stderr)
	}

	var response claudeOutput
	if err := json.Unmarshal(out, &response); err != nil {
		// --bare モードでは result フィールドなしに直接テキストが返る場合もあるためフォールバック
		response.Result = strings.TrimSpace(string(out))
	}

	// result の中の JSON を抽出（マークダウンコードブロックが混入した場合に対応）
	raw := response.Result
	if idx := strings.Index(raw, "{"); idx >= 0 {
		raw = raw[idx:]
	}
	if idx := strings.LastIndex(raw, "}"); idx >= 0 {
		raw = raw[:idx+1]
	}

	var quiz GeneratedQuiz
	if err := json.Unmarshal([]byte(raw), &quiz); err != nil {
		return nil, fmt.Errorf("failed to parse quiz JSON: %w\nraw: %s", err, raw)
	}
	if len(quiz.Questions) == 0 {
		return nil, fmt.Errorf("no questions generated")
	}
	return &quiz, nil
}

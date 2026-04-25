package handlers

import (
	"archive/zip"
	"bytes"
	"context"
	"database/sql"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/takeshun256/quiz-generator/ai"
	"github.com/takeshun256/quiz-generator/models"
)

type Job struct {
	ID        string
	Status    string // "running" | "done" | "error"
	QuizSetID int64
	Error     string
	Done      chan struct{}
}

var (
	jobs   = map[string]*Job{}
	jobsMu sync.Mutex
)

func newJobID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func IndexHandler(tmpl *Renderer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tmpl.ExecuteTemplate(w, "index.html", nil)
	}
}

func GenerateHandler(tmpl *Renderer, db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			r.ParseForm()
		}

		var sourceText string

		// テキスト貼付
		if t := r.FormValue("source_text"); t != "" {
			sourceText = t
		}

		// ファイルアップロード
		if f, fh, err := r.FormFile("source_file"); err == nil {
			defer f.Close()
			b, err := io.ReadAll(f)
			if err == nil {
				ext := strings.ToLower(filepath.Ext(fh.Filename))
				if ext == ".pptx" {
					text, err := extractPPTX(b)
					if err != nil {
						http.Error(w, "PPTXの読み込みに失敗しました: "+err.Error(), http.StatusBadRequest)
						return
					}
					sourceText = text
				} else {
					sourceText = string(b)
				}
			}
		}

		// Obsidianパス
		if p := r.FormValue("source_path"); p != "" {
			b, err := os.ReadFile(p)
			if err != nil {
				http.Error(w, "ファイルを読み込めませんでした: "+err.Error(), http.StatusBadRequest)
				return
			}
			sourceText = string(b)
		}

		if sourceText == "" {
			http.Error(w, "ソーステキストを入力してください", http.StatusBadRequest)
			return
		}

		count, _ := strconv.Atoi(r.FormValue("count"))
		if count <= 0 || count > 30 {
			count = 10
		}
		format := ai.QuizFormat(r.FormValue("format"))
		if format == "" {
			format = ai.FormatMix
		}
		timeLimit, _ := strconv.Atoi(r.FormValue("time_limit"))
		if timeLimit < 0 {
			timeLimit = 0
		}

		jobID := newJobID()
		job := &Job{ID: jobID, Status: "running", Done: make(chan struct{})}
		jobsMu.Lock()
		jobs[jobID] = job
		jobsMu.Unlock()

		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()

			quiz, err := ai.GenerateQuiz(ctx, sourceText, count, format)
			if err != nil {
				log.Printf("GenerateQuiz error: %v", err)
				jobsMu.Lock()
				job.Status = "error"
				job.Error = err.Error()
				jobsMu.Unlock()
				close(job.Done)
				return
			}

			quizSetID, err := models.CreateQuizSet(db, quiz.Title, sourceText, timeLimit)
			if err != nil {
				log.Printf("CreateQuizSet error: %v", err)
				jobsMu.Lock()
				job.Status = "error"
				job.Error = err.Error()
				jobsMu.Unlock()
				close(job.Done)
				return
			}

			for i, q := range quiz.Questions {
				err := models.CreateQuestion(db, models.Question{
					QuizSetID:   quizSetID,
					Type:        q.Type,
					Question:    q.Question,
					Options:     q.Options,
					Correct:     q.Correct,
					Explanation: q.Explanation,
					Position:    i + 1,
				})
				if err != nil {
					log.Printf("CreateQuestion error: %v", err)
				}
			}

			jobsMu.Lock()
			job.Status = "done"
			job.QuizSetID = quizSetID
			jobsMu.Unlock()
			close(job.Done)
		}()

		http.Redirect(w, r, "/generating/"+jobID, http.StatusSeeOther)
	}
}

func GeneratingPageHandler(tmpl *Renderer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		jobID := chi.URLParam(r, "jobId")
		tmpl.ExecuteTemplate(w, "generating.html", map[string]any{"JobID": jobID})
	}
}

func GeneratingStatusHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		jobID := chi.URLParam(r, "jobId")
		jobsMu.Lock()
		job, ok := jobs[jobID]
		jobsMu.Unlock()

		if !ok {
			http.Error(w, "job not found", http.StatusNotFound)
			return
		}

		switch job.Status {
		case "done":
			w.Header().Set("HX-Redirect", fmt.Sprintf("/quiz/%d/start", job.QuizSetID))
			w.WriteHeader(http.StatusOK)
		case "error":
			w.Write([]byte(fmt.Sprintf(`<div class="text-center py-16 text-red-400">
				<div class="text-4xl mb-4">❌</div>
				<p>生成に失敗しました</p>
				<p class="text-sm mt-2">%s</p>
				<a href="/" class="mt-4 inline-block text-indigo-400 hover:underline">やり直す</a>
			</div>`, job.Error)))
		default:
			// 生成中: htmx に何もスワップさせない（204 = No Content）
			w.WriteHeader(http.StatusNoContent)
		}
	}
}

// extractPPTX はPPTXファイルのバイト列からテキストを抽出する。
// PPTXはZIPアーカイブで、ppt/slides/slide*.xml にスライドテキストが含まれる。
func extractPPTX(data []byte) (string, error) {
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", fmt.Errorf("ZIPとして開けません: %w", err)
	}

	var sb strings.Builder
	for _, f := range r.File {
		// スライドXMLのみ対象
		if !strings.HasPrefix(f.Name, "ppt/slides/slide") || !strings.HasSuffix(f.Name, ".xml") {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			continue
		}
		text, err := extractXMLText(rc)
		rc.Close()
		if err != nil {
			continue
		}
		if text != "" {
			sb.WriteString(text)
			sb.WriteString("\n\n")
		}
	}

	result := strings.TrimSpace(sb.String())
	if result == "" {
		return "", fmt.Errorf("テキストが見つかりませんでした")
	}
	return result, nil
}

// extractXMLText はXMLリーダーから <a:t> 要素のテキストを抽出する。
func extractXMLText(r io.Reader) (string, error) {
	var sb strings.Builder
	dec := xml.NewDecoder(r)
	inText := false
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return sb.String(), nil
		}
		switch t := tok.(type) {
		case xml.StartElement:
			// <a:t> はDrawingML のテキストラン要素
			if t.Name.Local == "t" {
				inText = true
			}
		case xml.EndElement:
			if t.Name.Local == "t" {
				inText = false
				sb.WriteString(" ")
			}
		case xml.CharData:
			if inText {
				sb.Write(t)
			}
		}
	}
	return strings.TrimSpace(sb.String()), nil
}

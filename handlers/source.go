package handlers

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
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
		if f, _, err := r.FormFile("source_file"); err == nil {
			defer f.Close()
			b, err := io.ReadAll(f)
			if err == nil {
				sourceText = string(b)
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

			quizSetID, err := models.CreateQuizSet(db, quiz.Title, sourceText)
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
			w.Header().Set("HX-Redirect", fmt.Sprintf("/quiz/%d", job.QuizSetID))
			w.WriteHeader(http.StatusOK)
		case "error":
			w.Write([]byte(fmt.Sprintf(`<div class="text-center py-16 text-red-400">
				<div class="text-4xl mb-4">❌</div>
				<p>生成に失敗しました</p>
				<p class="text-sm mt-2">%s</p>
				<a href="/" class="mt-4 inline-block text-indigo-400 hover:underline">やり直す</a>
			</div>`, job.Error)))
		default:
			w.WriteHeader(http.StatusOK)
		}
	}
}

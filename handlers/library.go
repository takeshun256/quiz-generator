package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/takeshun256/quiz-generator/models"
)

type AttemptView struct {
	models.Attempt
	Percent int
}

type QuizSetWithAttempts struct {
	models.QuizSet
	Attempts []AttemptView
}

func LibraryHandler(tmpl *Renderer, db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sets, err := models.ListQuizSets(db)
		if err != nil {
			http.Error(w, "DB error", http.StatusInternalServerError)
			return
		}
		var items []QuizSetWithAttempts
		for _, s := range sets {
			raw, _ := models.GetAttempts(db, s.ID)
			var avs []AttemptView
			for _, a := range raw {
				pct := 0
				if a.Total > 0 {
					pct = a.Score * 100 / a.Total
				}
				avs = append(avs, AttemptView{Attempt: a, Percent: pct})
			}
			items = append(items, QuizSetWithAttempts{QuizSet: s, Attempts: avs})
		}
		tmpl.ExecuteTemplate(w, "library.html", map[string]any{
			"Items": items,
		})
	}
}

func LibraryDeleteHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		models.DeleteQuizSet(db, id)
		w.WriteHeader(http.StatusOK)
	}
}

type ExportQuestion struct {
	Type        string   `json:"type"`
	Question    string   `json:"question"`
	Options     []string `json:"options"`
	Correct     string   `json:"correct"`
	Explanation string   `json:"explanation"`
}

type ExportData struct {
	Title     string           `json:"title"`
	CreatedAt time.Time        `json:"created_at"`
	Questions []ExportQuestion `json:"questions"`
}

func LibraryExportHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)

		quizSet, err := models.GetQuizSet(db, id)
		if err != nil {
			http.Error(w, "quiz not found", http.StatusNotFound)
			return
		}

		questions, err := models.GetQuestions(db, id)
		if err != nil {
			http.Error(w, "questions not found", http.StatusInternalServerError)
			return
		}

		var eqs []ExportQuestion
		for _, q := range questions {
			eqs = append(eqs, ExportQuestion{
				Type:        q.Type,
				Question:    q.Question,
				Options:     q.Options,
				Correct:     q.Correct,
				Explanation: q.Explanation,
			})
		}

		data := ExportData{
			Title:     quizSet.Title,
			CreatedAt: quizSet.CreatedAt,
			Questions: eqs,
		}

		b, _ := json.MarshalIndent(data, "", "  ")
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="quiz-%d.json"`, id))
		w.Write(b)
	}
}

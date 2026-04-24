package handlers

import (
	"database/sql"
	"html/template"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/takeshun256/quiz-generator/models"
)

func QuizPlayHandler(tmpl *template.Template, db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		quizSetID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		if err != nil {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}

		quizSet, err := models.GetQuizSet(db, quizSetID)
		if err != nil {
			http.Error(w, "quiz not found", http.StatusNotFound)
			return
		}

		question, err := models.GetQuestion(db, quizSetID, 1)
		if err != nil {
			http.Error(w, "question not found", http.StatusNotFound)
			return
		}

		total := quizSet.Count
		tmpl.ExecuteTemplate(w, "quiz.html", map[string]any{
			"QuizSet":  quizSet,
			"Question": question,
			"Current":  1,
			"Total":    total,
			"Progress": 100 / total,
		})
	}
}

func QuizQuestionHandler(tmpl *template.Template, db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		quizSetID, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		position, _ := strconv.Atoi(r.URL.Query().Get("position"))

		quizSet, err := models.GetQuizSet(db, quizSetID)
		if err != nil {
			http.Error(w, "quiz not found", http.StatusNotFound)
			return
		}

		question, err := models.GetQuestion(db, quizSetID, position)
		if err != nil {
			http.Error(w, "question not found", http.StatusNotFound)
			return
		}

		total := quizSet.Count
		tmpl.ExecuteTemplate(w, "question-partial", map[string]any{
			"QuizSet":  quizSet,
			"Question": question,
			"Current":  position,
			"Total":    total,
			"Progress": position * 100 / total,
		})
	}
}

func QuizAnswerHandler(tmpl *template.Template, db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		quizSetID, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		r.ParseForm()

		questionID, _ := strconv.ParseInt(r.FormValue("question_id"), 10, 64)
		chosen := r.FormValue("chosen")
		position, _ := strconv.Atoi(r.FormValue("position"))

		quizSet, err := models.GetQuizSet(db, quizSetID)
		if err != nil {
			http.Error(w, "quiz not found", http.StatusNotFound)
			return
		}

		question, err := models.GetQuestion(db, quizSetID, position)
		if err != nil {
			http.Error(w, "question not found", http.StatusNotFound)
			return
		}

		isCorrect := chosen == question.Correct
		models.SaveAnswer(db, quizSetID, questionID, chosen, isCorrect)

		total := quizSet.Count
		isLast := position >= total
		next := position + 1

		tmpl.ExecuteTemplate(w, "answer-partial", map[string]any{
			"QuizSet":  quizSet,
			"Question": question,
			"Chosen":   chosen,
			"IsLast":   isLast,
			"Next":     next,
			"Current":  position,
			"Total":    total,
		})
	}
}

type ResultItem struct {
	Question  models.Question
	Chosen    string
	IsCorrect bool
}

func QuizResultHandler(tmpl *template.Template, db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		quizSetID, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)

		quizSet, err := models.GetQuizSet(db, quizSetID)
		if err != nil {
			http.Error(w, "quiz not found", http.StatusNotFound)
			return
		}

		questions, err := models.GetQuestions(db, quizSetID)
		if err != nil {
			http.Error(w, "questions not found", http.StatusNotFound)
			return
		}

		answers, err := models.GetAnswers(db, quizSetID)
		if err != nil {
			answers = map[int64]string{}
		}

		var items []ResultItem
		score := 0
		for _, q := range questions {
			chosen := answers[q.ID]
			isCorrect := chosen == q.Correct
			if isCorrect {
				score++
			}
			items = append(items, ResultItem{Question: q, Chosen: chosen, IsCorrect: isCorrect})
		}

		total := len(questions)
		scorePercent := 0
		if total > 0 {
			scorePercent = score * 100 / total
		}

		tmpl.ExecuteTemplate(w, "result.html", map[string]any{
			"QuizSet":      quizSet,
			"Items":        items,
			"Score":        score,
			"Total":        total,
			"ScorePercent": scorePercent,
		})
	}
}

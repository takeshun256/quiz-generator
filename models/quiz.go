package models

import (
	"database/sql"
	"encoding/json"
	"time"
)

type QuizSet struct {
	ID         int64
	Title      string
	SourceText string
	TimeLimit  int // seconds per question, 0 = no limit
	CreatedAt  time.Time
	Count      int
}

type Question struct {
	ID          int64
	QuizSetID   int64
	Type        string
	Question    string
	Options     []string
	Correct     string
	Explanation string
	Position    int
}

type Attempt struct {
	ID         int64
	QuizSetID  int64
	Score      int
	Total      int
	FinishedAt time.Time
}

func CreateQuizSet(db *sql.DB, title, sourceText string, timeLimit int) (int64, error) {
	res, err := db.Exec(
		`INSERT INTO quiz_sets (title, source_text, time_limit) VALUES (?, ?, ?)`,
		title, sourceText, timeLimit,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func CreateQuestion(db *sql.DB, q Question) error {
	opts, err := json.Marshal(q.Options)
	if err != nil {
		return err
	}
	_, err = db.Exec(
		`INSERT INTO questions (quiz_set_id, type, question, options, correct, explanation, position)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		q.QuizSetID, q.Type, q.Question, string(opts), q.Correct, q.Explanation, q.Position,
	)
	return err
}

func GetQuestion(db *sql.DB, quizSetID int64, position int) (*Question, error) {
	row := db.QueryRow(
		`SELECT id, quiz_set_id, type, question, options, correct, explanation, position
		 FROM questions WHERE quiz_set_id = ? AND position = ?`,
		quizSetID, position,
	)
	return scanQuestion(row)
}

func GetQuestions(db *sql.DB, quizSetID int64) ([]Question, error) {
	rows, err := db.Query(
		`SELECT id, quiz_set_id, type, question, options, correct, explanation, position
		 FROM questions WHERE quiz_set_id = ? ORDER BY position`,
		quizSetID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var qs []Question
	for rows.Next() {
		q, err := scanQuestion(rows)
		if err != nil {
			return nil, err
		}
		qs = append(qs, *q)
	}
	return qs, nil
}

func CountQuestions(db *sql.DB, quizSetID int64) (int, error) {
	var n int
	err := db.QueryRow(`SELECT COUNT(*) FROM questions WHERE quiz_set_id = ?`, quizSetID).Scan(&n)
	return n, err
}

func GetQuizSet(db *sql.DB, id int64) (*QuizSet, error) {
	row := db.QueryRow(
		`SELECT qs.id, qs.title, qs.source_text, qs.time_limit, qs.created_at, COUNT(q.id)
		 FROM quiz_sets qs LEFT JOIN questions q ON q.quiz_set_id = qs.id
		 WHERE qs.id = ? GROUP BY qs.id`,
		id,
	)
	var s QuizSet
	err := row.Scan(&s.ID, &s.Title, &s.SourceText, &s.TimeLimit, &s.CreatedAt, &s.Count)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func ListQuizSets(db *sql.DB) ([]QuizSet, error) {
	rows, err := db.Query(
		`SELECT qs.id, qs.title, qs.source_text, qs.time_limit, qs.created_at, COUNT(q.id)
		 FROM quiz_sets qs LEFT JOIN questions q ON q.quiz_set_id = qs.id
		 GROUP BY qs.id ORDER BY qs.created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var sets []QuizSet
	for rows.Next() {
		var s QuizSet
		if err := rows.Scan(&s.ID, &s.Title, &s.SourceText, &s.TimeLimit, &s.CreatedAt, &s.Count); err != nil {
			return nil, err
		}
		sets = append(sets, s)
	}
	return sets, nil
}

func DeleteQuizSet(db *sql.DB, id int64) error {
	_, err := db.Exec(`DELETE FROM quiz_sets WHERE id = ?`, id)
	return err
}

func SaveAnswer(db *sql.DB, quizSetID, questionID int64, chosen string, isCorrect bool) error {
	correct := 0
	if isCorrect {
		correct = 1
	}
	_, err := db.Exec(
		`INSERT INTO answers (quiz_set_id, question_id, chosen, is_correct) VALUES (?, ?, ?, ?)`,
		quizSetID, questionID, chosen, correct,
	)
	return err
}

func GetAnswers(db *sql.DB, quizSetID int64) (map[int64]string, error) {
	rows, err := db.Query(
		`SELECT question_id, chosen FROM answers WHERE quiz_set_id = ? ORDER BY answered_at`,
		quizSetID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	m := make(map[int64]string)
	for rows.Next() {
		var qid int64
		var chosen string
		if err := rows.Scan(&qid, &chosen); err != nil {
			return nil, err
		}
		m[qid] = chosen
	}
	return m, nil
}

func SaveAttempt(db *sql.DB, quizSetID int64, score, total int) error {
	_, err := db.Exec(
		`INSERT INTO attempts (quiz_set_id, score, total) VALUES (?, ?, ?)`,
		quizSetID, score, total,
	)
	return err
}

func GetAttempts(db *sql.DB, quizSetID int64) ([]Attempt, error) {
	rows, err := db.Query(
		`SELECT id, quiz_set_id, score, total, finished_at
		 FROM attempts WHERE quiz_set_id = ? ORDER BY finished_at DESC LIMIT 10`,
		quizSetID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var attempts []Attempt
	for rows.Next() {
		var a Attempt
		if err := rows.Scan(&a.ID, &a.QuizSetID, &a.Score, &a.Total, &a.FinishedAt); err != nil {
			return nil, err
		}
		attempts = append(attempts, a)
	}
	return attempts, nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanQuestion(s scanner) (*Question, error) {
	var q Question
	var opts string
	err := s.Scan(&q.ID, &q.QuizSetID, &q.Type, &q.Question, &opts, &q.Correct, &q.Explanation, &q.Position)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(opts), &q.Options); err != nil {
		return nil, err
	}
	return &q, nil
}

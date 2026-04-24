package main

import (
	"html/template"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/takeshun256/quiz-generator/handlers"
	"github.com/takeshun256/quiz-generator/models"
)

func main() {
	// DB初期化
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./quiz.db"
	}
	db, err := models.InitDB(dbPath)
	if err != nil {
		log.Fatalf("DB init error: %v", err)
	}
	defer db.Close()

	// テンプレート読み込み
	tmpl, err := template.ParseGlob("templates/*.html")
	if err != nil {
		log.Fatalf("template parse error: %v", err)
	}

	// ルーター
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// 静的ファイル
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// ルート
	r.Get("/", handlers.IndexHandler(tmpl))
	r.Post("/generate", handlers.GenerateHandler(tmpl, db))
	r.Get("/generating/{jobId}", handlers.GeneratingPageHandler(tmpl))
	r.Get("/generating/{jobId}/status", handlers.GeneratingStatusHandler())

	r.Get("/quiz/{id}", handlers.QuizPlayHandler(tmpl, db))
	r.Get("/quiz/{id}/question", handlers.QuizQuestionHandler(tmpl, db))
	r.Post("/quiz/{id}/answer", handlers.QuizAnswerHandler(tmpl, db))
	r.Get("/quiz/{id}/result", handlers.QuizResultHandler(tmpl, db))

	r.Get("/library", handlers.LibraryHandler(tmpl, db))
	r.Delete("/library/{id}", handlers.LibraryDeleteHandler(db))
	r.Get("/library/{id}/export", handlers.LibraryExportHandler(db))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("🚀 Starting server on http://localhost:%s", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatal(err)
	}
}

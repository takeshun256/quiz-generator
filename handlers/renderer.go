package handlers

import (
	"html/template"
	"io"
)

// Renderer はリクエストごとに base.html + ページテンプレートをパースし、
// Go template の named block 上書き問題を回避する。
type Renderer struct {
	Dir string // templates ディレクトリのパス
}

func (r *Renderer) ExecuteTemplate(w io.Writer, name string, data any) error {
	var tmpl *template.Template
	var err error

	switch name {
	case "question-partial", "answer-partial":
		// パーシャルは base.html 不要、quiz.html のみ
		tmpl, err = template.ParseFiles(r.Dir + "/quiz.html")
		if err != nil {
			return err
		}
		return tmpl.ExecuteTemplate(w, name, data)
	default:
		// フルページ: base.html + 対象ページをセットでパース
		tmpl, err = template.ParseFiles(r.Dir+"/base.html", r.Dir+"/"+name)
		if err != nil {
			return err
		}
		return tmpl.ExecuteTemplate(w, "base.html", data)
	}
}

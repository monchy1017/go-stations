package router

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"

	"github.com/TechBowl-japan/go-stations/handler"
	"github.com/TechBowl-japan/go-stations/handler/middleware"
	"github.com/TechBowl-japan/go-stations/service"
)

type Contextkey string

const OSContextKey Contextkey = "os"

func NewRouter(todoDB *sql.DB) *http.ServeMux {
	// register routes
	mux := http.NewServeMux()

	// Basic認証情報を取得
	basicAuthUserID := os.Getenv("BASIC_AUTH_USER_ID")
	basicAuthPassword := os.Getenv("BASIC_AUTH_PASSWORD")

	// /healthzエンドポイントを登録
	healthzHnadler := &handler.HealthzHandler{}
	mux.Handle("/healthz", healthzHnadler)

	// TODOService インスタンスを作成
	todoService := &service.TODOService{DB: todoDB}

	// /todos エンドポイントを登録(ミドルウェアでBasic認証を追加, station04)
	todoHandler := &handler.TODOHandler{SVC: todoService}
	mux.Handle("/todos", middleware.BasicAuth(basicAuthUserID, basicAuthPassword, todoHandler))

	// /do-panic エンドポイントを登録
	mux.HandleFunc("/do-panic", middleware.PanicHandler)

	// /test-os エンドポイントを登録
	mux.HandleFunc("/test-os", func(w http.ResponseWriter, r *http.Request) {
		// ミドルウェアで追加されたOS情報をコンテキストから取得
		osInfo := r.Context().Value(middleware.OSContextKey)
		if osInfo == nil {
			http.Error(w, "OS information not found in context", http.StatusInternalServerError)
			return
		}

		// 取得したOS情報をレスポンスに含める
		userAgent := r.UserAgent()
		fmt.Fprintf(w, "User-Agent: %s\nDetected OS: %s", userAgent, osInfo)
	})

	return mux
}

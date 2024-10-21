package middleware

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/mileusna/useragent"
)

// WithValueを使うためのキーの型(衝突を避けるために型を定義)
type Contextkey struct{}

var OSContextKey = &Contextkey{}

type LogData struct {
	Timestamp time.Time `json:"timestamp"`
	Latency   int64     `json:"latency"` // ミリ秒単位で処理時間を記録
	Path      string    `json:"path"`    // リクエストのURLパス
	OS        string    `json:"os"`      // OS情報
}

func Recovery(h http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		// TODO: ここに実装をする
		defer func() {
			// panicが発生した時にrecoverする
			if err := recover(); err != nil {
				log.Printf("panic: %v", err)
				if w.Header().Get("Content-Type") == "" {
					w.Header().Set("Content-Type", "text/plain; charset=utf-8")
				}
				w.WriteHeader(http.StatusInternalServerError)

				// エラーメッセージをレスポンスに書き込む
				_, writeErr := w.Write([]byte("Internal Server Error"))
				if writeErr != nil {
					log.Printf("write error: %v", writeErr)
				}
			}
		}()
		// hからServeHTTPを呼び出してhttpリクエストをchainさせる
		h.ServeHTTP(w, r)
	}
	return http.HandlerFunc(fn)
}

func PanicHandler(w http.ResponseWriter, r *http.Request) {
	panic("Panic!")
}

func AddOSContext(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ua := useragent.Parse(r.UserAgent())
		os := ua.OS
		ctx := context.WithValue(r.Context(), OSContextKey, os)
		// fmt.Printf("os: %v\n", os)
		h.ServeHTTP(w, r.WithContext(ctx))
	})
}

func LoggingMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()
		h.ServeHTTP(w, r)
		endTime := time.Now()
		latency := endTime.Sub(startTime).Microseconds()
		path := r.URL.Path

		osInfo := r.Context().Value(OSContextKey)
		if osInfo == nil {
			osInfo = "unknown OS"
		}

		logData := LogData{
			Timestamp: startTime,
			Latency:   latency,
			Path:      path,
			OS:        osInfo.(string),
		}

		logDataJSON, err := json.Marshal(logData)
		if err != nil {
			log.Printf("failed to marshal log data: %v", err)
			return
		}
		log.Println(string(logDataJSON))
	})
}

func BasicAuth(validUser, validPassword string, h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// BasicAuthメソッドを使ってユーザー名とパスワードを取得
		user, password, ok := r.BasicAuth()
		if !ok || user != validUser || password != validPassword {
			//認証に失敗したらHTTPステータスコード401を返す
			w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		// 認証に成功したら次のハンドラを呼び出す
		h.ServeHTTP(w, r)
	})
}

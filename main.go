package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/TechBowl-japan/go-stations/db"
	"github.com/TechBowl-japan/go-stations/handler/middleware"
	"github.com/TechBowl-japan/go-stations/handler/router"
	"github.com/joho/godotenv"
)

func main() {
	err := realMain()
	if err != nil {
		log.Fatalln("main: failed to exit successfully, err =", err)
	}
}

func realMain() error {
	loadErr := godotenv.Load()
	if loadErr != nil {
		log.Println("failed to load .env file")
	}

	// config values
	const (
		defaultPort   = ":8080"
		defaultDBPath = ".sqlite3/todo.db"
	)

	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = defaultDBPath
	}

	// set time zone
	var err error
	time.Local, err = time.LoadLocation("Asia/Tokyo")
	if err != nil {
		return err
	}

	// 環境変数からUserIDとPasswordを取得
	userID := os.Getenv("BASIC_AUTH_USER_ID")
	password := os.Getenv("BASIC_AUTH_PASSWORD")
	if userID == "" || password == "" {
		log.Fatalln("BASIC_AUTH_USER_ID and BASIC_AUTH_PASSWORD are required")
	}

	// set up sqlite3
	todoDB, err := db.NewDB(dbPath)
	if err != nil {
		return err
	}
	defer todoDB.Close()

	// NOTE: 新しいエンドポイントの登録はrouter.NewRouterの内部で行うようにする
	mux := router.NewRouter(todoDB)

	// サーバーに渡すmuxを、Recoveryミドルウェアでラップする(station01)
	// AddOSContextミドルウェアでラップして、OS情報をリクエストコンテキストに格納する(station02)
	// LoggingMiddlewareミドルウェアでラップして、リクエストのログを出力する(station03)
	wrappedMux := middleware.AddOSContext(middleware.LoggingMiddleware(middleware.Recovery(mux)))

	//以下 graceful shutdownのためのコード

	server := &http.Server{
		Addr:    port,
		Handler: wrappedMux,
	}

	// シグナルハンドリング
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop() // deferでstopを呼び出すことで、プログラムが終了する際にシグナルの監視を終了する

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done() // 終了時にwg.Done()を呼び出す
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("server.ListenAndServe: %v", err)
		}
	}()

	// シグナルを待機
	<-ctx.Done()
	log.Println("Server is shutting down...")
	stop() // シグナルの監視を終了

	// シャットダウンのタイムアウトを設定
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("server.Shutdown: %v", err)
	}

	// 全てのgoroutineが終了するまで待機
	wg.Wait()
	log.Println("Server exited gracefully")

	return nil
}

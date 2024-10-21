package main

import (
	"log"
	"net/http"
	"os"
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

	// サーバーをlistenする
	err = http.ListenAndServe(port, wrappedMux)
	if err != nil {
		return err
	}
	return nil
}

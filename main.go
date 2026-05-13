package main

import (
	"log"
	"net/http"
	"github.com/rs/cors"
)

func main() {
	// ログ設定
	logPath := "logs/order.log"
	os.MkdirAll(filepath.Dir(logPath), 0755)
	f, err := os.OpenFile(logPath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer f.Close()
	log.SetOutput(f)

	// DB初期化
	InitDB()

	// ルーティング
	http.HandleFunc("/orders", OrderHandler)        // POST: 注文, GET: 一覧
	http.HandleFunc("/orders/status", StatusHandler) // GET: 状態別取得
	http.HandleFunc("/orders/update", UpdateHandler) // POST: 状態更新

	log.Println("Server started on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// --- グローバル変数 ---
var db *sql.DB

// --- 構造体定義 ---
type RequestItem struct {
	MenuName  string `json:"menuName"`
	UnitPrice int    `json:"unitPrice"`
	Quantity  int    `json:"quantity"`
}

type OrderRequest struct {
	TerminalNo  string        `json:"terminalNo"`
	MessageType string        `json:"messageType"`
	TotalAmount int           `json:"totalAmount"`
	Items       []RequestItem `json:"items"`
}

type OrderResponse struct {
	Result      string `json:"result"`
	OrderNo     string `json:"orderNo"`
	OrderStatus string `json:"orderStatus"`
	TotalAmount int    `json:"totalAmount"`
	Message     string `json:"message"`
}

// --- メイン関数 ---
func main() {
	// ログ設定
	logPath := "logs/order.log"
	os.MkdirAll(filepath.Dir(logPath), 0755)
	logFile, _ := os.OpenFile(logPath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	defer logFile.Close()
	log.SetOutput(io.MultiWriter(os.Stdout, logFile))

	// DB初期化
	initDB()

	// ルーティング
	mux := http.NewServeMux()
	mux.HandleFunc("/api/orders", apiOrdersHandler)
	mux.HandleFunc("/api/orders/", apiOrderDetailHandler)

	// サーバ起動
	fmt.Println("サーバー起動: http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", corsMiddleware(mux)))
}

// --- データベース処理 ---
func initDB() {
	var err error
	db, err = sql.Open("sqlite", "order.db")
	if err != nil {
		log.Fatal(err)
	}

	query := `
	CREATE TABLE IF NOT EXISTS order_items (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		order_no TEXT NOT NULL,
		terminal_no TEXT NOT NULL,
		order_status TEXT NOT NULL,
		item_no INTEGER NOT NULL,
		menu_name TEXT NOT NULL,
		unit_price INTEGER NOT NULL,
		quantity INTEGER NOT NULL,
		subtotal INTEGER NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`
	db.Exec(query)
}

func generateOrderNo() string {
	now := time.Now()
	datePart := now.Format("0102")
	var count int
	db.QueryRow("SELECT COUNT(DISTINCT order_no) FROM order_items WHERE order_no LIKE ?", datePart+"-%").Scan(&count)
	return fmt.Sprintf("%s-%03d", datePart, count+1)
}

// --- ハンドラー処理 ---
func apiOrdersHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == "POST" {
		var req OrderRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			sendError(w, "Invalid JSON")
			return
		}
		log.Printf("[INCOMING] %v", req)

		// 入力チェック
		if req.TerminalNo == "" || req.MessageType != "ORDER_CONFIRM" || len(req.Items) == 0 {
			sendError(w, "Validation Failed")
			return
		}

		orderNo := generateOrderNo()
		status := "オーダー受信"

		for i, item := range req.Items {
			subtotal := item.UnitPrice * item.Quantity
			db.Exec(`INSERT INTO order_items (order_no, terminal_no, order_status, item_no, menu_name, unit_price, quantity, subtotal) 
				VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
				orderNo, req.TerminalNo, status, i+1, item.MenuName, item.UnitPrice, item.Quantity, subtotal)
		}

		res := OrderResponse{"OK", orderNo, status, req.TotalAmount, "注文を受け付けました"}
		log.Printf("[OUTGOING] %v", res)
		json.NewEncoder(w).Encode(res)

	} else if r.Method == "GET" {
		status := r.URL.Query().Get("status")
		query := "SELECT order_no, MAX(terminal_no), MAX(order_status), SUM(subtotal) FROM order_items"
		if status != "" {
			query += " WHERE order_status = '" + status + "'"
		}
		query += " GROUP BY order_no"

		rows, _ := db.Query(query)
		var list []map[string]interface{}
		for rows.Next() {
			var no, term, st string
			var total int
			rows.Scan(&no, &term, &st, &total)
			list = append(list, map[string]interface{}{"orderNo": no, "terminalNo": term, "status": st, "totalAmount": total})
		}
		json.NewEncoder(w).Encode(list)
	}
}

func apiOrderDetailHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/orders/"), "/")
	orderNo := parts[0]

	if r.Method == "GET" {
		rows, _ := db.Query("SELECT item_no, menu_name, unit_price, quantity, subtotal FROM order_items WHERE order_no = ?", orderNo)
		var items []map[string]interface{}
		for rows.Next() {
			var ino, pr, qty, sub int
			var name string
			rows.Scan(&ino, &name, &pr, &qty, &sub)
			items = append(items, map[string]interface{}{"itemNo": ino, "menuName": name, "unitPrice": pr, "quantity": qty, "subtotal": sub})
		}
		json.NewEncoder(w).Encode(items)
	} else if r.Method == "PUT" {
		var body struct{ OrderStatus string `json:"orderStatus"` }
		json.NewDecoder(r.Body).Decode(&body)
		db.Exec("UPDATE order_items SET order_status = ? WHERE order_no = ?", body.OrderStatus, orderNo)
		log.Printf("[UPDATE] %s -> %s", orderNo, body.OrderStatus)
		json.NewEncoder(w).Encode(map[string]string{"result": "OK"})
	}
}

// --- ユーティリティ ---
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, PUT, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func sendError(w http.ResponseWriter, msg string) {
	w.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(w).Encode(map[string]string{"result": "NG", "message": msg})
}
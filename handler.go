package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
)

// १०.१ /api/orders (दर्ता र सूची)
func ApiOrdersHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	if r.Method == "POST" {
		var req OrderRequest
		json.NewDecoder(r.Body).Decode(&req)
		log.Printf("[IN] %v", req)

		if err := validateOrder(req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"result": "NG", "message": err.Error()})
			return
		}

		orderNo := generateOrderNo()
		status := "オーダー受信"
		for i, item := range req.Items {
			db.Exec(`INSERT INTO order_items (order_no, terminal_no, order_status, item_no, menu_name, unit_price, quantity, subtotal) 
				VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
				orderNo, req.TerminalNo, status, i+1, item.MenuName, item.UnitPrice, item.Quantity, item.UnitPrice*item.Quantity)
		}

		res := OrderResponse{"OK", orderNo, status, req.TotalAmount, "注文を受け付けました"}
		log.Printf("[OUT] %v", res)
		json.NewEncoder(w).Encode(res)

	} else if r.Method == "GET" {
		// १०.२ / १०.३ सूची取得
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

// १०.४ र १०.५ विवरण र अवस्था अपडेट
func ApiOrderDetailHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	parts := strings.Split(r.URL.Path, "/")
	orderNo := parts[3]

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
		var body struct { OrderStatus string `json:"orderStatus"` }
		json.NewDecoder(r.Body).Decode(&body)
		
		_, err := db.Exec("UPDATE order_items SET order_status = ? WHERE order_no = ?", body.OrderStatus, orderNo)
		if err != nil { return }
		
		log.Printf("[UPDATE] No=%s, Status=%s", orderNo, body.OrderStatus)
		json.NewEncoder(w).Encode(map[string]string{"result": "OK", "orderNo": orderNo, "newStatus": body.OrderStatus})
	}
}
import (
    "encoding/json"
    "fmt"
    "net/http"
    "os"
    "time"
)

type Item struct {
    MenuName  string `json:"menuName"`
    UnitPrice int    `json:"unitPrice"`
    Quantity  int    `json:"quantity"`
}

type OrderRequest struct {
    TerminalNo  string `json:"terminalNo"`
    TotalAmount int    `json:"totalAmount"`
    Items       []Item `json:"items"`
}

func orderHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    var req OrderRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    // 6.1 Order Number Logic (MMDD-NNN)
    // Using seconds as NNN for this demonstration to ensure uniqueness
    orderNo := time.Now().Format("0102") + "-" + fmt.Sprintf("%03d", time.Now().Second())

    // 4.4 Logging
    logEntry := fmt.Sprintf("[%s] Order: %s, Terminal: %s, Total: %d\n",
        time.Now().Format("2006-01-02 15:04:05"), orderNo, req.TerminalNo, req.TotalAmount)
    f, _ := os.OpenFile("logs/order.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
    f.WriteString(logEntry)
    f.Close()

    // 5.1 DB Saving
    for i, item := range req.Items {
        subtotal := item.UnitPrice * item.Quantity
        db.Exec(`INSERT INTO order_items (order_no, terminal_no, order_status, item_no, menu_name, unit_price, quantity, subtotal)
            VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
            orderNo, req.TerminalNo, "オーダー受信", i+1, item.MenuName, item.UnitPrice, item.Quantity, subtotal)
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]string{"orderNo": orderNo, "status": "Accepted"})
}
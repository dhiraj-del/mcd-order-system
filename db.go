import (
    "fmt"
    "net/http"
)

func main() {
    initDB()

    http.HandleFunc("/api/orders", orderHandler)

    fmt.Println("Server starting on http://localhost:8080&quot;)
    http.ListenAndServe(":8080", nil)
}

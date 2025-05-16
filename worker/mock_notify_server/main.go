package main

import (
	"encoding/json"
	"exchange-wallet-service/httpclient"
	"fmt"
	"io"
	"log"
	"net/http"
)

type NotifyRequest struct {
	Txn []httpclient.Transaction `json:"txn"`
}

func main() {
	http.HandleFunc("/exchange-wallet/notify", func(w http.ResponseWriter, r *http.Request) {
		log.Println("📩 Received a request")

		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "failed to read body", http.StatusInternalServerError)
			return
		}
		defer r.Body.Close()

		var req NotifyRequest
		if err := json.Unmarshal(body, &req); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			log.Println("❌ Invalid JSON:", err)
			return
		}

		// 打印格式化的 JSON
		fmt.Println("🧾 Parsed JSON request:")
		pretty, _ := json.MarshalIndent(req, "", "  ")
		fmt.Println(string(pretty))

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"success":true}`))
	})

	addr := "127.0.0.1:9997/exchange-wallet/notify"
	log.Println("🚀 Mock Notify Server listening on", addr)
	if err := http.ListenAndServe("127.0.0.1:9997", nil); err != nil {
		log.Fatal("❌ Server failed:", err)
	}
}

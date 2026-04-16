package handler

import "net/http"

func setCORS(res http.ResponseWriter) {
	res.Header().Set("Access-Control-Allow-Origin", "*")
	res.Header().Set("Access-Control-Allow-Methods", "POST, DELETE, OPTIONS")
	res.Header().Set("Access-Control-Allow-Headers", "*")
}

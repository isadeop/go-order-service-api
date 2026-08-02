package controllers

import (
	"encoding/json"
	"net/http"
)

// decodeJSONBody decodifica o corpo da requisição no destino informado.
// JSON inválido já escreve a resposta 400 e retorna false.
func decodeJSONBody(w http.ResponseWriter, r *http.Request, dst any) bool {

	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		http.Error(w, "json inválido", http.StatusBadRequest)
		return false
	}

	return true
}

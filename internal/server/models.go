package server

import (
	"bigbat/internal/config"
	"bigbat/internal/openai"
	"encoding/json"
	"net/http"
)

func (a *App) handleModels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	data := make([]openai.ModelObject, 0, len(config.DefaultModelList)+len(config.ModelAliasMap))
	for _, modelName := range config.DefaultModelList {
		data = append(data, openai.ModelObject{ID: modelName, Object: "model"})
	}
	added := make(map[string]struct{}, len(config.DefaultModelList))
	for _, modelName := range config.DefaultModelList {
		added[modelName] = struct{}{}
	}
	for alias := range config.ModelAliasMap {
		if _, ok := added[alias]; ok {
			continue
		}
		data = append(data, openai.ModelObject{ID: alias, Object: "model"})
	}
	resp := openai.ModelListResponse{Object: "list", Data: data}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

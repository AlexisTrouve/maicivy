// Package fakeagents — une FAUSSE passerelle IA pour le dossier vivant : le banc local
// (cmd/dossier-live, DOSSIER_LIVE_FAKE_AGENTS=1) et les tests de bout en bout des routes (api).
package fakeagents

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"
)

// QUOI : une FAUSSE passerelle IA — elle répond à chaque
// tâche des agents (`dossier-vivant/<tâche>` en tête du prompt système) par une sortie conforme au
// contrat, sur le protocole /v1/messages.
//
// POURQUOI : l'E2E du visiteur doit traverser TOUT le chemin de production — la route publique, le
// handler, le service d'agents, le vrai AIService (HTTP, décodage, coût), le flux SSE, la montagne
// et les fiches réelles, puis le PDF — sans dépendre d'un modèle (durée, coût, sortie variable).
// Seule la réponse du modèle est simulée ; le motif est celui de dossiervivant/run_test.go. Les
// tests des routes (api/dossier_vivant_record_test.go) s'en servent pour la même raison : un run
// RÉEL du serveur, dont on vérifie ensuite le PDF.
//
// COMMENT : chaque appel est retenu `hold` (Hold pour le banc : les sous-agents se chevauchent et
// le flux arrive au fil de l'eau, comme en vrai). Les projets cités sont de vraies cartes de la montagne ; le code
// du service valide tout id contre elle, comme pour un vrai modèle.

// Hold — la durée d'un appel sur le banc.
const Hold = 300 * time.Millisecond

// Kept — projets retenus ; Dropped — écarté (la montagne réelle les porte tous).
var (
	Kept    = []string{"melodicode", "maicivy", "envoia"}
	Dropped = "liveconf"
)

func answer(task, prompt string) any {
	switch task {
	case "plan":
		return map[string]any{"plan": "quatre tâches en parallèle.", "tasks": map[string]string{
			"reader": "reformule le besoin.", "researcher": "trouve les preuves.", "architect": "dessine la chaîne.", "writer": "monte le dossier."}}
	case "reader":
		return map[string]any{"notes": []string{"un volume de demandes à traiter", "une équipe réduite", "trace de chaque décision"},
			"gaps": []string{"Quel outil aujourd'hui ?"}, "summary": "besoin reformulé, une question."}
	case "filter":
		return map[string]any{"candidates": append(append([]string{}, Kept...), Dropped)}
	case "verdict":
		if strings.Contains(prompt, "id: "+Dropped) {
			return map[string]any{"verdict": "drop", "reason": "aucune délégation", "why": "", "score": 10}
		}
		return map[string]any{"verdict": "keep", "reason": "agents en production", "why": "Des agents en production, mesurés.", "score": 80}
	case "refs":
		return map[string]any{"refs": []string{"envoia"}}
	case "lesson":
		return map[string]any{"useful": true, "lesson": "planifier, puis exécuter sans LLM"}
	case "design":
		return map[string]any{
			"stages":     []string{"la demande entre, citée telle quelle", "un modèle léger la trie", "le répondeur cite la base", "un humain reprend les cas sensibles"},
			"branches":   []string{"doute du trieur → file humaine"},
			"guardrails": []string{"La demande est citée, jamais exécutée.", "Contrôle sans LLM.", "Humain sur règle."},
			"check":      "Couvre les trois exigences", "summary": "chaîne en quatre étapes.",
		}
	case "schema":
		return map[string]any{
			"notes": []string{"Le modèle trie, le code tranche", "L'humain voit le doute, pas tout"},
			"nodes": []map[string]any{
				{"id": "sn-in", "title": "Demande", "subtitle": "citée, jamais exécutée", "col": 0, "row": 0},
				{"id": "sn-triage", "title": "Trieur", "subtitle": "modèle léger", "col": 1, "row": 0},
				{"id": "sn-answer", "title": "Répondeur", "subtitle": "cite la base", "col": 2, "row": 0},
				{"id": "sn-human", "title": "File humaine", "subtitle": "brief joint", "col": 1, "row": 2, "human": true},
			},
			"edges": []map[string]any{
				{"from": "sn-in", "to": "sn-triage", "step": 1},
				{"from": "sn-triage", "to": "sn-answer", "step": 2},
				{"from": "sn-triage", "to": "sn-human", "dashed": true, "step": 3},
			},
			"journal": "Journal des décisions", "label": "Chaîne proposée, de la demande à la réponse",
			"summary": "schéma en quatre nœuds, trois garde-fous posés.",
		}
	case "section-need":
		return map[string]any{"lead": "Traiter chaque demande en minutes, sans perdre la trace.", "items": []string{"Source citée.", "Humain sur règle.", "Trace lisible."}}
	case "closing":
		return map[string]any{"contribution": []string{"L’architecture.", "Les garde-fous.", "Le coût mesuré."}, "next": "Un appel de 30 minutes, puis un prototype."}
	}
	return map[string]any{}
}

// NewGateway — la fausse passerelle, sur un port local libre ; chaque appel retenu `hold`. À fermer
// par l'appelant (Close) ; le banc la garde toute sa vie.
func NewGateway(hold time.Duration) *httptest.Server {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Model  string `json:"model"`
			System []struct {
				Text string `json:"text"`
			} `json:"system"`
			Messages []struct {
				Content []struct {
					Text string `json:"text"`
				} `json:"content"`
			} `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || len(body.System) == 0 || len(body.Messages) == 0 || len(body.Messages[0].Content) == 0 {
			http.Error(w, `{"type":"error","error":{"type":"invalid_request_error","message":"forme inattendue"}}`, http.StatusBadRequest)
			return
		}
		task := strings.TrimPrefix(strings.SplitN(body.System[0].Text, "\n", 2)[0], "dossier-vivant/")
		time.Sleep(hold)
		text, _ := json.Marshal(answer(task, body.Messages[0].Content[0].Text))
		resp, _ := json.Marshal(map[string]any{
			"id": "msg_fake", "type": "message", "role": "assistant", "model": body.Model,
			"content":     []any{map[string]any{"type": "text", "text": string(text)}},
			"stop_reason": "end_turn",
			"usage":       map[string]any{"input_tokens": 1200, "output_tokens": 300},
		})
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(resp)
	}))
	return srv
}

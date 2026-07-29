package services

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/anthropics/anthropic-sdk-go"
)

// Verrouille le contrat de validateLang : le paramètre `language` (obligatoire sur tous les tools
// du chat) doit être servable, sinon une erreur explicite est renvoyée au LLM (listant les langues
// dispo + invitant à répondre en anglais). Régression de la demande "language systématique + erreur".

func TestValidateLang_Supported(t *testing.T) {
	for _, l := range []string{"fr", "en", "de", "it", "zh"} {
		got, err := validateLang(map[string]interface{}{"language": l})
		if err != nil {
			t.Fatalf("validateLang(%q) a renvoyé une erreur: %v", l, err)
		}
		if got != l {
			t.Fatalf("validateLang(%q) = %q, attendu %q", l, got, l)
		}
	}
}

func TestValidateLang_Normalizes(t *testing.T) {
	got, err := validateLang(map[string]interface{}{"language": "  EN "})
	if err != nil || got != "en" {
		t.Fatalf("validateLang('  EN ') = (%q, %v), attendu ('en', nil)", got, err)
	}
}

func TestValidateLang_UnsupportedMentionsListAndEnglish(t *testing.T) {
	for _, l := range []string{"es", "ja", "pt", "ka", "ru"} {
		_, err := validateLang(map[string]interface{}{"language": l})
		if err == nil {
			t.Fatalf("validateLang(%q) attendait une erreur, reçu nil", l)
		}
		msg := err.Error()
		// L'erreur doit lister les langues dispo...
		if !strings.Contains(msg, chatLangList) {
			t.Errorf("erreur pour %q sans la liste %q: %s", l, chatLangList, msg)
		}
		// ...et proposer l'anglais comme repli.
		if !strings.Contains(strings.ToLower(msg), "english") && !strings.Contains(msg, "(en)") {
			t.Errorf("erreur pour %q devrait proposer l'anglais: %s", l, msg)
		}
	}
}

func TestValidateLang_MissingOrInvalid(t *testing.T) {
	cases := []map[string]interface{}{
		{},                // pas de clé language
		{"language": ""},  // vide
		{"language": 123}, // mauvais type
	}
	for _, input := range cases {
		if _, err := validateLang(input); err == nil {
			t.Errorf("validateLang(%v) attendait une erreur, reçu nil", input)
		}
	}
}

// allChatTools liste tous les tools exposés à Claude — sert à vérifier que la garde de langue
// s'applique SYSTÉMATIQUEMENT (aucun tool n'y échappe).
var allChatTools = []string{
	"get_project", "list_projects", "list_skills", "get_experience",
	"show_project", "show_projects", "show_skills", "show_experience",
	"search_projects", "show_blog_article", "show_blog_list", "add_tip",
	"suggest_followups",
}

// TI : pour CHAQUE tool, une langue non supportée doit court-circuiter en erreur AVANT tout accès
// aux données. On peut donc utiliser un ChatService aux dépendances nil (portfolio/blog jamais
// touchés) — si la garde laissait passer, on aurait un nil-panic, ce qui ferait échouer le test.
func TestExecuteTool_RejectsUnsupportedLanguage(t *testing.T) {
	cs := &ChatService{} // portfolio/blog nil exprès
	// JSON avec tous les paramètres possibles : peu importe, la validation langue tombe en premier.
	input := json.RawMessage(`{"name":"maicivy","query":"rust","slug":"x","text":"tip","language":"es"}`)
	for _, name := range allChatTools {
		res, err := cs.executeTool(name, input)
		if err == nil {
			t.Errorf("%s: langue 'es' non supportée → attendait une erreur, reçu nil (res=%v)", name, res)
			continue
		}
		if !strings.Contains(err.Error(), chatLangList) {
			t.Errorf("%s: l'erreur devrait lister %q, reçu: %s", name, chatLangList, err.Error())
		}
	}
}

// TI : un `language` manquant est aussi rejeté pour tous les tools (param obligatoire).
func TestExecuteTool_RejectsMissingLanguage(t *testing.T) {
	cs := &ChatService{}
	input := json.RawMessage(`{"name":"maicivy","query":"rust","slug":"x","text":"tip"}`) // pas de language
	for _, name := range allChatTools {
		if _, err := cs.executeTool(name, input); err == nil {
			t.Errorf("%s: language manquant → attendait une erreur, reçu nil", name)
		}
	}
}

// classifyChatError catégorise l'erreur Claude en un CODE STABLE (pas un texte en dur — le frontend
// traduit ensuite dans la langue du visiteur, cf. règle i18n du projet). Verrou de la classification
// pour ne pas régresser silencieusement vers un code générique sur un vrai dépassement de contexte.
func TestClassifyChatError_ContextTooLong(t *testing.T) {
	cases := []string{
		"anthropic: Request too large: message exceeds token limit",
		"context length exceeded",
		"413 Payload Too Large",
		"prompt is too large for this model",
	}
	for _, msg := range cases {
		if got := classifyChatError(msg); got != "context_too_long" {
			t.Errorf("classifyChatError(%q) = %q, attendu \"context_too_long\"", msg, got)
		}
	}
}

func TestClassifyChatError_Generic(t *testing.T) {
	cases := []string{
		"connection reset by peer",
		"anthropic: internal server error (500)",
		"",
	}
	for _, msg := range cases {
		if got := classifyChatError(msg); got != "generic" {
			t.Errorf("classifyChatError(%q) = %q, attendu \"generic\"", msg, got)
		}
	}
}

func TestEstimateTokens(t *testing.T) {
	cases := map[string]int{
		"":                      0,
		"abcd":                  1,  // 4 chars / 4
		strings.Repeat("a", 40): 10, // 40 chars / 4
	}
	for s, want := range cases {
		if got := estimateTokens(s); got != want {
			t.Errorf("estimateTokens(%q) = %d, attendu %d", s, got, want)
		}
	}
}

// messageText extrait le texte d'un anthropic.MessageParam construit via NewUserMessage/
// NewAssistantMessage(NewTextBlock(...)) — helper de test uniquement.
func messageText(m anthropic.MessageParam) string {
	if len(m.Content) == 0 || m.Content[0].OfText == nil {
		return ""
	}
	return m.Content[0].OfText.Text
}

// TestBuildMessages_KeepsAllWhenUnderBudget : un historique court (bien sous 40k tokens estimés)
// n'est PAS tronqué — tous les messages sont conservés, dans l'ordre chronologique.
func TestBuildMessages_KeepsAllWhenUnderBudget(t *testing.T) {
	cs := &ChatService{}
	history := []ChatMessage{
		{Role: "user", Content: "salut"},
		{Role: "assistant", Content: "bonjour !"},
		{Role: "user", Content: "ça va ?"},
	}
	msgs := cs.buildMessages(history, "et toi ?")

	if len(msgs) != 4 { // 3 historiques + le nouveau message
		t.Fatalf("len(msgs) = %d, attendu 4 (aucun message ne devrait être coupé)", len(msgs))
	}
	want := []string{"salut", "bonjour !", "ça va ?", "et toi ?"}
	for i, w := range want {
		if got := messageText(msgs[i]); got != w {
			t.Errorf("msgs[%d] = %q, attendu %q", i, got, w)
		}
	}
	if msgs[0].Role != anthropic.MessageParamRoleUser || msgs[1].Role != anthropic.MessageParamRoleAssistant {
		t.Errorf("rôles non préservés: msgs[0].Role=%v msgs[1].Role=%v", msgs[0].Role, msgs[1].Role)
	}
}

// TestBuildMessages_EmptyHistory : historique vide → seul le nouveau message est envoyé, pas de panic.
func TestBuildMessages_EmptyHistory(t *testing.T) {
	cs := &ChatService{}
	msgs := cs.buildMessages(nil, "premier message")
	if len(msgs) != 1 || messageText(msgs[0]) != "premier message" {
		t.Fatalf("buildMessages(nil, ...) = %+v, attendu 1 message = le nouveau", msgs)
	}
}

// TestBuildMessages_DropsOldestWhenOverBudget verrouille le NOUVEAU comportement (budget en tokens,
// pas un nombre fixe de messages) : seuls les messages les plus ANCIENS sont coupés une fois le
// budget de 40k tokens dépassé — le reste de l'historique récent est conservé intact, contrairement
// à l'ancien plafond "6 derniers messages" qui aurait coupé bien plus agressivement.
func TestBuildMessages_DropsOldestWhenOverBudget(t *testing.T) {
	cs := &ChatService{}
	// m0, seul, dépasse déjà largement le budget une fois combiné à m1+m2+m3 (35000 + 3*3000 = 44000 >
	// 40000) → doit être coupé. m1+m2+m3 (9000 tokens, marge large) tiennent confortablement seuls.
	big := strings.Repeat("a", 35_000*approxCharsPerToken)
	small := strings.Repeat("b", 3_000*approxCharsPerToken)
	history := []ChatMessage{
		{Role: "user", Content: big + "-m0-oldest"},            // doit être coupé
		{Role: "assistant", Content: small + "-m1-first-kept"}, // gardé
		{Role: "user", Content: small + "-m2"},                 // gardé
		{Role: "assistant", Content: small + "-m3-mostrecent"}, // gardé
	}
	msgs := cs.buildMessages(history, "nouvelle question")

	if len(msgs) != 4 { // m1, m2, m3, + nouveau message (m0 coupé)
		t.Fatalf("len(msgs) = %d, attendu 4 (m0 le plus ancien doit être coupé)", len(msgs))
	}
	if strings.Contains(messageText(msgs[0]), "m0-oldest") {
		t.Error("m0 (le plus ancien) n'aurait pas dû être conservé — budget 40k dépassé")
	}
	if !strings.Contains(messageText(msgs[0]), "m1-first-kept") {
		t.Errorf("msgs[0] devrait être m1 (premier survivant, ordre chronologique préservé), got: %.30q", messageText(msgs[0]))
	}
	if !strings.Contains(messageText(msgs[len(msgs)-1]), "nouvelle question") {
		t.Error("le nouveau message doit toujours être en dernier, jamais coupé par le budget")
	}
}

// TestExecuteTool_AddTip_EchoesTextAndIcon verrouille un bug réel découvert en implémentant
// suggest_followups : add_tip renvoyait un stub {"ok": true} au lieu d'échoer le texte/icône que le
// LLM a choisi. Le FRONTEND lit `text`/`icon` depuis CE tool_result (page.tsx, case 'add_tip') — avec
// le stub, `tipData.text` était toujours undefined → le tip n'était JAMAIS affiché, silencieusement.
func TestExecuteTool_AddTip_EchoesTextAndIcon(t *testing.T) {
	cs := &ChatService{}
	input := json.RawMessage(`{"text":"Astuce utile","icon":"💡","language":"fr"}`)

	result, err := cs.executeTool("add_tip", input)
	if err != nil {
		t.Fatalf("add_tip a renvoyé une erreur: %v", err)
	}

	data, ok := result.(map[string]string)
	if !ok {
		t.Fatalf("add_tip devrait renvoyer map[string]string, got %T: %+v", result, result)
	}
	if data["text"] != "Astuce utile" {
		t.Errorf("text = %q, attendu %q", data["text"], "Astuce utile")
	}
	if data["icon"] != "💡" {
		t.Errorf("icon = %q, attendu %q", data["icon"], "💡")
	}
}

// TestExecuteTool_SuggestFollowups_ExtractsQuestions : le tool renvoie la liste de questions telle
// que fournie par le LLM (échoée, même pattern que add_tip — le frontend l'affiche directement dans
// le LeftPanel, en remplacement du pool statique de hints, tant que de nouvelles suggestions arrivent).
func TestExecuteTool_SuggestFollowups_ExtractsQuestions(t *testing.T) {
	cs := &ChatService{}
	input := json.RawMessage(`{"questions":["Quels sont tes projets en Rust ?","Et en Go ?"],"language":"fr"}`)

	result, err := cs.executeTool("suggest_followups", input)
	if err != nil {
		t.Fatalf("suggest_followups a renvoyé une erreur: %v", err)
	}

	data, ok := result.(map[string][]string)
	if !ok {
		t.Fatalf("suggest_followups devrait renvoyer map[string][]string, got %T: %+v", result, result)
	}
	want := []string{"Quels sont tes projets en Rust ?", "Et en Go ?"}
	if len(data["questions"]) != len(want) {
		t.Fatalf("questions = %v, attendu %v", data["questions"], want)
	}
	for i, q := range want {
		if data["questions"][i] != q {
			t.Errorf("questions[%d] = %q, attendu %q", i, data["questions"][i], q)
		}
	}
}

// TestExecuteTool_SuggestFollowups_IgnoresNonStringItems : un item non-string dans le tableau (LLM
// erratique) est ignoré plutôt que de faire planter le parsing.
func TestExecuteTool_SuggestFollowups_IgnoresNonStringItems(t *testing.T) {
	cs := &ChatService{}
	input := json.RawMessage(`{"questions":["Vraie question ?",42,null],"language":"fr"}`)

	result, err := cs.executeTool("suggest_followups", input)
	if err != nil {
		t.Fatalf("suggest_followups a renvoyé une erreur: %v", err)
	}
	data := result.(map[string][]string)
	if len(data["questions"]) != 1 || data["questions"][0] != "Vraie question ?" {
		t.Errorf("questions = %v, attendu 1 seul élément valide", data["questions"])
	}
}

// --- accumulateStream : vrai streaming token par token ---
//
// Avant ce fix, Chat() attendait la réponse COMPLÈTE de Claude (appel non-streaming) puis l'envoyait
// d'un seul ChatEventText — le curseur "streaming" du frontend était cosmétique. accumulateStream
// consomme un stream d'events Anthropic et forwarde chaque delta de texte EN TEMPS RÉEL, tout en
// accumulant le Message final (même shape qu'un appel non-streaming, pour ne pas changer le reste de
// la boucle agentic). Testé avec un stream FAKE (fixtures JSON réalistes) — pas de dépendance réseau.

// fakeStream implémente streamEventIterator à partir d'une liste d'events pré-construits.
type fakeStream struct {
	events []anthropic.MessageStreamEventUnion
	idx    int
	err    error
}

func (f *fakeStream) Next() bool {
	if f.idx >= len(f.events) {
		return false
	}
	f.idx++
	return true
}
func (f *fakeStream) Current() anthropic.MessageStreamEventUnion { return f.events[f.idx-1] }
func (f *fakeStream) Err() error                                 { return f.err }

// parseStreamEvent parse un JSON d'event SSE Anthropic (format réel documenté) en MessageStreamEventUnion.
func parseStreamEvent(t *testing.T, raw string) anthropic.MessageStreamEventUnion {
	t.Helper()
	var e anthropic.MessageStreamEventUnion
	if err := json.Unmarshal([]byte(raw), &e); err != nil {
		t.Fatalf("parseStreamEvent: %v (raw=%s)", err, raw)
	}
	return e
}

func TestAccumulateStream_ForwardsTextDeltasInRealTimeAndAccumulatesFinalMessage(t *testing.T) {
	events := []anthropic.MessageStreamEventUnion{
		parseStreamEvent(t, `{"type":"message_start","message":{"id":"msg_1","type":"message","role":"assistant","content":[],"model":"claude-haiku-4-5-20251001","stop_reason":null,"stop_sequence":null,"usage":{"input_tokens":10,"output_tokens":0}}}`),
		parseStreamEvent(t, `{"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`),
		parseStreamEvent(t, `{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Bon"}}`),
		parseStreamEvent(t, `{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"jour !"}}`),
		parseStreamEvent(t, `{"type":"content_block_stop","index":0}`),
		parseStreamEvent(t, `{"type":"message_delta","delta":{"stop_reason":"end_turn","stop_sequence":null},"usage":{"output_tokens":5}}`),
		parseStreamEvent(t, `{"type":"message_stop"}`),
	}

	eventCh := make(chan ChatEvent, 10)
	msg, err := accumulateStream(&fakeStream{events: events}, eventCh)
	close(eventCh)

	if err != nil {
		t.Fatalf("accumulateStream a renvoyé une erreur: %v", err)
	}

	// Les deltas doivent avoir été forwardés EN TEMPS RÉEL, dans l'ordre, PAS un seul bloc groupé.
	var got []ChatEvent
	for e := range eventCh {
		got = append(got, e)
	}
	if len(got) != 2 {
		t.Fatalf("attendu 2 ChatEventText distincts (streaming réel), got %d: %+v", len(got), got)
	}
	if got[0].Delta != "Bon" || got[1].Delta != "jour !" {
		t.Errorf("deltas = %q, %q — attendu \"Bon\" puis \"jour !\" séparément", got[0].Delta, got[1].Delta)
	}

	// Le Message final doit être correctement accumulé (même shape qu'un appel non-streaming).
	if len(msg.Content) != 1 {
		t.Fatalf("msg.Content devrait avoir 1 bloc, got %d", len(msg.Content))
	}
	if msg.Content[0].Text != "Bonjour !" {
		t.Errorf("msg.Content[0].Text = %q, attendu \"Bonjour !\" (accumulé)", msg.Content[0].Text)
	}
	if msg.StopReason != "end_turn" {
		t.Errorf("msg.StopReason = %q, attendu \"end_turn\"", msg.StopReason)
	}
}

func TestAccumulateStream_AssemblesToolUseInputFromDeltas(t *testing.T) {
	events := []anthropic.MessageStreamEventUnion{
		parseStreamEvent(t, `{"type":"message_start","message":{"id":"msg_2","type":"message","role":"assistant","content":[],"model":"claude-haiku-4-5-20251001","stop_reason":null,"stop_sequence":null,"usage":{"input_tokens":10,"output_tokens":0}}}`),
		parseStreamEvent(t, `{"type":"content_block_start","index":0,"content_block":{"type":"tool_use","id":"tool_1","name":"get_project","input":{}}}`),
		parseStreamEvent(t, `{"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"{\"name\": \""}}`),
		parseStreamEvent(t, `{"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"maicivy\"}"}}`),
		parseStreamEvent(t, `{"type":"content_block_stop","index":0}`),
		parseStreamEvent(t, `{"type":"message_delta","delta":{"stop_reason":"tool_use","stop_sequence":null},"usage":{"output_tokens":8}}`),
		parseStreamEvent(t, `{"type":"message_stop"}`),
	}

	eventCh := make(chan ChatEvent, 10)
	msg, err := accumulateStream(&fakeStream{events: events}, eventCh)
	close(eventCh)

	if err != nil {
		t.Fatalf("accumulateStream a renvoyé une erreur: %v", err)
	}

	// Aucun delta de TEXTE pour un tool_use pur — pas de ChatEventText parasite.
	for e := range eventCh {
		t.Errorf("événement inattendu (pas de texte dans cette réponse): %+v", e)
	}

	if len(msg.Content) != 1 {
		t.Fatalf("msg.Content devrait avoir 1 bloc, got %d", len(msg.Content))
	}
	var input map[string]string
	if err := json.Unmarshal(msg.Content[0].Input, &input); err != nil {
		t.Fatalf("input JSON mal assemblé: %v (raw=%s)", err, msg.Content[0].Input)
	}
	if input["name"] != "maicivy" {
		t.Errorf("input assemblé = %+v, attendu name=maicivy (fragments partial_json concaténés)", input)
	}
	if msg.StopReason != "tool_use" {
		t.Errorf("msg.StopReason = %q, attendu \"tool_use\"", msg.StopReason)
	}
}

func TestAccumulateStream_PropagatesStreamError(t *testing.T) {
	boom := fmt.Errorf("connection reset by peer")
	eventCh := make(chan ChatEvent, 5)
	_, err := accumulateStream(&fakeStream{err: boom}, eventCh)
	close(eventCh)

	if err == nil || err.Error() != boom.Error() {
		t.Errorf("accumulateStream devrait propager l'erreur du stream, got %v", err)
	}
}

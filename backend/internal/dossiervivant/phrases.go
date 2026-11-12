package dossiervivant

import "fmt"

// QUOI : les lignes que le CODE écrit au nom des agents — statuts (« lit l'index »), compteurs
// (« 79 projets indexés »), relais de l'orchestrateur, titres des sections du dossier.
//
// POURQUOI ici et pas dans le modèle : ces lignes accompagnent des actions du code (un statut part
// AVANT l'appel de modèle qu'il annonce) ou portent des chiffres tirés des données. Les confier au
// modèle coûterait un appel et laisserait un chiffre au modèle — exactement ce que le dossier
// s'interdit. Ce sont pourtant des CONTENUS d'agent (pas de l'interface) : ils suivent la langue
// du run, comme tout ce que les agents écrivent.
//
// COMMENT : une table clé → langue → texte, cinq langues comme l'interface (fr source). Une langue
// inconnue sert l'anglais (règle du projet : jamais le français par défaut pour un visiteur).

var phrasebook = map[string]map[string]string{
	"reading":   {"fr": "lit la demande", "en": "reading the request", "de": "liest die Anfrage", "it": "legge la richiesta", "zh": "阅读需求"},
	"planning":  {"fr": "répartit le travail", "en": "splitting the work", "de": "verteilt die Arbeit", "it": "distribuisce il lavoro", "zh": "分配任务"},
	"waiting":   {"fr": "attend les retours", "en": "waiting for reports", "de": "wartet auf Rückmeldungen", "it": "attende i resoconti", "zh": "等待汇报"},
	"checking":  {"fr": "vérifie le dossier", "en": "checking the dossier", "de": "prüft das Dossier", "it": "verifica il dossier", "zh": "核对方案"},
	"delivered": {"fr": "dossier remis", "en": "dossier delivered", "de": "Dossier übergeben", "it": "dossier consegnato", "zh": "方案已交付"},

	"relay.need":         {"fr": "le besoin est prêt, première section.", "en": "the need is ready, first section.", "de": "der Bedarf steht, erster Abschnitt.", "it": "il bisogno è pronto, prima sezione.", "zh": "需求已就绪，开始第一节。"},
	"relay.architecture": {"fr": "l’architecture est prête.", "en": "the architecture is ready.", "de": "die Architektur steht.", "it": "l’architettura è pronta.", "zh": "架构已就绪。"},
	"relay.proofs":       {"fr": "les preuves sont prêtes, chiffres des fiches.", "en": "the proofs are ready, figures from the project files.", "de": "die Belege stehen, Zahlen aus den Projektblättern.", "it": "le prove sono pronte, cifre dalle schede.", "zh": "证据已就绪，数据取自项目档案。"},
	"check.start":        {"fr": "chaque preuve contre sa fiche.", "en": "each proof against its project file.", "de": "jeder Beleg gegen sein Projektblatt.", "it": "ogni prova contro la sua scheda.", "zh": "逐条核对证据与项目档案。"},
	"check.done":         {"fr": "%d preuves, chiffres lus sur les fiches.", "en": "%d proofs, figures read from the project files.", "de": "%d Belege, Zahlen aus den Projektblättern.", "it": "%d prove, cifre lette dalle schede.", "zh": "%d 条证据，数据均取自项目档案。"},
	"ready":              {"fr": "votre dossier de conception est prêt.", "en": "your design dossier is ready.", "de": "Ihr Konzeptdossier ist fertig.", "it": "il suo dossier di progettazione è pronto.", "zh": "您的设计方案已准备就绪。"},

	"reader.gaps": {"fr": "relève ce qui manque", "en": "noting what is missing", "de": "notiert, was fehlt", "it": "annota cosa manca", "zh": "列出缺失信息"},
	"reader.done": {"fr": "Besoin reformulé, envoyé", "en": "Need restated, sent", "de": "Bedarf umformuliert, gesendet", "it": "Bisogno riformulato, inviato", "zh": "需求已重述并发送"},

	"research.index":      {"fr": "lit l’index", "en": "reading the index", "de": "liest den Index", "it": "legge l’indice", "zh": "阅读项目索引"},
	"research.indexed":    {"fr": "%d projets indexés", "en": "%d projects indexed", "de": "%d Projekte im Index", "it": "%d progetti indicizzati", "zh": "已索引 %d 个项目"},
	"research.filtering":  {"fr": "filtre les projets", "en": "filtering projects", "de": "filtert die Projekte", "it": "filtra i progetti", "zh": "筛选项目"},
	"research.candidates": {"fr": "%d candidats", "en": "%d candidates", "de": "%d Kandidaten", "it": "%d candidati", "zh": "%d 个候选项目"},
	"research.reading":    {"fr": "lit les fiches", "en": "reading project files", "de": "liest die Projektblätter", "it": "legge le schede", "zh": "阅读项目档案"},
	"research.ranking":    {"fr": "classe les preuves", "en": "ranking the proofs", "de": "ordnet die Belege", "it": "ordina le prove", "zh": "给证据排序"},
	"research.ranked":     {"fr": "%d preuves principales sur %d retenues", "en": "%d main proofs out of %d kept", "de": "%d Hauptbelege von %d behaltenen", "it": "%d prove principali su %d tenute", "zh": "保留的 %[2]d 个中 %[1]d 个为主要证据"},
	"research.done":       {"fr": "Preuves envoyées", "en": "Proofs sent", "de": "Belege gesendet", "it": "Prove inviate", "zh": "证据已发送"},
	"research.summary":    {"fr": "%d projets retenus, %d en preuve.", "en": "%d projects kept, %d as proof.", "de": "%d Projekte behalten, %d als Beleg.", "it": "%d progetti tenuti, %d come prova.", "zh": "保留 %d 个项目，%d 个作为证据。"},

	"arch.reading":     {"fr": "lit des architectures", "en": "studying architectures", "de": "studiert Architekturen", "it": "studia architetture", "zh": "研究已有架构"},
	"arch.designing":   {"fr": "conçoit la chaîne", "en": "designing the pipeline", "de": "entwirft die Kette", "it": "progetta la catena", "zh": "设计流程"},
	"arch.stages":      {"fr": "Chaîne en %d étapes, %d embranchements", "en": "Pipeline in %d stages, %d branches", "de": "Kette in %d Schritten, %d Verzweigungen", "it": "Catena in %d fasi, %d diramazioni", "zh": "流程共 %d 步，%d 个分支"},
	"arch.commission":  {"fr": "confie le schéma au Dessinateur", "en": "handing the diagram to the Drafter", "de": "übergibt das Schema dem Zeichner", "it": "affida lo schema al Disegnatore", "zh": "将架构图交给绘图员"},
	"arch.order":       {"fr": "dessine la chaîne : %d étapes, %d garde-fous.", "en": "draw the pipeline: %d stages, %d guardrails.", "de": "zeichne die Kette: %d Schritte, %d Leitplanken.", "it": "disegna la catena: %d fasi, %d paletti.", "zh": "请绘制流程：%d 步，%d 道护栏。"},
	"arch.waitDiagram": {"fr": "attend le schéma", "en": "waiting for the diagram", "de": "wartet auf das Schema", "it": "attende lo schema", "zh": "等待架构图"},
	"arch.guardrails":  {"fr": "pose les garde-fous", "en": "setting guardrails", "de": "setzt Leitplanken", "it": "fissa i paletti", "zh": "设置护栏"},
	"arch.checking":    {"fr": "vérifie le besoin", "en": "checking against the need", "de": "prüft gegen den Bedarf", "it": "verifica rispetto al bisogno", "zh": "对照需求核查"},
	"arch.chain":       {"fr": "Chaîne : %s", "en": "Pipeline: %s", "de": "Kette: %s", "it": "Catena: %s", "zh": "流程：%s"},
	"arch.done":        {"fr": "Architecture envoyée", "en": "Architecture sent", "de": "Architektur gesendet", "it": "Architettura inviata", "zh": "架构已发送"},

	"diag.reading":  {"fr": "lit la commande de l’Architecte", "en": "reading the Architect’s brief", "de": "liest den Auftrag des Architekten", "it": "legge l’incarico dell’Architetto", "zh": "阅读架构师的委托"},
	"diag.checking": {"fr": "contrôle la grille", "en": "checking the grid", "de": "prüft das Raster", "it": "controlla la griglia", "zh": "校验网格"},
	"diag.drawn":    {"fr": "Schéma : %d étapes, %d liens", "en": "Diagram: %d steps, %d links", "de": "Schema: %d Schritte, %d Verbindungen", "it": "Schema: %d fasi, %d collegamenti", "zh": "架构图：%d 步，%d 条连线"},
	"diag.rejected": {"fr": "Schéma refusé : hors du contrat de la grille", "en": "Diagram rejected: outside the grid contract", "de": "Schema abgelehnt: verletzt den Rastervertrag", "it": "Schema respinto: fuori dal contratto della griglia", "zh": "架构图被拒：不符合网格约定"},
	"diag.done":     {"fr": "Schéma remis à l’Architecte", "en": "Diagram handed to the Architect", "de": "Schema an den Architekten übergeben", "it": "Schema consegnato all’Architetto", "zh": "架构图已交给架构师"},

	"writer.opening":            {"fr": "ouvre le dossier", "en": "opening the dossier", "de": "öffnet das Dossier", "it": "apre il dossier", "zh": "新建方案"},
	"writer.plan":               {"fr": "Plan du dossier : cinq sections", "en": "Dossier outline: five sections", "de": "Gliederung: fünf Abschnitte", "it": "Schema del dossier: cinque sezioni", "zh": "方案大纲：五个章节"},
	"writer.wait.need":          {"fr": "attend le besoin", "en": "waiting for the need", "de": "wartet auf den Bedarf", "it": "attende il bisogno", "zh": "等待需求"},
	"writer.wait.architecture":  {"fr": "attend l’architecture", "en": "waiting for the architecture", "de": "wartet auf die Architektur", "it": "attende l’architettura", "zh": "等待架构"},
	"writer.wait.proofs":        {"fr": "attend les preuves", "en": "waiting for the proofs", "de": "wartet auf die Belege", "it": "attende le prove", "zh": "等待证据"},
	"writer.write.need":         {"fr": "écrit le besoin", "en": "writing the need", "de": "schreibt den Bedarf", "it": "scrive il bisogno", "zh": "撰写需求"},
	"writer.write.architecture": {"fr": "écrit l’architecture", "en": "writing the architecture", "de": "schreibt die Architektur", "it": "scrive l’architettura", "zh": "撰写架构"},
	"writer.write.proofs":       {"fr": "écrit les preuves", "en": "writing the proofs", "de": "schreibt die Belege", "it": "scrive le prove", "zh": "撰写证据"},
	"writer.write.contribution": {"fr": "écrit l’apport", "en": "writing the contribution", "de": "schreibt den Beitrag", "it": "scrive il contributo", "zh": "撰写贡献"},
	"writer.write.next":         {"fr": "écrit la suite", "en": "writing next steps", "de": "schreibt die nächsten Schritte", "it": "scrive i prossimi passi", "zh": "撰写下一步"},
	"writer.sectionDone":        {"fr": "Section %d écrite", "en": "Section %d written", "de": "Abschnitt %d geschrieben", "it": "Sezione %d scritta", "zh": "第 %d 节完成"},
	"writer.done":               {"fr": "Dossier rédigé", "en": "Dossier written", "de": "Dossier geschrieben", "it": "Dossier scritto", "zh": "方案撰写完成"},

	"section." + SecNeed:         {"fr": "Le besoin, reformulé", "en": "The need, restated", "de": "Der Bedarf, neu formuliert", "it": "Il bisogno, riformulato", "zh": "需求重述"},
	"section." + SecArchitecture: {"fr": "L’architecture proposée", "en": "The proposed architecture", "de": "Die vorgeschlagene Architektur", "it": "L’architettura proposta", "zh": "建议架构"},
	"section." + SecProofs:       {"fr": "Les projets qui le prouvent", "en": "The projects that prove it", "de": "Die Projekte, die es belegen", "it": "I progetti che lo dimostrano", "zh": "佐证项目"},
	"section." + SecContribution: {"fr": "Ce qu’Alexis apporte", "en": "What Alexis brings", "de": "Was Alexis einbringt", "it": "Cosa porta Alexis", "zh": "Alexis 的贡献"},
	"section." + SecNext:         {"fr": "Prochaine étape", "en": "Next step", "de": "Nächster Schritt", "it": "Prossimo passo", "zh": "下一步"},

	// Une fiche qui ne tient pas dans un appel (callsize.go) : elle n'est pas jugée, et cela se voit.
	"brief.tooLong": {"fr": "%s : fiche trop longue pour un appel, non jugée", "en": "%s: project file too long for one call, not judged", "de": "%s: Projektblatt zu lang für einen Aufruf, nicht bewertet", "it": "%s: scheda troppo lunga per una chiamata, non valutata", "zh": "%s：项目档案过长，无法在一次调用中评估"},

	// Séparateur « Nom : raison » des notes de verdict (typographie de chaque langue).
	"colon": {"fr": " : ", "en": ": ", "de": ": ", "it": ": ", "zh": "："},
}

// languageNames — la langue d'écriture, nommée pour le modèle (consignes en anglais).
var languageNames = map[string]string{"fr": "French", "en": "English", "de": "German", "it": "Italian", "zh": "Simplified Chinese"}

// RunLanguage — la langue d'un run : celle de l'interface si elle est servie, sinon l'anglais.
func RunLanguage(lang string) string {
	if _, ok := languageNames[lang]; ok {
		return lang
	}
	return "en"
}

// phrases — le carnet d'un run, dans sa langue.
type phrases string

func (p phrases) get(key string, args ...any) string {
	line := phrasebook[key][string(p)]
	if len(args) == 0 {
		return line
	}
	return fmt.Sprintf(line, args...)
}

package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

// OrigineMail nomme QUI a émis un courrier. Le but est qu'un récapitulatif se lise sans connaître le
// code : « Malt » et « WanMira » désignent ce que la personne qui lit reconnaît, pas le service Go.
type OrigineMail string

const (
	// OrigineMalt : offres Malt captées en IMAP puis transférées (le flux qui compte au quotidien).
	OrigineMalt OrigineMail = "malt"
	// OrigineWanMira : newsletter du blog. Les articles sont écrits par WanMira, d'où le nom —
	// c'est l'origine du CONTENU, la seule que quelqu'un reconnaîtra dans un salon Discord.
	OrigineWanMira OrigineMail = "wanmira"
	// OrigineRappelMalt : le rappel hebdomadaire « mets à jour ta disponibilité ».
	OrigineRappelMalt OrigineMail = "rappel_malt"
	// OrigineAutre : place tenue pour un futur émetteur, afin qu'un nouvel envoi soit compté même si
	// personne n'a pensé à lui créer une origine.
	OrigineAutre OrigineMail = "autre"
)

// originesConnues fixe l'ORDRE d'affichage du récapitulatif. Une liste explicite plutôt qu'un
// parcours de map : un digest dont les lignes dansent d'un jour à l'autre se lit mal.
var originesConnues = []OrigineMail{OrigineMalt, OrigineWanMira, OrigineRappelMalt, OrigineAutre}

// libellesOrigines : ce que voit le lecteur. Séparé de la clé Redis, qui elle ne doit jamais changer.
var libellesOrigines = map[OrigineMail]string{
	OrigineMalt:       "Malt (offres transférées)",
	OrigineWanMira:    "WanMira (newsletter blog)",
	OrigineRappelMalt: "Rappel disponibilité Malt",
	OrigineAutre:      "Autre",
}

const (
	// Fenêtre de silence après une alerte, PAR origine. Une panne de relai fait échouer tous les
	// mails en attente d'un coup : sans cette fenêtre, la première vraie panne poste onze pings et
	// la suivante ne sera plus lue.
	cooldownAlerteMail = time.Hour
	// Les compteurs journaliers ne servent qu'au récapitulatif du jour ; on les garde une semaine
	// pour pouvoir relire un week-end, pas davantage.
	retentionCompteursMail = 8 * 24 * time.Hour
)

// MailWatch tient la comptabilité des envois de courrier et alerte quand ça casse.
//
// POURQUOI ici et pas en lisant les logs de postfix : la panne du 2026-08-30 n'avait RIEN injecté
// dans postfix — le relai était injoignable, les mails n'ont jamais quitté le backend. Un moniteur
// posé sur le MTA ne l'aurait pas vue. Le seul endroit qui sait qu'un envoi devait avoir lieu est
// l'émetteur lui-même.
type MailWatch struct {
	redis *redis.Client
	notif *DiscordNotifier // peut être nil : on compte alors sans notifier personne
}

// NewMailWatch rend nil sans Redis — sans compteur partagé il n'y a rien à récapituler. Un notifier
// nil est en revanche parfaitement valide : les chiffres restent disponibles.
func NewMailWatch(rdb *redis.Client, notif *DiscordNotifier) *MailWatch {
	if rdb == nil {
		return nil
	}
	return &MailWatch{redis: rdb, notif: notif}
}

// CompteursMail = ce qui s'est passé sur une journée, par origine.
type CompteursMail struct {
	Envoyes map[OrigineMail]int
	Echecs  map[OrigineMail]int
}

// Total rend le nombre d'événements du jour, toutes origines confondues. Sert à décider s'il y a
// quelque chose à dire — une journée à zéro ne mérite aucun message.
func (c CompteursMail) Total() int {
	n := 0
	for _, v := range c.Envoyes {
		n += v
	}
	for _, v := range c.Echecs {
		n += v
	}
	return n
}

func (w *MailWatch) cle(genre string, jour time.Time, origine OrigineMail) string {
	return fmt.Sprintf("maicivy:mail:%s:%s:%s", genre, jour.UTC().Format("2006-01-02"), origine)
}

// incrementer compte un événement. Volontairement silencieux en cas d'échec Redis : la comptabilité
// ne doit jamais faire tomber un envoi de courrier, elle l'observe.
func (w *MailWatch) incrementer(ctx context.Context, genre string, origine OrigineMail, jour time.Time) {
	cle := w.cle(genre, jour, origine)
	if err := w.redis.Incr(ctx, cle).Err(); err != nil {
		log.Warn().Err(err).Str("cle", cle).Msg("mailwatch: compteur non incrémenté")
		return
	}
	w.redis.Expire(ctx, cle, retentionCompteursMail)
}

// Succes enregistre un envoi réussi. Ne poste RIEN sur le moment : un envoi qui marche n'a pas à
// interrompre qui que ce soit, il apparaîtra dans le récapitulatif du jour.
func (w *MailWatch) Succes(ctx context.Context, origine OrigineMail, maintenant time.Time) {
	if w == nil {
		return
	}
	w.incrementer(ctx, "envoyes", origine, maintenant)
}

// Echec enregistre un envoi cassé ET alerte immédiatement, dans la limite d'une alerte par heure et
// par origine (cf. cooldownAlerteMail).
func (w *MailWatch) Echec(ctx context.Context, origine OrigineMail, motif string, maintenant time.Time) {
	if w == nil {
		return
	}
	w.incrementer(ctx, "echecs", origine, maintenant)

	if w.notif == nil {
		return
	}

	// SetNX = pose la marque seulement si personne ne l'a posée : c'est le cooldown ET le verrou,
	// en une opération atomique.
	pose, err := w.redis.SetNX(ctx, "maicivy:mail:alerte:"+string(origine), "1", cooldownAlerteMail).Result()
	if err != nil {
		log.Warn().Err(err).Msg("mailwatch: cooldown illisible — on alerte quand même")
	} else if !pose {
		return // déjà alerté pour cette origine dans l'heure
	}

	msg := fmt.Sprintf("🔴 **Envoi de mail en échec — %s**\nMotif : `%s`\n"+
		"_Prochaine alerte pour cette origine dans 1 h au plus tôt._",
		libelleOrigine(origine), motif)
	if err := w.notif.Post(ctx, msg, true); err != nil {
		log.Warn().Err(err).Msg("mailwatch: alerte Discord non postée")
	}
}

// AlerterBlocage signale des mails en attente depuis trop longtemps.
//
// POURQUOI cette alerte EN PLUS de Echec : pendant les 13 jours de panne, les échecs se produisaient
// bien — mais uniquement dans les logs du conteneur. Ici le signal ne vient pas d'un envoi raté mais
// d'un ÉTAT anormal : « il y a du courrier qui attend depuis trop longtemps », ce qui reste vrai
// tant que le problème dure, même si plus personne ne retente.
func (w *MailWatch) AlerterBlocage(ctx context.Context, enAttente int, plusVieux time.Time) {
	if w == nil || w.notif == nil || enAttente <= 0 {
		return
	}

	pose, err := w.redis.SetNX(ctx, "maicivy:mail:alerte:blocage", "1", cooldownAlerteMail).Result()
	if err != nil {
		log.Warn().Err(err).Msg("mailwatch: cooldown blocage illisible — on alerte quand même")
	} else if !pose {
		return
	}

	msg := fmt.Sprintf("🔴 **%d mail(s) en attente de transfert**\n"+
		"Le plus ancien attend depuis le %s. Le relai SMTP ou le service d'envoi ne répond pas.",
		enAttente, plusVieux.UTC().Format("02/01/2006 15:04 UTC"))
	if err := w.notif.Post(ctx, msg, true); err != nil {
		log.Warn().Err(err).Msg("mailwatch: alerte blocage non postée")
	}
}

// cleDerniereReleve garde l'instant de la dernière relève IMAP réussie (secondes Unix, sans TTL :
// c'est son ÂGE qui est le signal, une clé qui expirerait se tairait précisément quand ça casse).
const cleDerniereReleve = "maicivy:mail:releve:derniere"

// MarquerReleve note qu'une relève IMAP vient de réussir.
//
// POURQUOI : du 17 au 19/09/2026 la relève ne lisait plus rien, et aucune alerte ne pouvait le voir —
// un mail jamais relevé n'existe pas en base, donc ni Echec ni AlerterBlocage n'ont de prise dessus.
// La seule trace possible est l'âge de la dernière relève réussie (cf. jobs.MailWatchJob.VerifierReleve).
func (w *MailWatch) MarquerReleve(ctx context.Context, maintenant time.Time) {
	if w == nil {
		return
	}
	if err := w.redis.Set(ctx, cleDerniereReleve, maintenant.Unix(), 0).Err(); err != nil {
		log.Warn().Err(err).Msg("mailwatch: relève non marquée")
	}
}

// DerniereReleve relit l'instant de la dernière relève réussie ; ok=false si aucune n'a jamais été
// marquée (boîte non configurée, ou premier démarrage).
func (w *MailWatch) DerniereReleve(ctx context.Context) (time.Time, bool, error) {
	secondes, err := w.redis.Get(ctx, cleDerniereReleve).Int64()
	if err == redis.Nil {
		return time.Time{}, false, nil
	}
	if err != nil {
		return time.Time{}, false, fmt.Errorf("mailwatch: lecture dernière relève: %w", err)
	}
	return time.Unix(secondes, 0), true, nil
}

// AlerterReleveMuette signale une relève IMAP qui ne réussit plus depuis trop longtemps. Même
// fenêtre de silence que les autres alertes : une panne qui dure ne pingue qu'une fois par heure.
func (w *MailWatch) AlerterReleveMuette(ctx context.Context, derniere time.Time) {
	if w == nil || w.notif == nil {
		return
	}

	pose, err := w.redis.SetNX(ctx, "maicivy:mail:alerte:releve", "1", cooldownAlerteMail).Result()
	if err != nil {
		log.Warn().Err(err).Msg("mailwatch: cooldown relève illisible — on alerte quand même")
	} else if !pose {
		return
	}

	msg := fmt.Sprintf("🔴 **Relève IMAP muette**\n"+
		"Dernière relève réussie le %s. Les nouveaux mails (offres Malt) ne sont plus lus ; "+
		"voir `mailbox: échec du poll` dans les logs du backend.",
		derniere.UTC().Format("02/01/2006 15:04 UTC"))
	if err := w.notif.Post(ctx, msg, true); err != nil {
		log.Warn().Err(err).Msg("mailwatch: alerte relève non postée")
	}
}

// Compteurs relit la journée.
func (w *MailWatch) Compteurs(ctx context.Context, jour time.Time) (CompteursMail, error) {
	c := CompteursMail{
		Envoyes: map[OrigineMail]int{},
		Echecs:  map[OrigineMail]int{},
	}
	for _, o := range originesConnues {
		env, err := w.redis.Get(ctx, w.cle("envoyes", jour, o)).Int()
		if err != nil && err != redis.Nil {
			return c, fmt.Errorf("mailwatch: lecture envoyés %s: %w", o, err)
		}
		ech, err := w.redis.Get(ctx, w.cle("echecs", jour, o)).Int()
		if err != nil && err != redis.Nil {
			return c, fmt.Errorf("mailwatch: lecture échecs %s: %w", o, err)
		}
		c.Envoyes[o] = env
		c.Echecs[o] = ech
	}
	return c, nil
}

// PosterDigest publie le récapitulatif du jour, SANS ping, et rend true s'il a effectivement posté.
//
// Deux garde-fous : une journée sans le moindre événement ne produit aucun message (règle explicite
// d'Alexi — ne pas déranger pour rien), et une marque par date empêche le job horaire de republier
// le même récapitulatif toute la journée. La marque n'est posée qu'APRÈS un envoi réussi, pour qu'un
// Discord momentanément indisponible n'escamote pas le récapitulatif du jour.
func (w *MailWatch) PosterDigest(ctx context.Context, jour time.Time) (bool, error) {
	if w == nil {
		return false, nil
	}
	compte, err := w.Compteurs(ctx, jour)
	if err != nil {
		return false, err
	}
	if compte.Total() == 0 || w.notif == nil {
		return false, nil
	}

	cleJour := "maicivy:mail:digest:" + jour.UTC().Format("2006-01-02")
	dejaFait, err := w.redis.Exists(ctx, cleJour).Result()
	if err == nil && dejaFait > 0 {
		return false, nil
	}

	if err := w.notif.Post(ctx, w.rendreDigest(compte, jour), false); err != nil {
		return false, fmt.Errorf("mailwatch: digest non posté: %w", err)
	}
	w.redis.Set(ctx, cleJour, "1", retentionCompteursMail)
	return true, nil
}

// rendreDigest met en forme le récapitulatif. Les origines à zéro sont TAISÉES : une liste de zéros
// noie la seule ligne qui porte de l'information.
func (w *MailWatch) rendreDigest(c CompteursMail, jour time.Time) string {
	var b strings.Builder
	fmt.Fprintf(&b, "📬 **Courrier maicivy — %s**\n", jour.UTC().Format("02/01/2006"))

	for _, o := range originesConnues {
		env, ech := c.Envoyes[o], c.Echecs[o]
		if env == 0 && ech == 0 {
			continue
		}
		fmt.Fprintf(&b, "• %s : **%d** envoyé(s)", libelleOrigine(o), env)
		if ech > 0 {
			fmt.Fprintf(&b, " — ⚠️ %d échec(s)", ech)
		}
		b.WriteString("\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

func libelleOrigine(o OrigineMail) string {
	if l, ok := libellesOrigines[o]; ok {
		return l
	}
	return string(o)
}

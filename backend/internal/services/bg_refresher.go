package services

import (
	"context"
	"sync/atomic"
)

// bgRefresher garantit qu'UN SEUL rafraîchissement de cache tourne à la fois (single-flight in-process).
//
// QUOI : un verrou non bloquant autour d'une tâche de refetch lancée en arrière-plan.
// POURQUOI : en stale-while-revalidate, chaque requête qui voit un cache périmé déclencherait sinon sa
// PROPRE goroutine de refetch. Sous trafic, N visiteurs simultanés sur un cache expiré = N appels
// parallèles à Gitea/GitLab (effet « troupeau ») — exactement le « repull la planète » qu'on veut tuer.
// maicivy tourne en UN SEUL container backend, donc un flag atomique en mémoire suffit : pas besoin
// d'un lock Redis distribué (qu'on ajouterait seulement le jour où on scale à plusieurs réplicas).
// COMMENT : CompareAndSwap false→true ; si le swap échoue, un refresh est déjà en cours → on abandonne
// silencieusement (les autres requêtes s'appuieront sur lui). La goroutine remet le flag à false en
// sortie (defer), même en cas de panique.
type bgRefresher struct {
	running atomic.Bool
}

// trigger lance fn en arrière-plan SI aucun refresh n'est déjà en cours. Non bloquant : retourne
// immédiatement (le but est que la requête HTTP rende le cache stale sans attendre).
//
// fn reçoit un contexte DÉTACHÉ (context.Background) — surtout PAS le ctx de la requête : celle-ci a
// déjà retourné le stale, donc son contexte est annulé, et un refetch/écriture-cache attaché à lui
// mourrait en plein vol (le cache ne serait jamais rafraîchi). Le détachement est ici, à la source,
// pour qu'aucun appelant n'ait à y penser.
func (b *bgRefresher) trigger(fn func(ctx context.Context)) {
	// Un seul gagnant : le premier à passer false→true. Les autres voient `running` déjà à true.
	if !b.running.CompareAndSwap(false, true) {
		return
	}
	go func() {
		defer b.running.Store(false)
		fn(context.Background())
	}()
}

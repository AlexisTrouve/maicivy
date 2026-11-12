package dossiervivant

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"
)

// QUOI : l'émission du flux. Les agents appellent Emit depuis leurs goroutines ; l'émetteur sérialise
// en trames SSE (`event: <type>` / `data: <JSON>`) dans un canal que le handler HTTP vide vers le
// client.
//
// POURQUOI un canal : cinq agents émettent en même temps et une connexion HTTP n'a qu'un écrivain.
// Le canal est le point de sérialisation ; l'ordre d'arrivée dans le canal EST l'ordre du flux.
//
// COMMENT la déconnexion est gérée : l'émetteur vit sous le contexte de la CONNEXION (pas celui du
// travail des agents). Client parti → le contexte tombe → Emit rend la main sans bloquer, et le
// handler annule le run : on ne paie pas des tokens que personne ne regarde. Inversement, un run
// arrêté sur budget garde sa connexion : son `run.failed` part bien.

// Emitter — reçoit les événements d'un run. Sûr en concurrence.
type Emitter interface{ Emit(Event) }

// FormatSSE — une trame SSE complète pour un événement.
func FormatSSE(e Event) ([]byte, error) {
	data, err := encodeEvent(e)
	if err != nil {
		return nil, err
	}
	return []byte(fmt.Sprintf("event: %s\ndata: %s\n\n", e.kind(), data)), nil
}

// StreamEmitter — émetteur vers un canal de trames, borné par le contexte de la connexion.
type StreamEmitter struct {
	ctx context.Context
	out chan []byte
}

// NewStreamEmitter — `buffer` absorbe les rafales (un lot de blocs, cinq usages d'affilée) sans
// faire attendre les agents sur l'écriture réseau.
func NewStreamEmitter(ctx context.Context, buffer int) *StreamEmitter {
	return &StreamEmitter{ctx: ctx, out: make(chan []byte, buffer)}
}

func (s *StreamEmitter) Emit(e Event) {
	frame, err := FormatSSE(e)
	if err != nil {
		// Les événements sont des structs du paquet : un échec d'encodage est un bug, pas une entrée.
		log.Error().Err(err).Msg("dossier vivant: trame non émise")
		return
	}
	select {
	case s.out <- frame:
	case <-s.ctx.Done():
	}
}

// Frames — les trames, dans l'ordre d'émission. Fermé par Close.
func (s *StreamEmitter) Frames() <-chan []byte { return s.out }

// Close — à appeler une fois le run terminé (Service.Run ne rend la main qu'une fois toutes ses
// goroutines finies : plus aucun Emit ne peut suivre).
func (s *StreamEmitter) Close() { close(s.out) }

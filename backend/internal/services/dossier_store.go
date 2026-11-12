package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/redis/go-redis/v9"
)

// dossier_store.go — la conservation des dossiers remis par les runs réels du dossier vivant.
//
// QUOI : Save garde la copie serveur d'un dossier (DossierRecord) sous un identifiant neuf ; Load la
// relit. La route publique du PDF d'un run (POST /dossier-vivant/run/pdf) n'imprime que ce qui a été
// gardé ici : le navigateur ne fournit plus que l'identifiant.
//
// CHOIX :
//   - Redis, avec une durée de vie de 24 h (dossierStoreTTL). Le dossier ne vit à l'écran que dans
//     l'onglet du visiteur (aucun historique) : 24 h couvrent « je télécharge plus tard dans la
//     journée » sans garder indéfiniment le besoin d'un visiteur. Expiré = oublié par Redis.
//   - Identifiant : 128 bits de crypto/rand, en hexadécimal (32 caractères). Non devinable : un
//     dossier n'est lisible que par qui a reçu son flux. Toute autre forme est refusée avant Redis.
//   - Seul un run ARRIVÉ À LA REMISE est gardé (l'appelant n'enregistre qu'à run.completed) : un run
//     qui échoue ne laisse aucun dossier imprimable.

// dossierStoreTTL — durée de vie d'un dossier remis.
const dossierStoreTTL = 24 * time.Hour

// dossierStoreKey — préfixe des clés (versionné : un changement de forme de DossierRecord change de clé).
const dossierStoreKey = "dossiervivant:dossier:v1:"

var dossierRecordIDRe = regexp.MustCompile(`^[0-9a-f]{32}$`)

// ErrDossierNotFound — identifiant inconnu, mal formé ou expiré (404 DOSSIER_NOT_FOUND). Les trois
// cas sont indiscernables à dessein : Redis a oublié un dossier expiré, et un id mal formé n'a jamais
// désigné quoi que ce soit.
var ErrDossierNotFound = errors.New("dossier vivant: dossier inconnu ou expiré")

type DossierStore struct {
	redis *redis.Client
}

func NewDossierStore(r *redis.Client) *DossierStore { return &DossierStore{redis: r} }

// Save — garde le dossier pour dossierStoreTTL et rend son identifiant.
func (s *DossierStore) Save(ctx context.Context, rec *DossierRecord) (string, error) {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("dossier vivant: identifiant : %w", err)
	}
	id := hex.EncodeToString(raw)
	data, err := json.Marshal(rec)
	if err != nil {
		return "", fmt.Errorf("dossier vivant: encodage du dossier : %w", err)
	}
	if err := s.redis.Set(ctx, dossierStoreKey+id, data, dossierStoreTTL).Err(); err != nil {
		return "", fmt.Errorf("dossier vivant: écriture du dossier : %w", err)
	}
	return id, nil
}

// Load — le dossier `id`, ou ErrDossierNotFound.
func (s *DossierStore) Load(ctx context.Context, id string) (*DossierRecord, error) {
	if !dossierRecordIDRe.MatchString(id) {
		return nil, ErrDossierNotFound
	}
	data, err := s.redis.Get(ctx, dossierStoreKey+id).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, ErrDossierNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("dossier vivant: lecture du dossier : %w", err)
	}
	var rec DossierRecord
	if err := json.Unmarshal(data, &rec); err != nil {
		return nil, fmt.Errorf("dossier vivant: dossier illisible : %w", err)
	}
	return &rec, nil
}

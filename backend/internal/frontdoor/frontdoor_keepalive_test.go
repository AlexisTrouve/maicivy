package frontdoor

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"maicivy/internal/middleware"
)

// Régression du 500 « the server closed connection before returning the first response byte »
// (165 octets, mesuré dans le journal nginx du 11/11 sur /favicon.ico et /_next/image).
//
// QUOI : une rafale de requêtes ouvre plusieurs connexions keep-alive vers le front ; on attend
//   que le SERVEUR les ferme pour inactivité, puis une seconde rafale doit passer en 200, toute.
// POURQUOI : Node (Next standalone) ferme une connexion inactive au bout de 5 s ; fasthttp la garde
//   10 s par défaut. Entre les deux, le front-door réutilise des connexions mortes. fasthttp
//   rejoue un GET au plus 5 fois — une page qui charge une rafale d'images après une pause tire
//   plus de 5 connexions mortes du pool et la requête sort en 500. C'est pour ça que le défaut
//   était intermittent : il faut une pause de 5 à 10 s, puis une rafale.
// COMMENT : faux front en net/http avec IdleTimeout = idleServeur (le rôle des 5 s de Node,
//   rapportés à l'échelle du test) ; la pause tombe entre ce délai et les 10 s de fasthttp.
func TestFrontdoor_FrontKeepAliveFermeParLeServeur(t *testing.T) {
	const idleServeur = 3 * time.Second // > upstreamMaxIdle, < 10 s du défaut fasthttp

	frontend := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(20 * time.Millisecond) // force des connexions parallèles pendant la rafale
		_, _ = w.Write([]byte("FRONTEND"))
	}))
	frontend.Config.IdleTimeout = idleServeur
	frontend.Start()
	defer frontend.Close()

	mr, _ := miniredis.Run()
	defer mr.Close()
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	app := New(Config{
		Sus:         middleware.SusConfig{Redis: rdb},
		BackendURL:  frontend.URL,
		FrontendURL: frontend.URL,
	})

	// rafale — n requêtes simultanées, chacune d'une IP distincte (le checkpoint ne doit pas
	// throttler : on mesure le transport, pas l'anti-abus). Rend les statuts non-200.
	rafale := func(tour string, n int) []string {
		var mu sync.Mutex
		var echecs []string
		var wg sync.WaitGroup
		for i := 0; i < n; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				req := httptest.NewRequest("GET", fmt.Sprintf("/_next/image?i=%d", i), nil)
				req.Header.Set("X-Real-IP", fmt.Sprintf("10.0.%d.%d", i/250, i%250+1))
				resp, err := app.Test(req, -1)
				if err != nil {
					mu.Lock()
					echecs = append(echecs, fmt.Sprintf("%s #%d: %v", tour, i, err))
					mu.Unlock()
					return
				}
				b, _ := io.ReadAll(resp.Body)
				if resp.StatusCode != 200 {
					mu.Lock()
					echecs = append(echecs, fmt.Sprintf("%s #%d: %d %s", tour, i, resp.StatusCode, b))
					mu.Unlock()
				}
			}(i)
		}
		wg.Wait()
		return echecs
	}

	if e := rafale("1re rafale", 30); len(e) > 0 {
		t.Fatalf("la 1re rafale (connexions neuves) doit passer : %v", e)
	}
	time.Sleep(idleServeur + 700*time.Millisecond) // le serveur a fermé ses connexions inactives
	if e := rafale("2e rafale", 30); len(e) > 0 {
		t.Fatalf("%d requêtes sur 30 en échec après la pause (connexions mortes réutilisées) :\n%v", len(e), e)
	}
}

package middleware

// Régression du panic de prod du 2026-10-11 (14:16 UTC) : les goroutines analytics recevaient
// c.Context(), le *fasthttp.RequestCtx de la requête. fasthttp le recycle dès que le handler a
// rendu ; la goroutine, encore en vol, appelait Done() dessus → nil pointer → SIGSEGV, process tué.
// Le test vérifie la CAUSE de façon déterministe (le type du contexte transmis), pas une course.

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"

	"maicivy/internal/models"
)

// appelObserve : le contexte reçu, et son état AU MOMENT de l'appel (après, le cancel différé de
// l'appelant l'a légitimement annulé).
type appelObserve struct {
	ctx context.Context
	err error
}

// puitsEspion enregistre le contexte reçu par chaque écriture analytics.
type puitsEspion struct{ appels chan appelObserve }

func (p *puitsEspion) MarkVisitorActive(ctx context.Context, _ uuid.UUID) error {
	p.appels <- appelObserve{ctx, ctx.Err()}
	return nil
}

func (p *puitsEspion) TrackEvent(ctx context.Context, _ *models.AnalyticsEvent) error {
	p.appels <- appelObserve{ctx, ctx.Err()}
	return nil
}

func TestAnalytics_GoroutinesDetacheesDuContexteDeRequete(t *testing.T) {
	espion := &puitsEspion{appels: make(chan appelObserve, 4)}
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("visitor_id", uuid.New())
		return c.Next()
	})
	app.Use(NewAnalytics(espion).Handler())
	app.Get("/*", func(c *fiber.Ctx) error { return c.SendString("ok") })

	// Une page (pageview) : déclenche les DEUX goroutines (visiteur actif + événement).
	resp, err := app.Test(httptest.NewRequest("GET", "/fr/cv", nil))
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode)

	for i := 0; i < 2; i++ {
		select {
		case a := <-espion.appels:
			_, estRequete := a.ctx.(*fasthttp.RequestCtx)
			require.False(t, estRequete, "contexte de requête fasthttp transmis à une goroutine : recyclé après la réponse → panic")
			_, aUnDelai := a.ctx.Deadline()
			require.True(t, aUnDelai, "une écriture de fond doit être bornée dans le temps")
			require.NoError(t, a.err, "le contexte ne doit pas être annulé par la fin de la requête")
		case <-time.After(2 * time.Second):
			t.Fatalf("écriture analytics %d jamais reçue", i+1)
		}
	}
}

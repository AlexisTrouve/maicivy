package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http/httptest"
	"testing"

	"maicivy/internal/config"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// TestExportPDF_AccepteZh : /cv/export?lang=zh passe la validation de langue et demande le CV
// zh au service (l'erreur simulée du service court-circuite chromedp, inutile ici).
func TestExportPDF_AccepteZh(t *testing.T) {
	app := fiber.New()
	m := new(MockCVService)
	app.Get("/cv/export", (&CVHandler{cvService: m}).ExportPDF)
	m.On("GetAdaptiveCV", mock.Anything, "fullstack", "zh").Return(nil, errors.New("sentinelle"))

	resp, err := app.Test(httptest.NewRequest("GET", "/cv/export?lang=zh", nil))
	assert.NoError(t, err)
	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), "sentinelle", "zh doit atteindre le service, pas être refusé")
	m.AssertExpectations(t)
}

// TestExportPDF_RefuseDe : une langue sans gabarit PDF reste refusée explicitement.
func TestExportPDF_RefuseDe(t *testing.T) {
	app := fiber.New()
	app.Get("/cv/export", (&CVHandler{cvService: new(MockCVService)}).ExportPDF)
	resp, err := app.Test(httptest.NewRequest("GET", "/cv/export?lang=de", nil))
	assert.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)
}

// TestListCVs_ListeZh : le catalogue expose une entrée zh par thème.
func TestListCVs_ListeZh(t *testing.T) {
	app := fiber.New()
	m := new(MockCVService)
	m.On("GetAvailableThemes").Return([]config.CVTheme{{ID: "fullstack", Name: "Full-Stack"}})
	app.Get("/cv/list", (&CVHandler{cvService: m}).ListCVs)

	resp, err := app.Test(httptest.NewRequest("GET", "/cv/list", nil))
	assert.NoError(t, err)
	var out struct {
		Cvs []struct{ Lang, PDFURL string } `json:"cvs"`
	}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
	langs := []string{}
	for _, e := range out.Cvs {
		langs = append(langs, e.Lang)
	}
	assert.ElementsMatch(t, []string{"fr", "en", "zh"}, langs)
}

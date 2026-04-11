package services

import (
	"context"
	"encoding/base64"
	"fmt"
	"html/template"
	"io"
	"strings"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	"github.com/rs/zerolog/log"

	"maicivy/internal/models"
)

type PDFLetterService struct {
	templates *template.Template
}

func NewPDFLetterService(templatesPath string) (*PDFLetterService, error) {
	// Load templates
	tmpl, err := template.ParseGlob(templatesPath + "/*.html")
	if err != nil {
		return nil, fmt.Errorf("failed to load templates: %w", err)
	}

	return &PDFLetterService{
		templates: tmpl,
	}, nil
}

// GeneratePDF : génère PDF d'une lettre
func (s *PDFLetterService) GeneratePDF(ctx context.Context, letter models.LetterResponse, writer io.Writer) error {
	// 1. Render HTML from template
	html, err := s.renderHTML(letter)
	if err != nil {
		return fmt.Errorf("failed to render HTML: %w", err)
	}

	// 2. Convert HTML to PDF via chromedp
	return s.htmlToPDF(ctx, html, writer)
}

// renderHTML : génère HTML depuis template
func (s *PDFLetterService) renderHTML(letter models.LetterResponse) (string, error) {
	templateName := "letter_motivation.html"
	if letter.Type == models.LetterTypeAntiMotivation {
		templateName = "letter_anti_motivation.html"
	}

	var buf strings.Builder
	data := struct {
		Content     string
		CompanyName string
		Date        string
		Type        string
	}{
		Content:     letter.Content,
		CompanyName: letter.CompanyInfo.Name,
		Date:        letter.GeneratedAt.Format("02 January 2006"),
		Type:        string(letter.Type),
	}

	if err := s.templates.ExecuteTemplate(&buf, templateName, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// htmlToPDF : convertit HTML en PDF via Chrome headless
func (s *PDFLetterService) htmlToPDF(ctx context.Context, html string, writer io.Writer) error {
	// --no-sandbox requis en environnement Docker (pas de namespaces PID/réseau disponibles)
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-setuid-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
	)
	allocCtx, cancel := chromedp.NewExecAllocator(ctx, opts...)
	defer cancel()

	// Create chromedp context
	allocCtx, cancel = chromedp.NewContext(allocCtx)
	defer cancel()

	// Timeout protection
	allocCtx, cancel = context.WithTimeout(allocCtx, 30*time.Second)
	defer cancel()

	var pdfBuf []byte

	// Execute chromedp tasks
	err := chromedp.Run(allocCtx,
		chromedp.Navigate("about:blank"),
		chromedp.ActionFunc(func(ctx context.Context) error {
			// Encoder HTML en base64 pour préserver UTF-8
			encoded := base64.StdEncoding.EncodeToString([]byte(html))
			script := fmt.Sprintf(`
				const bytes = Uint8Array.from(atob('%s'), c => c.charCodeAt(0));
				const html = new TextDecoder('utf-8').decode(bytes);
				document.open('text/html', 'replace');
				document.write(html);
				document.close();
			`, encoded)
			return chromedp.Evaluate(script, nil).Do(ctx)
		}),
		chromedp.ActionFunc(func(ctx context.Context) error {
			// Attendre que le document soit chargé
			time.Sleep(500 * time.Millisecond)
			return nil
		}),
		chromedp.ActionFunc(func(ctx context.Context) error {
			// Générer PDF avec options pour préserver le style
			var err error
			pdfBuf, _, err = page.PrintToPDF().
				WithPrintBackground(true).
				WithPreferCSSPageSize(true).
				Do(ctx)
			return err
		}),
	)

	if err != nil {
		log.Error().Err(err).Msg("chromedp error during PDF generation")
		return fmt.Errorf("chromedp error: %w", err)
	}

	// Write PDF to output
	_, err = writer.Write(pdfBuf)
	return err
}

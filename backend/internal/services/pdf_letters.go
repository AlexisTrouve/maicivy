package services

import (
	"context"
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
	// Create chromedp context
	allocCtx, cancel := chromedp.NewContext(ctx)
	defer cancel()

	// Timeout protection
	allocCtx, cancel = context.WithTimeout(allocCtx, 30*time.Second)
	defer cancel()

	var pdfBuf []byte

	// Execute chromedp tasks
	err := chromedp.Run(allocCtx,
		chromedp.Navigate("about:blank"),
		chromedp.ActionFunc(func(ctx context.Context) error {
			// Set HTML content
			script := fmt.Sprintf(`
				document.open();
				document.write(%s);
				document.close();
			`, escapeJSString(html))

			return chromedp.Evaluate(script, nil).Do(ctx)
		}),
		chromedp.ActionFunc(func(ctx context.Context) error {
			// Wait for page load
			time.Sleep(500 * time.Millisecond)
			return nil
		}),
		chromedp.ActionFunc(func(ctx context.Context) error {
			// Print to PDF
			var err error
			pdfBuf, _, err = page.PrintToPDF().Do(ctx)
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

// escapeJSString : escape string pour JS
func escapeJSString(s string) string {
	// Simple escaping for JavaScript template literal
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "`", "\\`")
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\r", "\\r")
	s = strings.ReplaceAll(s, "$", "\\$")
	return "`" + s + "`"
}

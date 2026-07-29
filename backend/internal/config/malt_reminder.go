package config

import (
	"os"
	"strings"
	"time"
)

// MaltReminderConfig — rappel email périodique (2x/semaine par défaut) pour penser à mettre à jour
// la disponibilité sur Malt (aucune API Malt pour l'automatiser — geste manuel qu'on oublie
// facilement). To vide → feature désactivée, même convention que le reste de maicivy.
type MaltReminderConfig struct {
	To        string
	FromEmail string
	FromName  string
	Days      []time.Weekday
	HourUTC   int
}

var weekdayNames = map[string]time.Weekday{
	"sun": time.Sunday, "mon": time.Monday, "tue": time.Tuesday, "wed": time.Wednesday,
	"thu": time.Thursday, "fri": time.Friday, "sat": time.Saturday,
}

func LoadMaltReminderConfig() *MaltReminderConfig {
	return &MaltReminderConfig{
		To:        os.Getenv("MALT_REMINDER_TO"),
		FromEmail: getEnvOrDefault("MALT_REMINDER_FROM_EMAIL", "mailbox@etheryale.com"),
		FromName:  getEnvOrDefault("MALT_REMINDER_FROM_NAME", "Mailbox Etheryale"),
		// Défaut : lundi + jeudi, 1h UTC = 9h Asia/Shanghai — étalé sur la semaine, avant le début
		// de journée. Ajustable sans rebuild via MALT_REMINDER_DAYS ("mon,thu") / _HOUR_UTC.
		Days:    parseWeekdays(getEnvOrDefault("MALT_REMINDER_DAYS", "mon,thu")),
		HourUTC: getEnvAsIntOrDefault("MALT_REMINDER_HOUR_UTC", 1),
	}
}

// Configured dit si le rappel a un destinataire — sinon désactivé proprement (pas de job démarré).
func (c *MaltReminderConfig) Configured() bool { return c.To != "" }

// parseWeekdays parse une liste CSV de jours ("mon,thu") — jours invalides/inconnus ignorés
// silencieusement plutôt que de planter au démarrage sur une valeur d'env mal formée.
func parseWeekdays(csv string) []time.Weekday {
	var days []time.Weekday
	for _, p := range strings.Split(csv, ",") {
		p = strings.ToLower(strings.TrimSpace(p))
		if d, ok := weekdayNames[p]; ok {
			days = append(days, d)
		}
	}
	return days
}

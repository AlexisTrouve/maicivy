package models

import (
	"github.com/lib/pq"
)

// SkillLevel représente le niveau de maîtrise
type SkillLevel string

const (
	SkillLevelBeginner     SkillLevel = "beginner"
	SkillLevelIntermediate SkillLevel = "intermediate"
	SkillLevelAdvanced     SkillLevel = "advanced"
	SkillLevelExpert       SkillLevel = "expert"
)

// Skill représente une compétence technique
type Skill struct {
	BaseModel

	// Informations principales
	Name            string         `gorm:"type:varchar(100);not null;uniqueIndex" json:"name" validate:"required,min=2,max=100"`
	Level           SkillLevel     `gorm:"type:varchar(20);not null" json:"level" validate:"required,oneof=beginner intermediate advanced expert"`
	Category        string         `gorm:"type:varchar(100);index" json:"category" validate:"required"`
	Tags            pq.StringArray `gorm:"type:text[]" json:"tags"`
	YearsExperience int            `gorm:"default:0" json:"yearsExperience" validate:"min=0,max=50"`

	// Métadonnées
	Description string `gorm:"type:text" json:"description" validate:"max=500"`
	Featured    bool   `gorm:"default:false" json:"featured"`
	Icon        string `gorm:"type:varchar(100)" json:"icon"` // Icon name (ex: "golang", "react")
}

// TableName override le nom de table par défaut
func (Skill) TableName() string {
	return "skills"
}

// LevelScore retourne un score numérique pour le niveau
func (s *Skill) LevelScore() int {
	switch s.Level {
	case SkillLevelExpert:
		return 4
	case SkillLevelAdvanced:
		return 3
	case SkillLevelIntermediate:
		return 2
	case SkillLevelBeginner:
		return 1
	default:
		return 0
	}
}

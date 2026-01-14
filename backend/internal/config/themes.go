package config

// CVTheme représente un thème de CV avec ses tags prioritaires
type CVTheme struct {
	ID          string             `json:"id"`
	Name        string             `json:"name"`
	Description string             `json:"description"`
	Icon        string             `json:"icon"`               // Emoji or icon identifier
	TagWeights  map[string]float64 `json:"tagWeights"` // tag → poids (0.0-1.0)
}

// GetAvailableThemes retourne tous les thèmes configurés
func GetAvailableThemes() map[string]CVTheme {
	return map[string]CVTheme{
		"backend": {
			ID:          "backend",
			Name:        "Backend Developer",
			Description: "Focus sur développement backend, APIs, bases de données",
			Icon:        "\U0001F5A5",
			TagWeights: map[string]float64{
				"go":            1.0,
				"api":           1.0,
				"backend":       1.0,
				"postgresql":    0.9,
				"redis":         0.9,
				"docker":        0.8,
				"microservices": 0.8,
				"restful":       0.8,
				"grpc":          0.7,
				"kubernetes":    0.7,
				"nodejs":        0.6,
				"python":        0.5,
			},
		},
		"cpp": {
			ID:          "cpp",
			Name:        "C++ Developer",
			Description: "Focus sur développement C++, systèmes bas niveau",
			Icon:        "\U0001F527",
			TagWeights: map[string]float64{
				"c++":          1.0,
				"cpp":          1.0,
				"c":            0.9,
				"systems":      0.9,
				"embedded":     0.8,
				"performance":  0.8,
				"optimization": 0.8,
				"low-level":    0.8,
				"memory":       0.7,
				"algorithms":   0.7,
				"qt":           0.6,
				"boost":        0.6,
			},
		},
		"artistique": {
			ID:          "artistique",
			Name:        "Creative & Artistic",
			Description: "Focus sur projets créatifs, design, visualisation",
			Icon:        "\U0001F3A8",
			TagWeights: map[string]float64{
				"design":      1.0,
				"art":         1.0,
				"creative":    1.0,
				"ui":          0.9,
				"ux":          0.9,
				"3d":          0.9,
				"graphics":    0.9,
				"animation":   0.8,
				"threejs":     0.8,
				"webgl":       0.8,
				"photoshop":   0.7,
				"illustrator": 0.7,
				"blender":     0.7,
			},
		},
		"fullstack": {
			ID:          "fullstack",
			Name:        "Full-Stack Developer",
			Description: "Focus sur développement full-stack, frontend + backend",
			Icon:        "\U0001F4BB",
			TagWeights: map[string]float64{
				"fullstack":  1.0,
				"frontend":   0.9,
				"backend":    0.9,
				"react":      0.9,
				"nextjs":     0.9,
				"typescript": 0.9,
				"api":        0.8,
				"nodejs":     0.8,
				"go":         0.8,
				"postgresql": 0.7,
				"tailwind":   0.7,
				"docker":     0.6,
			},
		},
		"devops": {
			ID:          "devops",
			Name:        "DevOps Engineer",
			Description: "Focus sur infrastructure, CI/CD, monitoring",
			Icon:        "\U0001F433",
			TagWeights: map[string]float64{
				"devops":     1.0,
				"docker":     1.0,
				"kubernetes": 1.0,
				"ci/cd":      1.0,
				"terraform":  0.9,
				"ansible":    0.9,
				"aws":        0.9,
				"monitoring": 0.8,
				"prometheus": 0.8,
				"grafana":    0.8,
				"nginx":      0.7,
				"jenkins":    0.7,
				"gitops":     0.7,
			},
		},
	}
}

// GetTheme retourne un thème spécifique ou nil
func GetTheme(themeID string) *CVTheme {
	themes := GetAvailableThemes()
	if theme, exists := themes[themeID]; exists {
		return &theme
	}
	return nil
}

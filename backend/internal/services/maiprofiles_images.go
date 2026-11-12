package services

import "context"

// MPFStoreImage — une entrée du store d'images de maiProFiles (GET /images).
//
// Le store tient les captures, diagrammes et vidéos téléversés ; une entrée rattachée à un
// projet porte `linked_to.type == "project"` et l'id du projet. Le fichier se sert par
// `/images/<id>` (206 sur les requêtes Range, ce qui rend les vidéos lisibles partout).
type MPFStoreImage struct {
	ID       string `json:"id"`
	Filename string `json:"filename"`
	// OriginalName — nom du fichier téléversé (« Grove.jpg »). Sert à reconnaître une capture déjà
	// présente dans le livre sous son propre nom (`/portfolio-images/Grove.jpg`).
	OriginalName string `json:"original_name"`
	ContentType  string `json:"content_type"`
	Alt          string `json:"alt"`
	LinkedTo     struct {
		Type    string `json:"type"`
		ID      string `json:"id"`
		Section string `json:"section"`
	} `json:"linked_to"`
}

// ListImages — tout le store en UNE requête.
//
// POURQUOI pas `/images?linked_to=<id>` par projet : 75 requêtes par rafraîchissement sur le
// worker unique de maiProFiles, alors que le store complet tient en une réponse (~390 entrées).
// Le tri par projet se fait chez nous.
func (c *MaiProFilesClient) ListImages(ctx context.Context) ([]MPFStoreImage, error) {
	var images []MPFStoreImage
	if err := c.get(ctx, "/images", "", &images); err != nil {
		return nil, err
	}
	return images, nil
}

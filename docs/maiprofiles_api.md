# maiProFiles API — intégration maicivy

Base URL : `https://maiprofiles.etheryale.com`
Credentials : `../maiProFiles/.env` → `API_KEY` pour les writes.

## Endpoints utiles pour maicivy

### Page About / Header du CV
```http
GET /profile
```
Retourne `name`, `headline`, `bio.short`, `bio.full`, `skills`, `experience_years`, `links`.
Utiliser `bio.short` pour le tagline, `skills.strong` pour les badges.

### Projets affichés sur le CV / portfolio section
```http
GET /projects                        # liste (sans description longue)
GET /projects/{id}                   # détail complet avec description.portfolio
GET /search?q=rust                   # filtrer par techno / tag
```

### Images de projet
```http
GET /images?linked_to={project_id}           # toutes les images d'un projet
GET /images?linked_to={id}&section=hero      # image principale
GET /images/{image_id}                       # binaire direct (src="...")
```

## Exemple — récupérer les projets "production" avec leur image hero
```python
import httpx

base = "https://maiprofiles.etheryale.com"
projects = httpx.get(f"{base}/projects").json()
prod = [p for p in projects if p.get("status") == "production"]

for p in prod:
    images = httpx.get(f"{base}/images", params={"linked_to": p["id"], "section": "hero"}).json()
    p["hero_url"] = f"{base}/images/{images[0]['id']}" if images else None
```

## Stats globales (pour le header CV)
```http
GET /stats
# → {"projects": 23, "total_loc": 95000, "stack": {"python": 12, "rust": 8, ...}}
```

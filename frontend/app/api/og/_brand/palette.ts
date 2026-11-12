/**
 * Palette « montagne de nuit » des images générées (aperçus de lien).
 *
 * POURQUOI des littéraux et pas les variables CSS du site : ImageResponse (satori) ne lit ni
 * globals.css ni Tailwind. Les valeurs sont donc recopiées ICI, une fois, depuis leurs sources —
 * `--background` / `--foreground` / `--muted-foreground` (globals.css, surface publique), l'or et
 * les surfaces de tailwind.config.ts, et le bandeau du CV sombre (backend, #0c1424). Une teinte du
 * site qui change doit changer ici aussi : l'aperçu est la première image du site qu'on voit.
 */
export const night = {
  /** Fond de page (= surface.base, = page du CV). */
  page: '#060911',
  /** Bandeau du CV : le ciel un cran plus clair, pour le halo derrière le sommet. */
  band: '#0c1424',
  /** Carte posée (= cartes du CV) : fond des tuiles. */
  card: '#111a2b',
} as const;

export const ink = {
  /** Texte principal : hsl(41 48% 88%), le crème des titres serif du site. */
  cream: '#efe6d2',
  /** Texte secondaire : hsl(212 9% 63.7%). */
  muted: '#9ca3ac',
} as const;

/** L'or est un SIGNAL (cf. tailwind.config.ts) : filets, sommet, accroche — jamais un aplat. */
export const gold = {
  base: '#d9b45f',
  soft: '#ecd8a4',
  dim: '#7a6329',
  bright: '#f0d590',
} as const;

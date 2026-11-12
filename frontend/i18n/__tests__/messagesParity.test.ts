import fr from '../../messages/fr.json';
import en from '../../messages/en.json';
import de from '../../messages/de.json';
import itMsgs from '../../messages/it.json';
import zh from '../../messages/zh.json';

// Garde-fou des 5 fichiers de messages (FR = source).
// POURQUOI : une clé manquante s'affiche brute à l'écran, un placeholder ICU renommé casse
// l'interpolation, et une valeur restée en français/anglais dans zh met en échec une lectrice
// sinophone (panel admin utilisé au quotidien). Ces trois défauts passent inaperçus en revue de diff.

type Dict = Record<string, unknown>;
const locales: Record<string, Dict> = { fr, en, de, it: itMsgs, zh };

// Aplatit l'arbre en { "a.b.0.c": "valeur" } — les tableaux (chat.hints, tech[]) inclus.
function flatten(o: unknown, prefix = '', out: Record<string, string> = {}): Record<string, string> {
  if (typeof o === 'string') out[prefix] = o;
  else if (o && typeof o === 'object') {
    for (const [k, v] of Object.entries(o as Dict)) flatten(v, prefix ? `${prefix}.${k}` : k, out);
  }
  return out;
}

// Noms d'arguments ICU de premier niveau : `{name}` et `{count, plural, …}`. Les branches de
// pluriel (`{moins d’un an…}`, `{# ans}`) ne matchent pas (espace ou non-ASCII après l'accolade).
function placeholders(s: string): string[] {
  return Array.from(new Set(Array.from(s.matchAll(/\{(\w+)\s*[,}]/g), (m) => m[1]))).sort();
}

const flat = Object.fromEntries(Object.entries(locales).map(([l, d]) => [l, flatten(d)]));
const frKeys = Object.keys(flat.fr).sort();

// Mots latins tolérés dans zh : marques, noms propres, sigles et termes tech que le chinois
// technique garde tels quels. Tout autre mot latin = chaîne non traduite.
const ZH_LATIN_WHITELIST = new Set(
  (
    'AI API Alexi Alexis Trouvé Anthropic Aria C++ CSP CTO Chart.js Claude Clearbit Cloudflare Cogesco ' +
    'Cookie DevOps Discord Docker Drifterra DuckDuckGo Enter Fiber Framer Motion GDPR GIF GORM GPT-4 Git ' +
    'GitHub GitLab Gitea Go Google HSTS HttpOnly IP JPEG LinkedIn Malt Markdown Microsoft Next.js Node ORM ' +
    'OWASP Top PDF PNG PostgreSQL RSS React Redis SHA256 SameSite Secure Token TypeScript URL Upwork ' +
    'User-Agent VPS WebP WebSocket Wikipedia X-Frame-Options XSS admin bluemonday maiProFiles maicivy ' +
    'maicivy_session malt set_draft set_rate you@email.com vs ' +
    // Profils du CV (cv.themes) : « Technomancer » est la signature du positionnement, gardée en
    // anglais comme `montagne.voice.tagline` ; MCP et CI/CD sont des sigles techniques.
    'Technomancer MCP CI CD ' +
    // Passe 2 de la refonte : exemples d'entreprises (lettres), sigles et noms d'outils du schéma
    // d'architecture, que le chinois technique garde en latin.
    'Mistral Doctolib CDN Web TLS HTTP JSON App Router Radix SQL SHA-256 services scraper.go'
  ).split(' '),
);
// Clés dont la valeur est un identifiant interne (jamais affiché) ou une marque.
// `montagne.voice.tagline` : signature de titre (« Agent Orchestrator — Technomancer ») gardée en
// anglais dans les 5 langues, comme une marque — décision de la maquette AC-2.
const ZH_SKIP_KEY = (k: string) => k.endsWith('.category') || k === 'admin.brand' || k === 'montagne.voice.tagline';

describe('messages — parité des 5 locales', () => {
  it.each(['en', 'de', 'it', 'zh'])('%s a exactement les clés de fr', (l) => {
    expect(Object.keys(flat[l]).sort()).toEqual(frKeys);
  });

  it.each(['en', 'de', 'it', 'zh'])('%s garde les placeholders ICU de fr', (l) => {
    const mismatches = frKeys
      .filter((k) => placeholders(flat.fr[k]).join() !== placeholders(flat[l][k] ?? '').join())
      .map((k) => `${k}: fr=${placeholders(flat.fr[k])} ${l}=${placeholders(flat[l][k] ?? '')}`);
    expect(mismatches).toEqual([]);
  });

  it('zh ne contient aucun mot latin hors liste blanche', () => {
    const offenders: string[] = [];
    for (const [k, v] of Object.entries(flat.zh)) {
      if (ZH_SKIP_KEY(k)) continue;
      // Retire la syntaxe ICU (`{x}`, `{count, plural, =0 {` / `other {`) et les balises de texte
      // riche (`<unit>…</unit>`, rendues par t.rich) avant d'inspecter le texte.
      const text = v
        .replace(/\{\w+(\s*,\s*\w+\s*,?)?\s*\}?/g, ' ')
        .replace(/(=\d+|other|one)\s*\{/g, ' ')
        .replace(/<\/?\w+>/g, ' ');
      // Lettres latines accentuées incluses, mais pas × (U+00D7) ni ÷ (U+00F7), qui sont des symboles.
      for (const m of text.matchAll(/[A-Za-zÀ-ÖØ-öø-ÿ][\w@.+#&'’À-ÖØ-öø-ÿ-]*/g)) {
        const word = m[0].replace(/[.']+$/, '');
        if (!ZH_LATIN_WHITELIST.has(word)) offenders.push(`${k}: «${word}» dans «${v}»`);
      }
    }
    expect(offenders).toEqual([]);
  });
});

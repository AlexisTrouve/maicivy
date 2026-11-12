import type { MetadataRoute } from 'next';
import { locales, defaultLocale } from '@/i18n/config';
import { publishedRoutes } from '@/lib/publication';

// Sitemap dynamique servi sur /sitemap.xml (déclaré dans public/robots.txt, qui pointait jusque-là
// vers un 404). POURQUOI : Google découvre/indexe via le sitemap ; sans lui, il crawle à l'aveugle
// et rate des pages (notamment les articles de blog). On y déclare AUSSI le hreflang par URL
// (alternates.languages) — la méthode recommandée par Google pour un site multilingue, et qui évite
// le footgun d'un canonical statique au niveau layout (qui dé-indexerait /cv, /blog…).

const BASE = process.env.NEXT_PUBLIC_BASE_URL || 'https://maicivy.etheryale.com';

// Routes statiques PUBLIQUES indexables. On exclut /admin (privé) et les outils internes
// (analytics, gitstats, letters) qui n'ont pas de valeur SEO. Les pages à bascule (lib/publication.ts)
// n'y entrent qu'une fois publiées : même drapeau que leur noindex.
const STATIC_ROUTES = ['', 'cv', 'chat', 'blog', 'portfolio', ...publishedRoutes()];

// languagesFor : map hreflang {fr, en, de, it, zh, x-default} pour un sous-chemin donné. Chaque URL
// du sitemap déclare ainsi ses équivalents dans les autres langues → Google les relie au lieu de les
// traiter comme du contenu dupliqué. x-default = la locale par défaut (servie aux langues non ciblées).
function languagesFor(path: string): Record<string, string> {
  const suffix = path ? `/${path}` : '';
  const langs: Record<string, string> = {};
  for (const l of locales) {
    langs[l] = `${BASE}/${l}${suffix}`;
  }
  langs['x-default'] = `${BASE}/${defaultLocale}${suffix}`;
  return langs;
}

// fetchBlogPosts : liste les articles publiés (slug + date de MAJ) pour les inclure dans le sitemap.
// On tape l'API publique same-origin. Cache 1h (revalidate) : le sitemap n'a pas besoin d'être temps
// réel, et ça évite un appel API à chaque crawl. Échec → liste vide (le sitemap reste valide sans le
// blog plutôt que de planter le rendu).
//
// TOUTES les pages : l'API borne per_page à 50 et ramène toute valeur hors borne à 10. Mesuré le
// 2026-10-11 : l'ancien `per_page=100` ne déclarait que 10 articles sur 138 à Google. On parcourt
// donc les pages (50 par page) jusqu'à total_pages. Une page en échec → liste partielle jusque-là.
const SITEMAP_PAGE = 50;
async function fetchBlogPosts(): Promise<{ slug: string; updatedAt: string }[]> {
  const out: { slug: string; updatedAt: string }[] = [];
  try {
    for (let page = 1; ; page++) {
      const res = await fetch(`${BASE}/api/v1/blog/posts?page=${page}&per_page=${SITEMAP_PAGE}`, {
        next: { revalidate: 3600 },
      });
      if (!res.ok) break;
      const data = await res.json();
      const posts: { slug?: string; updated_at?: string }[] = data.posts || [];
      for (const p of posts) {
        if (p.slug) out.push({ slug: p.slug, updatedAt: p.updated_at || '' });
      }
      if (posts.length === 0 || page >= (data.total_pages || 1)) break;
    }
  } catch {
    // Panne réseau : on garde ce qui a été lu.
  }
  return out;
}

// fetchPortfolio : liste les projets CURÉS pour que chaque page projet soit indexable.
// POURQUOI ça compte : ces pages sont la seule forme indexable du contenu projet — ailleurs il ne
// vit que dans le chat, invisible pour un moteur. Même discipline que le blog : échec → liste vide,
// le sitemap reste valide sans le portfolio plutôt que de faire planter le rendu.
async function fetchPortfolioProjects(): Promise<string[]> {
  try {
    const res = await fetch(`${BASE}/api/v1/portfolio`, { next: { revalidate: 3600 } });
    if (!res.ok) return [];
    const data = await res.json();
    return (data.projects || [])
      .map((p: { id?: string }) => p.id)
      .filter((id: string | undefined): id is string => Boolean(id));
  } catch {
    return [];
  }
}

export default async function sitemap(): Promise<MetadataRoute.Sitemap> {
  const now = new Date();
  const entries: MetadataRoute.Sitemap = [];

  // 1. Routes statiques × locales (home en priorité maximale).
  for (const path of STATIC_ROUTES) {
    const languages = languagesFor(path);
    for (const l of locales) {
      entries.push({
        url: `${BASE}/${l}${path ? `/${path}` : ''}`,
        lastModified: now,
        changeFrequency: path === '' ? 'weekly' : 'monthly',
        priority: path === '' ? 1.0 : 0.8,
        alternates: { languages },
      });
    }
  }

  // 2. Articles de blog × locales (le moteur de contenu — chaque article doit être indexable).
  const posts = await fetchBlogPosts();
  for (const post of posts) {
    const path = `blog/${post.slug}`;
    const languages = languagesFor(path);
    const lastModified = post.updatedAt ? new Date(post.updatedAt) : now;
    for (const l of locales) {
      entries.push({
        url: `${BASE}/${l}/${path}`,
        lastModified,
        changeFrequency: 'monthly',
        priority: 0.7,
        alternates: { languages },
      });
    }
  }

  // 3. Pages projet du portfolio × locales.
  const projets = await fetchPortfolioProjects();
  for (const id of projets) {
    const path = `portfolio/${id}`;
    const languages = languagesFor(path);
    for (const l of locales) {
      entries.push({
        url: `${BASE}/${l}/${path}`,
        lastModified: now,
        changeFrequency: 'monthly',
        priority: 0.7,
        alternates: { languages },
      });
    }
  }

  return entries;
}

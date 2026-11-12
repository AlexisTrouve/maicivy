import Link from 'next/link';
import { getTranslations } from 'next-intl/server';
import { blogHref } from './blogHref';

/**
 * QUOI : filtrage des articles par thème curated.
 *
 * POURQUOI des liens et plus des boutons : le filtre et la pagination sont désormais calculés
 * côté backend sur le catalogue ENTIER (avant : filtrage client sur les 50 premiers articles
 * seulement). Chaque filtre est une URL (?theme=), rendue côté serveur et partageable.
 *
 * COMMENT : composant « bête » — il reçoit les thèmes déjà comptés (sur tout le catalogue), le
 * thème actif et le total. Choisir un thème repart toujours de la page 1. Cliquer le thème actif
 * le retire. Les data-testid sont stables pour l'E2E (chaque lien : libellé puis compteur, deux
 * <span> — le test lit le dernier).
 *
 * FORME : téléphone d'abord — une rangée d'onglets au-dessus de la grille ; ≥ lg, une colonne à
 * gauche. Le thème actif est « où l'on est » : la seule place de l'or ici, un filet (dessous sur
 * téléphone, à gauche en colonne), jamais un aplat.
 */

export interface TagCount {
  tag: string; // clé stable du thème (ex : "games") — aussi utilisée pour le data-testid
  count: number;
}

interface TagFilterProps {
  tags: TagCount[];
  selected: string | null;
  total: number;
  locale: string;
}

const baseLien =
  'relative inline-flex min-h-11 shrink-0 items-center justify-between gap-3 rounded-md px-3 text-sm transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring lg:flex lg:w-full';
// Filet d'or : sous l'onglet sur téléphone, sur l'arête gauche en colonne (≥ lg).
const actif =
  "text-foreground after:absolute after:inset-x-3 after:bottom-0 after:h-px after:bg-gold after:content-[''] lg:bg-secondary lg:after:inset-x-auto lg:after:inset-y-2 lg:after:left-0 lg:after:h-auto lg:after:w-px";
const repos = 'text-muted-foreground hover:bg-secondary/60 hover:text-foreground';

export async function TagFilter({ tags, selected, total, locale }: TagFilterProps) {
  const t = await getTranslations({ locale, namespace: 'blog' });

  const entrees = [
    { key: 'all', href: blogHref(locale), label: t('all'), count: total, active: selected === null, testId: 'tag-all' },
    ...tags.map(({ tag, count }) => ({
      key: tag,
      href: blogHref(locale, { theme: selected === tag ? null : tag }),
      label: t(`themes.${tag}`),
      count,
      active: selected === tag,
      testId: `tag-${tag}`,
    })),
  ];

  return (
    <aside data-testid="tag-filter" className="lg:w-52 lg:shrink-0">
      <div className="lg:sticky lg:top-24">
        <h2 className="font-sans text-xs font-medium uppercase tracking-[0.16em] text-muted-foreground">
          {t('filterTitle')}
        </h2>
        <nav
          aria-label={t('filterTitle')}
          className="-mx-1 mt-2 flex gap-1 overflow-x-auto border-b border-border px-1 lg:mx-0 lg:mt-3 lg:flex-col lg:overflow-visible lg:border-b-0 lg:px-0"
        >
          {entrees.map((e) => (
            <Link
              key={e.key}
              href={e.href}
              data-testid={e.testId}
              aria-current={e.active ? 'page' : undefined}
              className={`${baseLien} ${e.active ? actif : repos}`}
            >
              <span className="truncate">{e.label}</span>
              <span className="text-xs tabular-nums text-muted-foreground">{e.count}</span>
            </Link>
          ))}
        </nav>
      </div>
    </aside>
  );
}

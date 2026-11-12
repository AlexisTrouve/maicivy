import { getTranslations } from 'next-intl/server';

import type { Montagne, MontagneNode } from '@/lib/montagne-api';
import { monogramSvg } from '@/lib/montagne/monogram';
import { portfolioImageUrl } from '@/lib/portfolio-api';

/**
 * MontagneList — la montagne en texte : tous les projets, strate par strate, avec leurs liens.
 *
 * QUOI : liste rendue côté serveur, dans le HTML initial. Chaque projet porte son icône, son nom,
 * sa mini-description, son statut, ce qu'il sert à construire et ce avec quoi il est construit
 * (ancres internes), et le lien vers sa page portfolio s'il est curé.
 * POURQUOI : c'est la forme indexable de la montagne (SEO), la version sans JavaScript, et la
 * version lisible au lecteur d'écran. Le canvas n'en est que la mise en relief.
 * E2E : `data-project-id` / `data-curated` / `data-stratum` sur chaque entrée — le test y lit les
 * données au lieu de les coder en dur.
 */
export async function MontagneList({ locale, montagne }: { locale: string; montagne: Montagne }) {
  const t = await getTranslations({ locale, namespace: 'montagne' });

  const byId = new Map(montagne.nodes.map((n) => [n.id, n]));
  // Index des liens dans les deux sens : « sert à construire » et « construit avec ».
  const builds = new Map<string, MontagneNode[]>();
  const builtWith = new Map<string, MontagneNode[]>();
  for (const l of montagne.links) {
    const from = byId.get(l.from);
    const to = byId.get(l.to);
    if (!from || !to) continue;
    builds.set(from.id, [...(builds.get(from.id) ?? []), to]);
    builtWith.set(to.id, [...(builtWith.get(to.id) ?? []), from]);
  }

  // Libellé de statut : clé i18n si elle existe, sinon la valeur brute (statut ajouté côté
  // maiProFiles après ce code) plutôt qu'une clé manquante affichée.
  const statusLabel = (status: string) => (t.has(`status.${status}`) ? t(`status.${status}`) : status);

  const strata = [...montagne.strata].filter((s) => s.count > 0).sort((a, b) => a.level - b.level);

  return (
    <section
      aria-labelledby="montagne-list-title"
      data-testid="montagne-list"
      className="mx-auto max-w-6xl px-4 py-14"
    >
      <h2 id="montagne-list-title" className="font-heading text-2xl font-semibold text-slate-100 md:text-3xl">
        {t('list.title')}
      </h2>
      <p className="mt-2 max-w-3xl text-sm text-slate-400">{t('list.intro')}</p>
      <p className="mt-1 text-xs text-slate-500" data-testid="montagne-list-count">
        {t('list.count', { count: montagne.nodes.length })}
      </p>

      {strata.map((stratum) => {
        const nodes = montagne.nodes
          .filter((n) => n.stratum === stratum.id)
          .sort((a, b) => a.name.localeCompare(b.name, locale));
        return (
          <div key={stratum.id} className="mt-10" data-testid="montagne-list-stratum" data-stratum={stratum.id}>
            <h3 className="flex items-baseline gap-3 border-b border-white/5 pb-2">
              <span className="font-heading text-lg font-semibold text-slate-100">{t(`strata.${stratum.id}`)}</span>
              <span className="text-xs text-slate-500">{t(`strataHint.${stratum.id}`)}</span>
            </h3>
            <ul className="mt-4 grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
              {nodes.map((node) => (
                <li
                  key={node.id}
                  id={`projet-${node.id}`}
                  data-testid="montagne-list-item"
                  data-project-id={node.id}
                  data-curated={node.curated ? 'true' : 'false'}
                  data-stratum={node.stratum}
                  className="scroll-mt-24 rounded-lg border border-white/5 bg-[#0b111c] p-4"
                >
                  <div className="flex items-start gap-3">
                    {node.icon_url ? (
                      // eslint-disable-next-line @next/next/no-img-element -- icône 40 px servie par maiProFiles : l'optimiseur n'apporte rien
                      <img src={portfolioImageUrl(node.icon_url)} alt="" width={40} height={40} className="h-10 w-10 shrink-0 rounded-full" />
                    ) : (
                      <span
                        aria-hidden="true"
                        className="block h-10 w-10 shrink-0 [&>svg]:h-10 [&>svg]:w-10"
                        // SVG produit par monogramSvg, lettres échappées : pas d'injection possible.
                        dangerouslySetInnerHTML={{ __html: monogramSvg(node.monogram, node.stratum) }}
                      />
                    )}
                    <div className="min-w-0">
                      <h4 className="font-medium leading-tight text-slate-100">{node.name}</h4>
                      <p className="mt-0.5 text-xs text-slate-500">{statusLabel(node.status)}</p>
                    </div>
                  </div>
                  {node.mini && <p className="mt-3 text-sm text-slate-300">{node.mini}</p>}
                  <RelationLine label={t('list.builds')} nodes={builds.get(node.id)} testId="montagne-list-builds" />
                  <RelationLine label={t('list.builtWith')} nodes={builtWith.get(node.id)} testId="montagne-list-built-with" />
                  {node.curated && (
                    // Ancre HTML et non le <Link> client de next-intl — mesuré le 2026-10-30 : en dev,
                    // l'hydratation de cette liste échouait 14 fois sur 36 (« Hydration failed »),
                    // TOUJOURS sur la première entrée curée, c'est-à-dire la première qui embarquait
                    // un composant client. Avec une ancre, 0 sur 12. La liste est rendue côté
                    // serveur, qui connaît déjà la locale : le <Link> n'apportait que la navigation
                    // client et la prélecture — 25 pages portfolio préchargées au défilement d'une
                    // liste de référence, un coût sans usage.
                    <a
                      href={`/${locale}/portfolio/${node.id}`}
                      data-testid="montagne-list-portfolio"
                      className="mt-3 inline-block text-sm text-slate-200 underline decoration-white/20 underline-offset-4 hover:decoration-white/60"
                    >
                      {t('list.openPortfolio')} →
                    </a>
                  )}
                </li>
              ))}
            </ul>
          </div>
        );
      })}
    </section>
  );
}

/** Une ligne de relations : libellé + ancres vers les entrées de la liste. Rien si vide. */
function RelationLine({ label, nodes, testId }: { label: string; nodes?: MontagneNode[]; testId: string }) {
  if (!nodes || nodes.length === 0) return null;
  return (
    <p className="mt-2 text-xs text-slate-400" data-testid={testId}>
      <span className="text-slate-500">{label}</span>{' '}
      {nodes.map((n, i) => (
        <span key={n.id}>
          {i > 0 && ', '}
          <a href={`#projet-${n.id}`} className="text-slate-300 hover:text-white hover:underline">
            {n.name}
          </a>
        </span>
      ))}
    </p>
  );
}

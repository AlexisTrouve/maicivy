import { Fragment, type CSSProperties } from 'react';
import Image from 'next/image';
import { getTranslations } from 'next-intl/server';

import type { Montagne, MontagneNode, MontagneSummitCard } from '@/lib/montagne-api';
import { summitCover } from '@/lib/montagne/art';
import { STRATUM_TINT } from '@/lib/montagne/monogram';
import { GEOMETRIES, layoutMountain, type MountainLayout } from '@/lib/montagne/pyramid';
import { nameSegments } from '@/lib/montagne/wordBreaks';
import { portfolioImageUrl } from '@/lib/portfolio-api';

import { MontagneBackdrop } from './MontagneBackdrop';
import styles from './montagne.module.css';

/**
 * MontagnePeak — la montagne des projets, rendue côté SERVEUR en HTML + SVG.
 *
 * QUOI : les œuvres du sommet en grandes cartes (leur ILLUSTRATION en couverture — Eikon,
 * lib/montagne/art.ts —, sinon leur icône en grand ; les PREUVES s'ouvrent dans la fleur), puis
 * tous les autres projets en tuiles carrées, en rangs qui s'élargissent vers le bas, reliés par
 * les liens réels « sert à construire » en filets dorés, posés sur la montagne peinte
 * (MontagneBackdrop) dont la pointe naît sous les cartes et le pied dans la brume dorée.
 *
 * POURQUOI sans canvas : la vue doit être instantanée (dans le HTML), accessible (de vrais liens,
 * lus dans l'ordre de la montagne) et touchable au pouce. Le canvas 3D reste une vue secondaire
 * (« vue réseau », MontagneExplorer), chargée au clic seulement.
 *
 * COMMENT :
 *  - positions précalculées pour chaque géométrie (pyramid.ts › GEOMETRIES), posées en variables
 *    CSS (--x0/--y0, --x1/--y1, --x2/--y2) ; montagne.module.css choisit par media query. Les
 *    filets, eux, ont un SVG par géométrie (on ne peut pas mettre une variable CSS dans un
 *    attribut `d`) — ils sont décoratifs (aria-hidden) : les liens se LISENT dans la liste
 *    (MontagneList). La roche n'est plus dessinée : c'est l'illustration de fond.
 *  - chaque carte et tuile est un <a> : sans JavaScript, il mène à la page projet (curé) ou à
 *    l'entrée de la liste ; avec JavaScript, MontagneExplorer intercepte le clic (attribut
 *    `data-open-flower`) et ouvre la fleur.
 *  - les projets ABSORBÉS par une carte (`members`, cf. montagne_sommet.go) n'ont pas de tuile ;
 *    leurs filets partent de la carte (pyramid.ts).
 *  - E2E : `montagne-summit-card` / `montagne-tile` portent `data-project-id`, la carte porte
 *    `data-visual` (art | icon) et `data-members`, sa couverture `montagne-summit-cover` ; les
 *    filets portent `data-from` / `data-to`.
 */
export async function MontagnePeak({ locale, montagne }: { locale: string; montagne: Montagne }) {
  const t = await getTranslations({ locale, namespace: 'montagne' });
  const absorbed = new Map(montagne.summit.flatMap((c) => c.members.map((m) => [m, c.id] as const)));
  const geometryInput = { nodes: montagne.nodes, links: montagne.links, summit: montagne.summit.map((c) => c.id), absorbed };
  const layouts = GEOMETRIES.map((g) => layoutMountain(geometryInput, g));
  const byId = new Map(montagne.nodes.map((n) => [n.id, n]));
  const cardById = new Map(montagne.summit.map((c) => [c.id, c]));

  // Variables du conteneur : taille et mesures de chaque géométrie.
  const containerVars: Record<string, string> = {};
  GEOMETRIES.forEach((g, i) => {
    Object.assign(containerVars, {
      [`--w${i}`]: `${layouts[i].width}px`,
      [`--h${i}`]: `${layouts[i].height}px`,
      [`--tile${i}`]: `${g.tile}px`,
      [`--cw${i}`]: `${g.cardWidth}px`,
      [`--ch${i}`]: `${g.cardHeight}px`,
      // Nom sous la tuile : un peu plus étroit que la cellule, pour garder un blanc entre voisins.
      [`--label${i}`]: `${g.cell - 4}px`,
      [`--fs${i}`]: `${i === 0 ? 10 : i === 1 ? 11 : 12}px`,
      // Bas des cartes : la pointe de la montagne peinte s'y pose (MontagneBackdrop).
      [`--peak${i}`]: `${layouts[i].peakBottom}px`,
    });
  });

  /** Position d'un élément dans les trois géométries. */
  const placeVars = (id: string, kind: 'cards' | 'tiles'): CSSProperties => {
    const v: Record<string, string> = {};
    layouts.forEach((l, i) => {
      const box = l[kind].find((b) => b.id === id)!;
      v[`--x${i}`] = `${box.x}px`;
      v[`--y${i}`] = `${box.y}px`;
    });
    return v as CSSProperties;
  };

  /** Cible sans JavaScript : la page projet si elle existe, sinon l'entrée de la liste. */
  const fallbackHref = (n: MontagneNode) => (n.curated ? `/${locale}/portfolio/${n.id}` : `#projet-${n.id}`);

  const summit = layouts[0].cards.map((c) => byId.get(c.id)!);
  const tiles = layouts[0].tiles.map((b) => byId.get(b.id)!);

  return (
    <section aria-labelledby="montagne-peak-title" data-testid="montagne-peak" className="relative overflow-x-clip px-2 pb-6">
      <h2 id="montagne-peak-title" className="sr-only">
        {t('peak.title')}
      </h2>
      <div className={styles.mountain} style={containerVars as CSSProperties} data-testid="montagne-mountain">
        {/* Dans le conteneur : son cadrage se calcule sur les mesures de la géométrie active. */}
        <MontagneBackdrop locale={locale} />
        {layouts.map((l, i) => (
          <MountainLinks key={i} layout={l} className={styles[`g${i}`]} />
        ))}

        <ol aria-label={t('peak.summit')} className="list-none">
          {summit.map((n) => (
            <li key={n.id} className={`${styles.placed} ${styles.card}`} style={placeVars(n.id, 'cards')}>
              <SummitCard node={n} card={cardById.get(n.id)!} href={fallbackHref(n)} coverAlt={t.has(`art.covers.${n.id}`) ? t(`art.covers.${n.id}`) : ''} />
            </li>
          ))}
        </ol>

        <ol aria-label={t('peak.mountain')} className="list-none">
          {tiles.map((n) => (
            <li key={n.id} className={styles.placed} style={placeVars(n.id, 'tiles')}>
              <a
                href={fallbackHref(n)}
                data-testid="montagne-tile"
                data-project-id={n.id}
                data-stratum={n.stratum}
                data-open-flower=""
                className={`${styles.tile} group relative flex items-center justify-center rounded-lg border bg-[#0a0f1a]/90 shadow-[0_6px_18px_-8px_rgba(0,0,0,0.9)] transition-colors hover:border-gold focus:outline-none focus-visible:border-gold focus-visible:ring-2 focus-visible:ring-gold/60`}
                // Bordure teintée par la strate (repère de couche, jamais de l'or : l'or est le
                // signal du survol et des filets).
                style={{ borderColor: `${STRATUM_TINT[n.stratum]}88` }}
              >
                <TileIcon node={n} />
                <span className={`${styles.tileName} text-slate-300 group-hover:text-white`}>
                  {nameSegments(n.name).map((seg, i) => (
                    <Fragment key={i}>
                      {i > 0 && <wbr />}
                      {seg}
                    </Fragment>
                  ))}
                </span>
              </a>
            </li>
          ))}
        </ol>
      </div>
    </section>
  );
}

/**
 * Carte d'une œuvre du sommet : son ILLUSTRATION en couverture (lib/montagne/art.ts) — une image
 * d'ambiance, pas une preuve : les preuves (captures, démo vidéo, pages du site) s'ouvrent dans
 * la fleur au clic (MontagneProofFlower). Sans illustration, son icône en grand : jamais un cadre
 * vide.
 */
function SummitCard({ node, card, href, coverAlt }: { node: MontagneNode; card: MontagneSummitCard; href: string; coverAlt: string }) {
  const cover = summitCover(node.id);
  return (
    <a
      href={href}
      data-testid="montagne-summit-card"
      data-project-id={node.id}
      data-visual={cover ? 'art' : 'icon'}
      data-members={card.members.join(' ')}
      data-open-flower=""
      className={`${styles.summitCard} group flex h-full w-full flex-col overflow-hidden rounded-lg border border-gold/70 bg-[#0a0f1a] transition-[border-color,box-shadow] hover:border-gold focus:outline-none focus-visible:ring-2 focus-visible:ring-gold/70`}
    >
      <span className="relative block min-h-0 flex-1 overflow-hidden">
        {cover ? (
          <Image
            src={cover}
            alt={coverAlt}
            fill
            data-testid="montagne-summit-cover"
            // Largeur d'une carte selon la géométrie (pyramid.ts) : 168 / 164 / 236 px.
            sizes="(min-width: 1280px) 236px, 168px"
            className="object-cover transition-transform duration-500 group-hover:scale-[1.04]"
          />
        ) : (
          // Pas d'illustration : l'icône (ou le monogramme) en grand, sur un halo — la carte reste pleine.
          <span className="flex h-full w-full items-center justify-center bg-[radial-gradient(ellipse_at_center,rgba(30,41,66,0.9),#0a0f1a_70%)]">
            <TileIcon node={node} large />
          </span>
        )}
      </span>
      <span
        className={`${styles.serif} block truncate border-t border-gold/40 bg-gradient-to-b from-[#0b111d] to-[#060910] px-2 py-1.5 text-center text-[17px] leading-tight text-[#efe6d2] xl:py-2 xl:text-[22px]`}
      >
        {node.name}
      </span>
    </a>
  );
}

/** Icône d'un projet : l'icône définitive (Eikon) si elle existe, sinon le monogramme en lettres. */
function TileIcon({ node, large = false }: { node: MontagneNode; large?: boolean }) {
  if (node.icon_url) {
    return (
      // eslint-disable-next-line @next/next/no-img-element -- icône servie par maiProFiles, quelques px : l'optimiseur n'apporte rien
      <img
        src={portfolioImageUrl(node.icon_url)}
        alt=""
        aria-hidden="true"
        className={large ? 'h-20 w-20 xl:h-24 xl:w-24' : 'h-[62%] w-[62%]'}
      />
    );
  }
  return (
    <span
      aria-hidden="true"
      className={`${styles.serif} font-semibold tracking-wide ${large ? 'text-6xl xl:text-7xl' : 'text-[19px] xl:text-[22px]'}`}
      style={{ color: large ? '#ecd8a4' : '#e7dfcc' }}
    >
      {node.monogram}
    </span>
  );
}

/**
 * Les filets d'une géométrie : un tracé doré par lien réel. Décoratif (aria-hidden) ;
 * `overflow: visible` laisse un filet qui part d'une carte déborder du cadre si besoin.
 * Discrets au repos, allumés quand on survole ou focalise une carte ou une tuile : l'opacité vit
 * dans montagne.module.css (.link / .linkEnd), l'allumage est posé par useLinkHighlight
 * (attribut `data-lit` sur les tracés et les points du projet actif).
 */
function MountainLinks({ layout, className }: { layout: MountainLayout; className: string }) {
  const { width, height, links } = layout;
  return (
    <svg
      className={`${className} pointer-events-none absolute left-0 top-0`}
      width={width}
      height={height}
      viewBox={`0 0 ${width} ${height}`}
      overflow="visible"
      aria-hidden="true"
      data-testid="montagne-links"
    >
      {/* Filets « sert à construire » : l'or est ici le signal — un lien réel, rien d'autre. */}
      <g>
        {links.map((l) => (
          <path key={`${l.from}>${l.to}`} className={styles.link} data-from={l.from} data-to={l.to} d={l.d} />
        ))}
      </g>
      {/* Point d'arrivée de chaque filet : ce qu'il sert à construire. Il porte les deux
          extrémités de son filet (data-end-*), pour s'allumer avec lui. */}
      <g>
        {links.map((l) => (
          <circle key={`${l.from}>${l.to}`} className={styles.linkEnd} data-end-from={l.from} data-end-to={l.to} cx={l.x2} cy={l.y2} r="2" />
        ))}
      </g>
    </svg>
  );
}

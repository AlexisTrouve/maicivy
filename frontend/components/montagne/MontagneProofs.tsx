'use client';

import Image from 'next/image';
import { useTranslations } from 'next-intl';

import { Link } from '@/i18n/navigation';
import type { MontagneNode, MontagneProof, MontagneSummitCard } from '@/lib/montagne-api';
import { siteCapture } from '@/lib/montagne/sitePages';
import { portfolioImageUrl } from '@/lib/portfolio-api';

/**
 * MontagneProofs — les preuves d'une carte du sommet, dans sa fleur.
 *
 * QUOI : une galerie par règle (montagne_sommet.go), jamais une capture isolée :
 *  - built_games : les jeux bâtis sur le moteur, chacun nommé, menant à sa page projet ;
 *  - video       : la démo, lisible avec ses contrôles, légendée par maiProFiles ;
 *  - stages      : la chaîne global → local, une légende i18n par étage qui dit la filiation ;
 *  - site        : les pages de ce site, chaque capture menant à la vraie page.
 * POURQUOI les preuves viennent toutes du backend : l'ordre, le filtrage (jeux, vrais visuels)
 * et l'absence d'un étage sans visuel y sont décidés et testés. Ici on ne fait que dessiner — un
 * étage absent n'a donc ni titre ni cadre vide.
 * E2E : `montagne-proofs` (data-rule), `montagne-proof` (data-proof-project, data-stage),
 * `montagne-proof-video`, `montagne-proof-page` (data-page).
 */
export function MontagneProofs({ card, node }: { card: MontagneSummitCard; node: MontagneNode }) {
  const t = useTranslations('montagne.proofs');
  if (card.proofs.length === 0) return null;

  return (
    <section data-testid="montagne-proofs" data-rule={card.rule} aria-label={t('label')} className="w-full">
      {card.rule === 'built_games' && (
        <>
          <ProofHeading>{t('builtGames', { name: node.name })}</ProofHeading>
          <ul className="grid list-none grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
            {card.proofs.map((p) => (
              <li key={`${p.project_id}-${p.url}`}>
                <ProjectProof proof={p} caption={p.project_name ?? ''} />
              </li>
            ))}
          </ul>
        </>
      )}

      {card.rule === 'video' &&
        card.proofs.map((p) => (
          <figure key={p.url} className="overflow-hidden rounded-xl border border-white/10 bg-[#0b111c]">
            <video
              data-testid="montagne-proof-video"
              controls
              playsInline
              // « metadata » et jamais « none » : avec « none », Chrome ignore le clic (mesuré).
              preload="metadata"
              poster={p.poster_url ? portfolioImageUrl(p.poster_url) : undefined}
              src={portfolioImageUrl(p.url)}
              aria-label={p.alt || t('video')}
              className="aspect-video w-full bg-black"
            />
            {p.alt && <figcaption className="px-4 py-3 text-sm leading-relaxed text-slate-300">{p.alt}</figcaption>}
          </figure>
        ))}

      {card.rule === 'stages' && (
        <ul className="flex list-none flex-col gap-4">
          {card.proofs.map((p) => (
            <li key={`${p.stage}-${p.url}`}>
              <ProjectProof proof={p} caption={t(`stages.${p.stage}`, { project: p.project_name ?? '' })} wide />
            </li>
          ))}
        </ul>
      )}

      {card.rule === 'site' && (
        <>
          <ProofHeading>{t('site.title')}</ProofHeading>
          <ul className="grid list-none grid-cols-2 gap-3">
            {card.proofs.map((p) => (
              <li key={p.page}>
                <SitePageProof proof={p} label={t(`site.pages.${p.page}`)} />
              </li>
            ))}
          </ul>
        </>
      )}
    </section>
  );
}

function ProofHeading({ children }: { children: React.ReactNode }) {
  return <h3 className="mb-3 text-xs font-semibold uppercase tracking-[0.18em] text-slate-400">{children}</h3>;
}

/**
 * Une preuve portée par un projet (jeu, étage) : son visuel ENTIER (object-contain — une preuve
 * se voit en entier), sa légende, et un lien vers la page du projet s'il en a une, sinon vers
 * l'image en grand.
 */
function ProjectProof({ proof, caption, wide = false }: { proof: MontagneProof; caption: string; wide?: boolean }) {
  const t = useTranslations('montagne');
  const image = portfolioImageUrl(proof.url);
  const body = (
    <>
      <span className={`relative block w-full bg-[#05080f] ${wide ? 'aspect-[4/3] md:aspect-[16/9]' : 'aspect-[4/3]'}`}>
        <Image
          src={image}
          alt={proof.alt || caption}
          fill
          sizes={wide ? '(min-width: 1024px) 960px, 100vw' : '(min-width: 1024px) 320px, (min-width: 640px) 50vw, 100vw'}
          unoptimized={proof.vector}
          className="object-contain"
        />
      </span>
      <span className="block px-3 py-2.5 text-sm leading-snug text-slate-200">
        {caption}
        <span className="mt-0.5 block text-xs text-slate-500">{proof.curated ? t('proofs.openProject') : t('flower.openImage')} →</span>
      </span>
    </>
  );
  const cls =
    'block overflow-hidden rounded-xl border border-white/10 bg-[#0b111c] transition-colors hover:border-white/30 focus:outline-none focus-visible:ring-2 focus-visible:ring-gold/60';
  return (
    <figure data-testid="montagne-proof" data-proof-project={proof.project_id} data-stage={proof.stage}>
      {proof.curated ? (
        <Link href={`/portfolio/${proof.project_id}`} className={cls}>
          {body}
        </Link>
      ) : (
        <a href={image} target="_blank" rel="noopener noreferrer" className={cls}>
          {body}
        </a>
      )}
    </figure>
  );
}

/**
 * Une page de ce site : sa capture téléphone sous 768 px, grand écran au-delà (<picture>), et un
 * lien vers la VRAIE page dans la langue courante (Link de next-intl préfixe la locale).
 */
function SitePageProof({ proof, label }: { proof: MontagneProof; label: string }) {
  const capture = siteCapture(proof.page!);
  return (
    <Link
      href={proof.path!}
      data-testid="montagne-proof-page"
      data-page={proof.page}
      className="block overflow-hidden rounded-xl border border-white/10 bg-[#0b111c] transition-colors hover:border-white/30 focus:outline-none focus-visible:ring-2 focus-visible:ring-gold/60"
    >
      <picture className="block aspect-[9/16] w-full overflow-hidden md:aspect-[16/10]">
        <source media="(min-width: 768px)" srcSet={capture.desktop} />
        {/* eslint-disable-next-line @next/next/no-img-element -- <picture> à deux sources (téléphone / grand écran), déjà en webp optimisé : next/image ne sait pas choisir une source par media query */}
        <img src={capture.mobile} alt={label} loading="lazy" className="h-full w-full object-cover object-top" />
      </picture>
      <span className="block px-3 py-2.5 text-sm text-slate-200">{label} →</span>
    </Link>
  );
}

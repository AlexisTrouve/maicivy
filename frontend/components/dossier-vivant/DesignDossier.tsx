'use client';

import { useLayoutEffect, useRef, useState } from 'react';
import { useLocale, useTranslations } from 'next-intl';

import { easeIO, easeOut, lerp, prog, reveal } from '@/lib/dossier-vivant/anim';
import { CH, typingDuration } from '@/lib/dossier-vivant/choreography';
import { SECTION_IDS, type DossierBlock, type SectionId } from '@/lib/dossier-vivant/events';
import { researchDoneAt, type RunState, type SectionView } from '@/lib/dossier-vivant/runState';
import type { SceneNode } from '@/lib/dossier-vivant/scene';
import { CONTACT_EMAIL } from '@/lib/montagne/links';
import { portfolioImageUrl } from '@/lib/portfolio-api';

import { Icon } from './icons';
import { PipelineDiagram } from './PipelineDiagram';
import { ProjectMark } from './ProjectMark';
import { StateLine } from './StateLine';
import styles from './dossier-vivant.module.css';
import { useDossierPdf } from './useDossierPdf';
import { useDossierSkills, type SkillsState } from './useDossierSkills';
import type { OpenProject } from './useProjectFlower';

/** Ce que le front sait d'un projet cité en preuve : lu sur sa fiche réelle (backend → maiProFiles). */
export interface ProjectFacts {
  status: string;
  loc: number | null;
  tests: number | null;
}

/** Hauteurs de la feuille (px du conteneur de l'équipe) : sous les colonnes, sur leurs corps. */
export interface SheetStops {
  work: number;
  mid: number;
}

type Timed<B> = { at: number; block: B };
type ProofBlock = Extract<DossierBlock, { type: 'proof' }>;
type ItemBlock = Extract<DossierBlock, { type: 'item' }>;

/**
 * DesignDossier — le livrable : la feuille du dossier de conception, qui s'écrit section par
 * section pendant que les agents travaillent, monte sur leurs colonnes quand ils ont fini, et prend
 * toute la colonne à la remise.
 *
 * QUOI : en-tête (titre, destinataire, état, « Télécharger le dossier » — l'action principale de
 * l'écran une fois le dossier prêt, seule en or plein), puis les cinq sections du flux. Les
 * chiffres du tableau des preuves (statut, lignes, tests) viennent des fiches RÉELLES (`facts`),
 * jamais du flux.
 * COMMENT :
 *  - les paragraphes se tapent ; le reste du texte est déjà posé, invisible, pour que la feuille
 *    ait sa hauteur finale dès l'arrivée du bloc (pas de saut pendant la frappe) ;
 *  - pendant le parcours, la feuille déborde sous la colonne : elle suit la section en cours
 *    d'écriture par une translation calculée sur les positions mesurées ; à la remise elle revient
 *    en haut, puis se DÉROULE (`unrolled`) : elle prend la hauteur de son contenu, plus aucun
 *    ascenseur interne — c'est la PAGE qui défile (DossierVivant agrandit la zone qui la porte) ;
 *  - télécharger = un VRAI PDF, mis en page A4 par le backend à partir de l'état du dossier
 *    (useDossierPdf, pdfExport.ts) à partir de SA copie du dossier ; `pdfEndpoint` dit quelle
 *    route publique : celle du rejeu (avec sa mention) ou celle du run réel du visiteur.
 * Téléphone : même feuille, plus étroite (CSS) — en-tête empilé, bouton pleine largeur, tableaux
 * repliés en fiches où chaque valeur porte son en-tête (`data-label`) ; à la remise, la boîte de
 * l'équipe s'étend à tout l'écran (DossierVivant) et la feuille se lit en défilant la page au doigt.
 * Ligne de preuve : un clic (ou le bouton du projet, au clavier) ouvre la fleur du projet ; le lien
 * de la démo garde son propre rôle.
 * Après les preuves, la section « Compétences mobilisées » (SkillsSection) : composée par le
 * backend à partir des preuves écrites (useDossierSkills), la même que celle du PDF.
 * E2E : `dv-dossier` (`data-ready`, `data-unrolled`), `dv-section` (`data-section`, `data-status`), `dv-proof-row`
 * (`data-project-id`), `dv-download` (`data-pdf` : idle | busy | failed | expired), `dv-skills`
 * (`data-status` : loading | ready | failed), `dv-skill-row` (`data-sources` : ids des preuves).
 */
export function DesignDossier({
  state,
  now,
  stops,
  nodes,
  facts,
  elapsedLabel,
  costLabel,
  pdfEndpoint,
  skillsEndpoint,
  unrolled,
  onOpen,
  canOpen,
}: {
  state: RunState;
  now: number;
  stops: SheetStops;
  nodes: ReadonlyMap<string, SceneNode>;
  facts: Readonly<Record<string, ProjectFacts>>;
  elapsedLabel: string;
  costLabel: string;
  pdfEndpoint: string;
  /** Route des compétences (publique, mêmes données pour le rejeu et un run réel). */
  skillsEndpoint: string;
  /** Remise terminée (feuille en place) : la feuille prend la hauteur de son contenu. */
  unrolled: boolean;
  onOpen: OpenProject;
  canOpen: (id: string) => boolean;
}) {
  const t = useTranslations('dossierVivant.dossier');
  const bodyRef = useRef<HTMLDivElement>(null);
  const innerRef = useRef<HTMLDivElement>(null);
  const [boxes, setBoxes] = useState<{ body: number; secs: Partial<Record<SectionId, { top: number; bottom: number }>> }>({ body: 0, secs: {} });

  const dossier = state.dossier;
  const blockCount = dossier ? SECTION_IDS.reduce((n, id) => n + dossier.sections[id].blocks.length, 0) : 0;
  // La section des compétences arrive après coup (réponse du backend) : elle change aussi la hauteur.
  const [skillsKey, setSkillsKey] = useState('');

  // Mesure des sections à chaque bloc arrivé (pas à chaque image) : cibles du suivi d'écriture.
  // Plus un ResizeObserver : le schéma ne prend sa hauteur qu'APRÈS avoir mesuré sa propre largeur
  // (PipelineDiagram), un rendu plus tard que l'arrivée de son bloc. La translation du suivi ne
  // change aucune taille : l'observateur ne se déclenche pas à chaque image.
  useLayoutEffect(() => {
    const body = bodyRef.current;
    const inner = innerRef.current;
    if (!body || !inner) return;
    const measure = () => {
      const secs: Partial<Record<SectionId, { top: number; bottom: number }>> = {};
      inner.querySelectorAll<HTMLElement>('[data-section]').forEach((el) => {
        secs[el.dataset.section as SectionId] = { top: el.offsetTop, bottom: el.offsetTop + el.offsetHeight };
      });
      setBoxes({ body: body.clientHeight, secs });
    };
    measure();
    if (typeof ResizeObserver === 'undefined') return;
    const ro = new ResizeObserver(measure);
    ro.observe(inner);
    return () => ro.disconnect();
  }, [blockCount, stops.work, stops.mid, skillsKey]);

  // À la remise, la feuille revient en haut une fois pour toutes (un défilement interne fait pendant
  // le parcours ne doit pas rester) : déroulée, elle se lit en défilant la page.
  useLayoutEffect(() => {
    if (unrolled && bodyRef.current) bodyRef.current.scrollTop = 0;
  }, [unrolled]);

  // En-tête de la feuille (le PDF a le sien, composé par le serveur avec les mêmes messages).
  const forLine = state.visitor ? t('for', { name: state.visitor.name, affiliation: state.visitor.affiliation }) : '';
  // Le compte d'agents vient du run (orchestrateur + sous-agents annoncés) : cinq en rejeu, six
  // quand le Dessinateur est de l'équipe — jamais un nombre en dur qui mentirait sur l'un des deux.
  const preparedLine = t('prepared', { time: elapsedLabel, cost: costLabel, count: state.team.length + 1 });
  const pdf = useDossierPdf(state, pdfEndpoint);
  const skills = useDossierSkills(state, skillsEndpoint);

  if (!dossier || !state.visitor) return null;

  const dIn = easeOut(prog(now, dossier.openedAt, CH.dossierIn));
  const doneAt = researchDoneAt(state);
  const slideAt = doneAt === null ? Infinity : doneAt + CH.slideDelay;
  const up = easeIO(prog(now, slideAt, CH.slideD));
  const full = state.completedAt === null ? 0 : easeIO(prog(now, state.completedAt, CH.fullD));
  const y = lerp(lerp(stops.work, stops.mid, up), 0, full) + (1 - dIn) * 30;

  // Suivi de la section en cours d'écriture : chacune doit tenir dans la partie découverte.
  let off = 0;
  for (const id of SECTION_IDS) {
    const sec = dossier.sections[id];
    const box = boxes.secs[id];
    if (sec.writingAt === null || !box) continue;
    const visible = boxes.body - (sec.writingAt < slideAt ? stops.work : stops.mid);
    const target = Math.max(0, Math.min(box.bottom - visible + 8, box.top - 4));
    off = lerp(off, target, easeIO(prog(now, sec.writingAt, CH.scrollD)));
  }
  if (state.completedAt !== null) off = lerp(off, 0, easeIO(prog(now, state.completedAt, CH.fullD)));

  const ready = state.completedAt !== null && now >= state.completedAt;
  const readyStatus = ready ? 'done' : 'working';
  const readyLabel = ready ? t('state.ready') : state.reviewAt !== null ? t('state.reviewing') : t('state.drafting');
  const download = state.completedAt === null ? null : reveal(now, state.completedAt + 0.2, 0.5);

  return (
    <article
      className={styles.dossier}
      style={{ transform: `translateY(${y.toFixed(1)}px)`, opacity: dIn, visibility: dIn > 0 ? 'visible' : 'hidden' }}
      aria-labelledby="dv-dossier-title"
      data-testid="dv-dossier"
      data-ready={ready ? '' : undefined}
      data-unrolled={unrolled ? '' : undefined}
    >
      <div className={styles.dosHead}>
        <div>
          <h2 id="dv-dossier-title">{t('title')}</h2>
          <p data-testid="dv-dossier-subtitle">{ready ? `${forLine}.\n${preparedLine}` : forLine}</p>
        </div>
        <div className={styles.dosSide}>
          <StateLine status={readyStatus} label={readyLabel} now={now} className={styles.readyState} testId="dv-dossier-state" />
          {download && (
            <button
              type="button"
              className={`${styles.btnGold} ${styles.noPrint}`}
              style={download}
              onClick={pdf.download}
              aria-busy={pdf.status === 'busy'}
              data-testid="dv-download"
              data-pdf={pdf.status}
            >
              {/* L'échec se lit au glyphe ✕ autant qu'au libellé ; l'attente, à l'arc qui tourne. */}
              <Icon name={pdf.status === 'failed' || pdf.status === 'expired' ? 'x' : pdf.status === 'busy' ? 'arc' : 'dl'} />
              <span>{pdf.status === 'busy' ? t('pdf.busy') : pdf.status === 'failed' ? t('pdf.failed') : pdf.status === 'expired' ? t('pdf.expired') : t('download')}</span>
            </button>
          )}
        </div>
      </div>
      <div className={styles.dosBody} ref={bodyRef}>
        <div className={styles.dosInner} ref={innerRef} style={{ transform: off > 0.5 ? `translateY(${(-off).toFixed(1)}px)` : undefined }}>
          {SECTION_IDS.map((id) => [
            <Section
              key={id}
              id={id}
              sec={dossier.sections[id]}
              now={now}
              nodes={nodes}
              facts={facts}
              organisation={state.visitor!.organisation}
              onOpen={onOpen}
              canOpen={canOpen}
            />,
            // Les compétences suivent les preuves dont elles sont tirées (même ordre que le PDF).
            id === 'proofs' && (
              <SkillsSection key="skills" skills={skills} at={dossier.sections.proofs.doneAt ?? 0} now={now} nodes={nodes} onLayout={setSkillsKey} />
            ),
          ])}
        </div>
      </div>
    </article>
  );
}

function Section({
  id,
  sec,
  now,
  nodes,
  facts,
  organisation,
  onOpen,
  canOpen,
}: {
  id: SectionId;
  sec: SectionView;
  now: number;
  nodes: ReadonlyMap<string, SceneNode>;
  facts: Readonly<Record<string, ProjectFacts>>;
  organisation: string;
  onOpen: OpenProject;
  canOpen: (id: string) => boolean;
}) {
  const t = useTranslations('dossierVivant.dossier');
  const note = sec.status === 'writing' ? t('section.writing') : sec.status === 'waiting' ? t(`section.waitingFor.${sec.waitingFor}`) : '';

  // Les blocs consécutifs de même famille se regroupent : puces dans une liste, preuves dans un tableau.
  const groups: (Timed<DossierBlock> | Timed<ItemBlock>[] | Timed<ProofBlock>[])[] = [];
  for (const b of sec.blocks) {
    const last = groups[groups.length - 1];
    if ((b.block.type === 'item' || b.block.type === 'proof') && Array.isArray(last) && last[0].block.type === b.block.type) {
      (last as Timed<DossierBlock>[]).push(b);
    } else if (b.block.type === 'item' || b.block.type === 'proof') {
      groups.push([b] as Timed<ItemBlock>[] | Timed<ProofBlock>[]);
    } else {
      groups.push(b);
    }
  }

  return (
    <section
      className={`${styles.sec} ${id === 'proofs' ? styles.wide : ''}`}
      aria-labelledby={`dv-sec-${id}`}
      data-testid="dv-section"
      data-section={id}
      data-status={sec.status}
    >
      <h3 id={`dv-sec-${id}`}>
        {sec.title} <small>{note}</small>
      </h3>
      {groups.map((g, i) => {
        if (Array.isArray(g)) {
          if (g[0].block.type === 'item') {
            return (
              <ul key={i}>
                {(g as Timed<ItemBlock>[]).map((b, k) => (
                  <li key={k} style={reveal(now, b.at, CH.blockReveal)}>
                    <Icon name="chev" />
                    <span>{b.block.text}</span>
                  </li>
                ))}
              </ul>
            );
          }
          return (
            <ProofTable
              key={i}
              rows={g as Timed<ProofBlock>[]}
              headAt={(sec.writingAt ?? 0) + 0.2}
              now={now}
              nodes={nodes}
              facts={facts}
              onOpen={onOpen}
              canOpen={canOpen}
            />
          );
        }
        return <Block key={i} timed={g} now={now} nodes={nodes} organisation={organisation} />;
      })}
    </section>
  );
}

/** Texte tapé : la part déjà écrite, puis le reste posé invisible (hauteur finale dès le départ). */
function TypedText({ text, at, now }: { text: string; at: number; now: number }) {
  const n = Math.round(text.length * prog(now, at, typingDuration(text)));
  return (
    <>
      {text.slice(0, n)}
      <span style={{ visibility: 'hidden' }} aria-hidden="true">
        {text.slice(n)}
      </span>
    </>
  );
}

function Block({ timed, now, nodes, organisation }: { timed: Timed<DossierBlock>; now: number; nodes: ReadonlyMap<string, SceneNode>; organisation: string }) {
  const t = useTranslations('dossierVivant.dossier');
  const { at, block } = timed;
  switch (block.type) {
    case 'lead':
      return (
        <p className={styles.lead}>
          <TypedText text={block.text} at={at} now={now} />
        </p>
      );
    case 'paragraph':
      return (
        <p>
          <TypedText text={block.text} at={at} now={now} />
        </p>
      );
    case 'pipeline':
      return <PipelineDiagram nodes={block.nodes} edges={block.edges} journal={block.journal} label={block.label} at={at} now={now} />;
    case 'also':
      return (
        <p className={styles.also} style={reveal(now, at, CH.blockReveal)}>
          {t('proofs.also', { names: block.projectIds.map((id) => nodes.get(id)?.name ?? id).join(', ') })}
        </p>
      );
    case 'contact':
      return (
        <div className={`${styles.nextActions} ${styles.noPrint}`} style={reveal(now, at, CH.blockReveal)}>
          <a
            className={styles.btnLine}
            href={`mailto:${CONTACT_EMAIL}?subject=${encodeURIComponent(t('contactSubject', { organisation }))}`}
            data-testid="dv-contact"
          >
            <Icon name="mail" />
            {t('contact')}
          </a>
        </div>
      );
    default:
      return null;
  }
}

/**
 * SkillsSection — « Compétences mobilisées » : une ligne par compétence (nom, niveau, preuves d'où
 * elle vient), puis les technologies que plusieurs preuves partagent. Tout vient du backend ; ici,
 * seulement les libellés (niveaux, en-têtes) et les noms des projets, lus sur la montagne.
 * Rien tant que les preuves ne sont pas écrites ; aucune section si la sélection est vide.
 */
function SkillsSection({
  skills,
  at,
  now,
  nodes,
  onLayout,
}: {
  skills: SkillsState;
  at: number;
  now: number;
  nodes: ReadonlyMap<string, SceneNode>;
  onLayout: (key: string) => void;
}) {
  const t = useTranslations('dossierVivant.dossier.skills');
  const data = skills.status === 'ready' ? skills.data : null;
  // Le parent remesure les sections quand celle-ci change de forme (chargement → contenu).
  useLayoutEffect(() => onLayout(`${skills.status}:${data?.skills.length ?? 0}`), [skills.status, data, onLayout]);
  if (skills.status === 'idle' || (data && data.skills.length === 0 && data.technologies.length === 0)) return null;

  const names = (ids: string[]) => ids.map((id) => nodes.get(id)?.name ?? id).join(', ');
  const level = (l: string) => (t.has(`levels.${l}`) ? t(`levels.${l}`) : l);
  const note = skills.status === 'loading' ? t('loading') : skills.status === 'failed' ? t('failed') : '';
  return (
    <section
      className={`${styles.sec} ${styles.wide}`}
      aria-labelledby="dv-sec-skills"
      data-section="skills"
      data-testid="dv-skills"
      data-status={skills.status}
      style={reveal(now, at, CH.blockReveal)}
    >
      <h3 id="dv-sec-skills">
        {t('title')} <small>{note}</small>
      </h3>
      {data && (
        <>
          <p className={styles.skillsLead}>{t('lead')}</p>
          <table className={`${styles.proofs} ${styles.skills}`}>
            <thead>
              <tr>
                <th scope="col">{t('skill')}</th>
                <th scope="col">{t('level')}</th>
                <th scope="col">{t('source')}</th>
              </tr>
            </thead>
            <tbody>
              {data.skills.map((s) => (
                <tr key={s.name} data-testid="dv-skill-row" data-sources={s.projects.join(',')}>
                  <td>
                    <b className={styles.skillName}>{s.name}</b>
                    {/* La marque or a toujours son glyphe et son libellé : l'état se lit sans la couleur. */}
                    {s.need && <span className={styles.skillNeed}>◆ {t('need')}</span>}
                  </td>
                  <td data-label={t('level')}>{level(s.level)}</td>
                  <td className={styles.skillSources} data-label={t('source')}>
                    {names(s.projects)}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
          {data.technologies.length > 0 && (
            <p className={styles.also}>
              {t('technologies')} : {data.technologies.map((x) => `${x.name} (${names(x.projects)})`).join(' · ')}
            </p>
          )}
        </>
      )}
    </section>
  );
}

function ProofTable({
  rows,
  headAt,
  now,
  nodes,
  facts,
  onOpen,
  canOpen,
}: {
  rows: Timed<ProofBlock>[];
  headAt: number;
  now: number;
  nodes: ReadonlyMap<string, SceneNode>;
  facts: Readonly<Record<string, ProjectFacts>>;
  onOpen: OpenProject;
  canOpen: (id: string) => boolean;
}) {
  const t = useTranslations('dossierVivant.dossier.proofs');
  const tStatus = useTranslations('montagne.status');
  const locale = useLocale();
  const fmt = new Intl.NumberFormat(locale);
  const statusLabel = (s: string) => (tStatus.has(s) ? tStatus(s) : s);
  return (
    <table className={styles.proofs}>
      <thead style={reveal(now, headAt, CH.blockReveal)}>
        <tr>
          <th scope="col">{t('project')}</th>
          <th scope="col">{t('status')}</th>
          <th scope="col" className={styles.num}>
            {t('lines')}
          </th>
          <th scope="col" className={styles.num}>
            {t('tests')}
          </th>
          <th scope="col">{t('demo')}</th>
        </tr>
      </thead>
      <tbody>
        {rows.map(({ at, block }, k) => {
          const n = nodes.get(block.projectId);
          const f = facts[block.projectId];
          const openable = canOpen(block.projectId);
          // Le projet : marque, nom, raison. Dans un bouton quand il a une fleur — son nom
          // accessible est ce texte même (la marque est décorative).
          const project = (
            <span className={styles.proj}>
              <ProjectMark node={n} monoClass={styles.projMono} iconClass={styles.projIcon} />
              <span>
                <b>{n?.name ?? block.projectId}</b>
                <span>{block.why}</span>
              </span>
            </span>
          );
          return (
            // Clé = rang de la ligne : la liste ne fait que s'allonger, et un vrai run peut citer
            // deux fois le même projet (le projectId seul donnait deux lignes de même clé).
            // Toute la ligne ouvre la fleur à la souris (cible large) ; le bouton du projet porte
            // le clavier et reprend le focus à la fermeture. Le lien de la démo garde son rôle.
            <tr
              key={k}
              style={reveal(now, at, CH.blockReveal)}
              data-testid="dv-proof-row"
              data-project-id={block.projectId}
              data-openable={openable ? '' : undefined}
              onClick={
                openable
                  ? (e) => {
                      if ((e.target as Element).closest('a')) return;
                      onOpen(block.projectId, e.currentTarget.querySelector('button') ?? e.currentTarget);
                    }
                  : undefined
              }
            >
              <td>
                {openable ? (
                  <button type="button" className={styles.projButton} aria-haspopup="dialog">
                    {project}
                  </button>
                ) : (
                  project
                )}
              </td>
              {/* `data-label` : l'en-tête de la colonne, que le téléphone affiche devant la valeur quand
                  le tableau se replie en fiches (CSS) ; le grand écran ne le lit pas. */}
              <td data-label={t('status')}>{f ? statusLabel(f.status) : '—'}</td>
              <td className={styles.num} data-label={t('lines')}>
                {f?.loc ? fmt.format(f.loc) : '—'}
              </td>
              <td className={styles.num} data-label={t('tests')}>
                {f?.tests ? fmt.format(f.tests) : '—'}
              </td>
              <td data-label={t('demo')}>
                {block.demo ? (
                  <a href={block.demo.kind === 'video' ? portfolioImageUrl(block.demo.url) : block.demo.url} target="_blank" rel="noopener noreferrer">
                    {t(block.demo.kind)}
                  </a>
                ) : (
                  <span className={styles.none}>{t('none')}</span>
                )}
              </td>
            </tr>
          );
        })}
      </tbody>
    </table>
  );
}

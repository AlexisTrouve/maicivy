'use client';

import { useLocale, useTranslations } from 'next-intl';

import { easeIO, easeOut, prog, reveal } from '@/lib/dossier-vivant/anim';
import { CH } from '@/lib/dossier-vivant/choreography';
import type { AgentId, SubAgentId } from '@/lib/dossier-vivant/events';
import { usageAt, type AgentView, type LogEntry, type MessageView, type RunState, type Usage } from '@/lib/dossier-vivant/runState';
import type { StageLayout } from '@/lib/dossier-vivant/scene';

import { Icon, type IconName } from './icons';
import { StateLine } from './StateLine';
import styles from './dossier-vivant.module.css';

/** Hauteur de l'éventail orchestrateur → colonnes (px). */
const FAN_H = 34;
/**
 * Téléphone : pas d'éventail (des colonnes sur deux rangées n'ont pas d'axe commun où le tracer),
 * un simple écart. Les envois restent lisibles dans le fil de l'orchestrateur.
 */
const COMPACT_GAP = 8;

/** Espace vertical entre l'orchestrateur et les colonnes. */
export function fanHeight(compact: boolean): number {
  return compact ? COMPACT_GAP : FAN_H;
}

/**
 * Colonnes par rangée. Grand écran et téléphone couché : une rangée, une colonne par sous-agent
 * (couché, la hauteur manque pour deux rangées, la largeur non). Téléphone en portrait : deux de
 * front (trois pour l'équipe à cinq d'un run réel) — sous ~110 px une colonne n'affiche plus rien.
 */
function columnsPerRow(count: number, layout: StageLayout): number {
  if (layout !== 'compact') return count;
  return count <= 4 ? 2 : 3;
}
/** Écart entre colonnes (px) — le même que `.cols { gap }`. */
const COL_GAP = 12;

/** Totaux formatés de l'équipe, partagés avec le dossier (sa ligne de remise). */
export interface TeamTotals {
  usage: Usage;
  elapsedLabel: string;
  costLabel: string;
}

/**
 * Sens d'un message dans le fil, déduit de ses interlocuteurs : choisit glyphe, teinte et paquet.
 * `side` — d'un sous-agent à un autre (l'Architecte et son Dessinateur) : il ne passe pas par
 * l'orchestrateur, donc aucun paquet dans l'éventail ; le fil le montre « De → À ».
 */
type Direction = 'in' | 'out' | 'self' | 'down' | 'up' | 'side';
function directionOf(m: MessageView): Direction {
  if (m.from === 'visitor') return 'in';
  if (m.to === 'visitor') return 'out';
  if (m.from === m.to) return 'self';
  if (m.from === 'orchestrator') return 'down';
  return m.to === 'orchestrator' ? 'up' : 'side';
}
const DIRECTION_ICON: Record<Direction, IconName> = { in: 'note', out: 'check', self: 'chev', down: 'down', up: 'up', side: 'chev' };
const LOG_ICON: Record<LogEntry['kind'], IconName> = { keep: 'plus', drop: 'x', question: 'q', done: 'check', note: 'chev' };

/** Point d'un chemin de l'éventail (cubique verticale), à l'avancement u. */
function fanPoint(x1: number, x2: number, u: number): [number, number] {
  const v = 1 - u;
  const x = v * v * v * x1 + 3 * v * v * u * x1 + 3 * v * u * u * x2 + u * u * u * x2;
  const y = 3 * v * v * u * (FAN_H * 0.6) + 3 * v * u * u * (FAN_H * 0.4) + u * u * u * FAN_H;
  return [x, y];
}

/**
 * AgentTeam — l'équipe au travail : l'orchestrateur en tête (fil des messages, totaux), l'éventail
 * où passent les paquets, et une colonne par sous-agent du run (`state.team` : quatre en rejeu,
 * cinq en run réel avec le Dessinateur) — état, appel d'outil en cours, journal. Le dossier (DesignDossier) se pose par-dessus, dans le même conteneur.
 * Téléphone : mêmes éléments, resserrés, sans éventail — en portrait (`compact`) l'orchestrateur
 * sur trois lignes (dernier message seul, CSS) et les colonnes en grille de deux ; couché
 * (`landscape`) l'orchestrateur sur deux lignes et demie et les colonnes sur une rangée.
 *
 * Noms d'agents et libellés d'état : interface (next-intl). Activités, appels et notes : contenu
 * du flux, affiché tel quel. Le modèle n'est affiché que si le flux le donne (`run.started`).
 * E2E : `dv-orchestrator`, `dv-agent-column` (`data-agent`, `data-status`), `dv-agent-state`,
 * `dv-thread-message`, `dv-totals`.
 */
export function AgentTeam({
  state,
  now,
  width,
  orchHeight,
  colsHeight,
  totals,
  layout,
}: {
  state: RunState;
  now: number;
  width: number;
  orchHeight: number;
  colsHeight: number;
  totals: TeamTotals;
  /** Disposition (scene.ts › layoutFor) : sur téléphone, pas d'éventail ; en portrait, colonnes en grille. */
  layout: StageLayout;
}) {
  const compact = layout !== 'wide';
  const t = useTranslations('dossierVivant.team');
  const locale = useLocale();
  const fmt = new Intl.NumberFormat(locale);
  const tokens = (u: Usage) => fmt.format(Math.round(u.tokens / 10) * 10);
  const shortName = (id: AgentId) => t(`agents.${id}.short`);
  const statusLabel = (a: AgentView) => a.activity ?? t(`status.${a.status}`);

  const orch = state.agents.orchestrator;
  const visitorName = state.visitor?.name ?? '';
  const shown = state.messages.slice(-3);

  // Paquets de l'éventail : un envoi part de l'orchestrateur, un retour y remonte.
  const team = state.team;
  const colW = (width - (team.length - 1) * COL_GAP) / team.length;
  const fanX = (i: number) => colW / 2 + i * (colW + COL_GAP);
  const live = new Set<number>();
  const packets = state.messages
    .map((m) => ({ m, dir: directionOf(m) }))
    .filter(({ m, dir }) => (dir === 'down' || dir === 'up') && now - m.at >= 0 && now - m.at <= CH.packetD)
    .slice(-6)
    .map(({ m, dir }) => {
      const agent = (dir === 'down' ? m.to : m.from) as SubAgentId;
      const i = team.indexOf(agent);
      live.add(i);
      const k = easeIO((now - m.at) / CH.packetD);
      const [x, y] = fanPoint(width / 2, fanX(i), dir === 'down' ? k : 1 - k);
      // Clé = rang du message (runState) : deux envois du même paquet réseau partagent leur `at`.
      return { key: m.id, x, y, dir };
    });

  return (
    <>
      <div className={styles.orch} data-testid="dv-orchestrator">
        <div>
          <h2 className={styles.agentName}>{t('agents.orchestrator.name')}</h2>
          <p className={styles.model}>{orch.model ?? t('role.orchestrator')}</p>
          <StateLine status={orch.status} label={statusLabel(orch)} now={now} />
        </div>
        <ul className={styles.threadList} aria-live="polite" aria-label={t('thread')}>
          {shown.map((m) => {
            const dir = directionOf(m);
            const who =
              dir === 'in' || dir === 'out'
                ? visitorName
                : dir === 'self'
                  ? t(`topics.${m.topic ?? 'plan'}`)
                  : dir === 'side'
                    ? t('between', { from: shortName(m.from as AgentId), to: shortName(m.to as AgentId) })
                    : shortName((dir === 'down' ? m.to : m.from) as AgentId);
            const e = easeOut(prog(now, m.at, CH.msgReveal));
            return (
              <li
                key={m.id}
                data-dir={dir}
                data-testid="dv-thread-message"
                style={{ opacity: e, transform: `translateY(${((1 - e) * 6).toFixed(2)}px)` }}
              >
                <Icon name={DIRECTION_ICON[dir]} />
                <b>{who}</b>
                <span className={styles.msg}>{m.text}</span>
              </li>
            );
          })}
        </ul>
        <div className={styles.totals} role="status" data-testid="dv-totals">
          <span>
            <b className={styles.num}>{tokens(totals.usage)}</b> {t('tokens')}
          </span>
          <span>
            <b className={styles.num}>{totals.costLabel}</b> {t('usd')}
          </span>
          <span>
            <b className={styles.num}>{totals.elapsedLabel}</b>
          </span>
        </div>
      </div>

      {!compact && (
      <svg className={styles.fan} style={{ top: orchHeight, width, height: FAN_H }} viewBox={`0 0 ${width} ${FAN_H}`} aria-hidden="true">
        {team.map((id, i) => {
          const x1 = width / 2;
          const x2 = fanX(i);
          return (
            <path
              key={id}
              className={styles.fanPath}
              data-live={live.has(i) ? '' : undefined}
              d={`M${x1},0C${x1},${FAN_H * 0.6} ${x2},${FAN_H * 0.4} ${x2},${FAN_H}`}
            />
          );
        })}
        {packets.map((p) => (
          <circle key={p.key} className={styles.packet} data-dir={p.dir} cx={p.x.toFixed(1)} cy={p.y.toFixed(1)} r={3} />
        ))}
      </svg>
      )}

      <div
        className={styles.cols}
        style={{ top: orchHeight + fanHeight(compact), height: colsHeight, gridTemplateColumns: `repeat(${columnsPerRow(team.length, layout)}, minmax(0, 1fr))` }}
      >
        {team.map((id) => {
          const a = state.agents[id];
          const call = a.call ? a.call.text.slice(0, Math.max(0, Math.round((now - a.call.at) * CH.toolCps))) : '';
          return (
            <section key={id} className={styles.col} aria-label={t(`agents.${id}.name`)} data-testid="dv-agent-column" data-agent={id} data-status={a.status}>
              <div>
                <h3 className={styles.colName}>
                  <span className={styles.nLong}>{t(`agents.${id}.name`)}</span>
                  <span className={styles.nShort}>{shortName(id)}</span>
                </h3>
                <p className={styles.colMeta}>
                  <span>{a.model ?? t('role.subAgent')}</span>
                  <span>
                    <span className={styles.num}>{tokens(usageAt(state, id, now))}</span> {t('tokens')}
                  </span>
                </p>
                <StateLine status={a.status} label={statusLabel(a)} now={now} testId="dv-agent-state" />
              </div>
              <p className={styles.act} data-testid="dv-agent-call">
                {call}
              </p>
              <ul className={styles.log}>
                {a.log.map((entry, i) => (
                  <li key={i} data-kind={entry.kind} style={reveal(now, entry.at, CH.logReveal, 8)}>
                    <Icon name={LOG_ICON[entry.kind]} />
                    <span>{entry.text}</span>
                  </li>
                ))}
              </ul>
            </section>
          );
        })}
      </div>
    </>
  );
}

'use client';

import { useLocale, useTranslations } from 'next-intl';

import type { RunFailureReason } from '@/lib/dossier-vivant/events';

import { Icon } from './icons';
import styles from './dossier-vivant.module.css';

/** Ce que la barre annonce : rien de lancé, le rejeu Tessera, ou un run réel (modèle, arrêt). */
export type BarRun =
  | { kind: 'idle' }
  | { kind: 'replay' }
  /** `over` : le round est fini (dossier remis ou run arrêté) — « Rejouer » peut alors en relancer un. */
  | { kind: 'live'; model?: string; failure: RunFailureReason | null; over: boolean };

/**
 * ReplayBar — la barre fine du dossier vivant : la mention de ce qui se joue et les contrôles de
 * lecture à droite, la ligne de progression en bas. Le nom (retour à l'accueil) n'est visible que sur
 * téléphone couché, où l'en-tête du site est masqué — ailleurs il le porte déjà (CSS `.wordmark`).
 * TROIS MENTIONS, chacune vraie de ce qu'elle annonce :
 *  - AVANT le départ (`idle`) : l'équipe tourne EN DIRECT sur le besoin du visiteur (« Lancer
 *    l'équipe », chemin par défaut) ; le bouton de lecture devient « Voir la démo » et lance le
 *    rejeu enregistré — la démo reste à un clic, sans être le défaut ;
 *  - REJEU (`replay`) : « rejouée, sans appel IA » à demeure — un enregistrement ne doit jamais
 *    passer pour une réponse au besoin du visiteur ; pause et rejouer actifs ;
 *  - DIRECT (`live`) : « équipe en direct » et le modèle annoncé par le flux. La pause n'y a pas de
 *    sens (on ne met pas des agents en pause) : désactivée. « Rejouer » relance un NOUVEAU round
 *    réel sur le même besoin, mais seulement une fois le round fini — pendant qu'il tourne, ce
 *    serait couper un run déjà payé pour en payer un second. Un arrêt du run (`run.failed`)
 *    remplace la mention, avec le glyphe ✕.
 * Doctrine de l'or : le filet en tête (surface d'Alexis) et la progression ; aucun bouton en or ici.
 * E2E : `dv-pause`, `dv-replay`, `dv-caption` (`data-mode` idle | replay | live, `data-failed`).
 */
export function ReplayBar({
  progress,
  playing,
  onToggle,
  onReplay,
  run,
}: {
  progress: number;
  playing: boolean;
  /** Avant le départ : lance la démo. Pendant le rejeu : pause / reprise. */
  onToggle: () => void;
  /** Rejeu : repart du début de la démo. Direct, round fini : relance un round réel. */
  onReplay: () => void;
  run: BarRun;
}) {
  const t = useTranslations('dossierVivant.bar');
  const tLive = useTranslations('dossierVivant.live');
  // Le nom d'Alexis : même clé que l'en-tête du site (nav.brand), une seule traduction.
  const tNav = useTranslations('nav');
  const locale = useLocale();
  const failure = run.kind === 'live' ? run.failure : null;
  return (
    <header className={styles.bar}>
      <a className={styles.wordmark} href={`/${locale}`} data-testid="dv-wordmark">
        {tNav('brand')}
      </a>
      <div className={styles.controls}>
        <span
          className={`${styles.caption} ${failure ? styles.captionFailed : ''}`}
          data-testid="dv-caption"
          data-mode={run.kind}
          data-failed={failure ?? undefined}
          role={failure ? 'alert' : undefined}
        >
          {run.kind === 'idle' ? (
            tLive('captionIdle')
          ) : run.kind === 'replay' ? (
            t('caption')
          ) : failure ? (
            <>
              <span aria-hidden="true">✕</span> {tLive(`failed.${failure}`)}
            </>
          ) : run.model ? (
            tLive('caption', { model: run.model })
          ) : (
            tLive('captionPending')
          )}
        </span>
        <button type="button" className={styles.btnQuiet} onClick={onToggle} disabled={run.kind === 'live'} data-testid="dv-pause">
          <Icon name={playing ? 'pause' : 'play'} />
          <span>{run.kind === 'idle' ? t('demo') : playing ? t('pause') : t('play')}</span>
        </button>
        <button type="button" className={styles.btnQuiet} onClick={onReplay} disabled={run.kind === 'idle' || (run.kind === 'live' && !run.over)} data-testid="dv-replay">
          <Icon name="replay" />
          <span>{t('replay')}</span>
        </button>
      </div>
      <div
        className={styles.progress}
        style={{ transform: `scaleX(${progress.toFixed(4)})` }}
        role="progressbar"
        aria-label={t('progress')}
        aria-valuemin={0}
        aria-valuemax={100}
        aria-valuenow={Math.round(progress * 100)}
      />
    </header>
  );
}

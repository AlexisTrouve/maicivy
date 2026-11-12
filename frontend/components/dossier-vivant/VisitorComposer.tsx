'use client';

import { forwardRef, type CSSProperties } from 'react';
import { useTranslations } from 'next-intl';

import type { RunFailureReason } from '@/lib/dossier-vivant/events';

import { Icon } from './icons';
import styles from './dossier-vivant.module.css';

/**
 * VisitorComposer — l'ouverture : la voix d'Alexis (la phrase de /montagne), puis le champ où le
 * visiteur décrit son besoin et le bouton qui lance l'équipe.
 *
 * Deux régimes :
 *  - ÉDITION (`edit` fourni — le défaut, pour tout visiteur) : besoin et présentation éditables,
 *    pré-remplis par l'exemple Tessera ; « Lancer l'équipe » part sur de VRAIS agents. Un refus
 *    avant le départ (saisie hors bornes, service indisponible) s'affiche sous le champ, glyphe ✕
 *    compris : la couleur seule ne dit pas l'état.
 *  - REJEU (`edit` absent, la démo lancée depuis la barre) : le champ montre le besoin du scénario
 *    enregistré, en LECTURE SEULE — laisser écrire un autre besoin pendant que Tessera se rejoue
 *    serait mentir.
 * Doctrine de l'or : « Lancer l'équipe » est l'action principale de cet écran, la seule en or plein.
 * E2E : `dv-hero`, `dv-need`, `dv-who`, `dv-launch`, `dv-live-error`.
 */
export const VisitorComposer = forwardRef<
  HTMLTextAreaElement,
  {
    need: string;
    who: string;
    onLaunch: () => void;
    style: CSSProperties;
    launchFocusable: boolean;
    /** Régime d'édition : modifie le besoin et la présentation ; `error` = refus avant le départ. */
    edit?: { onNeed: (v: string) => void; onWho: (v: string) => void; error: RunFailureReason | null };
  }
>(function VisitorComposer({ need, who, onLaunch, style, launchFocusable, edit }, needRef) {
  const t = useTranslations('dossierVivant.hero');
  const tLive = useTranslations('dossierVivant.live');
  const tVoice = useTranslations('montagne.voice');
  return (
    <section className={styles.hero} style={style} aria-labelledby="dv-hero-title" data-testid="dv-hero">
      <h1 id="dv-hero-title">{tVoice('title')}</h1>
      <form
        className={styles.composer}
        onSubmit={(e) => {
          e.preventDefault();
          onLaunch();
        }}
      >
        <label htmlFor="dv-need">{t('prompt')}</label>
        <textarea
          id="dv-need"
          ref={needRef}
          className={styles.need}
          value={need}
          readOnly={!edit}
          onChange={edit ? (e) => edit.onNeed(e.target.value) : undefined}
          maxLength={2000}
          data-editable={edit ? '' : undefined}
          aria-describedby={edit?.error ? 'dv-who dv-live-error' : 'dv-who'}
          data-testid="dv-need"
        />
        <div className={styles.composerRow}>
          <input
            id="dv-who"
            className={styles.who}
            value={who}
            readOnly={!edit}
            onChange={edit ? (e) => edit.onWho(e.target.value) : undefined}
            maxLength={120}
            data-editable={edit ? '' : undefined}
            aria-label={t('whoLabel')}
            data-testid="dv-who"
          />
          <button type="submit" className={styles.btnGold} tabIndex={launchFocusable ? 0 : -1} data-testid="dv-launch">
            <span>{t('launch')}</span>
            <Icon name="arrow" />
          </button>
        </div>
        {edit?.error && (
          <p id="dv-live-error" className={styles.liveError} role="alert" data-testid="dv-live-error">
            <span aria-hidden="true">✕</span> {tLive(`failed.${edit.error}`)}
          </p>
        )}
      </form>
    </section>
  );
});

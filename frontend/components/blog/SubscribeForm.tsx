'use client';

import React, { useEffect, useId, useState } from 'react';
import { useTranslations } from 'next-intl';
import { blogApi } from '@/lib/api';

// SubscribeForm — capture email avec granularité par topic.
//
// POURQUOI : un lecteur qui finit un article n'a aujourd'hui AUCUN moyen de revenir (retour user
// réel). Il peut ne s'abonner qu'aux sujets qui l'intéressent (ex: que Drifterra) ; aucune case
// cochée = il reçoit tout. Phase 1 = on enregistre l'abonné ; l'envoi des emails viendra en phase 2.
//
// FORME : une feuille posée sur la nuit, filet d'or en tête (surface d'Alexis) ; le bouton
// « S'abonner » est L'action principale de la page article — la seule en or. Les sujets sont des
// interrupteurs (aria-pressed) dont l'état se lit au glyphe ✓, pas à la seule couleur.
export function SubscribeForm() {
  const t = useTranslations('blog.follow');
  const emailId = useId();
  // Libellés lisibles des topics (clé = project_name renvoyé par l'API), traduits dans
  // blog.follow.topics. Un topic inconnu retombe sur sa clé brute : un nouveau projet s'affiche
  // quand même sans toucher au code.
  const topicLabels = (t.raw('topics') ?? {}) as Record<string, string>;
  const [topics, setTopics] = useState<string[]>([]);
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [email, setEmail] = useState('');
  const [status, setStatus] = useState<'idle' | 'loading' | 'ok' | 'error'>('idle');

  // Charger les topics suivables (dérivés en live des posts publiés). Échec silencieux : le form
  // reste utilisable sans cases (= abonnement à tout).
  useEffect(() => {
    blogApi.getTopics().then((r) => setTopics(r.topics || [])).catch(() => setTopics([]));
  }, []);

  const toggle = (topic: string) => {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(topic)) next.delete(topic);
      else next.add(topic);
      return next;
    });
  };

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    setStatus('loading');
    try {
      // Aucune case cochée → topics: [] → l'abonné reçoit TOUT.
      await blogApi.subscribe(email.trim(), Array.from(selected));
      setStatus('ok');
    } catch {
      setStatus('error');
    }
  };

  // Feuille commune aux deux états : filet d'or sur l'arête haute (un signal, pas un aplat).
  const sheet =
    "relative rounded-lg border border-border bg-card p-6 before:absolute before:inset-x-6 before:top-0 before:h-px before:bg-gold/60 before:content-['']";

  if (status === 'ok') {
    return (
      <div role="status" data-testid="subscribe-success" className={`${sheet} flex items-start gap-3 text-foreground`}>
        <span aria-hidden="true" className="mt-0.5 text-gold">✓</span>
        <p>{t('success')}</p>
      </div>
    );
  }

  return (
    <form onSubmit={submit} data-testid="subscribe-form" className={sheet}>
      <h2 className="text-2xl font-semibold text-foreground">{t('title')}</h2>
      <p className="mt-1 text-sm leading-relaxed text-muted-foreground">{t('intro')}</p>

      {topics.length > 0 && (
        <div role="group" aria-label={t('topicsLabel')} className="mt-5 flex flex-wrap gap-2">
          {topics.map((topic) => {
            const on = selected.has(topic);
            return (
              <button
                type="button"
                key={topic}
                onClick={() => toggle(topic)}
                aria-pressed={on}
                data-testid={`subscribe-topic-${topic}`}
                className={`inline-flex min-h-10 items-center gap-1.5 rounded-md border px-3 text-sm transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring ${
                  on
                    ? 'border-foreground/60 bg-secondary text-foreground'
                    : 'border-border text-muted-foreground hover:border-muted-foreground hover:text-foreground'
                }`}
              >
                {on && <span aria-hidden="true">✓</span>}
                {topicLabels[topic] || topic}
              </button>
            );
          })}
        </div>
      )}

      <div className="mt-5 flex flex-col gap-2 sm:flex-row">
        <label htmlFor={emailId} className="sr-only">
          {t('emailLabel')}
        </label>
        <input
          id={emailId}
          type="email"
          required
          autoComplete="email"
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          placeholder={t('emailPlaceholder')}
          data-testid="subscribe-email"
          // Un champ est un CREUX dans la feuille : fond de page, pas la surface de la carte.
          // 16 px : en dessous, iOS zoome la page au focus.
          className="h-12 min-w-0 flex-1 rounded-md border border-border bg-background px-3 text-base text-foreground placeholder:text-muted-foreground/70 focus-visible:border-gold/70 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/40"
        />
        <button
          type="submit"
          disabled={status === 'loading'}
          data-testid="subscribe-submit"
          className="inline-flex h-12 items-center justify-center rounded-md border border-gold bg-gold px-6 font-semibold text-primary-foreground transition-colors hover:bg-gold-bright disabled:cursor-wait disabled:opacity-70 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-gold-bright focus-visible:ring-offset-2 focus-visible:ring-offset-card"
        >
          {status === 'loading' ? t('sending') : t('subscribe')}
        </button>
      </div>

      {status === 'error' && (
        <p role="alert" data-testid="subscribe-error" className="mt-3 flex items-start gap-2 text-sm text-destructive">
          <span aria-hidden="true">✕</span>
          {t('error')}
        </p>
      )}
    </form>
  );
}

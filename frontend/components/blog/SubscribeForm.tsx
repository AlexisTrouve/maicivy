'use client';

import React, { useEffect, useState } from 'react';
import { blogApi } from '@/lib/api';

// Libellés lisibles pour les topics (clé = project_name renvoyé par l'API). Un topic inconnu
// retombe sur sa clé brute, donc un nouveau projet s'affiche quand même sans toucher au code.
const TOPIC_LABELS: Record<string, string> = {
  Drifterra: 'Drifterra (le jeu)',
  tech: 'Dev / tech',
  veille: 'Veille tech',
};

interface SubscribeFormProps {
  locale?: string;
}

// SubscribeForm — capture email avec granularité par topic.
//
// POURQUOI : un lecteur qui finit un article n'a aujourd'hui AUCUN moyen de revenir (retour user
// réel). Il peut ne s'abonner qu'aux sujets qui l'intéressent (ex: que Drifterra) ; aucune case
// cochée = il reçoit tout. Phase 1 = on enregistre l'abonné ; l'envoi des emails viendra en phase 2.
export function SubscribeForm({ locale = 'fr' }: SubscribeFormProps) {
  const fr = locale === 'fr';
  const [topics, setTopics] = useState<string[]>([]);
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [email, setEmail] = useState('');
  const [status, setStatus] = useState<'idle' | 'loading' | 'ok' | 'error'>('idle');

  // Charger les topics suivables (dérivés en live des posts publiés). Échec silencieux : le form
  // reste utilisable sans cases (= abonnement à tout).
  useEffect(() => {
    blogApi.getTopics().then((r) => setTopics(r.topics || [])).catch(() => setTopics([]));
  }, []);

  const toggle = (t: string) => {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(t)) next.delete(t);
      else next.add(t);
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

  if (status === 'ok') {
    return (
      <div className="rounded-xl border border-green-200 dark:border-green-800 bg-green-50 dark:bg-green-900/20 p-6 text-green-800 dark:text-green-300">
        {fr
          ? 'Inscrit. Tu recevras les nouveaux articles que tu as choisis.'
          : "You're in. You'll get the new posts you picked."}
      </div>
    );
  }

  return (
    <form
      onSubmit={submit}
      className="rounded-xl border border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800/50 p-6"
    >
      <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-1">
        {fr ? 'Suivre par email' : 'Follow by email'}
      </h3>
      <p className="text-sm text-gray-600 dark:text-gray-400 mb-4">
        {fr
          ? 'Reçois les nouveaux articles. Choisis tes sujets (rien de coché = tout).'
          : 'Get new posts. Pick your topics (none checked = everything).'}
      </p>

      {topics.length > 0 && (
        <div className="flex flex-wrap gap-2 mb-4">
          {topics.map((t) => {
            const on = selected.has(t);
            return (
              <button
                type="button"
                key={t}
                onClick={() => toggle(t)}
                aria-pressed={on}
                className={`text-sm px-3 py-1 rounded-full border transition-colors ${
                  on
                    ? 'bg-blue-600 border-blue-600 text-white'
                    : 'bg-white dark:bg-gray-700 border-gray-300 dark:border-gray-600 text-gray-700 dark:text-gray-300 hover:border-blue-400'
                }`}
              >
                {TOPIC_LABELS[t] || t}
              </button>
            );
          })}
        </div>
      )}

      <div className="flex flex-col sm:flex-row gap-2">
        <input
          type="email"
          required
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          placeholder={fr ? 'ton@email.com' : 'you@email.com'}
          className="flex-1 px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-900 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500"
        />
        <button
          type="submit"
          disabled={status === 'loading'}
          className="px-5 py-2 rounded-lg bg-blue-600 hover:bg-blue-700 disabled:opacity-60 text-white font-medium transition-colors"
        >
          {status === 'loading' ? '…' : fr ? "S'abonner" : 'Subscribe'}
        </button>
      </div>

      {status === 'error' && (
        <p className="mt-2 text-sm text-red-600 dark:text-red-400">
          {fr ? 'Oups, réessaie (email valide ?).' : 'Oops, try again (valid email?).'}
        </p>
      )}
    </form>
  );
}

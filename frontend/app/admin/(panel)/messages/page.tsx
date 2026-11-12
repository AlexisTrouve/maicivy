'use client';

import { useState } from 'react';
import { useTranslations } from 'next-intl';
import { Panel, PageTitle, SectionLabel, btnPrimary, btnSuccess, fieldArea, fieldInline } from '@/components/admin/ui';

const API = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

// Outil « Messages plateforme » du panneau admin. Colle une mission (Malt/LinkedIn/Upwork) → le
// backend (POST /messages/generate, sync ~5s) renvoie un message prêt à envoyer. credentials:include
// → cookie admin = owner → modèle Opus, pas de rate-limit (vs version publique Haiku plafonnée).
export default function AdminMessagesTool() {
  const t = useTranslations('admin');
  const [mission, setMission] = useState('');
  const [platform, setPlatform] = useState('malt');
  const [tjm, setTjm] = useState('');
  const [lang, setLang] = useState('fr');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [content, setContent] = useState('');
  const [copied, setCopied] = useState(false);

  async function generate(e: React.FormEvent) {
    e.preventDefault();
    setLoading(true);
    setError('');
    setContent('');
    setCopied(false);
    try {
      const body: Record<string, unknown> = { mission, platform, lang };
      if (tjm) body.tjm = parseInt(tjm, 10); // optionnel — omis si vide (backend: 50–5000)
      const res = await fetch(`${API}/api/v1/messages/generate`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify(body),
      });
      if (!res.ok) {
        let msg = `${t('errorPrefix')} ${res.status}`;
        try {
          const j = await res.json();
          if (j?.error) msg = j.error;
        } catch {
          /* corps non-JSON */
        }
        setError(msg);
        return;
      }
      const j = await res.json();
      setContent(j.content || '');
    } catch {
      setError(t('login.networkError'));
    } finally {
      setLoading(false);
    }
  }

  async function copy() {
    await navigator.clipboard.writeText(content);
    setCopied(true);
  }

  return (
    <div className="mx-auto grid h-full w-full max-w-6xl gap-8 lg:grid-cols-2" data-testid="admin-messages-tool">
      {/* Saisie */}
      <form onSubmit={generate} className="flex flex-col gap-4">
        <PageTitle sub={<span data-testid="msg-help">{t('messages.help')}</span>}>{t('messages.title')}</PageTitle>
        <textarea
          value={mission}
          onChange={(e) => setMission(e.target.value)}
          placeholder={t('messages.missionPlaceholder')}
          rows={12}
          data-testid="msg-mission"
          // flex-1 : cf. commentaire de l'outil CV — les deux colonnes descendent jusqu'en bas
          // plutôt que de se tasser en haut d'un grand écran.
          className={`${fieldArea} flex-1`}
        />
        <div className="flex flex-wrap items-center gap-3">
          <select value={platform} onChange={(e) => setPlatform(e.target.value)} data-testid="msg-platform" className={`${fieldInline} cursor-pointer`}>
            <option value="malt">Malt</option>
            <option value="linkedin">LinkedIn</option>
            <option value="upwork">Upwork</option>
          </select>
          <input
            type="number"
            value={tjm}
            onChange={(e) => setTjm(e.target.value)}
            placeholder={t('messages.tjmPlaceholder')}
            data-testid="msg-tjm"
            // tabular-nums : un tarif se compare d'un coup d'œil, les chiffres doivent s'aligner.
            className={`${fieldInline} w-28 tabular-nums`}
          />
          <select value={lang} onChange={(e) => setLang(e.target.value)} data-testid="msg-lang" className={`${fieldInline} cursor-pointer`}>
            <option value="fr">{t('outputLang.fr')}</option>
            <option value="en">{t('outputLang.en')}</option>
          </select>
          <button type="submit" disabled={loading || mission.trim().length < 20} data-testid="msg-generate" className={btnPrimary}>
            {loading ? t('messages.generating') : t('messages.generate')}
          </button>
        </div>
        {error && (
          <p data-testid="msg-error" className="text-sm text-rose-400">
            {error}
          </p>
        )}
      </form>

      {/* Résultat — le livrable, donc filet doré. */}
      {/* Pleine hauteur : la colonne de saisie descend jusqu'en bas, le résultat l'accompagne — et
          un message généré est long, il a besoin de cette place pour se lire sans défiler. */}
      <Panel hairline className="flex min-h-[300px] flex-col overflow-hidden">
        {content ? (
          <>
            <div className="flex items-center justify-between gap-3 border-b border-slate-800/80 px-4 py-2.5">
              <SectionLabel>{t('messages.resultTitle')}</SectionLabel>
              <button onClick={copy} data-testid="msg-copy" className={btnSuccess}>
                {/* La traduction porte déjà son « ✓ » (clé messages.copied). */}
                {copied ? t('messages.copied') : t('messages.copy')}
              </button>
            </div>
            {/* Le message part dans un formulaire de plateforme : il se lit comme une lettre, pas
                comme du code. D'où la police du corps de texte et un interligne aéré, alors qu'un
                <pre> hérite par défaut d'une chasse fixe serrée. */}
            <pre
              data-testid="msg-result"
              className="flex-1 overflow-auto whitespace-pre-wrap p-5 font-sans text-[15px] leading-[1.7] text-slate-100"
            >
              {content}
            </pre>
          </>
        ) : (
          <div className="flex flex-1 items-center justify-center p-8 text-center text-sm text-slate-500">
            {loading ? t('messages.loading') : t('messages.empty')}
          </div>
        )}
      </Panel>
    </div>
  );
}

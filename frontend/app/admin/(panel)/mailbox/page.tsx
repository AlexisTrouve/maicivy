'use client';

import { useCallback, useEffect, useState } from 'react';
import { useLocale, useTranslations } from 'next-intl';
import { mailboxApi } from '@/lib/api';
import type { MailboxEmail, MailboxEmailSummary, MailboxTranslation } from '@/lib/types';
import { splitMailboxBody, shortLinkLabel } from '@/lib/mailboxBody';
import { Panel, btnGhost, btnNeutral, btnPrimary, field } from '@/components/admin/ui';

const PER_PAGE = 20;

// Outil « Boîte mail » du panneau admin. Liste (sidebar, paginée/filtrée) + détail (panneau), sur le
// modèle de chat/page.tsx. Consulter un mail le marque lu côté serveur (convention client mail) ; le
// retry de transfert peut échouer si le service IMAP/SMTP n'est pas configuré (503 backend).
export default function AdminMailboxTool() {
  const t = useTranslations('admin.mailbox');
  const locale = useLocale();

  const [emails, setEmails] = useState<MailboxEmailSummary[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [totalPages, setTotalPages] = useState(1);
  const [platform, setPlatform] = useState('');
  const [unreadOnly, setUnreadOnly] = useState(false);
  const [listLoading, setListLoading] = useState(true);
  const [listFailed, setListFailed] = useState(false);

  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [detail, setDetail] = useState<MailboxEmail | null>(null);
  const [detailLoading, setDetailLoading] = useState(false);
  const [retrying, setRetrying] = useState(false);
  const [showCoT, setShowCoT] = useState(false);

  // Traduction à la demande (cf. GetOrTranslateMailboxEmail côté backend) — jamais automatique tant
  // qu'aucun cache n'existe : translation=null + pas de traduction en cours → bouton "Traduire".
  const [translation, setTranslation] = useState<MailboxTranslation | null>(null);
  const [translating, setTranslating] = useState(false);
  const [translationFailed, setTranslationFailed] = useState(false);
  const [showOriginal, setShowOriginal] = useState(false);

  // POURQUOI pas `t` dans les deps : useTranslations renvoie une closure non-mémoïsée dans le mock de
  // test — l'inclure ferait recréer loadList (et re-déclencher le useEffect ci-dessous) à CHAQUE
  // render, en boucle. Le message d'erreur est traduit au render (cf. listFailed plus bas), pas ici.
  const loadList = useCallback(
    async (targetPage: number) => {
      setListLoading(true);
      setListFailed(false);
      try {
        const res = await mailboxApi.list({
          page: targetPage,
          perPage: PER_PAGE,
          platform: platform || undefined,
          unread: unreadOnly || undefined,
        });
        setEmails(res.emails);
        setTotal(res.total);
        setPage(res.page);
        setTotalPages(res.total_pages);
      } catch {
        setListFailed(true);
      } finally {
        setListLoading(false);
      }
    },
    [platform, unreadOnly],
  );

  useEffect(() => {
    loadList(1);
  }, [loadList]);

  async function selectEmail(id: string) {
    setSelectedId(id);
    setDetailLoading(true);
    setShowCoT(false);
    setShowOriginal(false);
    setTranslationFailed(false);
    try {
      const d = await mailboxApi.getById(id);
      setDetail(d);
      // Le backend marque le mail lu à la consultation — refléter localement sans re-fetch la liste.
      setEmails((prev) => prev.map((e) => (e.id === id ? { ...e, read: true } : e)));
    } catch {
      setDetail(null);
    } finally {
      setDetailLoading(false);
    }
  }

  // Check cache-only (jamais de traduction déclenchée ici) à chaque changement de mail OU de langue
  // d'interface — "on sert auto si c'est en cache" : si une traduction existe déjà pour cette langue,
  // on la montre directement, sans repasser par un clic. `fr` = langue source, rien à vérifier.
  useEffect(() => {
    if (!detail) return;
    if (locale === 'fr') {
      setTranslation(null);
      return;
    }
    let cancelled = false;
    setTranslation(null);
    mailboxApi
      .getTranslation(detail.id, locale)
      .then((t) => {
        if (!cancelled) setTranslation(t);
      })
      .catch(() => {
        // 404 = rien en cache pour l'instant, comportement normal — le bouton "Traduire" reste
        // disponible pour déclencher la traduction à la demande. Pas une erreur à afficher.
      });
    return () => {
      cancelled = true;
    };
  }, [detail?.id, locale]);

  async function translateNow() {
    if (!detail) return;
    setTranslating(true);
    setTranslationFailed(false);
    try {
      const t = await mailboxApi.translateNow(detail.id, locale);
      setTranslation(t);
    } catch {
      setTranslationFailed(true);
    } finally {
      setTranslating(false);
    }
  }

  async function toggleRead(e: React.MouseEvent) {
    e.stopPropagation();
    if (!detail) return;
    const next = !detail.read;
    await mailboxApi.setRead(detail.id, next);
    setDetail({ ...detail, read: next });
    setEmails((prev) => prev.map((em) => (em.id === detail.id ? { ...em, read: next } : em)));
  }

  async function retryForward() {
    if (!detail) return;
    setRetrying(true);
    try {
      await mailboxApi.retryForward(detail.id);
      const refreshed = await mailboxApi.getById(detail.id);
      setDetail(refreshed);
      setEmails((prev) =>
        prev.map((e) =>
          e.id === refreshed.id
            ? {
                ...e,
                forwarded_at: refreshed.forwarded_at,
                forward_error: refreshed.forward_error,
                forward_blocked: refreshed.forward_blocked,
              }
            : e,
        ),
      );
    } finally {
      setRetrying(false);
    }
  }

  function formatDate(iso: string) {
    try {
      return new Date(iso).toLocaleString(locale);
    } catch {
      return iso;
    }
  }

  return (
    <div className="flex h-[calc(100vh-4rem)] gap-4" data-testid="admin-mailbox-tool">
      {/* Liste */}
      <Panel className="flex w-96 shrink-0 flex-col overflow-hidden">
        <div className="border-b border-slate-800/80 p-3">
          <h1 className="font-heading text-base font-semibold text-slate-100">{t('title')}</h1>
          <p className="mt-0.5 text-xs text-slate-500">{t('help')}</p>
          {/* flex-wrap + min-w-0 : un <input> a une largeur intrinsèque (~20 caractères) sous
              laquelle flex-1 refuse de le réduire. Dans une colonne de 384 px, le libellé de la case
              à cocher se retrouvait poussé contre le bord du panneau selon la longueur de sa
              traduction — et cinq langues ne donnent pas la même longueur. */}
          <div className="mt-3 flex flex-wrap items-center gap-2">
            <input
              value={platform}
              onChange={(e) => setPlatform(e.target.value)}
              placeholder={t('platformFilterPlaceholder')}
              data-testid="mailbox-filter-platform"
              className={`${field} min-w-0 flex-1 py-1.5 text-xs`}
            />
            <label className="flex shrink-0 cursor-pointer items-center gap-1.5 text-xs text-slate-400 hover:text-slate-200">
              <input
                type="checkbox"
                checked={unreadOnly}
                onChange={(e) => setUnreadOnly(e.target.checked)}
                data-testid="mailbox-filter-unread"
                className="accent-gold"
              />
              {t('unreadOnly')}
            </label>
          </div>
        </div>

        <ul className="flex-1 overflow-auto" data-testid="mailbox-list">
          {listLoading && <li className="p-4 text-xs text-slate-500">{t('loading')}</li>}
          {!listLoading && listFailed && <li className="p-4 text-xs text-rose-400">{t('loadError')}</li>}
          {!listLoading && !listFailed && emails.length === 0 && (
            <li className="p-4 text-xs text-slate-500" data-testid="mailbox-empty">
              {t('empty')}
            </li>
          )}
          {emails.map((e) => (
            <li
              key={e.id}
              onClick={() => selectEmail(e.id)}
              data-testid={`mailbox-item-${e.id}`}
              // Le mail ouvert porte la même barre dorée que la navigation : dans une liste de vingt
              // lignes qui se ressemblent, un simple fond plus clair se perd au défilement.
              className={`relative cursor-pointer border-b border-slate-800/50 py-2.5 pl-3 pr-3 transition-colors ${
                selectedId === e.id ? 'bg-slate-800/60' : 'hover:bg-slate-800/30'
              }`}
            >
              {selectedId === e.id && (
                <span aria-hidden="true" className="absolute inset-y-1.5 left-0 w-[3px] rounded-full bg-gold" />
              )}
              <div className="flex items-center justify-between gap-2">
                <span className={`truncate text-sm ${e.read ? 'text-slate-400' : 'font-semibold text-slate-50'}`}>
                  {e.from_address}
                </span>
                <span className="shrink-0 rounded border border-slate-700/70 bg-slate-800/60 px-1.5 py-0.5 text-[10px] text-slate-400">
                  {e.platform}
                </span>
              </div>
              <div className="truncate text-xs text-slate-500">{e.subject || t('noSubject')}</div>
              <div className="mt-1 flex items-center justify-between text-[10px] text-slate-600">
                <span>{formatDate(e.received_at)}</span>
                <span className="flex items-center gap-1.5">
                  {e.is_opportunity && typeof e.relevance_score === 'number' && (
                    <span className={e.forward_blocked ? 'text-amber-400' : 'text-slate-500'}>
                      {e.relevance_score}/100
                    </span>
                  )}
                  {e.forward_blocked && (
                    <span className="font-semibold text-amber-400" data-testid={`mailbox-blocked-${e.id}`}>
                      {t('blocked')}
                    </span>
                  )}
                  {!e.forward_blocked && e.forwarded_at && <span>{t('forwarded')}</span>}
                  {!e.forward_blocked && !e.forwarded_at && e.forward_error && <span>{t('forwardFailed')}</span>}
                </span>
              </div>
            </li>
          ))}
        </ul>

        <div className="flex items-center justify-between border-t border-slate-800/80 p-2 text-xs text-slate-500">
          <button onClick={() => loadList(page - 1)} disabled={page <= 1 || listLoading} data-testid="mailbox-prev-page" className={btnGhost}>
            ←
          </button>
          <span className="tabular-nums">{t('pageIndicator', { page, totalPages, total })}</span>
          <button
            onClick={() => loadList(page + 1)}
            disabled={page >= totalPages || listLoading}
            data-testid="mailbox-next-page"
            className={btnGhost}
          >
            →
          </button>
        </div>
      </Panel>

      {/* Détail */}
      <Panel hairline className="flex min-w-0 flex-1 flex-col p-5" testid="mailbox-detail">
        {!selectedId && (
          <div className="flex flex-1 items-center justify-center text-center text-sm text-slate-500">
            {t('selectPrompt')}
          </div>
        )}
        {selectedId && detailLoading && (
          <div className="flex flex-1 items-center justify-center text-sm text-slate-500">{t('loading')}</div>
        )}
        {selectedId && !detailLoading && detail && (() => {
          const useTranslation = Boolean(translation) && !showOriginal;
          const shownSubject = useTranslation ? translation!.subject : detail.subject;
          const shownBody = useTranslation ? translation!.body : detail.body_text;
          return (
          <div className="flex flex-1 flex-col gap-3 overflow-auto">
            <div className="flex items-start justify-between gap-3">
              <div className="min-w-0">
                <h2 className="truncate font-heading text-lg font-semibold text-slate-50">{shownSubject || t('noSubject')}</h2>
                <p className="truncate text-xs text-slate-500">
                  {detail.from_address} · {formatDate(detail.received_at)}
                </p>
              </div>
              <span className="shrink-0 rounded-md border border-slate-700/70 bg-slate-800/60 px-2 py-1 text-xs text-slate-400">
                {detail.platform}
              </span>
            </div>

            {locale !== 'fr' && (
              <div className="flex items-center gap-2 text-xs">
                {translation ? (
                  <button
                    onClick={() => setShowOriginal((v) => !v)}
                    data-testid="mailbox-toggle-translation"
                    className="font-medium text-sky-400 underline decoration-dotted underline-offset-2 hover:text-sky-300"
                  >
                    {showOriginal ? t('showTranslation') : t('showOriginal')}
                  </button>
                ) : (
                  <button
                    onClick={translateNow}
                    disabled={translating}
                    data-testid="mailbox-translate-button"
                    className={`${btnNeutral} px-2.5 py-1 text-xs`}
                  >
                    {translating ? t('translating') : t('translate')}
                  </button>
                )}
                {translationFailed && (
                  <span className="text-rose-400" data-testid="mailbox-translation-error">
                    {t('translationFailed')}
                  </span>
                )}
              </div>
            )}

            {detail.is_opportunity && typeof detail.relevance_score === 'number' && (
              <div
                className={`rounded-lg border px-3 py-2.5 text-xs ${
                  detail.forward_blocked
                    ? 'border-amber-700/70 bg-amber-950/30 text-amber-300'
                    : 'border-slate-800 bg-slate-950/60 text-slate-400'
                }`}
                data-testid="mailbox-relevance"
              >
                <div className="flex flex-wrap items-center justify-between gap-2">
                  <span>
                    <span className="font-semibold">{t('relevanceScore', { score: detail.relevance_score })}</span>
                    {detail.relevance_reason && <span> — {detail.relevance_reason}</span>}
                  </span>
                  {detail.relevance_link && (
                    <a
                      href={detail.relevance_link}
                      target="_blank"
                      rel="noopener noreferrer"
                      data-testid="mailbox-relevance-link"
                      // Le lien vers la mission est ce pour quoi ce mail existe : c'est l'action
                      // principale du panneau, donc l'or.
                      className={`${btnPrimary} shrink-0 px-2.5 py-1 text-[11px]`}
                    >
                      {t('viewMission')} ↗
                    </a>
                  )}
                </div>
                {detail.forward_blocked && <div className="mt-1">{t('blockedExplain')}</div>}
                {detail.relevance_cot && (
                  <div className="mt-2">
                    <button
                      onClick={() => setShowCoT((v) => !v)}
                      data-testid="mailbox-toggle-cot"
                      className="text-[11px] font-medium underline decoration-dotted hover:text-slate-200"
                    >
                      {showCoT ? t('hideReasoning') : t('showReasoning')}
                    </button>
                    {showCoT && (
                      <pre
                        className="mt-1.5 whitespace-pre-wrap break-words rounded-md border border-slate-800 bg-slate-950/70 p-2.5 text-[11px] leading-relaxed text-slate-400"
                        data-testid="mailbox-cot"
                      >
                        {detail.relevance_cot}
                      </pre>
                    )}
                  </div>
                )}
              </div>
            )}

            <div className="flex flex-wrap items-center gap-2">
              <button onClick={toggleRead} data-testid="mailbox-toggle-read" className={`${btnNeutral} px-3 py-1.5 text-xs`}>
                {detail.read ? t('markUnread') : t('markRead')}
              </button>
              <button
                onClick={retryForward}
                disabled={retrying}
                data-testid="mailbox-retry-forward"
                className={`${btnNeutral} px-3 py-1.5 text-xs`}
              >
                {retrying ? t('retrying') : detail.forward_blocked ? t('forceForward') : t('retryForward')}
              </button>
              {/* Pas de glyphe ajouté ici : la traduction porte DÉJÀ son « ✓ » (cf. messages/*.json,
                  clé mailbox.forwarded). En rajouter un affichait « ✓ Transféré ✓ ». */}
              {detail.forwarded_at && <span className="text-xs text-emerald-400">{t('forwarded')}</span>}
              {!detail.forwarded_at && detail.forward_error && (
                <span className="text-xs text-rose-400" data-testid="mailbox-forward-error">
                  {detail.forward_error}
                </span>
              )}
            </div>

            {/* Un mail se lit comme du texte, pas comme un journal machine : police du corps et
                interligne aéré, malgré le <pre> qui préserve les retours à la ligne d'origine. */}
            <pre
              className="flex-1 whitespace-pre-wrap break-words rounded-lg border border-slate-800 bg-slate-950/60 p-4 font-sans text-sm leading-[1.7] text-slate-200"
              data-testid="mailbox-body"
            >
              {shownBody
                ? splitMailboxBody(shownBody).map((seg, i) =>
                    seg.type === 'link' ? (
                      <a
                        key={i}
                        href={seg.value}
                        target="_blank"
                        rel="noopener noreferrer"
                        title={seg.value}
                        data-testid="mailbox-body-link"
                        className="text-sky-400 underline decoration-dotted underline-offset-2 hover:text-sky-300"
                      >
                        🔗 {shortLinkLabel(seg.value)}
                      </a>
                    ) : (
                      <span key={i}>{seg.value}</span>
                    ),
                  )
                : t('emptyBody')}
            </pre>
          </div>
          );
        })()}
      </Panel>
    </div>
  );
}

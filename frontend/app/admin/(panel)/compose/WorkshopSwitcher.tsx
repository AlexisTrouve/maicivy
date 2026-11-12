'use client';

import { useCallback, useEffect, useRef, useState } from 'react';
import { useLocale, useTranslations } from 'next-intl';
import { Archive, ArchiveRestore, Check, ChevronDown, MoreHorizontal, Pencil } from 'lucide-react';

const API = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

// Sélecteur d'atelier — l'historique de travail de l'opératrice.
//
// POURQUOI cet écran en avait besoin : « nouvel atelier » enterrait le précédent. L'opératrice traite
// plusieurs annonces dans la journée, revient sur l'une d'elles quand un client répond, et n'avait
// aucun moyen de la retrouver. Le travail existait pourtant déjà en base.
//
// POURQUOI un menu déroulant depuis le titre plutôt qu'une colonne permanente : changer d'atelier est
// un geste OCCASIONNEL, contrairement au chat où l'on saute de conversation en conversation. Une
// colonne fixe prendrait en permanence de la largeur au message et à sa traduction — les deux choses
// qu'elle lit vraiment — pour une action faite quelques fois par jour. Et le manque le plus criant
// n'était pas la liste : c'était de ne pas savoir SUR QUELLE annonce on travaille. Le titre en tête
// d'écran répond à ça en permanence, la liste n'est que son prolongement.
//
// POURQUOI on archive et on ne supprime jamais (décision d'Alexi) : un message envoyé à un client est
// une trace qu'on peut vouloir relire des mois plus tard. Un archivage se défait, pas une suppression.

// WorkshopSummary : ce que renvoie la liste. Volontairement SANS le contenu (annonce, message, fil) —
// le fil embarque les captures d'écran en base64, et les charger pour tous les ateliers à chaque
// ouverture de l'écran coûtait déjà des centaines de kilo-octets avant même qu'on garde l'historique.
export interface WorkshopSummary {
  id: string;
  title: string;
  rate: number;
  has_draft: boolean;
  checks_pass: number;
  checks_total: number;
  sent_at?: string;
  archived_at?: string;
  updated_at: string;
}

export default function WorkshopSwitcher({
  currentId,
  currentTitle,
  onOpen,
  onCurrentChanged,
}: {
  currentId: string | null;
  currentTitle: string;
  /** Charger un atelier — c'est le parent qui détient l'état de l'écran. */
  onOpen: (id: string) => void;
  /** L'atelier COURANT a été renommé ou archivé : le parent doit rafraîchir son affichage. */
  onCurrentChanged: (patch: { title?: string; archived?: boolean }) => void;
}) {
  const t = useTranslations('admin.compose');
  const locale = useLocale();

  const [open, setOpen] = useState(false);
  const [archived, setArchived] = useState(false);
  const [items, setItems] = useState<WorkshopSummary[]>([]);
  const [loading, setLoading] = useState(false);
  const [menuFor, setMenuFor] = useState<string | null>(null);
  const [renaming, setRenaming] = useState<string | null>(null);
  const [renameValue, setRenameValue] = useState('');
  const rootRef = useRef<HTMLDivElement>(null);

  const load = useCallback(
    async (wantArchived: boolean) => {
      setLoading(true);
      try {
        const res = await fetch(`${API}/api/v1/admin/compose/drafts?archived=${wantArchived}`, {
          credentials: 'include',
        });
        if (!res.ok) return;
        const { drafts } = await res.json();
        setItems(Array.isArray(drafts) ? drafts : []);
      } catch {
        // Liste indisponible : on laisse l'existant à l'écran plutôt que de la vider. Une liste vide
        // se lirait comme « tu n'as aucun atelier », ce qui serait un mensonge alarmant.
      } finally {
        setLoading(false);
      }
    },
    [],
  );

  // Chargée à l'OUVERTURE et non au montage : inutile d'aller chercher l'historique sur un écran où
  // l'on vient rédiger. C'est aussi ce qui garantit une liste à jour à chaque consultation.
  useEffect(() => {
    if (open) void load(archived);
  }, [open, archived, load]);

  // Fermeture au clic extérieur et à Échap — un panneau flottant qu'on ne peut pas fermer autrement
  // qu'en retrouvant son bouton est un piège.
  useEffect(() => {
    if (!open) return;
    const onDown = (e: MouseEvent) => {
      if (!rootRef.current?.contains(e.target as Node)) {
        setOpen(false);
        setMenuFor(null);
      }
    };
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        setOpen(false);
        setMenuFor(null);
      }
    };
    document.addEventListener('mousedown', onDown);
    document.addEventListener('keydown', onKey);
    return () => {
      document.removeEventListener('mousedown', onDown);
      document.removeEventListener('keydown', onKey);
    };
  }, [open]);

  // patchMeta — renommer / archiver / marquer envoyé. Endpoint SÉPARÉ de la sauvegarde d'état : une
  // modification de métadonnée ne doit pas réémettre l'annonce, le message et tout le fil (volume
  // absurde, et surtout risque d'écraser un état plus récent).
  const patchMeta = useCallback(
    async (id: string, body: Record<string, unknown>) => {
      try {
        const res = await fetch(`${API}/api/v1/admin/compose/drafts/${id}/meta`, {
          method: 'PATCH',
          headers: { 'Content-Type': 'application/json' },
          credentials: 'include',
          body: JSON.stringify(body),
        });
        if (!res.ok) return;
        await load(archived);
        // Si c'est l'atelier ouvert qui change, l'écran doit suivre — sinon le titre affiché en tête
        // contredirait la liste juste en dessous.
        if (id === currentId) {
          onCurrentChanged({
            title: typeof body.title === 'string' ? body.title : undefined,
            archived: typeof body.archived === 'boolean' ? body.archived : undefined,
          });
        }
      } catch {
        /* échec réseau : la liste reste telle quelle, l'opératrice peut réessayer */
      }
    },
    [archived, currentId, load, onCurrentChanged],
  );

  const formatDate = (iso: string) => {
    try {
      return new Date(iso).toLocaleDateString(locale, { day: 'numeric', month: 'short' });
    } catch {
      return iso.slice(0, 10);
    }
  };

  const label = currentTitle.trim() || t('untitled');

  return (
    <div className="relative min-w-0" ref={rootRef}>
      {/* Le titre de l'atelier courant EST le bouton d'ouverture. Deux informations pour un seul
          élément : où je suis, et comment aller ailleurs. */}
      <button
        type="button"
        data-testid="compose-workshop-button"
        aria-expanded={open}
        aria-label={t('openWorkshops')}
        onClick={() => setOpen((v) => !v)}
        className="flex min-w-0 items-center gap-1.5 rounded-lg px-2 py-1 text-left text-sm font-medium text-slate-100 transition-colors hover:bg-slate-800/60"
      >
        <span className="truncate">{label}</span>
        <ChevronDown aria-hidden="true" className={`h-3.5 w-3.5 shrink-0 text-slate-500 transition-transform ${open ? 'rotate-180' : ''}`} />
      </button>

      {open && (
        <div
          data-testid="compose-workshop-list"
          className="absolute left-0 top-full z-30 mt-2 w-[22rem] overflow-hidden rounded-xl border border-slate-800 bg-surface-raised shadow-panel-gold"
        >
          {/* Deux onglets plutôt qu'un filtre : « en cours » et « archivés » sont deux intentions
              distinctes, et l'archive doit rester à portée sans encombrer la vue courante. */}
          <div className="flex border-b border-slate-800 text-[11px]">
            {[false, true].map((tab) => (
              <button
                key={String(tab)}
                type="button"
                data-testid={`compose-workshop-tab-${tab ? 'archived' : 'active'}`}
                onClick={() => {
                  setArchived(tab);
                  setMenuFor(null);
                }}
                className={`flex-1 px-3 py-2 font-medium uppercase tracking-wider transition-colors ${
                  archived === tab ? 'bg-gold/10 text-gold-soft' : 'text-slate-500 hover:text-slate-300'
                }`}
              >
                {tab ? t('archivedTab') : t('activeTab')}
              </button>
            ))}
          </div>

          <ul className="max-h-[26rem] overflow-y-auto py-1">
            {items.length === 0 && !loading && (
              <li className="px-3 py-4 text-center text-xs text-slate-500">
                {archived ? t('archivedEmpty') : t('workshopsEmpty')}
              </li>
            )}

            {items.map((w) => (
              <li key={w.id} className="relative">
                <div
                  className={`flex items-start gap-2 px-3 py-2 transition-colors ${
                    w.id === currentId ? 'bg-slate-800/50' : 'hover:bg-slate-800/30'
                  }`}
                >
                  {renaming === w.id ? (
                    // Renommage EN PLACE plutôt qu'une boîte de dialogue : elle voit la ligne qu'elle
                    // modifie, au milieu des autres, donc elle peut choisir un nom qui s'en distingue.
                    <input
                      autoFocus
                      value={renameValue}
                      data-testid={`compose-workshop-rename-input-${w.id}`}
                      aria-label={t('renamePrompt')}
                      onChange={(e) => setRenameValue(e.target.value)}
                      onKeyDown={(e) => {
                        if (e.key === 'Enter') {
                          void patchMeta(w.id, { title: renameValue });
                          setRenaming(null);
                        }
                        if (e.key === 'Escape') setRenaming(null);
                      }}
                      onBlur={() => setRenaming(null)}
                      className="min-w-0 flex-1 rounded border border-gold/60 bg-surface-inset px-2 py-1 text-sm text-slate-100 outline-none"
                    />
                  ) : (
                    <button
                      type="button"
                      data-testid={`compose-workshop-open-${w.id}`}
                      onClick={() => {
                        onOpen(w.id);
                        setOpen(false);
                      }}
                      className="min-w-0 flex-1 text-left"
                    >
                      <div className="flex items-center gap-1.5">
                        {w.id === currentId && (
                          <span aria-hidden="true" className="h-1.5 w-1.5 shrink-0 rounded-full bg-gold" />
                        )}
                        <span className="truncate text-sm text-slate-100">{w.title?.trim() || t('untitled')}</span>
                      </div>
                      {/* Ligne de repères : quand, combien, prêt ou non, parti ou non. Tout ce qu'il
                          faut pour choisir sans ouvrir. */}
                      <div className="mt-0.5 flex flex-wrap items-center gap-x-2 gap-y-0.5 text-[11px] text-slate-500">
                        <span>{formatDate(w.updated_at)}</span>
                        {w.rate > 0 && <span className="tabular-nums text-slate-400">{t('rate', { amount: w.rate })}</span>}
                        {!w.has_draft && <span className="italic">{t('noDraft')}</span>}
                        {w.has_draft && w.checks_total > 0 && (
                          <span
                            className={
                              w.checks_pass === w.checks_total ? 'text-emerald-400' : 'text-amber-400'
                            }
                          >
                            ✓ {t('checksShort', { pass: w.checks_pass, total: w.checks_total })}
                          </span>
                        )}
                        {w.sent_at && (
                          <span className="inline-flex items-center gap-0.5 rounded bg-emerald-500/15 px-1 text-emerald-300">
                            <Check aria-hidden="true" className="h-2.5 w-2.5" />
                            {t('sent')}
                          </span>
                        )}
                      </div>
                    </button>
                  )}

                  <button
                    type="button"
                    aria-label={t('rowMenu')}
                    data-testid={`compose-workshop-menu-${w.id}`}
                    onClick={() => setMenuFor((m) => (m === w.id ? null : w.id))}
                    className="shrink-0 rounded p-1 text-slate-600 transition-colors hover:bg-slate-700/60 hover:text-slate-200"
                  >
                    <MoreHorizontal aria-hidden="true" className="h-4 w-4" />
                  </button>
                </div>

                {menuFor === w.id && (
                  <div className="mx-3 mb-2 overflow-hidden rounded-lg border border-slate-700 bg-surface-inset text-xs">
                    <button
                      type="button"
                      data-testid={`compose-workshop-rename-${w.id}`}
                      onClick={() => {
                        setRenameValue(w.title || '');
                        setRenaming(w.id);
                        setMenuFor(null);
                      }}
                      className="flex w-full items-center gap-2 px-3 py-2 text-left text-slate-300 hover:bg-slate-700/50"
                    >
                      <Pencil aria-hidden="true" className="h-3.5 w-3.5" />
                      {t('rename')}
                    </button>
                    <button
                      type="button"
                      data-testid={`compose-workshop-sent-${w.id}`}
                      onClick={() => {
                        void patchMeta(w.id, { sent: !w.sent_at });
                        setMenuFor(null);
                      }}
                      className="flex w-full items-center gap-2 px-3 py-2 text-left text-slate-300 hover:bg-slate-700/50"
                    >
                      <Check aria-hidden="true" className="h-3.5 w-3.5" />
                      {w.sent_at ? t('unmarkSent') : t('markSent')}
                    </button>
                    <button
                      type="button"
                      data-testid={`compose-workshop-archive-${w.id}`}
                      onClick={() => {
                        void patchMeta(w.id, { archived: !w.archived_at });
                        setMenuFor(null);
                      }}
                      className="flex w-full items-center gap-2 px-3 py-2 text-left text-slate-300 hover:bg-slate-700/50"
                    >
                      {w.archived_at ? (
                        <ArchiveRestore aria-hidden="true" className="h-3.5 w-3.5" />
                      ) : (
                        <Archive aria-hidden="true" className="h-3.5 w-3.5" />
                      )}
                      {w.archived_at ? t('unarchive') : t('archive')}
                    </button>
                  </div>
                )}
              </li>
            ))}
          </ul>
        </div>
      )}
    </div>
  );
}

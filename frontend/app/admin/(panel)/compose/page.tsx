'use client';

import { useCallback, useEffect, useRef, useState } from 'react';
import { useLocale, useTranslations } from 'next-intl';
import { streamChat, type ChatImage, type ChatMessage } from '@/lib/chat-api';
import { extractImageFiles, fileToChatImage, isSupportedImage } from '@/lib/chat-images';
import { Panel, SectionLabel, btnGhost, btnPrimary, btnSuccess, field, fieldArea } from '@/components/admin/ui';
import WorkshopSwitcher from './WorkshopSwitcher';

const API = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

// Signature attendue en fin de message. Donnée de profil, pas une string d'UI — elle n'est jamais
// affichée telle quelle, elle sert au contrôle mécanique côté backend.
const SIGNATURE = 'Alexis';

// ToolCall : un appel de tool tel qu'affiché à l'opérateur. On garde l'input complet pour qu'il puisse
// déplier et voir CE QUE l'agent a réellement écrit ou fixé — pas seulement qu'il a agi.
interface ToolCall {
  name: string;
  input: unknown;
}

// ThreadItem : un tour d'affichage. Les tours d'agent transportent leur raisonnement et leurs tools ;
// les tours d'opérateur n'ont que du texte. Le champ `content` est le seul repris dans l'historique
// envoyé au modèle — le reste est de la présentation.
interface ThreadItem {
  role: 'user' | 'assistant';
  content: string;
  thinking?: string;
  tools?: ToolCall[];
  // images : captures jointes par l'opérateur à ce tour. Conservées dans le fil AFFICHÉ (vignettes)
  // mais PAS renvoyées à l'API aux tours suivants — cf. toHistory : le modèle les a déjà vues, les
  // relire à chaque tour multiplierait le coût en tokens de vision sans rien apporter.
  images?: ChatImage[];
}

// toHistory : réduit le fil d'affichage à ce que l'API attend. Le raisonnement n'y retourne pas — le
// backend le réinjecte lui-même depuis les blocs de la réponse précédente.
function toHistory(items: ThreadItem[]): ChatMessage[] {
  return items.map(({ role, content }) => ({ role, content }));
}

// Repliable : bloc de texte long affiché en une ligne dépliable.
// POURQUOI <details> natif plutôt qu'un état React : accessible au clavier, replié par défaut sans
// code, et un fil de dix tours ne coûte pas dix états.
function Repliable({
  label,
  text,
  testid,
  open,
}: {
  label: string;
  text?: string;
  testid: string;
  open?: boolean;
}) {
  if (!text) return null;
  return (
    <details open={open} data-testid={testid} className="rounded-md border border-slate-800 bg-surface-inset">
      <summary className="cursor-pointer px-2 py-1 text-xs text-slate-400 hover:text-slate-200">
        {label}
      </summary>
      <p className="whitespace-pre-wrap px-2 pb-2 text-xs leading-relaxed text-slate-400">{text}</p>
    </details>
  );
}

// AgentActivity : signal visuel que l'agent travaille.
// POURQUOI une animation et pas un simple texte : un libellé fixe est indiscernable d'un écran gelé.
// Le mouvement est la seule chose qui distingue « ça calcule » de « c'est planté » — et un tour
// d'agent avec tools peut rester silencieux plusieurs dizaines de secondes.
function AgentActivity({ label }: { label: string }) {
  return (
    <div className="flex items-center gap-2 px-1 py-1 text-xs text-slate-400" data-testid="compose-activity">
      <span className="flex gap-1" aria-hidden="true">
        <span className="h-1.5 w-1.5 animate-bounce rounded-full bg-sky-400" />
        <span className="h-1.5 w-1.5 animate-bounce rounded-full bg-sky-400 [animation-delay:150ms]" />
        <span className="h-1.5 w-1.5 animate-bounce rounded-full bg-sky-400 [animation-delay:300ms]" />
      </span>
      <span className="animate-pulse">{label}</span>
    </div>
  );
}

// ToolLine : un appel de tool, en UNE ligne dépliable sur son input.
// POURQUOI montrer l'input : savoir que l'agent a appelé set_draft n'apprend rien ; voir CE QU'IL a
// écrit permet de comprendre pourquoi le message affiché a changé.
function ToolLine({ tool, label }: { tool: ToolCall; label: string }) {
  return (
    <details
      data-testid={`compose-toolcall-${tool.name}`}
      className="rounded border border-sky-900/60 bg-sky-950/30"
    >
      <summary className="cursor-pointer px-2 py-1 text-xs text-sky-400 hover:text-sky-200">
        {label}
      </summary>
      <pre className="overflow-x-auto px-2 pb-2 text-[11px] leading-relaxed text-sky-200/80">
        {JSON.stringify(tool.input, null, 2)}
      </pre>
    </details>
  );
}

// Atelier de rédaction d'un message de prospection, pour un opérateur qui NE LIT PAS la langue du
// message qu'il envoie.
//
// POURQUOI cet écran existe : le workflow précédent demandait deux outils (générateur public puis
// agent admin) avec un copier-coller manuel entre les deux, et l'agent rendait du markdown que
// l'opérateur devait nettoyer à la main sans comprendre pourquoi ces caractères étaient là.
//
// POURQUOI ni le draft ni la traduction ne sont éditables à la main : une édition manuelle
// contournerait le vérificateur et la traduction. L'opérateur se retrouverait avec du texte que rien
// n'a contrôlé et qu'il ne peut pas relire. En passant TOUTE mutation par l'agent (tools set_draft /
// set_rate), chaque changement est suivi d'un re-contrôle et d'une re-traduction — draft et traduction
// ne peuvent pas être désynchronisés.
//
// COMMENT : 1. l'opérateur colle l'annonce et lance, 2. l'agent rédige en appelant set_draft — le
// draft s'affiche DÈS l'event tool_call, sans attendre la fin du tour, 3. à la fin du tour on demande
// la vérification (traduction littérale + checklist) à un endpoint séparé, en contexte vierge,
// 4. l'opérateur ajuste en parlant à l'agent, et le cycle recommence.
export default function AdminComposeTool() {
  const t = useTranslations('admin.compose');
  const locale = useLocale();

  // --- Artefact ---
  // title : nom de l'atelier, produit une seule fois par la vérification dans la langue de
  // l'opératrice (cf. ComposeService.title côté backend). C'est son unique repère pour retrouver un
  // atelier dans son historique.
  const [title, setTitle] = useState('');
  const [announce, setAnnounce] = useState('');
  const [draft, setDraft] = useState('');
  const [translation, setTranslation] = useState('');
  const [rate, setRate] = useState<{ amount: number; rationale: string } | null>(null);
  const [mechanical, setMechanical] = useState<Array<{ id: string; status: string; detail?: string }>>([]);
  const [semantic, setSemantic] = useState<Array<{ id: string; label: string; status: string }>>([]);
  const [verifying, setVerifying] = useState(false);
  // verifyLimited : la dernière vérification a été REFUSÉE pour quota (429, prospecteur seulement).
  // La checklist affichée est alors celle du brouillon PRÉCÉDENT : sans cet avertissement, elle se
  // lirait comme la vérification du message courant.
  const [verifyLimited, setVerifyLimited] = useState(false);

  // --- Conversation ---
  // Le fil d'affichage porte PLUS que l'historique envoyé au modèle : chaque tour d'agent garde son
  // raisonnement et ses appels de tools. POURQUOI les attacher au tour plutôt qu'à un état global :
  // sinon ils disparaissent au tour suivant, et on ne peut plus les afficher AVANT le message auquel
  // ils ont mené.
  const [thread, setThread] = useState<ThreadItem[]>([]);
  const [input, setInput] = useState('');
  const [streamText, setStreamText] = useState('');
  const [streamThinking, setStreamThinking] = useState('');
  const [liveTools, setLiveTools] = useState<ToolCall[]>([]);
  const [busy, setBusy] = useState(false);
  const [copied, setCopied] = useState(false);
  // sentAt : date d'envoi de l'atelier ouvert (null = pas envoyé). copiedOnce : le message a été copié
  // dans cette session — le bouton « Envoyé » se met alors en avant, c'est le geste qui suit.
  const [sentAt, setSentAt] = useState<string | null>(null);
  const [copiedOnce, setCopiedOnce] = useState(false);

  // Captures en attente d'envoi, avec leur aperçu. POURQUOI garder l'aperçu séparément : l'image
  // envoyée est du base64 nu, l'aperçu a besoin du préfixe data: — les recomposer à l'affichage
  // coûterait une concaténation par rendu.
  const [pending, setPending] = useState<Array<{ id: string; image: ChatImage; preview: string }>>([]);
  const [dragging, setDragging] = useState(false);
  const [imageError, setImageError] = useState('');
  const fileInputRef = useRef<HTMLInputElement>(null);

  // Ancre de défilement : le fil est un conteneur à scroll propre, il faut le suivre à la main.
  const bottomRef = useRef<HTMLDivElement>(null);

  // draftRef : la vérification est déclenchée à la fin du tour, hors du cycle de rendu qui a posé le
  // draft. Passer par une ref évite de lire un état périmé dans la closure du callback onDone.
  const draftRef = useRef('');
  const translationRef = useRef('');
  const rateRef = useRef<number>(0);
  const rationaleRef = useRef('');
  const titleRef = useRef('');
  // draftIdRef : identifiant de la ligne persistée. Une ref et non un état — il ne s'affiche nulle
  // part et le relire depuis une closure ne doit pas dépendre d'un re-rendu.
  const draftIdRef = useRef<string | null>(null);
  // touchedRef : l'opérateur a-t-il commencé à travailler ? La restauration est asynchrone ; si elle
  // se résout alors qu'il a déjà collé une annonce, appliquer l'ancien atelier écraserait sa saisie
  // sous ses yeux, sans erreur ni explication. Une ref et non un état : on la lit depuis la closure
  // de la restauration, où un état serait figé à sa valeur de montage.
  const touchedRef = useRef(false);

  // Restauration au montage : sans elle, un simple rafraîchissement perd le message en cours. Pour un
  // opérateur qui ne relit pas le français et vient d'enchaîner plusieurs tours d'ajustement, c'est
  // tout le travail qui disparaît sans qu'il puisse le reconstituer.
  // applyWorkshop — installe à l'écran un atelier complet venu du serveur. Un seul chemin pour la
  // restauration au montage ET pour la bascule manuelle : deux implémentations divergeraient, et
  // c'est exactement le genre d'écart qui fait réapparaître une traduction sans son message.
  const applyWorkshop = useCallback((w: Record<string, unknown>) => {
    draftIdRef.current = (w.id as string) ?? null;
    const t = (w.title as string) || '';
    setTitle(t);
    titleRef.current = t;
    setAnnounce((w.announce as string) || '');
    setDraft((w.draft as string) || '');
    draftRef.current = (w.draft as string) || '';
    setTranslation((w.translation as string) || '');
    translationRef.current = (w.translation as string) || '';
    setSentAt((w.sent_at as string) || null);
    setCopiedOnce(false);
    const rate = (w.rate as number) || 0;
    if (rate) {
      setRate({ amount: rate, rationale: (w.rate_rationale as string) || '' });
      rateRef.current = rate;
      rationaleRef.current = (w.rate_rationale as string) || '';
    } else {
      setRate(null);
      rateRef.current = 0;
      rationaleRef.current = '';
    }
    // Les ateliers enregistrés avant l'ajout du raisonnement n'ont que role/content : les champs
    // absents restent indéfinis et le rendu les traite comme vides. Pas de migration nécessaire.
    setThread(Array.isArray(w.messages) ? (w.messages as ThreadItem[]) : []);
    const checks = (w.checks as { mechanical?: unknown[]; semantic?: unknown[] }) || {};
    setMechanical((checks.mechanical as typeof mechanical) || []);
    setSemantic((checks.semantic as typeof semantic) || []);
    setStreamText('');
    setStreamThinking('');
    setLiveTools([]);
  }, []);

  // openWorkshop — reprendre un atelier de l'historique.
  // POURQUOI une requête à l'unité : la liste ne transporte plus le contenu (le fil embarque les
  // captures en base64), donc on paie le poids d'un seul atelier, et uniquement quand on l'ouvre.
  const openWorkshop = useCallback(
    async (id: string) => {
      // L'opératrice a explicitement choisi cet atelier : une restauration tardive ne doit plus
      // venir le remplacer par le dernier modifié.
      touchedRef.current = true;
      try {
        const res = await fetch(`${API}/api/v1/admin/compose/drafts/${id}`, { credentials: 'include' });
        if (!res.ok) return;
        applyWorkshop(await res.json());
      } catch {
        /* échec réseau : l'écran garde l'atelier courant plutôt que de se vider */
      }
    },
    [applyWorkshop],
  );

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        // Deux temps : la liste (légère) donne l'identifiant du plus récent, puis on charge CE
        // contenu-là. Avant, la liste transportait les 10 derniers ateliers en entier — captures
        // comprises — pour n'en restaurer qu'un.
        const res = await fetch(`${API}/api/v1/admin/compose/drafts`, { credentials: 'include' });
        if (!res.ok) return;
        const { drafts } = await res.json();
        const latest = drafts?.[0];
        // touchedRef : si l'opérateur a commencé pendant que la requête volait, on ne touche à rien.
        // Mieux vaut ne pas restaurer que détruire un travail en cours.
        if (!latest?.id || cancelled || touchedRef.current) return;

        const full = await fetch(`${API}/api/v1/admin/compose/drafts/${latest.id}`, {
          credentials: 'include',
        });
        if (!full.ok || cancelled || touchedRef.current) return;
        applyWorkshop(await full.json());
      } catch {
        // Pas de restauration possible : on démarre sur un atelier vide plutôt que d'afficher une
        // erreur. Rien n'est perdu côté serveur, l'opérateur peut recharger.
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [applyWorkshop]);

  // addFiles — point d'entrée UNIQUE des trois gestes (glisser-déposer, sélecteur, collage).
  // POURQUOI centraliser : les trois produisent des File identiques, et une seule implémentation
  // garantit qu'ils se comportent pareil — y compris sur les erreurs.
  const addFiles = useCallback(
    async (files: File[]) => {
      if (files.length === 0) return;
      setImageError('');
      for (const file of files) {
        if (!isSupportedImage(file.type)) {
          setImageError(t('imageErrors.image_type_unsupported'));
          continue;
        }
        try {
          const image = await fileToChatImage(file);
          setPending((prev) => [
            ...prev,
            {
              id: `${file.name}-${prev.length}-${file.size}`,
              image,
              preview: `data:${image.media_type};base64,${image.data}`,
            },
          ]);
        } catch {
          setImageError(t('imageErrors.image_read_failed'));
        }
      }
    },
    [t],
  );

  // Collage presse-papier au niveau du DOCUMENT plutôt que sur le champ de saisie.
  // POURQUOI : les outils de capture de Windows, macOS et Linux déposent l'image dans le presse-papier
  // sans qu'on sache où le focus se trouve. On n'intercepte QUE s'il y a une image — un collage de
  // texte suit son cours normalement, y compris dans la zone d'annonce.
  useEffect(() => {
    const onPaste = (e: ClipboardEvent) => {
      const files = extractImageFiles(e.clipboardData);
      if (files.length === 0) return;
      e.preventDefault();
      void addFiles(files);
    };
    document.addEventListener('paste', onPaste);
    return () => document.removeEventListener('paste', onPaste);
  }, [addFiles]);

  // Suivi du bas du fil. `?.()` : jsdom n'implémente pas scrollIntoView, l'absence de la méthode ne
  // doit pas faire échouer les tests de rendu.
  useEffect(() => {
    bottomRef.current?.scrollIntoView?.({ behavior: 'smooth', block: 'end' });
  }, [thread, streamText, streamThinking, liveTools]);

  // persist — écrit l'état COMPLET de l'atelier. POURQUOI complet et non incrémental : le draft, sa
  // traduction et sa checklist décrivent la même version ; en sauvegarder un sous-ensemble ferait
  // réapparaître au rechargement une traduction qui ne correspond plus au message affiché.
  const persist = useCallback(
    async (items: ThreadItem[], checks: { mechanical: unknown[]; semantic: unknown[] }) => {
      const payload = {
        title: titleRef.current,
        announce,
        draft: draftRef.current,
        translation: translationRef.current,
        rate: rateRef.current,
        rate_rationale: rationaleRef.current,
        messages: items,
        checks,
      };
      try {
        if (!draftIdRef.current) {
          const res = await fetch(`${API}/api/v1/admin/compose/drafts`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            credentials: 'include',
            body: JSON.stringify(payload),
          });
          if (!res.ok) return;
          draftIdRef.current = (await res.json()).id ?? null;
          return;
        }
        await fetch(`${API}/api/v1/admin/compose/drafts/${draftIdRef.current}`, {
          method: 'PUT',
          headers: { 'Content-Type': 'application/json' },
          credentials: 'include',
          body: JSON.stringify(payload),
        });
      } catch {
        // Sauvegarde ratée : l'écran reste utilisable avec son état en mémoire. On ne bloque pas le
        // travail en cours pour un problème de persistance.
      }
    },
    [announce],
  );

  // verify — demande traduction + checklist au backend. POURQUOI un appel séparé du stream de l'agent :
  // le draft doit s'afficher immédiatement (l'opérateur voit le message bouger), et ces deux appels
  // modèle supplémentaires n'ont pas à ralentir le tour de l'agent.
  const verify = useCallback(
    async (currentDraft: string) => {
      if (!currentDraft.trim()) return null;
      setVerifying(true);
      try {
        const res = await fetch(`${API}/api/v1/admin/compose/verify`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          credentials: 'include',
          body: JSON.stringify({
            draft: currentDraft,
            signature: SIGNATURE,
            retained_rate: rateRef.current,
            operator_lang: locale,
            announce,
            // Nommer UNE seule fois : un titre régénéré à chaque tour changerait sous les yeux de
            // l'opératrice, or c'est son repère pour retrouver l'atelier dans la liste.
            want_title: titleRef.current.trim() === '',
          }),
        });
        setVerifyLimited(res.status === 429);
        if (!res.ok) return null;
        const j = await res.json();
        setTranslation(j.translation || '');
        translationRef.current = j.translation || '';
        if (j.title) {
          setTitle(j.title);
          titleRef.current = j.title;
        }
        setMechanical(j.mechanical || []);
        setSemantic(j.semantic || []);
        // Renvoyé pour que la sauvegarde persiste EXACTEMENT ce qui vient d'être calculé, sans
        // dépendre d'un re-rendu intermédiaire.
        return { mechanical: j.mechanical || [], semantic: j.semantic || [] };
      } catch {
        // Échec réseau : on garde l'état précédent plutôt que de vider la checklist. Une checklist
        // vide se lirait comme « rien à signaler », ce qui serait un mensonge.
      } finally {
        setVerifying(false);
      }
      return null;
    },
    [announce, locale],
  );

  // send — un tour de conversation avec l'agent, en mode compose.
  const send = useCallback(
    async (text: string) => {
      // Une capture SEULE suffit à envoyer : « regarde ça » est un geste légitime, l'image EST le
      // message. Exiger du texte forçait à écrire quelque chose d'inutile à chaque capture.
      if ((!text.trim() && pending.length === 0) || busy) return;
      // Les captures partent avec CE tour et sont retirées de la file : elles ne doivent pas se
      // rattacher au message suivant.
      const jointes = pending.map((p) => p.image);
      setPending([]);
      setImageError('');
      const withUser: ThreadItem[] = [
        ...thread,
        { role: 'user', content: text, ...(jointes.length ? { images: jointes } : {}) },
      ];
      setThread(withUser);
      setInput('');
      setStreamText('');
      setStreamThinking('');
      setLiveTools([]);
      setBusy(true);

      let assistantText = '';
      let assistantThinking = '';
      const assistantTools: ToolCall[] = [];
      let previousRate: { amount: number; rationale: string } | null = null;

      await streamChat(
        text,
        toHistory(thread),
        {
          onText: (delta) => {
            assistantText += delta;
            setStreamText(assistantText);
          },
          onThinking: (delta) => {
            assistantThinking += delta;
            setStreamThinking(assistantThinking);
          },
          onToolCall: (name, toolInput) => {
            assistantTools.push({ name, input: toolInput });
            setLiveTools([...assistantTools]);
            // Le draft est lu dans l'INPUT du tool_call, pas dans le tool_result : le backend ne
            // renvoie qu'un accusé au modèle (inutile de lui relire le message qu'il vient d'écrire).
            if (name === 'set_draft') {
              const message = (toolInput as { message?: string } | null)?.message;
              if (typeof message === 'string' && message.trim()) {
                setDraft(message);
                draftRef.current = message;
              }
            }
            if (name === 'set_rate') {
              // Mémorise le tarif en place : si le backend REFUSE ce montant (plancher dur), le
              // tool_result en erreur le restaure — un montant refusé ne doit jamais rester affiché.
              previousRate = rateRef.current ? { amount: rateRef.current, rationale: rationaleRef.current } : null;
              const payload = toolInput as { amount?: number; rationale?: string } | null;
              if (typeof payload?.amount === 'number') {
                setRate({ amount: payload.amount, rationale: payload.rationale ?? '' });
                rateRef.current = payload.amount;
                rationaleRef.current = payload.rationale ?? '';
              }
            }
          },
          onToolResult: (name, data) => {
            // Le résultat n'est qu'un accusé — SAUF un refus de set_rate (montant sous le plancher),
            // qui annule l'affichage optimiste fait au tool_call.
            if (name === 'set_rate' && (data as { error?: string } | null)?.error) {
              setRate(previousRate);
              rateRef.current = previousRate?.amount ?? 0;
              rationaleRef.current = previousRate?.rationale ?? '';
            }
          },
          onDone: () => {
            const complete: ThreadItem[] = [
              ...withUser,
              {
                role: 'assistant',
                content: assistantText,
                thinking: assistantThinking,
                tools: assistantTools,
              },
            ];
            setThread(complete);
            setStreamText('');
            setStreamThinking('');
            setLiveTools([]);
            setBusy(false);
            // Sauvegarde APRÈS la vérification : le draft, sa traduction et sa checklist décrivent la
            // même version, les persister ensemble évite de restaurer un état incohérent.
            void (async () => {
              const checks = await verify(draftRef.current);
              await persist(complete, checks ?? { mechanical: [], semantic: [] });
            })();
          },
          onError: () => {
            setBusy(false);
            setStreamText('');
            setStreamThinking('');
          },
        },
        undefined,
        { compose: true, images: jointes },
      );
    },
    [busy, thread, verify, persist, pending],
  );

  // start — premier tour : on donne l'annonce à l'agent et on lui demande le message.
  // Le texte de lancement vient de next-intl : il s'affiche dans le fil, donc c'est une string d'UI,
  // et il part dans la langue de l'opérateur — ce qui indique aussi à l'agent dans quelle langue lui
  // répondre (le message client, lui, reste en français).
  const start = () => {
    if (!announce.trim()) return;
    void send(t('kickoff', { announce }));
  };

  // reset — repart sur un atelier vierge. INDISPENSABLE : la restauration au montage ramène toujours
  // le dernier atelier, donc sans cette sortie l'opérateur ne pourrait jamais traiter une deuxième
  // annonce. On oublie l'identifiant plutôt que de supprimer la ligne : le travail précédent reste en
  // base, il n'est simplement plus celui en cours.
  const reset = () => {
    // touchedRef reste à true : après un reset explicite, une restauration tardive serait tout aussi
    // indésirable qu'après une saisie.
    touchedRef.current = true;
    draftIdRef.current = null;
    draftRef.current = '';
    translationRef.current = '';
    rateRef.current = 0;
    rationaleRef.current = '';
    titleRef.current = '';
    setTitle('');
    setAnnounce('');
    setDraft('');
    setTranslation('');
    setSentAt(null);
    setCopiedOnce(false);
    setRate(null);
    setMechanical([]);
    setSemantic([]);
    setThread([]);
    setStreamThinking('');
    setLiveTools([]);
    setStreamText('');
  };

  const copy = async () => {
    await navigator.clipboard.writeText(draft);
    setCopied(true);
    setCopiedOnce(true);
    window.setTimeout(() => setCopied(false), 2000);
  };

  // toggleSent — marque l'atelier ouvert comme envoyé (ou annule).
  //
  // POURQUOI ici, à côté de « Copier » : mesuré le 2026-10-12, sent_at n'avait été rempli sur AUCUN
  // des 25 ateliers. Le geste existait, mais caché dans le menu « … » de la liste des ateliers, loin
  // de l'endroit où l'opératrice finit son travail. Sans lui, aucun taux de réponse n'est mesurable.
  // COMMENT : même endpoint que la liste (PATCH …/meta, champ `sent`) ; l'état local ne change
  // qu'après la réponse du serveur, pour que le ✓ affiché soit une vérité et pas un espoir.
  const toggleSent = async () => {
    const id = draftIdRef.current;
    if (!id) return;
    const next = !sentAt;
    try {
      const res = await fetch(`${API}/api/v1/admin/compose/drafts/${id}/meta`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({ sent: next }),
      });
      if (!res.ok) return;
      setSentAt(next ? new Date().toISOString() : null);
    } catch {
      /* échec réseau : l'état affiché reste celui du serveur */
    }
  };

  // POURQUOI un GLYPHE en plus de la couleur : l'opératrice ne lit pas le libellé français des contrôles
  // mécaniques, elle lit l'état. Une pastille qui ne se distingue que par sa teinte oblige à comparer
  // les couleurs entre elles ; un signe se lit seul, et survit à un écran mal calibré ou à un daltonisme.
  const statusGlyph = (status: string) => (status === 'pass' ? '✓' : status === 'fail' ? '✕' : '–');

  const statusClass = (status: string) =>
    status === 'pass'
      ? 'border-emerald-600/60 bg-emerald-500/10 text-emerald-200'
      : status === 'fail'
        ? 'border-rose-500/60 bg-rose-500/10 text-rose-200'
        : 'border-slate-700 bg-slate-800/60 text-slate-400';

  return (
    // Hauteur BORNÉE au viewport (moins le p-8 du layout admin, soit 4rem).
    // POURQUOI pas h-full : le layout du panneau est en min-h-screen sans hauteur fixe, donc h-full se
    // résout contre un parent qui grandit — les enfants s'étiraient avec le contenu et le fil de
    // conversation poussait la page au lieu de défiler dans son cadre.
    <div
      className={`flex h-[calc(100vh-4rem)] flex-col gap-4 ${
        dragging ? 'rounded-lg outline-dashed outline-2 outline-sky-500' : ''
      }`}
      data-testid="compose-root"
      // Dépôt accepté sur TOUT l'écran plutôt qu'une zone étroite : c'est plus indulgent, et
      // l'opérateur n'a pas à viser. Le liseré signale que le lâcher sera pris en compte.
      onDragOver={(e) => {
        if (!e.dataTransfer.types.includes('Files')) return;
        e.preventDefault();
        setDragging(true);
      }}
      onDragLeave={(e) => {
        if (e.currentTarget.contains(e.relatedTarget as Node)) return;
        setDragging(false);
      }}
      onDrop={(e) => {
        const files = extractImageFiles(e.dataTransfer);
        if (files.length === 0) return;
        e.preventDefault();
        setDragging(false);
        void addFiles(files);
      }}
    >
      {/* --- Bandeau checklist ---
          Les contrôles MÉCANIQUES et le JUGEMENT sont volontairement séparés visuellement : une case
          cochée par du code est une preuve, un avis de modèle n'en est pas une, et l'opérateur doit
          voir la différence. */}
      <Panel hairline className="px-4 py-3" testid="compose-checklist">
        <div className="mb-3 flex items-center gap-3">
          {/* Le nom de l'atelier ouvert, en permanence et en tête d'écran. C'était le manque le plus
              criant : rien n'indiquait sur QUELLE annonce on travaillait, ni comment revenir à une
              autre. Le même élément porte les deux réponses. */}
          <WorkshopSwitcher
            currentId={draftIdRef.current}
            currentTitle={title}
            onOpen={(id) => void openWorkshop(id)}
            onCurrentChanged={(patch) => {
              if (patch.title !== undefined) {
                setTitle(patch.title);
                titleRef.current = patch.title;
              }
              // Archiver l'atelier OUVERT le range : on repart à vide plutôt que de continuer à
              // travailler sur quelque chose qui n'est plus dans la liste courante.
              if (patch.archived) reset();
            }}
          />
          <span aria-hidden="true" className="h-4 w-px bg-slate-800" />
          <SectionLabel>{t('checklistTitle')}</SectionLabel>
          {verifying && (
            <span className="animate-pulse text-[11px] text-slate-500">{t('verifying')}</span>
          )}
          {verifyLimited && !verifying && (
            <span className="text-[11px] text-rose-400" data-testid="compose-verify-limit">
              ✕ {t('verifyLimit')}
            </span>
          )}
          <div className="ml-auto flex items-center gap-2">
            {/* Le TJM est un chiffre que l'opératrice doit RECOPIER dans un formulaire : il mérite d'être
                lisible d'un coup d'œil, pas noyé dans une pastille de la même taille que le reste.
                Il est DORÉ et pas bleu : le bleu marque partout ailleurs ce que fait la machine,
                alors que ce nombre-ci est une consigne qui lui est adressée — la seule valeur de
                l'écran qu'elle doit reporter ailleurs à la main. */}
            {rate && (
              <span
                className="rounded-lg border border-gold/50 bg-gold/10 px-3 py-1.5 text-sm font-semibold tabular-nums text-gold-soft"
                title={rate.rationale}
                data-testid="compose-rate"
              >
                {t('rate', { amount: rate.amount })}
              </span>
            )}
            {draft !== '' && (
              <button
                type="button"
                data-testid="compose-new"
                onClick={reset}
                disabled={busy}
                className={`${btnGhost} border border-slate-800 text-xs hover:border-slate-700`}
              >
                {t('newDraft')}
              </button>
            )}
          </div>
        </div>

        {mechanical.length === 0 && semantic.length === 0 ? (
          <p className="text-xs text-slate-500">{t('checklistEmpty')}</p>
        ) : (
          <div className="flex flex-col gap-2.5">
            {/* Les deux natures sont désormais SÉPARÉES ET NOMMÉES. Une bordure en pointillés était
                trop discrète pour porter la différence entre une preuve (vérifiée par du code) et un
                avis de modèle — or c'est exactement ce que l'opératrice doit pouvoir distinguer. */}
            {mechanical.length > 0 && (
              <div className="flex flex-wrap items-center gap-2">
                <span className="w-24 shrink-0 text-[10px] uppercase tracking-wider text-slate-600">
                  {t('checksProven')}
                </span>
                {mechanical.map((c) => (
                  <span
                    key={c.id}
                    data-testid={`check-${c.id}`}
                    data-status={c.status}
                    title={c.detail}
                    className={`inline-flex items-center gap-1.5 rounded-lg border px-2.5 py-1 text-xs font-medium ${statusClass(c.status)}`}
                  >
                    <span aria-hidden="true" className="text-[13px] leading-none">
                      {statusGlyph(c.status)}
                    </span>
                    {t(`checks.${c.id}`)}
                    {c.detail ? <span className="opacity-60">· {c.detail}</span> : null}
                  </span>
                ))}
              </div>
            )}

            {/* Items dérivés de l'annonce : libellés rédigés par le juge dans la langue de
                l'opérateur, d'où l'absence de clé i18n ici (contenu généré, pas chrome d'UI). */}
            {semantic.length > 0 && (
              <div className="flex flex-wrap items-center gap-2">
                <span className="w-24 shrink-0 text-[10px] uppercase tracking-wider text-slate-600">
                  {t('checksJudged')}
                </span>
                {semantic.map((c) => (
                  <span
                    key={c.id}
                    data-testid={`check-semantic-${c.id}`}
                    data-status={c.status}
                    className={`inline-flex items-center gap-1.5 rounded-lg border border-dashed px-2.5 py-1 text-xs ${statusClass(c.status)}`}
                  >
                    <span aria-hidden="true" className="text-[13px] leading-none">
                      {statusGlyph(c.status)}
                    </span>
                    {c.label}
                  </span>
                ))}
              </div>
            )}
          </div>
        )}
      </Panel>

      <div className="grid min-h-0 flex-1 grid-cols-1 gap-4 lg:grid-cols-5">
        {/* --- Colonne gauche : le livrable et son miroir ---
            Les panneaux se dimensionnent à leur CONTENU, plafonnés puis défilants. Avec flex-1 sur
            chacun, un message court laissait deux grands cadres vides. */}
        <div className="flex min-h-0 flex-col gap-4 overflow-y-auto lg:col-span-3">
          {draft === '' ? (
            <Panel className="flex flex-col gap-3 p-4">
              <label htmlFor="announce">
                <SectionLabel>{t('announceLabel')}</SectionLabel>
              </label>
              <textarea
                id="announce"
                data-testid="compose-announce"
                value={announce}
                onChange={(e) => {
                  touchedRef.current = true;
                  setAnnounce(e.target.value);
                }}
                placeholder={t('announcePlaceholder')}
                className={`${fieldArea} min-h-48 flex-1`}
              />
              <button type="button" data-testid="compose-start" onClick={start} disabled={!announce.trim() || busy} className={`${btnPrimary} self-start`}>
                {t('start')}
              </button>
            </Panel>
          ) : (
            <>
              {/* LE LIVRABLE. C'est ce que l'opératrice fixe le plus longtemps et ce qu'elle copie : il a
                  droit à la plus grande taille de texte de l'écran et à un interligne généreux. En
                  `text-sm` serré, il se lisait comme un bloc de code alors que c'est une lettre. */}
              {/* LE livrable de tout l'écran : c'est le seul panneau à porter le filet doré ET une
                  bordure éclaircie. Tout le reste de la page existe pour produire ou contrôler ce
                  bloc — la hiérarchie visuelle doit le dire sans qu'on ait à lire. */}
              <Panel hairline className="flex flex-none flex-col p-4">
                <div className="mb-3 flex items-center gap-3">
                  <SectionLabel>{t('draftTitle')}</SectionLabel>
                  {/* Le bouton de copie est le geste FINAL de chaque session : il est traité comme
                      l'action principale, pas comme un lien discret dans un coin. Vert et non or —
                      l'or ouvre le travail, le vert le clôt. */}
                  <button
                    type="button"
                    data-testid="compose-copy"
                    onClick={copy}
                    className={`${btnSuccess} ml-auto px-3 py-1.5 text-xs ${
                      copied ? 'border-emerald-500/70 bg-emerald-500/25' : ''
                    }`}
                  >
                    {copied ? `✓ ${t('copied')}` : t('copy')}
                  </button>
                  {/* Le geste qui suit la copie : déclarer l'envoi. Glyphe ✓ en plus de la couleur
                      (l'opératrice lit l'état, pas le libellé). Mis en avant après une copie tant que
                      l'atelier n'est pas marqué ; jamais doré — l'or reste l'action principale. */}
                  <button
                    type="button"
                    data-testid="compose-sent"
                    data-sent={sentAt ? 'true' : 'false'}
                    onClick={() => void toggleSent()}
                    disabled={!draftIdRef.current}
                    title={sentAt ? t('unmarkSent') : t('markSent')}
                    className={`rounded-lg border px-3 py-1.5 text-xs font-medium transition-colors ${
                      sentAt
                        ? 'border-emerald-500/70 bg-emerald-500/25 text-emerald-200'
                        : copiedOnce
                          ? 'animate-pulse border-emerald-400 text-emerald-200 hover:bg-emerald-500/15'
                          : 'border-slate-700 text-slate-400 hover:border-slate-600 hover:text-slate-200'
                    }`}
                  >
                    {sentAt ? `✓ ${t('sent')}` : t('markSent')}
                  </button>
                </div>
                {/* Lecture seule à dessein — cf. commentaire de tête. */}
                <p
                  data-testid="compose-draft"
                  className="flex-1 overflow-auto whitespace-pre-wrap text-[15px] leading-[1.7] text-slate-100"
                >
                  {draft}
                </p>
              </Panel>

              {/* Le miroir de contrôle : même confort de lecture, mais visuellement SUBORDONNÉ — ce
                  n'est pas ce qu'elle envoie. D'où la surface plus sombre, le texte adouci, et
                  l'absence de filet doré. */}
              <Panel className="flex flex-none flex-col bg-surface-inset p-4">
                <SectionLabel className="mb-3 text-slate-600">{t('translationTitle')}</SectionLabel>
                <p
                  data-testid="compose-translation"
                  className="flex-1 overflow-auto whitespace-pre-wrap text-[15px] leading-[1.7] text-slate-400"
                >
                  {translation || <span className="text-slate-700">{t('translationPending')}</span>}
                </p>
              </Panel>
            </>
          )}
        </div>

        {/* --- Colonne droite : l'agent --- */}
        <Panel className="flex min-h-0 flex-col p-4 lg:col-span-2">
          <SectionLabel className="mb-3">{t('chatTitle')}</SectionLabel>

          {/* min-h-0 est indispensable : sans lui, un enfant flex refuse de rétrécir sous la taille de
              son contenu et le fil pousse la page au lieu de défiler dans son cadre. */}
          <div className="min-h-0 flex-1 space-y-3 overflow-y-auto pr-1" data-testid="compose-thread">
            {thread.map((m, i) =>
              m.role === 'user' ? (
                <div key={i} className="ml-auto max-w-[90%] space-y-1">
                  {/* Aperçu des captures DANS le fil : l'opérateur doit revoir ce qu'il a envoyé,
                      sinon un échange de dix tours devient illisible — on ne sait plus quelle
                      réponse répondait à quelle capture. */}
                  {(m.images ?? []).length > 0 && (
                    <div className="flex flex-wrap justify-end gap-1" data-testid={`compose-msg-images-${i}`}>
                      {(m.images ?? []).map((img, k) => (
                        // eslint-disable-next-line @next/next/no-img-element -- data: URL locale
                        <img
                          key={k}
                          src={`data:${img.media_type};base64,${img.data}`}
                          alt=""
                          className="h-20 w-20 rounded border border-sky-800 object-cover"
                        />
                      ))}
                    </div>
                  )}
                  {m.content && (
                    <div className="whitespace-pre-wrap rounded-xl border border-sky-800/50 bg-sky-900/40 px-3 py-2 text-sm leading-relaxed text-sky-50">
                      {m.content}
                    </div>
                  )}
                </div>
              ) : (
                <div key={i} className="space-y-1">
                  {/* Raisonnement puis tools AVANT le message : c'est l'ordre dans lequel l'agent a
                      travaillé, et l'opérateur peut suivre le cheminement plutôt que subir un
                      résultat opaque. Replié par défaut pour ne pas noyer la conversation. */}
                  <Repliable label={t('reasoning')} testid={`compose-thinking-${i}`} text={m.thinking} />
                  {(m.tools ?? []).map((tool, j) => (
                    <ToolLine key={j} tool={tool} label={t('toolCall', { name: tool.name })} />
                  ))}
                  {m.content && (
                    <div className="max-w-[90%] whitespace-pre-wrap rounded-xl border border-slate-800 bg-slate-800/60 px-3 py-2 text-sm leading-relaxed text-slate-200">
                      {m.content}
                    </div>
                  )}
                </div>
              ),
            )}

            {/* Tour en cours : même ordre — raisonnement, tools, puis texte. */}
            <Repliable label={t('reasoning')} testid="compose-thinking-live" text={streamThinking} open />
            {liveTools.map((tool, i) => (
              <ToolLine key={`live-${i}`} tool={tool} label={t('toolCall', { name: tool.name })} />
            ))}
            {streamText && (
              <div className="max-w-[90%] whitespace-pre-wrap rounded-xl border border-slate-800 bg-slate-800/60 px-3 py-2 text-sm leading-relaxed text-slate-200">
                {streamText}
              </div>
            )}
            {/* Indicateur d'activité visible pendant TOUT le tour, pas seulement avant le premier
                mot : les temps morts les plus longs sont entre deux appels de tools, une fois du
                texte déjà affiché. Sans lui, l'écran semble figé et on ne sait pas s'il travaille. */}
            {busy && <AgentActivity label={t('thinking')} />}
            <div ref={bottomRef} />
          </div>

          {/* Vignettes des captures en attente. Elles partent avec le PROCHAIN message : l'opérateur
              voit ce qu'il s'apprête à envoyer et peut retirer une capture avant de valider. */}
          {pending.length > 0 && (
            <div className="mt-2 flex flex-wrap gap-2" data-testid="compose-pending-images">
              {pending.map((p) => (
                <div key={p.id} className="relative">
                  {/* eslint-disable-next-line @next/next/no-img-element -- data: URL locale, jamais distante */}
                  <img
                    src={p.preview}
                    alt=""
                    className="h-16 w-16 rounded border border-slate-700 object-cover"
                  />
                  <button
                    type="button"
                    aria-label={t('removeImage')}
                    data-testid={`compose-remove-image-${p.id}`}
                    onClick={() => setPending((prev) => prev.filter((x) => x.id !== p.id))}
                    className="absolute -right-1 -top-1 h-5 w-5 rounded-full bg-slate-800 text-xs text-slate-300"
                  >
                    ×
                  </button>
                </div>
              ))}
            </div>
          )}

          {imageError && (
            <p className="mt-2 text-xs text-rose-400" data-testid="compose-image-error">
              {imageError}
            </p>
          )}

          <form
            className="mt-2 flex gap-2"
            onSubmit={(e) => {
              e.preventDefault();
              void send(input);
            }}
          >
            {/* Sélecteur de fichier classique — troisième geste, à côté du dépôt et du collage. */}
            <input
              ref={fileInputRef}
              type="file"
              accept="image/png,image/jpeg,image/gif,image/webp"
              multiple
              className="hidden"
              data-testid="compose-file-input"
              onChange={(e) => {
                void addFiles(Array.from(e.target.files ?? []));
                e.target.value = ''; // permet de re-sélectionner le même fichier ensuite
              }}
            />
            <button
              type="button"
              aria-label={t('attachImage')}
              title={t('attachImage')}
              data-testid="compose-attach"
              onClick={() => fileInputRef.current?.click()}
              disabled={busy}
              className="shrink-0 rounded-lg border border-slate-700/80 px-3 text-lg leading-none text-slate-400 transition-colors hover:border-slate-600 hover:text-slate-200 disabled:opacity-40"
            >
              +
            </button>
            <input
              data-testid="compose-chat-input"
              value={input}
              onChange={(e) => setInput(e.target.value)}
              placeholder={t('chatPlaceholder')}
              // Ouvert dès le départ : l'opérateur doit pouvoir coller une capture de l'ANNONCE sans
              // passer par la zone de texte. Le verrouiller jusqu'au premier brouillon rendait ce
              // geste impossible — la vignette apparaissait sans qu'on puisse l'envoyer.
              disabled={busy}
              className={`${field} min-w-0 flex-1 disabled:opacity-40`}
            />
            <button
              type="submit"
              data-testid="compose-chat-send"
              disabled={busy || (!input.trim() && pending.length === 0)}
              className={`${btnPrimary} shrink-0`}
            >
              {t('send')}
            </button>
          </form>
        </Panel>
      </div>
    </div>
  );
}

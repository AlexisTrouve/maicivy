'use client';

import { useState, useEffect, useTransition } from 'react';
import { useSearchParams } from 'next/navigation';
import { useRouter } from '@/i18n/navigation';
import { useTranslations } from 'next-intl';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Loader2 } from 'lucide-react';
import { Theme } from '@/lib/types';
import { isKnownCvTheme, orderCvThemes } from '@/lib/cvThemes';

// Fetch available themes from API
async function fetchThemes(): Promise<Theme[]> {
  const res = await fetch(`${process.env.NEXT_PUBLIC_API_URL}/api/v1/cv/themes`);
  if (!res.ok) throw new Error('Failed to fetch themes');
  const data = await res.json();
  return data.themes || [];
}

// Classes du déclencheur, partagées avec le squelette de CVThemeSelectorLazy : mêmes dimensions
// (h-11, pleine largeur du conteneur), donc aucun saut quand le vrai menu remplace le squelette.
// Champ « en creux » : fond de page sous le panneau, filet discret, l'or au focus seulement.
export const CV_THEME_TRIGGER_CLASS =
  'h-11 w-full rounded-lg border border-border bg-background px-4 text-left text-sm text-foreground transition-colors hover:border-muted-foreground/60 focus:ring-ring';

interface CVThemeSelectorProps {
  currentTheme: string;
  /** id posé sur le déclencheur, pour qu'un <label htmlFor> de la page le nomme. */
  triggerId?: string;
}

export default function CVThemeSelector({ currentTheme, triggerId }: CVThemeSelectorProps) {
  const router = useRouter();
  const searchParams = useSearchParams();
  const [themes, setThemes] = useState<Theme[]>([]);
  const [loading, setLoading] = useState(true);
  const [isPending, startTransition] = useTransition();
  const t = useTranslations('cv');

  useEffect(() => {
    fetchThemes()
      .then((list) => setThemes(orderCvThemes(list)))
      .catch(console.error)
      .finally(() => setLoading(false));
  }, []);

  // Libellés traduits d'un profil ; repli sur ceux de l'API pour un profil que l'interface ne
  // connaît pas encore (cf. lib/cvThemes.ts).
  const nameOf = (id: string, fallback?: string) =>
    isKnownCvTheme(id) ? t(`themes.${id}.name`) : fallback ?? id;
  const descriptionOf = (theme: Theme) =>
    isKnownCvTheme(theme.id) ? t(`themes.${theme.id}.description`) : theme.description;

  const handleThemeChange = (newTheme: string) => {
    if (newTheme === currentTheme) return;

    // Query passée en OBJET, jamais concaténée dans le pathname.
    //
    // POURQUOI : `router` vient du router LOCALISÉ de next-intl (i18n/navigation), pas de
    // next/navigation. Depuis next-intl v4, il n'accepte plus une chaîne contenant un `?` — il
    // traite tout l'argument comme un pathname, ne trouve pas la route « /cv?theme=backend » et
    // ne navigue PAS. Panne muette : aucune erreur console, le Select se referme sur le nouveau
    // libellé, l'URL et le CV ne bougent pas. La forme supportée est
    // `push({ pathname, query })` (cf. doc next-intl « Navigation APIs »).
    //
    // COMMENT : on repart des searchParams courants pour ne pas perdre les autres paramètres
    // éventuels, puis on écrase `theme`. Object.fromEntries suffit ici — aucun paramètre de cette
    // page n'est répété (une clé répétée serait réduite à sa dernière valeur).
    const query: Record<string, string> = Object.fromEntries(searchParams.entries());
    query.theme = newTheme;

    // startTransition : la navigation ne bloque pas l'UI, et `isPending` pilote l'état visuel
    // (spinner + curseur d'attente sur le trigger).
    startTransition(() => {
      router.push({ pathname: '/cv', query });
    });
  };

  return (
    <Select value={currentTheme} onValueChange={handleThemeChange} disabled={loading || isPending}>
      <SelectTrigger
        id={triggerId}
        aria-busy={loading || isPending}
        className={`${CV_THEME_TRIGGER_CLASS} ${isPending ? 'cursor-wait' : ''}`}
      >
        {/* Le libellé est rendu EXPLICITEMENT (enfant de SelectValue) au lieu de laisser Radix
            recopier le contenu de l'option choisie.
            POURQUOI : chaque option porte deux blocs (nom + description). Recopiés dans le
            déclencheur, ils tombaient sous le `[&>span]:line-clamp-1` du SelectTrigger shadcn,
            qui ne laissait AUCUNE ligne visible — le menu s'affichait vide (mesuré sur la prod :
            le texte était dans le DOM, invisible). Effet de bord utile : le libellé ne dépend
            plus de la liste des profils, il est juste avant même que /cv/themes réponde. */}
        <SelectValue placeholder={t('selectTheme')}>
          {/* Contenu INLINE uniquement (pas de flex ni de bloc) : c'est ce qui reste visible sous
              le line-clamp du déclencheur. */}
          {isPending && (
            <Loader2 className="mr-2 inline h-4 w-4 animate-spin align-[-3px] text-muted-foreground" aria-hidden="true" />
          )}
          {nameOf(currentTheme, themes.find((x) => x.id === currentTheme)?.name)}
        </SelectValue>
      </SelectTrigger>
      {/* Menu à la largeur EXACTE du déclencheur : en popper, Radix le dimensionne sinon sur la plus
          longue description, qui sortait de l'écran à 390 px (texte coupé à droite). Les
          descriptions passent donc à la ligne. */}
      <SelectContent className="w-[var(--radix-select-trigger-width)] border-border bg-popover">
        {themes.map((theme) => (
          <SelectItem key={theme.id} value={theme.id} className="py-2.5 focus:bg-secondary">
            <span className="block text-sm font-medium text-foreground">{nameOf(theme.id, theme.name)}</span>
            <span className="mt-0.5 block text-xs text-muted-foreground">{descriptionOf(theme)}</span>
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}

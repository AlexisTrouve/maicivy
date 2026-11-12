'use client';

import dynamic from 'next/dynamic';

// ============================================================================
// Chargement PARESSEUX du sélecteur de thème — sort Radix du JS bloquant de /cv.
// ----------------------------------------------------------------------------
// QUOI : enveloppe CVThemeSelector dans un `next/dynamic` avec `ssr: false`, pour que son code
// (et surtout ses dépendances) parte dans un chunk séparé, récupéré APRÈS le premier paint.
//
// POURQUOI ce fichier existe au lieu d'un dynamic() posé dans page.tsx : `ssr: false` n'est PAS
// autorisé dans un Server Component, et Next ne le signale pas — il l'ignore en silence. Mesuré :
// avec le dynamic() écrit dans page.tsx, le chunk de 510 kB était simplement RENOMMÉ et restait
// chargé au premier rendu (total JS inchangé, 1133 kB). Le dynamic doit donc être déclaré dans un
// composant client, d'où ce wrapper. Ne pas le replier dans page.tsx : ça annule le gain sans
// qu'aucune erreur n'apparaisse.
//
// CE QU'ON SORT : CVThemeSelector est le SEUL consommateur de @radix-ui/react-select du site, et
// Radix tire floating-ui avec lui. Ce chunk n'était chargé par aucune autre page (portfolio,
// gitstats, blog ne le référencent pas) — un demi-mégaoctet de JS bloquant pour un menu de
// 6 profils, sur le premier écran d'une page déjà lente.
//
// COMMENT : `ssr: false` est sans risque ici — le menu n'apporte rien d'indexable (la description
// du profil courant est, elle, rendue côté serveur par la page). Le `loading` occupe exactement la
// boîte du déclencheur (CV_THEME_TRIGGER_CLASS : h-11, pleine largeur) : aucun saut de mise en
// page, seul le moment d'arrivée du chunk change. Les dimensions sont RECOPIÉES plutôt
// qu'importées de CVThemeSelector : importer ce module ici ramènerait Radix dans le chunk principal.
// ============================================================================

const CVThemeSelector = dynamic(() => import('@/components/cv/CVThemeSelector'), {
  ssr: false,
  loading: () => (
    <div
      aria-hidden="true"
      data-testid="cv-theme-selector-placeholder"
      className="h-11 w-full animate-pulse rounded-lg border border-border bg-background"
    />
  ),
});

export default function CVThemeSelectorLazy({
  currentTheme,
  triggerId,
}: {
  currentTheme: string;
  triggerId?: string;
}) {
  return <CVThemeSelector currentTheme={currentTheme} triggerId={triggerId} />;
}

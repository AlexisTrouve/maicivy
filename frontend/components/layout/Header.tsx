'use client';

import { useEffect, useRef, useState } from 'react';
import { useTranslations } from 'next-intl';
import { Mail, Menu, X } from 'lucide-react';

import { Link, usePathname } from '@/i18n/navigation';
import { LanguageSwitcher } from '@/components/shared/LanguageSwitcher';
import { isImmersiveRoute } from '@/lib/immersiveRoutes';
import { CONTACT_EMAIL } from '@/lib/montagne/links';
import { PRIMARY_NAV, isActiveHref } from '@/lib/site/navigation';
import { cn } from '@/lib/utils';

/**
 * Header — l'en-tête UNIQUE du site public, dans le langage du bandeau de la montagne.
 *
 * QUOI : le nom d'Alexis en serif (retour à l'accueil = la montagne), la navigation principale
 * (lib/site/navigation.ts › PRIMARY_NAV), le contact (mailto) et la langue. Sous 1024 px, la nav
 * passe dans un menu téléphone ouvert par un bouton.
 * POURQUOI un seul en-tête : le bandeau de la montagne et l'ancien en-tête shadcn coexistaient,
 * chacun avec sa liste de liens — et la montagne ne menait nulle part. L'accueil EST désormais la
 * montagne et porte cet en-tête comme toutes les pages ; le dossier vivant aussi, avec sa barre de
 * rejeu collée dessous (lib/immersiveRoutes.ts).
 * Doctrine de l'or : le filet en tête signale la surface d'Alexis, la barre sous l'entrée active dit
 * « où l'on est ». Rien d'autre n'est doré ici — l'action principale appartient à chaque page.
 * Hauteur : h-16 EXACTEMENT (64 px, filet compris, il est posé en absolu) — /chat borne sa hauteur à
 * calc(100vh - 64px).
 * E2E : `site-header`, `site-nav-<id>`, `site-menu-toggle`, `site-menu`, `site-menu-<id>`,
 * `site-header-contact`, `site-header-language`.
 */

/** Lien neutre de la barre : lisible, sans couleur d'accent, cible tactile de 44 px. */
const barLink =
  'inline-flex min-h-[44px] items-center gap-1.5 rounded-md px-2.5 text-sm text-muted-foreground transition-colors hover:text-foreground focus:outline-none focus-visible:ring-1 focus-visible:ring-ring/70';

export function Header() {
  const pathname = usePathname();
  const t = useTranslations('nav');
  const [menuOpen, setMenuOpen] = useState(false);
  const toggleRef = useRef<HTMLButtonElement>(null);

  // Une navigation (lien du menu, retour arrière) referme le menu : le layout, lui, ne se remonte
  // pas entre deux pages, l'état survivrait sinon à la page qu'il a servi à quitter.
  useEffect(() => {
    setMenuOpen(false);
  }, [pathname]);

  // Échap referme le menu et rend le focus au bouton qui l'a ouvert (sinon il tombe sur <body>).
  useEffect(() => {
    if (!menuOpen) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key !== 'Escape') return;
      setMenuOpen(false);
      toggleRef.current?.focus();
    };
    document.addEventListener('keydown', onKey);
    return () => document.removeEventListener('keydown', onKey);
  }, [menuOpen]);

  // Sur l'accueil, l'en-tête N'EST PAS collant et ne crée AUCUN contexte d'empilement (ni z-index,
  // ni backdrop-blur) — comme l'ancien bandeau de la montagne, qui défilait avec la page.
  // POURQUOI : la vue réseau (z-250) et les fleurs (z-300) de la montagne sont des calques plein
  // écran piégés dans le contexte `isolate` de la page (indispensable à son ciel peint à z −1). Un
  // en-tête collant à z-40, frère de ce contexte, passerait AU-DESSUS d'eux et couvrirait leurs
  // boutons de fermeture. Non positionné en z, il est peint avant la page, et les calques le
  // recouvrent ; le menu téléphone et la liste des langues (z-50) restent, eux, au-dessus.
  // Même rendu sur les pages immersives (lib/immersiveRoutes.ts) : leur propre barre colle sous
  // lui, il défile donc avec la page au lieu de la recouvrir.
  const immersive = isImmersiveRoute(pathname);
  const flat = pathname === '/' || immersive;

  return (
    <header
      data-testid="site-header"
      className={cn(
        'h-16 border-b border-border/60 print:hidden',
        flat ? 'relative bg-background/70' : 'sticky top-0 z-40 bg-background/85 backdrop-blur',
        // Page immersive sur téléphone COUCHÉ (même mesure que scene.ts › LANDSCAPE_MAX_HEIGHT et le
        // CSS du dossier vivant) : 64 px d'en-tête sur ~290 px de haut ne laisseraient pas tenir
        // l'ouverture (mesuré sur Pixel 5 couché : le champ débordait de 25 px). La barre de la page,
        // qui porte le nom d'Alexis (retour à l'accueil), en tient lieu — comme avant.
        immersive && '[@media(orientation:landscape)_and_(max-height:500px)]:hidden',
      )}
    >
      {/* Filet doré : signale la surface, rien de plus (jamais un aplat). Absolu : ne compte pas
          dans les 64 px. */}
      <div aria-hidden="true" className="absolute inset-x-0 top-0 h-px bg-gradient-to-r from-transparent via-gold/60 to-transparent" />
      <div className="mx-auto flex h-full max-w-7xl items-center justify-between gap-2 px-3 sm:px-6">
        <Link
          href="/"
          data-testid="site-header-home"
          aria-label={t('brandHome')}
          className="min-w-0 truncate font-heading text-xl font-medium text-foreground focus:outline-none focus-visible:ring-1 focus-visible:ring-ring/70 sm:text-2xl"
        >
          {t('brand')}
        </Link>

        {/* Grand écran : la nav à plat. */}
        <nav aria-label={t('label')} className="hidden lg:block">
          <ul className="flex items-center gap-0.5">
            {PRIMARY_NAV.map((item) => {
              const active = isActiveHref(pathname, item.href);
              return (
                <li key={item.id}>
                  <Link
                    href={item.href}
                    data-testid={`site-nav-${item.id}`}
                    aria-current={active ? 'page' : undefined}
                    className={cn(
                      barLink,
                      'relative px-3',
                      // Actif : texte plein + barre dorée dessous (« où l'on est »).
                      active && 'text-foreground after:absolute after:inset-x-3 after:bottom-1.5 after:h-px after:bg-gold',
                    )}
                  >
                    {t(item.id)}
                  </Link>
                </li>
              );
            })}
          </ul>
        </nav>

        <div className="flex shrink-0 items-center gap-0.5 sm:gap-1">
          <a href={`mailto:${CONTACT_EMAIL}`} className={barLink} data-testid="site-header-contact">
            <Mail className="h-4 w-4" aria-hidden="true" />
            {/* Téléphone : l'icône seule à l'écran, le libellé pour le lecteur d'écran. */}
            <span className="sr-only sm:not-sr-only">{t('contact')}</span>
          </a>
          <div data-testid="site-header-language" className="text-muted-foreground">
            <LanguageSwitcher />
          </div>
          <button
            ref={toggleRef}
            type="button"
            data-testid="site-menu-toggle"
            aria-expanded={menuOpen}
            aria-controls="site-menu"
            aria-label={menuOpen ? t('closeMenu') : t('openMenu')}
            onClick={() => setMenuOpen((o) => !o)}
            className="inline-flex h-11 w-11 items-center justify-center rounded-md text-foreground transition-colors hover:bg-accent focus:outline-none focus-visible:ring-1 focus-visible:ring-ring/70 lg:hidden"
          >
            {menuOpen ? <X className="h-5 w-5" aria-hidden="true" /> : <Menu className="h-5 w-5" aria-hidden="true" />}
          </button>
        </div>
      </div>

      {/* Téléphone : le menu, sous la barre. Rendu seulement ouvert — une seule copie de chaque
          lien dans l'arbre d'accessibilité à la fois. Cibles de 48 px, une par ligne. */}
      {menuOpen && (
        <nav
          id="site-menu"
          data-testid="site-menu"
          aria-label={t('label')}
          className="absolute inset-x-0 top-full z-50 border-b border-border/60 bg-background/95 backdrop-blur lg:hidden"
        >
          <ul className="mx-auto max-w-7xl px-3 py-2 sm:px-6">
            {PRIMARY_NAV.map((item) => {
              const active = isActiveHref(pathname, item.href);
              return (
                <li key={item.id}>
                  <Link
                    href={item.href}
                    data-testid={`site-menu-${item.id}`}
                    aria-current={active ? 'page' : undefined}
                    onClick={() => setMenuOpen(false)}
                    className={cn(
                      'flex min-h-[48px] items-center border-l-2 pl-4 text-base transition-colors focus:outline-none focus-visible:ring-1 focus-visible:ring-ring/70',
                      active ? 'border-gold text-foreground' : 'border-transparent text-muted-foreground hover:text-foreground',
                    )}
                  >
                    {t(item.id)}
                  </Link>
                </li>
              );
            })}
          </ul>
        </nav>
      )}
    </header>
  );
}

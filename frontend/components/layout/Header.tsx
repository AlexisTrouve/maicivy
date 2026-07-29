'use client';

import { Link, usePathname } from '@/i18n/navigation';
import { useTranslations } from 'next-intl';
import { Button } from '@/components/ui/button';
import { cn } from '@/lib/utils';
import { Moon, Sun } from 'lucide-react';
import { useTheme } from '@/hooks/useTheme';
import { LanguageSwitcher } from '@/components/shared/LanguageSwitcher';
import { BackgroundSwitcher } from '@/components/background/BackgroundSwitcher';

export function Header() {
  const pathname = usePathname();
  const { theme, toggleTheme } = useTheme();
  const t = useTranslations('nav');
  const tCommon = useTranslations('common');

  const navigation = [
    { name: t('home'), href: '/' },
    { name: t('cv'), href: '/cv' },
    { name: t('letters'), href: '/letters' },
    { name: t('chat'), href: '/chat' },
    { name: t('analytics'), href: '/analytics' },
    { name: t('blog'), href: '/blog' },
    { name: t('gitstats'), href: '/gitstats' },
  ];

  return (
    <header className="sticky top-0 z-50 w-full border-b bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60">
      <div className="container flex h-16 items-center justify-between">
        <div className="flex items-center gap-8">
          <Link href="/" className="flex items-center gap-2 font-heading text-xl font-bold">
            {/* Logo maicivy — coins transparents, propre sur thème clair comme sombre */}
            <img src="/maicivy-logo.png" alt="" width={28} height={28} className="h-7 w-7" />
            maicivy
          </Link>

          {/* Nav en pastilles (segmented control) : chaque onglet est une surface arrondie cliquable.
              gap-1 (et non gap-6) car chaque pastille porte déjà son propre padding px-3 → on évite
              de doubler l'espacement et de faire déborder la barre sur md (7 onglets). */}
          <nav className="hidden md:flex items-center gap-1">
            {navigation.map((item) => (
              <Link
                key={item.href}
                href={item.href}
                className={cn(
                  // Forme commune : pastille arrondie, padding pour la surface cliquable, transition douce.
                  'rounded-full px-3 py-1.5 text-sm font-medium transition-colors',
                  pathname === item.href
                    // Actif : pastille PLEINE (fond primary + texte contrasté) → saute aux yeux.
                    ? 'bg-primary text-primary-foreground shadow-sm'
                    // Inactif : texte lisible, fond gris au survol pour signaler que c'est cliquable.
                    : 'text-muted-foreground hover:bg-muted hover:text-foreground'
                )}
              >
                {item.name}
              </Link>
            ))}
          </nav>
        </div>

        <div className="flex items-center gap-4">
          {/* Sélecteur de fond animé — desktop uniquement (les fonds ciblent desktop pour l'instant) */}
          <div className="hidden md:block">
            <BackgroundSwitcher />
          </div>
          <LanguageSwitcher />
          <Button
            variant="ghost"
            size="icon"
            onClick={toggleTheme}
            aria-label={tCommon('toggleTheme')}
          >
            {theme === 'dark' ? (
              <Sun className="h-5 w-5" />
            ) : (
              <Moon className="h-5 w-5" />
            )}
          </Button>
        </div>
      </div>
    </header>
  );
}

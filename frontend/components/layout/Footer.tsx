'use client';

import { useTranslations } from 'next-intl';
import { Github, Linkedin, Mail } from 'lucide-react';

import { Link } from '@/i18n/navigation';
import { LanguageSwitcher } from '@/components/shared/LanguageSwitcher';
import { CONTACT_EMAIL } from '@/lib/montagne/links';
import { GITHUB_URL, LINKEDIN_URL } from '@/lib/site/identity';
import { LEGAL_NAV, PRIMARY_NAV, WORKSHOP_NAV, type NavItem } from '@/lib/site/navigation';

/**
 * Footer — le pied de page UNIQUE du site public, sur TOUTES les pages (dossier vivant compris).
 *
 * QUOI : l'identité (nom serif, signature, phrase), puis quatre colonnes de liens — Explorer (la nav
 * principale, pour qui arrive en bas d'une longue page), Coulisses (Git stats, Analytics,
 * Architecture, Lettres), Contact (mailto comme le bandeau de la montagne, LinkedIn, GitHub) et
 * Légal — et la ligne finale : droits, avertissement IA, langue.
 * POURQUOI sur le dossier vivant aussi : sa barre ne porte que le rejeu ; sans ce pied, la page
 * ne menait qu'au portfolio et à la montagne.
 * POURQUOI statique : l'ancien pied interrogeait maiProFiles au navigateur pour un nom et trois
 * liens qui ne changent pas — une requête par page vue, et un français affiché aux visiteurs
 * de/it/zh tant qu'elle n'avait pas répondu. Les profils vivent dans lib/site/identity.ts.
 * Doctrine de l'or : aucun or ici (l'en-tête porte déjà le filet de la surface). Liens neutres,
 * cibles de 44 px.
 * E2E : `site-footer`, `site-footer-<id>`, `site-footer-language`.
 */

const footLink =
  'inline-flex min-h-[44px] items-center gap-2 text-sm text-muted-foreground transition-colors hover:text-foreground focus:outline-none focus-visible:ring-1 focus-visible:ring-ring/70';
const columnTitle = 'text-xs font-medium uppercase tracking-[0.18em] text-muted-foreground/80';

function LinkColumn({ title, items, ns }: { title: string; items: readonly NavItem[]; ns: (id: string) => string }) {
  return (
    <div>
      <h2 className={`${columnTitle} font-sans`}>{title}</h2>
      <ul className="mt-2">
        {items.map((item) => (
          <li key={item.id}>
            <Link href={item.href} className={footLink} data-testid={`site-footer-${item.id}`}>
              {ns(item.id)}
            </Link>
          </li>
        ))}
      </ul>
    </div>
  );
}

export function Footer() {
  const t = useTranslations('footer');
  const tNav = useTranslations('nav');
  // Signature et phrase d'Alexis : celles de la voix de la montagne (accueil), pas une copie — quand
  // il réécrira sa phrase, le pied suivra.
  const tVoice = useTranslations('montagne.voice');
  const year = new Date().getFullYear();

  return (
    <footer data-testid="site-footer" className="relative border-t border-border/60 bg-background/80 print:hidden">
      <div className="mx-auto max-w-7xl px-4 py-10 sm:px-6 md:py-14">
        <div className="grid grid-cols-2 gap-x-6 gap-y-8 md:grid-cols-4 lg:grid-cols-[1.4fr_repeat(4,minmax(0,1fr))]">
          {/* Identité : prend toute la largeur sur téléphone et tablette, une colonne au-delà. */}
          <div className="col-span-2 md:col-span-4 lg:col-span-1">
            <p className="font-heading text-2xl font-medium text-foreground">{tNav('brand')}</p>
            <p className="mt-1 text-[11px] font-medium uppercase tracking-[0.24em] text-muted-foreground">{tVoice('tagline')}</p>
            <p className="mt-3 max-w-xs text-sm leading-relaxed text-muted-foreground">{tVoice('title')}</p>
          </div>

          <LinkColumn title={t('explore')} items={PRIMARY_NAV} ns={tNav} />
          <LinkColumn title={t('workshop')} items={WORKSHOP_NAV} ns={tNav} />

          <div>
            <h2 className={`${columnTitle} font-sans`}>{t('contact')}</h2>
            <ul className="mt-2">
              <li>
                <a href={`mailto:${CONTACT_EMAIL}`} className={footLink} data-testid="site-footer-email">
                  <Mail className="h-4 w-4 shrink-0" aria-hidden="true" />
                  {t('email')}
                </a>
              </li>
              <li>
                <a href={LINKEDIN_URL} target="_blank" rel="noopener noreferrer" className={footLink} data-testid="site-footer-linkedin">
                  <Linkedin className="h-4 w-4 shrink-0" aria-hidden="true" />
                  {t('linkedin')}
                </a>
              </li>
              <li>
                <a href={GITHUB_URL} target="_blank" rel="noopener noreferrer" className={footLink} data-testid="site-footer-github">
                  <Github className="h-4 w-4 shrink-0" aria-hidden="true" />
                  {t('github')}
                </a>
              </li>
            </ul>
          </div>

          <LinkColumn title={t('legal')} items={LEGAL_NAV} ns={t} />
        </div>

        <div className="mt-10 flex flex-col gap-4 border-t border-border/60 pt-6 sm:flex-row sm:items-center sm:justify-between">
          <div className="space-y-1 text-xs leading-relaxed text-muted-foreground">
            <p>{t('rights', { year })}</p>
            <p className="max-w-2xl">{t('legalDisclaimer')}</p>
          </div>
          <div data-testid="site-footer-language" className="self-start text-muted-foreground sm:self-auto">
            <LanguageSwitcher placement="up" />
          </div>
        </div>
      </div>
    </footer>
  );
}

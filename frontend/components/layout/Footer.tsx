'use client';

import { Link } from '@/i18n/navigation';
import { useTranslations } from 'next-intl';
import { Github, Linkedin, Mail } from 'lucide-react';

export function Footer() {
  const currentYear = new Date().getFullYear();
  const t = useTranslations('footer');
  const tNav = useTranslations('nav');

  return (
    <footer className="border-t bg-muted/50">
      <div className="container py-8 md:py-12">
        <div className="grid grid-cols-1 gap-8 md:grid-cols-3">
          <div>
            <h3 className="font-heading text-lg font-semibold">maicivy</h3>
            <p className="mt-2 text-sm text-muted-foreground">
              {t('description')}
            </p>
          </div>

          <div>
            <h3 className="font-heading text-lg font-semibold">{t('navigation')}</h3>
            <ul className="mt-2 space-y-2 text-sm">
              <li>
                <Link href="/cv" className="text-muted-foreground hover:text-foreground">
                  {tNav('cv')}
                </Link>
              </li>
              <li>
                <Link href="/letters" className="text-muted-foreground hover:text-foreground">
                  {tNav('letters')}
                </Link>
              </li>
              <li>
                <Link href="/analytics" className="text-muted-foreground hover:text-foreground">
                  {tNav('analytics')}
                </Link>
              </li>
            </ul>
          </div>

          {/* TODO: Add real contact links
          <div>
            <h3 className="font-heading text-lg font-semibold">{t('contact')}</h3>
            <div className="mt-2 flex gap-4">
              <a href="https://github.com/USERNAME" target="_blank" rel="noopener noreferrer" className="text-muted-foreground hover:text-foreground">
                <Github className="h-5 w-5" />
              </a>
              <a href="https://linkedin.com/in/USERNAME" target="_blank" rel="noopener noreferrer" className="text-muted-foreground hover:text-foreground">
                <Linkedin className="h-5 w-5" />
              </a>
              <a href="mailto:EMAIL" className="text-muted-foreground hover:text-foreground">
                <Mail className="h-5 w-5" />
              </a>
            </div>
          </div>
          */}
        </div>

        <div className="mt-8 border-t pt-8 text-center text-sm text-muted-foreground">
          <p>&copy; {currentYear} maicivy. {t('rights')}</p>
        </div>
      </div>
    </footer>
  );
}

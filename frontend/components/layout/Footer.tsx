'use client';

import { Link } from '@/i18n/navigation';
import { useTranslations, useLocale } from 'next-intl';
import { Github, Linkedin, Mail, MapPin, MessageCircle, ExternalLink, Scale } from 'lucide-react';
import { useEffect, useState } from 'react';

const MAIPROFILES_URL = 'https://maiprofiles.etheryale.com';

interface ProfileContact {
  name: string;
  headline: string;
  location: string;
  email: string;
  github: string;
  gitea: string;
  linkedin: string;
  whatsapp: string;
  githubUrl: string;
  giteaUrl: string;
  linkedinUrl: string;
}

// Fallback statique — utilisé si le fetch maiprofiles échoue (CORS, timeout, etc.)
const FALLBACK_CONTACT: ProfileContact = {
  name: 'Alexis Trouvé',
  headline: 'Full-Stack Engineer & AI Specialist',
  location: 'France',
  email: 'alexistrouve.pro@gmail.com',
  github: 'AlexisTrouve',
  gitea: 'git.etheryale.com/StillHammer',
  linkedin: 'https://www.linkedin.com/in/alexis-trouve-432397a9/',
  whatsapp: '+33695110967',
  githubUrl: 'https://github.com/AlexisTrouve',
  giteaUrl: 'https://git.etheryale.com/StillHammer',
  linkedinUrl: 'https://www.linkedin.com/in/alexis-trouve-432397a9/',
};

// maiProfilesLang mappe une locale next-intl vers une langue supportée par maiProFiles.
// fr/en/ka sont seedés — les autres locales (it, de, zh...) fallback sur "fr" en attendant.
function maiProfilesLang(locale: string): string {
  const supported: Record<string, string> = { fr: 'fr', en: 'en', ka: 'ka' };
  return supported[locale] ?? 'fr';
}

export function Footer() {
  const currentYear = new Date().getFullYear();
  const t = useTranslations('footer');
  const tNav = useTranslations('nav');
  const locale = useLocale();
  const [contact, setContact] = useState<ProfileContact>(FALLBACK_CONTACT);

  // Fetch contact info depuis maiprofiles avec la locale courante de l'app.
  // On passe ?lang= pour que maiProFiles retourne les textes (headline, bio) dans la bonne langue.
  // useLocale() est re-exécuté à chaque changement de locale → le fetch se relance automatiquement.
  useEffect(() => {
    const lang = maiProfilesLang(locale);
    fetch(`${MAIPROFILES_URL}/profile?lang=${lang}`)
      .then(res => res.ok ? res.json() : null)
      .then(data => {
        if (!data) return;
        setContact({
          name: data.name || FALLBACK_CONTACT.name,
          headline: data.headline || FALLBACK_CONTACT.headline,
          location: data.location || FALLBACK_CONTACT.location,
          email: data.contact?.email || FALLBACK_CONTACT.email,
          github: data.contact?.github || FALLBACK_CONTACT.github,
          gitea: data.contact?.gitea || FALLBACK_CONTACT.gitea,
          linkedin: data.contact?.linkedin || FALLBACK_CONTACT.linkedin,
          whatsapp: data.contact?.whatsapp || FALLBACK_CONTACT.whatsapp,
          githubUrl: data.links?.github || FALLBACK_CONTACT.githubUrl,
          giteaUrl: data.links?.gitea || FALLBACK_CONTACT.giteaUrl,
          linkedinUrl: data.links?.linkedin || FALLBACK_CONTACT.linkedinUrl,
        });
      })
      .catch(() => {});
  // Re-fetch when the locale changes so the displayed data matches the current language
  }, [locale]);

  return (
    <footer className="border-t bg-muted/50">
      <div className="container py-10 md:py-14">
        <div className="grid grid-cols-1 gap-10 sm:grid-cols-2 lg:grid-cols-4">
          {/* About */}
          <div>
            <h3 className="font-heading text-lg font-semibold">{t('about')}</h3>
            <p className="mt-3 text-sm leading-relaxed text-muted-foreground">
              {t('aboutText')}
            </p>
            {contact && (
              <div className="mt-3 flex items-center gap-1.5 text-sm text-muted-foreground">
                <MapPin className="h-3.5 w-3.5 shrink-0" />
                <span>{contact.location}</span>
              </div>
            )}
          </div>

          {/* Navigation */}
          <div>
            <h3 className="font-heading text-lg font-semibold">{t('navigation')}</h3>
            <ul className="mt-3 space-y-2 text-sm">
              <li>
                <Link href="/cv" className="text-muted-foreground transition-colors hover:text-foreground">
                  {tNav('cv')}
                </Link>
              </li>
              <li>
                <Link href="/letters" className="text-muted-foreground transition-colors hover:text-foreground">
                  {tNav('letters')}
                </Link>
              </li>
              <li>
                <Link href="/chat" className="text-muted-foreground transition-colors hover:text-foreground">
                  {tNav('chat')}
                </Link>
              </li>
              <li>
                <Link href="/blog" className="text-muted-foreground transition-colors hover:text-foreground">
                  {tNav('blog')}
                </Link>
              </li>
              <li>
                <Link href="/gitstats" className="text-muted-foreground transition-colors hover:text-foreground">
                  {tNav('gitstats')}
                </Link>
              </li>
            </ul>
          </div>

          {/* Mentions légales */}
          <div>
            <h3 className="font-heading text-lg font-semibold">{t('legal')}</h3>
            <ul className="mt-3 space-y-2 text-sm">
              <li>
                <Link href="/legal" className="text-muted-foreground transition-colors hover:text-foreground">
                  {t('legalNotice')}
                </Link>
              </li>
              <li>
                <Link href="/privacy" className="text-muted-foreground transition-colors hover:text-foreground">
                  {t('privacy')}
                </Link>
              </li>
            </ul>
            <div className="mt-4 flex items-start gap-2 text-xs leading-relaxed text-muted-foreground/70">
              <Scale className="mt-0.5 h-3.5 w-3.5 shrink-0" />
              <span>{t('legalDisclaimer')}</span>
            </div>
          </div>

          {/* Contact — données dynamiques maiprofiles */}
          <div>
            <h3 className="font-heading text-lg font-semibold">{t('contact')}</h3>
            <div className="mt-3 space-y-3">
              <p className="text-sm font-medium">{contact.name}</p>
              <p className="text-xs text-muted-foreground">{contact.headline}</p>
              <div className="flex flex-col gap-2 text-sm">
                <a
                  href={`mailto:${contact.email}`}
                  className="flex items-center gap-2 text-muted-foreground transition-colors hover:text-foreground"
                >
                  <Mail className="h-4 w-4 shrink-0" />
                  <span>{contact.email}</span>
                </a>
                <a
                  href={contact.linkedinUrl || contact.linkedin}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="flex items-center gap-2 text-muted-foreground transition-colors hover:text-foreground"
                >
                  <Linkedin className="h-4 w-4 shrink-0" />
                  <span>LinkedIn</span>
                  <ExternalLink className="h-3 w-3 shrink-0 opacity-50" />
                </a>
                <a
                  href={contact.githubUrl}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="flex items-center gap-2 text-muted-foreground transition-colors hover:text-foreground"
                >
                  <Github className="h-4 w-4 shrink-0" />
                  <span>{contact.github}</span>
                  <ExternalLink className="h-3 w-3 shrink-0 opacity-50" />
                </a>
                <a
                  href={`https://wa.me/${contact.whatsapp.replace(/[^0-9]/g, '')}`}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="flex items-center gap-2 text-muted-foreground transition-colors hover:text-foreground"
                >
                  <MessageCircle className="h-4 w-4 shrink-0" />
                  <span>WhatsApp</span>
                  <ExternalLink className="h-3 w-3 shrink-0 opacity-50" />
                </a>
              </div>
            </div>
          </div>
        </div>

        {/* Copyright */}
        <div className="mt-10 border-t pt-8 text-center text-sm text-muted-foreground">
          <p>&copy; {currentYear} maicivy. {t('rights')}</p>
        </div>
      </div>
    </footer>
  );
}

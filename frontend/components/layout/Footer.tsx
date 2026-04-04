'use client';

import { Link } from '@/i18n/navigation';
import { useTranslations } from 'next-intl';
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

export function Footer() {
  const currentYear = new Date().getFullYear();
  const t = useTranslations('footer');
  const tNav = useTranslations('nav');
  const [contact, setContact] = useState<ProfileContact | null>(null);

  // Fetch contact info depuis maiprofiles
  useEffect(() => {
    fetch(`${MAIPROFILES_URL}/profile`)
      .then(res => res.ok ? res.json() : null)
      .then(data => {
        if (!data) return;
        setContact({
          name: data.name,
          headline: data.headline,
          location: data.location,
          email: data.contact?.email || '',
          github: data.contact?.github || '',
          gitea: data.contact?.gitea || '',
          linkedin: data.contact?.linkedin || '',
          whatsapp: data.contact?.whatsapp || '',
          githubUrl: data.links?.github || '',
          giteaUrl: data.links?.gitea || '',
          linkedinUrl: data.links?.linkedin || '',
        });
      })
      .catch(() => {});
  }, []);

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
            {contact ? (
              <div className="mt-3 space-y-3">
                <p className="text-sm font-medium">{contact.name}</p>
                <p className="text-xs text-muted-foreground">{contact.headline}</p>
                <div className="flex flex-col gap-2 text-sm">
                  {contact.email && (
                    <a
                      href={`mailto:${contact.email}`}
                      className="flex items-center gap-2 text-muted-foreground transition-colors hover:text-foreground"
                    >
                      <Mail className="h-4 w-4 shrink-0" />
                      <span>{contact.email}</span>
                    </a>
                  )}
                  {(contact.linkedinUrl || contact.linkedin) && (
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
                  )}
                  {contact.githubUrl && (
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
                  )}
                  {contact.whatsapp && (
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
                  )}
                </div>
              </div>
            ) : (
              <div className="mt-3 space-y-2">
                <div className="h-4 w-32 animate-pulse rounded bg-muted" />
                <div className="h-3 w-48 animate-pulse rounded bg-muted" />
                <div className="h-4 w-36 animate-pulse rounded bg-muted" />
              </div>
            )}
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

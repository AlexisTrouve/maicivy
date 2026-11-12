import { getTranslations } from 'next-intl/server';

import { CONTACT_EMAIL, cvExportUrl } from '@/lib/montagne/links';

import { Icon } from './icons';
import styles from './dossier-vivant.module.css';

/**
 * DossierFooter — le pied de la page du dossier vivant, sous la scène, toujours présent.
 *
 * QUOI : quatre portes vers la suite — le CV en PDF (même cible que MontagneVoice), le portfolio,
 * la montagne des projets (l'accueil) et le contact (mailto, comme l'en-tête du site). Le pied de
 * page commun du site (components/layout/Footer) suit, avec toute la navigation.
 * POURQUOI côté serveur : de simples liens, lisibles sans JavaScript ; DossierVivant (client) le
 * reçoit déjà rendu et le pose sous la scène.
 * Doctrine de l'or : AUCUN bouton or ici — l'action principale de l'écran reste « Lancer
 * l'équipe », puis « Télécharger le dossier ». Liens neutres, filet discret, cibles de 44 px.
 * E2E : `dv-footer`, `dv-footer-{cv,portfolio,montagne,contact}`.
 */
export async function DossierFooter({ locale }: { locale: string }) {
  const t = await getTranslations({ locale, namespace: 'dossierVivant.footer' });
  const links = [
    { id: 'cv', href: cvExportUrl(locale), icon: 'dl' as const },
    { id: 'portfolio', href: `/${locale}/portfolio`, icon: 'arrow' as const },
    // La montagne est l'accueil du site depuis la refonte de 2026-11.
    { id: 'montagne', href: `/${locale}`, icon: 'arrow' as const },
    { id: 'contact', href: `mailto:${CONTACT_EMAIL}`, icon: 'mail' as const },
  ];
  return (
    <footer className={`${styles.footer} ${styles.noPrint}`} data-testid="dv-footer">
      <p className={styles.footerLead}>{t('lead')}</p>
      <nav aria-label={t('label')}>
        <ul className={styles.footerLinks}>
          {links.map((l) => (
            <li key={l.id}>
              <a className={styles.footerLink} href={l.href} data-testid={`dv-footer-${l.id}`}>
                <Icon name={l.icon} />
                {t(l.id)}
              </a>
            </li>
          ))}
        </ul>
      </nav>
    </footer>
  );
}

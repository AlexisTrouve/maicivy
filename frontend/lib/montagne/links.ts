/**
 * links.ts — les cibles du bandeau de la montagne (CV PDF, contact).
 *
 * POURQUOI ici et pas importé de components/cv/ExportPDFButton : ce fichier-là est un module
 * 'use client' ; un Server Component qui en importe une fonction reçoit une RÉFÉRENCE client, pas
 * la fonction — l'appeler casse le rendu. Le bandeau est rendu côté serveur (sans JS), il lui faut
 * donc sa propre copie de la règle de langue du PDF.
 */

/** Langues où /cv/export sait rendre un PDF. Repli ANGLAIS, jamais français (règle du contenu). */
const PDF_LANGS = ['fr', 'en', 'zh'];

export function cvPdfLang(locale: string): string {
  return PDF_LANGS.includes(locale) ? locale : 'en';
}

/** Lien direct vers le PDF du CV standard (thème par défaut du backend). */
export function cvExportUrl(locale: string): string {
  const base = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';
  return `${base}/api/v1/cv/export?theme=fullstack&format=pdf&lang=${cvPdfLang(locale)}`;
}

/** Adresse de contact publique — la même que le pied de page du site (profil maiProFiles). */
export const CONTACT_EMAIL = 'alexistrouve.pro@gmail.com';

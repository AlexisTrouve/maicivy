/**
 * monogram.ts — l'icône provisoire d'un projet : deux lettres dans un disque.
 *
 * QUOI : une chaîne SVG, la MÊME pour la liste rendue côté serveur (inline) et pour le canvas
 * (chargée en image via data-URL). Une seule source de dessin : la liste et la montagne ne
 * peuvent pas diverger.
 * POURQUOI provisoire : les vraies icônes viendront d'Eikon via `icon_url` ; dès qu'il est
 * renseigné, il remplace ce monogramme partout (cf. nodeIconSrc).
 * COMMENT : disque sombre, anneau teinté par la strate (repère de couche, pas de l'or : l'or est
 * réservé au signal), lettres ivoire. Déterministe : les lettres viennent du backend.
 */

import type { MontagneNode, StratumId } from '@/lib/montagne-api';
import { portfolioImageUrl } from '@/lib/portfolio-api';

/**
 * Teinte d'anneau par strate — sourdes, froides en bas, chaudes en haut. Volontairement loin du
 * doré (#d9b45f) pour que la sélection, elle dorée, reste un signal.
 */
export const STRATUM_TINT: Record<StratumId, string> = {
  foundations: '#5b7083',
  tooling: '#6f7f6a',
  engines: '#8a6f8f',
  works: '#a0786a',
};

/** Échappe le texte injecté dans le SVG : les lettres viennent de données externes. */
function escapeXml(s: string): string {
  return s.replace(/[<>&"']/g, (c) => `&#${c.charCodeAt(0)};`);
}

/** SVG 64×64 du monogramme. */
export function monogramSvg(letters: string, stratum: StratumId): string {
  const tint = STRATUM_TINT[stratum] ?? STRATUM_TINT.tooling;
  return (
    `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 64 64" width="64" height="64">` +
    `<circle cx="32" cy="32" r="30" fill="#0b111c"/>` +
    `<circle cx="32" cy="32" r="28.5" fill="none" stroke="${tint}" stroke-width="3"/>` +
    `<text x="32" y="33" text-anchor="middle" dominant-baseline="middle" ` +
    `font-family="Poppins, Inter, system-ui, sans-serif" font-weight="600" font-size="24" ` +
    `fill="#efe9dc">${escapeXml(letters)}</text>` +
    `</svg>`
  );
}

/**
 * Source d'image d'un nœud : l'icône définitive si elle existe (chemin maiProFiles préfixé comme
 * toutes ses images), sinon le monogramme.
 */
export function nodeIconSrc(node: Pick<MontagneNode, 'icon_url' | 'monogram' | 'stratum'>): string {
  if (node.icon_url) return portfolioImageUrl(node.icon_url);
  return `data:image/svg+xml;charset=utf-8,${encodeURIComponent(monogramSvg(node.monogram, node.stratum))}`;
}

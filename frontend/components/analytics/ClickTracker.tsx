'use client';

import { useEffect } from 'react';

// ClickTracker — producteur de données pour la heatmap d'interactions.
//
// QUOI : écoute les clics sur les éléments interactifs (lien / bouton / [role=button] / [data-track])
// et POST leurs coordonnées au backend (`/api/v1/analytics/event`, event_type button_click|link_click).
// POURQUOI : le backend agrège déjà ces events en heatmap (`GET /analytics/heatmap`), MAIS rien ne les
// produisait côté client → la table d'events restait vide → la heatmap affichait « aucune donnée ».
// Ce composant est la pièce manquante. 100% réel : aucune donnée fabriquée, on remonte de vrais clics.
// COMMENT : un seul listener délégué en phase de capture sur `document` (monté une fois via le layout
// locale) ; coordonnées normalisées en POURCENTAGE du viewport (le front les repositionne en left/top %,
// cf. Heatmap.tsx) ; throttle anti-spam ; fire-and-forget (keepalive, erreurs ignorées — un 400 au tout
// premier hit avant que le cookie de session signé soit posé ne doit jamais gêner l'UX).
const API_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

// Throttle minimal entre deux events envoyés : absorbe les double-clics / rage-clicks sans noyer l'API
// (l'endpoint passe par le rate-limiter global ; on reste loin de la limite).
const MIN_INTERVAL_MS = 300;

export default function ClickTracker() {
  useEffect(() => {
    let lastSent = 0;

    const onClick = (e: MouseEvent) => {
      const now = Date.now();
      if (now - lastSent < MIN_INTERVAL_MS) return;

      // On ne tracke que les éléments interactifs : c'est ce qui a un sens pour « où les gens cliquent »
      // et ça évite de logger chaque clic dans le vide.
      const target = e.target as HTMLElement | null;
      const el = target?.closest('a, button, [role="button"], [data-track]') as HTMLElement | null;
      if (!el) return;

      lastSent = now;

      // Coordonnées en % du viewport (indépendant de la résolution). Garde-fou contre une largeur 0.
      const vw = window.innerWidth || 1;
      const vh = window.innerHeight || 1;
      const x = +((e.clientX / vw) * 100).toFixed(1);
      const y = +((e.clientY / vh) * 100).toFixed(1);

      const isLink = el.tagName === 'A';
      // Libellé de l'élément (pour le tooltip de la heatmap) : data-track explicite > aria-label > texte
      // > balise. Tronqué pour rester compact et ne pas stocker des pavés.
      const label = (
        el.getAttribute('data-track') ||
        el.getAttribute('aria-label') ||
        el.textContent ||
        el.tagName
      )
        .trim()
        .slice(0, 40);

      const body = JSON.stringify({
        event_type: isLink ? 'link_click' : 'button_click',
        page_url: window.location.pathname,
        event_data: { x, y, element: label },
      });

      // Fire-and-forget : on n'attend pas la réponse, on avale toute erreur (réseau, 400 sans session…).
      fetch(`${API_URL}/api/v1/analytics/event`, {
        method: 'POST',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
        body,
        keepalive: true,
      }).catch(() => {});
    };

    // Capture = true : on capte le clic même si un handler en aval fait stopPropagation().
    document.addEventListener('click', onClick, { capture: true });
    return () => document.removeEventListener('click', onClick, { capture: true });
  }, []);

  return null; // composant purement comportemental, aucun rendu
}

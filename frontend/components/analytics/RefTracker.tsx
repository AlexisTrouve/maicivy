'use client';

import { useEffect } from 'react';

const API = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

// RefTracker : attribution des visites apportées par un prospecteur.
//
// QUOI : si l'URL d'arrivée porte `?ref=<code>` (liens insérés par le Composer d'un prospecteur), on
// le signale UNE fois au backend (POST /ref), qui crédite le prospecteur.
// POURQUOI côté client : `?ref=` est sur l'URL de la PAGE, que le backend ne voit jamais (il ne reçoit
// que les appels d'API). Le backend déduplique par visiteur ; sessionStorage évite seulement de
// renvoyer le même code à chaque navigation interne.
export default function RefTracker() {
  useEffect(() => {
    const ref = new URLSearchParams(window.location.search).get('ref');
    if (!ref || !/^[a-z0-9]{1,16}$/i.test(ref)) return;
    const key = `maicivy:ref:${ref}`;
    if (sessionStorage.getItem(key)) return;
    sessionStorage.setItem(key, '1');
    void fetch(`${API}/api/v1/ref`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      credentials: 'include',
      body: JSON.stringify({ ref }),
    }).catch(() => {
      /* attribution best-effort : jamais d'erreur visible pour le visiteur */
    });
  }, []);
  return null;
}

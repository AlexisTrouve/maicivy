'use client';

import { useRouter } from 'next/navigation';
import { useTranslations } from 'next-intl';
import { LogOut } from 'lucide-react';

const API = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

// Bouton déconnexion : efface le cookie admin côté backend puis renvoie au login.
export default function LogoutButton() {
  const router = useRouter();
  const t = useTranslations('admin');
  async function logout() {
    // Les deux sessions (owner, prospecteur) ont chacune leur cookie et leur route : on ferme les deux,
    // le bouton n'a pas à savoir laquelle est ouverte.
    await Promise.all([
      fetch(`${API}/api/v1/admin/logout`, { method: 'POST', credentials: 'include' }),
      fetch(`${API}/api/v1/admin/prospect/logout`, { method: 'POST', credentials: 'include' }),
    ]);
    router.push('/admin/login');
    router.refresh();
  }
  return (
    <button
      onClick={logout}
      data-testid="admin-logout"
      // Icône + libellé : la déconnexion est irréversible côté session, elle doit rester
      // identifiable même quand le libellé est dans une langue qu'on ne lit pas.
      className="inline-flex items-center gap-1.5 rounded-lg border border-slate-800 px-2.5 py-1.5 text-xs text-slate-400 transition-colors hover:border-slate-700 hover:text-slate-200"
    >
      <LogOut aria-hidden="true" className="h-3.5 w-3.5" />
      {t('logout')}
    </button>
  );
}

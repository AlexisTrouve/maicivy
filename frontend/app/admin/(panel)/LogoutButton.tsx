'use client';

import { useRouter } from 'next/navigation';
import { useTranslations } from 'next-intl';

const API = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

// Bouton déconnexion : efface le cookie admin côté backend puis renvoie au login.
export default function LogoutButton() {
  const router = useRouter();
  const t = useTranslations('admin');
  async function logout() {
    await fetch(`${API}/api/v1/admin/logout`, { method: 'POST', credentials: 'include' });
    router.push('/admin/login');
    router.refresh();
  }
  return (
    <button
      onClick={logout}
      data-testid="admin-logout"
      className="rounded-md border border-slate-700 px-3 py-1.5 text-sm hover:bg-slate-800"
    >
      {t('logout')}
    </button>
  );
}

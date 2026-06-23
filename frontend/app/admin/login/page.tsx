'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { useTranslations } from 'next-intl';
import AdminLangSwitcher from '../AdminLangSwitcher';

// API publique (même origine que le site en prod) — le POST pose le cookie maicivy_admin côté backend.
const API = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

// Page de login du panneau admin. Poste le mot de passe au backend qui valide + pose le cookie owner
// signé. credentials:include → le Set-Cookie (httpOnly, Secure, SameSite=Lax) est conservé par le navigateur.
export default function AdminLoginPage() {
  const router = useRouter();
  const t = useTranslations('admin.login');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setLoading(true);
    setError('');
    try {
      const res = await fetch(`${API}/api/v1/admin/login`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({ password }),
      });
      if (res.ok) {
        // Cookie posé → on entre. refresh() force le re-render serveur du layout gardé.
        router.push('/admin');
        router.refresh();
      } else {
        setError(t('invalid'));
      }
    } catch {
      setError(t('networkError'));
    } finally {
      setLoading(false);
    }
  }

  return (
    <main className="flex min-h-screen items-center justify-center p-4">
      <form
        onSubmit={submit}
        data-testid="admin-login-form"
        className="w-full max-w-sm space-y-4 rounded-xl border border-slate-800 bg-slate-900 p-6 shadow-xl"
      >
        <div className="flex items-center justify-between">
          <h1 className="text-lg font-semibold">maicivy · admin</h1>
          <AdminLangSwitcher />
        </div>
        <input
          type="password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          placeholder={t('password')}
          autoFocus
          data-testid="admin-password"
          className="w-full rounded-md border border-slate-700 bg-slate-950 px-3 py-2 text-sm outline-none focus:border-blue-500"
        />
        {error && (
          <p data-testid="admin-error" className="text-sm text-red-400">
            {error}
          </p>
        )}
        <button
          type="submit"
          disabled={loading || !password}
          data-testid="admin-submit"
          className="w-full rounded-md bg-blue-600 px-3 py-2 text-sm font-medium hover:bg-blue-500 disabled:opacity-50"
        >
          {loading ? t('loading') : t('submit')}
        </button>
      </form>
    </main>
  );
}

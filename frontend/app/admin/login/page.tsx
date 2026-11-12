'use client';

import { useEffect, useRef, useState } from 'react';
import { useRouter } from 'next/navigation';
import { useTranslations } from 'next-intl';
import AdminLangSwitcher from '../AdminLangSwitcher';
import { btnPrimary, field } from '@/components/admin/ui';

// API publique (même origine que le site en prod) — le POST pose le cookie maicivy_admin côté backend.
const API = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

// Page de login du panneau admin. Poste le mot de passe au backend qui valide + pose le cookie owner
// signé. credentials:include → le Set-Cookie (httpOnly, Secure, SameSite=Lax) est conservé par le navigateur.
export default function AdminLoginPage() {
  const router = useRouter();
  const t = useTranslations('admin.login');
  const tp = useTranslations('admin.prospects');
  // Deux portes, une seule page : l'owner (mot de passe partagé, inchangé) et le prospecteur (nom ou
  // email + mot de passe personnel). Le mode PROSPECTEUR est celui par défaut : les prospecteurs ne
  // remarquaient pas le lien « je suis prospecteur » et tapaient leur mot de passe dans le champ owner.
  // L'owner, seul et averti, bascule via le lien.
  const [prospectMode, setProspectMode] = useState(true);
  const [login, setLogin] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  // Focus sur le PREMIER champ du mode actif, au chargement ET à chaque bascule. POURQUOI un effet et
  // pas autoFocus : le champ mot de passe reste monté d'un mode à l'autre, autoFocus ne rejouerait pas.
  const loginRef = useRef<HTMLInputElement>(null);
  const passwordRef = useRef<HTMLInputElement>(null);
  useEffect(() => {
    (prospectMode ? loginRef : passwordRef).current?.focus();
  }, [prospectMode]);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setLoading(true);
    setError('');
    try {
      const res = await fetch(`${API}/api/v1/admin/${prospectMode ? 'prospect/login' : 'login'}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify(prospectMode ? { login, password } : { password }),
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
      {/* Seule page du panneau vue avant d'être identifié : c'est elle qui pose le ton. D'où le
          traitement le plus appuyé de l'or — filet en tête et monogramme — alors que les écrans de
          travail, eux, le gardent en accent discret pour ne pas fatiguer à l'usage quotidien. */}
      <form
        onSubmit={submit}
        data-testid="admin-login-form"
        className="admin-hairline relative w-full max-w-sm space-y-5 rounded-2xl border border-slate-800/80 bg-slate-900/60 p-7 shadow-2xl shadow-black/40 backdrop-blur-sm"
      >
        <div className="flex items-start justify-between gap-3">
          <div className="flex items-center gap-2.5">
            <span
              aria-hidden="true"
              className="flex h-9 w-9 items-center justify-center rounded-lg border border-gold/40 bg-gold/10 font-heading text-lg font-bold text-gold"
            >
              m
            </span>
            <div className="leading-tight">
              <h1 className="font-heading text-base font-semibold text-slate-100">maicivy</h1>
              <div className="text-[10px] uppercase tracking-[0.18em] text-gold/70">admin</div>
            </div>
          </div>
          <AdminLangSwitcher />
        </div>
        {prospectMode && (
          <>
            <div className="text-sm font-medium text-slate-200">{tp('loginTitle')}</div>
            <input
              type="text"
              value={login}
              onChange={(e) => setLogin(e.target.value)}
              placeholder={tp('loginField')}
              autoComplete="username"
              ref={loginRef}
              data-testid="prospect-login"
              className={field}
            />
          </>
        )}
        <input
          type="password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          placeholder={t('password')}
          ref={passwordRef}
          data-testid="admin-password"
          className={field}
        />
        {error && (
          <p data-testid="admin-error" className="text-sm text-rose-400">
            {error}
          </p>
        )}
        <button
          type="submit"
          disabled={loading || !password || (prospectMode && !login.trim())}
          data-testid="admin-submit"
          className={`${btnPrimary} w-full`}
        >
          {loading ? t('loading') : t('submit')}
        </button>
        <button
          type="button"
          onClick={() => {
            setProspectMode(!prospectMode);
            setError('');
          }}
          data-testid="login-mode-switch"
          className="w-full text-center text-xs text-slate-500 hover:text-slate-300"
        >
          {prospectMode ? tp('switchToOwner') : tp('switchToProspect')}
        </button>
      </form>
    </main>
  );
}

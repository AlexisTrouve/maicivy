'use client';

import { use, useEffect, useState } from 'react';
import Link from 'next/link';
import { useTranslations } from 'next-intl';
import AdminLangSwitcher from '../../AdminLangSwitcher';
import { btnPrimary, field } from '@/components/admin/ui';

const API = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

// Acceptation d'une invitation prospecteur : le lien (à usage unique, expirant) ouvre cette page, où
// la personne invitée choisit son mot de passe. Hors du groupe (panel) : elle n'est pas encore connectée.
// Le compte n'est activé qu'ici — l'owner ne connaît jamais ce mot de passe.
export default function InvitePage({ params }: { params: Promise<{ token: string }> }) {
  // Next 15 : même une page client reçoit params en Promise — use() la déballe (l'accès synchrone
  // n'est plus qu'une compatibilité dépréciée).
  const { token } = use(params);
  const t = useTranslations('admin.prospects');
  const [name, setName] = useState<string | null>(null);
  const [invalid, setInvalid] = useState(false);
  const [password, setPassword] = useState('');
  // Saisie de confirmation : le mot de passe n'est connu que de la personne invitée (l'owner ne le voit
  // jamais) et le lien est à usage unique — une faute de frappe non détectée ici = compte inaccessible,
  // et il faut une nouvelle invitation. Arrivé le 15/10 au premier test réel.
  const [confirm, setConfirm] = useState('');
  const mismatch = confirm.length > 0 && confirm !== password;
  const [error, setError] = useState('');
  const [done, setDone] = useState(false);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    fetch(`${API}/api/v1/admin/invite/${encodeURIComponent(token)}`, { credentials: 'include' })
      .then(async (res) => {
        if (!res.ok) return setInvalid(true);
        setName((await res.json()).name);
      })
      .catch(() => setInvalid(true));
  }, [token]);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    // Garde redondante avec le bouton désactivé : Entrée dans un champ soumet aussi le formulaire.
    if (password !== confirm) return setError(t('passwordMismatch'));
    setLoading(true);
    setError('');
    try {
      const res = await fetch(`${API}/api/v1/admin/invite/${encodeURIComponent(token)}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({ password }),
      });
      if (res.ok) setDone(true);
      else if (res.status === 400) setError(t('passwordLength'));
      else setInvalid(true);
    } catch {
      setError(t('errorGeneric'));
    } finally {
      setLoading(false);
    }
  }

  return (
    <main className="flex min-h-screen items-center justify-center p-4">
      <div className="admin-hairline relative w-full max-w-sm space-y-5 rounded-2xl border border-slate-800/80 bg-slate-900/60 p-7 shadow-2xl shadow-black/40">
        <div className="flex items-start justify-between gap-3">
          <h1 className="font-heading text-base font-semibold text-slate-100">
            {name ? t('inviteTitle', { name }) : 'maicivy'}
          </h1>
          <AdminLangSwitcher />
        </div>
        {invalid ? (
          <p data-testid="invite-invalid" className="text-sm text-rose-400">✕ {t('inviteInvalid')}</p>
        ) : done ? (
          <div className="space-y-4">
            <p data-testid="invite-done" className="text-sm text-emerald-400">✓ {t('inviteDone')}</p>
            {/* Link (et non <a>) : navigation client vers une autre route de l'app — exigé par
                @next/next/no-html-link-for-pages, qui inspecte app/ depuis eslint-config-next 15. */}
            <Link href="/admin/login" data-testid="invite-go-login" className={`${btnPrimary} block w-full text-center`}>
              {t('inviteGoLogin')}
            </Link>
          </div>
        ) : (
          name && (
            <form onSubmit={submit} className="space-y-4" data-testid="invite-form">
              <p className="text-sm text-slate-400">{t('inviteHelp')}</p>
              <input
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                placeholder={t('invitePassword')}
                autoComplete="new-password"
                data-testid="invite-password"
                className={field}
              />
              <input
                type="password"
                value={confirm}
                onChange={(e) => setConfirm(e.target.value)}
                placeholder={t('invitePasswordConfirm')}
                autoComplete="new-password"
                data-testid="invite-password-confirm"
                className={field}
              />
              {mismatch && (
                <p data-testid="invite-mismatch" className="text-sm text-rose-400">✕ {t('passwordMismatch')}</p>
              )}
              {error && <p className="text-sm text-rose-400">✕ {error}</p>}
              <button type="submit" disabled={loading || password.length < 10 || password !== confirm} data-testid="invite-submit" className={`${btnPrimary} w-full`}>
                {t('inviteSubmit')}
              </button>
            </form>
          )
        )}
      </div>
    </main>
  );
}

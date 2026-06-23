import { cookies } from 'next/headers';
import { redirect } from 'next/navigation';
import LogoutButton from './LogoutButton';
import AdminNav from './AdminNav';
import AdminLangSwitcher from '../AdminLangSwitcher';

const ADMIN_COOKIE = 'maicivy_admin';

// URL backend interne (réseau docker) côté serveur.
function apiUrl(): string {
  return process.env.API_URL || 'http://maicivy-backend:8080';
}

// Guard server-side du panneau. POURQUOI valider via le backend (et pas juste la présence du cookie) :
// le cookie est signé HMAC côté serveur — seul le backend peut dire s'il est authentique et non
// expiré. On forwarde le cookie à /admin/me ; 200 = owner, sinon redirect login. Un cookie forgé
// passe peut-être ce guard de présence mais échoue ici (et de toute façon sur tout appel owner).
async function isAuthed(): Promise<boolean> {
  const token = cookies().get(ADMIN_COOKIE)?.value;
  if (!token) return false;
  try {
    const res = await fetch(`${apiUrl()}/api/v1/admin/me`, {
      headers: { Cookie: `${ADMIN_COOKIE}=${token}` },
      cache: 'no-store',
    });
    return res.ok;
  } catch {
    return false;
  }
}

// Shell du panneau (sidebar) + guard. Toute route sous (panel) hérite de ce gardien → ajouter un
// outil = créer app/admin/(panel)/<outil>/page.tsx, il sera protégé automatiquement.
export default async function PanelLayout({ children }: { children: React.ReactNode }) {
  if (!(await isAuthed())) {
    redirect('/admin/login');
  }
  return (
    <div className="flex min-h-screen">
      <aside className="flex w-56 shrink-0 flex-col border-r border-slate-800 bg-slate-900 p-4">
        <div className="mb-6 text-sm font-semibold text-slate-400">maicivy · admin</div>
        <AdminNav />
        {/* Bas de sidebar : sélecteur de langue + déconnexion, persistants sur tous les outils */}
        <div className="mt-auto flex items-center justify-between pt-4">
          <AdminLangSwitcher />
          <LogoutButton />
        </div>
      </aside>
      <main className="flex-1 p-8">{children}</main>
    </div>
  );
}

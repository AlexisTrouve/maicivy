import { cookies } from 'next/headers';
import { redirect } from 'next/navigation';
import LogoutButton from './LogoutButton';
import AdminNav from './AdminNav';
import RoleGate, { type AdminRole } from './RoleGate';
import AdminLangSwitcher from '../AdminLangSwitcher';

const ADMIN_COOKIE = 'maicivy_admin';
// Cookie de session d'un prospecteur (apporteur d'affaires) — cf. backend middleware/prospect_auth.go.
const PROSPECT_COOKIE = 'maicivy_prospect';

// URL backend interne (réseau docker) côté serveur.
function apiUrl(): string {
  return process.env.API_URL || 'http://maicivy-backend:8080';
}

// Guard server-side du panneau. POURQUOI valider via le backend (et pas juste la présence du cookie) :
// les cookies sont signés HMAC côté serveur — seul le backend peut dire s'ils sont authentiques, non
// expirés, et (pour un prospecteur) si le compte est toujours ACTIF : une révocation doit fermer le
// panneau au chargement suivant. On forwarde les deux cookies à /admin/me, qui renvoie le rôle.
async function currentRole(): Promise<AdminRole | null> {
  // Next 15 : cookies() est asynchrone.
  const jar = await cookies();
  const parts = [ADMIN_COOKIE, PROSPECT_COOKIE]
    .map((name) => {
      const v = jar.get(name)?.value;
      return v ? `${name}=${v}` : '';
    })
    .filter(Boolean);
  if (parts.length === 0) return null;
  try {
    const res = await fetch(`${apiUrl()}/api/v1/admin/me`, {
      headers: { Cookie: parts.join('; ') },
      cache: 'no-store',
    });
    if (!res.ok) return null;
    const j = await res.json();
    return j.role === 'prospect' ? 'prospect' : 'owner';
  } catch {
    return null;
  }
}

// Shell du panneau (sidebar) + guard. Toute route sous (panel) hérite de ce gardien → ajouter un
// outil = créer app/admin/(panel)/<outil>/page.tsx, il sera protégé automatiquement.
export default async function PanelLayout({ children }: { children: React.ReactNode }) {
  const role = await currentRole();
  if (!role) {
    redirect('/admin/login');
  }
  return (
    <div className="flex min-h-screen">
      {/* La sidebar RECULE (surface-inset, sous le fond de page) pour que les panneaux de travail
          ressortent. Une navigation aussi claire que le contenu se dispute l'attention avec lui. */}
      <aside className="relative flex w-60 shrink-0 flex-col border-r border-slate-800/80 bg-surface-inset p-4">
        {/* Filet doré sur l'arête droite de la sidebar — la seule ligne dorée pleine hauteur de
            l'écran. Elle sépare la navigation du travail, et signe la surface owner. Le dégradé
            l'éteint en haut et en bas pour qu'elle ne se lise pas comme une bordure de tableau. */}
        <span
          aria-hidden="true"
          className="pointer-events-none absolute inset-y-0 right-0 w-px bg-gradient-to-b from-transparent via-gold/60 to-transparent"
        />

        {/* Bloc identité. Le monogramme doré donne un point d'ancrage visuel en haut de l'écran —
            « maicivy · admin » en texte gris se lisait comme un commentaire oublié. */}
        <div className="mb-6 flex items-center gap-2.5">
          <span
            aria-hidden="true"
            className="flex h-9 w-9 items-center justify-center rounded-lg border border-gold-deep/60 bg-gradient-to-b from-gold-bright to-gold-deep font-heading text-lg font-bold text-slate-950 shadow-gold"
          >
            m
          </span>
          <div className="leading-tight">
            <div className="font-heading text-sm font-semibold text-slate-100">maicivy</div>
            <div className="text-[10px] uppercase tracking-[0.18em] text-gold/70">admin</div>
          </div>
        </div>

        <AdminNav role={role} />

        {/* Bas de sidebar : sélecteur de langue + déconnexion, persistants sur tous les outils.
            Séparés par un filet — sans lui, ils flottaient sous la navigation sans qu'on sache
            s'ils en faisaient partie. */}
        <div className="mt-auto flex items-center justify-between gap-2 border-t border-slate-800/70 pt-4">
          <AdminLangSwitcher />
          <LogoutButton />
        </div>
      </aside>
      {/* p-8 est CONTRACTUEL : compose, chat et mailbox bornent leur hauteur à `calc(100vh-4rem)`,
          soit exactement ce padding en haut et en bas. Le changer ici fait déborder ces trois écrans
          et casse l'E2E qui vérifie que la page ne défile pas verticalement. */}
      <main className="min-w-0 flex-1 p-8">
        {/* RoleGate : un prospecteur qui tape l'URL d'un outil owner voit un refus explicite au lieu
            d'un écran cassé par les 403 du backend (qui reste la vraie barrière). */}
        <RoleGate role={role}>{children}</RoleGate>
      </main>
    </div>
  );
}

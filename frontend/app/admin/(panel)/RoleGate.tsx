'use client';

import { usePathname } from 'next/navigation';
import { useTranslations } from 'next-intl';
import { ShieldOff } from 'lucide-react';
import { Panel } from '@/components/admin/ui';

export type AdminRole = 'owner' | 'prospect';

// Les seuls outils d'un prospecteur : CV depuis offre (/admin), Composer (/admin/compose) et Agent
// (/admin/chat — ses propres conversations, modèle visiteur, données publiques du portfolio).
// Tenu en miroir de ProspectAllowedAdminRoutes côté backend, qui reste l'autorité.
export const PROSPECT_PATHS = ['/admin', '/admin/compose', '/admin/chat'];

// RoleGate : remplace le contenu par un refus explicite quand un prospecteur ouvre l'URL d'un outil
// owner. POURQUOI côté client : le layout serveur ne connaît pas le chemin courant, et l'écran owner
// ne ferait qu'afficher des erreurs 403 en cascade.
export default function RoleGate({ role, children }: { role: AdminRole; children: React.ReactNode }) {
  const pathname = usePathname();
  const t = useTranslations('admin.prospects');
  if (role === 'prospect' && !PROSPECT_PATHS.includes(pathname)) {
    return (
      <Panel className="p-6">
        <div data-testid="admin-forbidden" className="flex items-center gap-3 text-slate-300">
          <ShieldOff aria-hidden="true" className="h-5 w-5 text-rose-400" />
          {t('forbidden')}
        </div>
      </Panel>
    );
  }
  return <>{children}</>;
}

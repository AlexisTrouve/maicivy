'use client';

import { Link } from '@/i18n/navigation';

// Fiche par défaut affichée quand aucun tool_result n'a encore été reçu
export function DefaultFiche() {
  return (
    <div className="p-6 space-y-6">
      {/* Bio */}
      <div>
        <div className="flex items-center gap-3 mb-3">
          <div className="w-12 h-12 rounded-full bg-primary/10 flex items-center justify-center text-xl">
            👨‍💻
          </div>
          <div>
            <h2 className="font-semibold text-foreground">Alexi</h2>
            <p className="text-sm text-muted-foreground">Développeur Full-Stack &amp; IA · Freelance</p>
          </div>
        </div>
        <p className="text-sm text-muted-foreground leading-relaxed">
          Développeur freelance full-stack avec une spécialisation en IA et systèmes agentiques.
          Je construis des produits complets — de l&apos;API au frontend.
        </p>
      </div>

      {/* Top skills */}
      <div>
        <h3 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider mb-2">
          Stack principale
        </h3>
        <div className="flex flex-wrap gap-1.5">
          {['Go', 'Next.js', 'TypeScript', 'Claude API', 'PostgreSQL', 'Redis', 'Docker'].map(
            (skill) => (
              <span
                key={skill}
                className="px-2 py-0.5 rounded-md bg-primary/10 text-primary text-xs font-medium"
              >
                {skill}
              </span>
            ),
          )}
        </div>
      </div>

      {/* Projets récents */}
      <div>
        <h3 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider mb-2">
          Projets récents
        </h3>
        <ul className="space-y-1.5">
          {[
            { name: 'maicivy', desc: 'CV IA interactif' },
            { name: 'aria', desc: 'Agent IA autonome' },
            { name: 'cogesco', desc: 'SEO IA SaaS' },
            { name: 'liveconf', desc: 'Conférences live' },
            { name: 'freelance-dashboard', desc: 'Gestion freelance' },
          ].map((p) => (
            <li key={p.name} className="flex items-center gap-2 text-sm">
              <span className="w-1.5 h-1.5 rounded-full bg-primary shrink-0" />
              <span className="font-medium">{p.name}</span>
              <span className="text-muted-foreground text-xs">— {p.desc}</span>
            </li>
          ))}
        </ul>
      </div>

      {/* CTA */}
      <div className="pt-2">
        <Link
          href="/letters"
          className="block w-full text-center rounded-lg bg-primary text-primary-foreground px-4 py-2 text-sm font-medium hover:bg-primary/90 transition-colors"
        >
          Générer une lettre de motivation →
        </Link>
      </div>

      {/* Hint */}
      <p className="text-xs text-muted-foreground text-center">
        💬 Posez une question pour voir les détails s&apos;afficher ici
      </p>
    </div>
  );
}

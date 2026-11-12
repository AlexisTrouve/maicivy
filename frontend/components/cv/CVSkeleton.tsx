// Squelette du corps du CV (expériences, compétences, projets) pendant le rendu en flux.
//
// Mêmes rythmes que la page finale (titre de section à filet, cartes posées sur `bg-card`) pour
// que l'arrivée du contenu ne fasse pas sauter la mise en page. Les blocs sont en `bg-muted`, la
// surface relevée du thème — aucune couleur propre.
//
// Largeurs des pastilles FIXES (et non `Math.random()`) : ce composant est rendu côté serveur ;
// un aléa y donnait un HTML différent à chaque rendu, donc un squelette qui « tremble » d'une
// visite à l'autre sans rien apporter.
const SKILL_WIDTHS = [96, 132, 84, 148, 110, 92, 124, 104];

export function CVSkeleton() {
  return (
    <div className="space-y-20 animate-pulse" data-testid="cv-skeleton" aria-hidden="true">
      {/* Expériences */}
      <section data-testid="cv-skeleton-section">
        <div className="mb-8 h-9 w-56 rounded bg-muted" />
        <div className="space-y-10">
          {[1, 2, 3].map((i) => (
            <div key={i} data-testid="cv-skeleton-experience" className="ml-9 h-48 rounded-lg border border-border bg-card md:ml-0 md:w-5/12" />
          ))}
        </div>
      </section>

      {/* Compétences */}
      <section data-testid="cv-skeleton-section">
        <div className="mb-8 h-9 w-48 rounded bg-muted" />
        <div className="flex min-h-[300px] flex-wrap content-center justify-center gap-3 rounded-xl border border-border bg-card p-6 md:p-10">
          {SKILL_WIDTHS.map((w, i) => (
            <div key={i} data-testid="cv-skeleton-skill" className="h-9 rounded-full bg-muted" style={{ width: `${w}px` }} />
          ))}
        </div>
      </section>

      {/* Projets */}
      <section data-testid="cv-skeleton-section">
        <div className="mb-8 h-9 w-40 rounded bg-muted" />
        <div className="grid grid-cols-1 gap-6 md:grid-cols-2 lg:grid-cols-3">
          {[1, 2, 3, 4, 5, 6].map((i) => (
            <div key={i} data-testid="cv-skeleton-project" className="h-56 rounded-lg border border-border bg-card" />
          ))}
        </div>
      </section>
    </div>
  );
}

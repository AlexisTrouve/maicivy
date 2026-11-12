'use client';

import { useTranslations } from 'next-intl';

interface Project {
  Name: string;
  Title: string;
  Category: string;
  ShortDesc: string;
  TechStack: string[];
}

interface ProjectListFicheProps {
  data: Project[];
  // Clic sur une carte → envoie un message dans le chat pour ouvrir la fiche détail de CE projet
  // (même mécanisme que les hints : la donnée ici n'a que Name/Title/Category/ShortDesc/TechStack,
  // pas les KeyFeatures/Stats/liens de la fiche détail — on redemande au LLM plutôt que de dupliquer
  // le fetch de données côté frontend).
  onProjectClick: (message: string) => void;
}

// Fiche liste de projets — affichée quand tool_result list_projects est reçu
export function ProjectListFiche({ data, onProjectClick }: ProjectListFicheProps) {
  const t = useTranslations('chat');
  return (
    <div className="p-6 space-y-4">
      <h2 className="text-lg font-bold text-foreground">{t('projects')}</h2>
      <div className="space-y-3">
        {data.map((project) => (
          <button
            key={project.Name}
            type="button"
            onClick={() => onProjectClick(t('openProjectMessage', { name: project.Title }))}
            className="w-full text-left rounded-lg border bg-card p-3 space-y-1.5 transition-colors
                       hover:border-primary/40 hover:bg-primary/5 cursor-pointer"
          >
            <div className="flex items-start justify-between gap-2">
              <h3 className="text-sm font-semibold text-foreground">{project.Title}</h3>
              <span className="shrink-0 text-xs px-2 py-0.5 rounded-full bg-primary/10 text-primary">
                {project.Category}
              </span>
            </div>
            <p className="text-xs text-muted-foreground leading-relaxed line-clamp-2">
              {project.ShortDesc}
            </p>
            <div className="flex flex-wrap gap-1">
              {(project.TechStack ?? []).slice(0, 4).map((tech) => (
                <span
                  key={tech}
                  className="px-1.5 py-0.5 rounded bg-muted text-muted-foreground text-xs"
                >
                  {tech}
                </span>
              ))}
              {(project.TechStack ?? []).length > 4 && (
                <span className="px-1.5 py-0.5 rounded bg-muted text-muted-foreground text-xs">
                  +{(project.TechStack ?? []).length - 4}
                </span>
              )}
            </div>
          </button>
        ))}
      </div>
    </div>
  );
}

'use client';

import { useTranslations } from 'next-intl';

interface StatItem {
  Label: string;
  Value: string;
}

interface ProjectData {
  Name: string;
  Title: string;
  Category: string;
  ShortDesc: string;
  KeyFeatures: string[];
  TechStack: string[];
  Stats: StatItem[];
  SkillsTags: string[];
  GithubURL?: string;
  DemoURL?: string;
}

interface ProjectFicheProps {
  data: ProjectData;
}

// Fiche détaillée d'un projet — affichée quand tool_result get_project est reçu
export function ProjectFiche({ data }: ProjectFicheProps) {
  const t = useTranslations('chat');
  return (
    <div className="p-6 space-y-5">
      {/* Header */}
      <div>
        <span className="inline-block px-2 py-0.5 rounded-md bg-primary/10 text-primary text-xs font-medium mb-2">
          {data.Category}
        </span>
        <h2 className="text-lg font-bold text-foreground leading-tight">{data.Title}</h2>
        <p className="mt-1.5 text-sm text-muted-foreground leading-relaxed">{data.ShortDesc}</p>
      </div>

      {/* Liens — GithubURL n'est renseigné QUE si le repo est public (github.com), cf. backend */}
      {(data.GithubURL || data.DemoURL) && (
        <div className="flex gap-2">
          {data.GithubURL && (
            <a
              href={data.GithubURL}
              target="_blank"
              rel="noopener noreferrer"
              className="flex-1 text-center rounded-lg border px-3 py-2 text-sm font-medium hover:bg-muted transition-colors"
            >
              {t('viewOnGithub')}
            </a>
          )}
          {data.DemoURL && (
            <a
              href={data.DemoURL}
              target="_blank"
              rel="noopener noreferrer"
              className="flex-1 text-center rounded-lg bg-primary text-primary-foreground px-3 py-2 text-sm font-medium hover:bg-primary/90 transition-colors"
            >
              {t('viewDemo')}
            </a>
          )}
        </div>
      )}

      {/* Tech stack */}
      <div>
        <h3 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider mb-2">
          🔧 {t('techStack')}
        </h3>
        <div className="flex flex-wrap gap-1.5">
          {data.TechStack.map((tech) => (
            <span
              key={tech}
              className="px-2 py-0.5 rounded-md bg-secondary text-secondary-foreground text-xs font-medium"
            >
              {tech}
            </span>
          ))}
        </div>
      </div>

      {/* Features */}
      {data.KeyFeatures && data.KeyFeatures.length > 0 && (
      <div>
        <h3 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider mb-2">
          ✓ {t('keyFeatures')}
        </h3>
        <ul className="space-y-1.5">
          {data.KeyFeatures.map((feature, i) => (
            <li key={i} className="flex items-start gap-2 text-sm text-foreground">
              <span className="text-primary mt-0.5 shrink-0">✓</span>
              <span>{feature}</span>
            </li>
          ))}
        </ul>
      </div>
      )}

      {/* Stats */}
      {data.Stats && data.Stats.length > 0 && (
        <div>
          <h3 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider mb-2">
            📊 Stats
          </h3>
          <div className="grid grid-cols-3 gap-2">
            {data.Stats.map((stat) => (
              <div
                key={stat.Label}
                className="rounded-lg bg-muted p-2.5 text-center"
              >
                <div className="text-sm font-bold text-foreground">{stat.Value}</div>
                <div className="text-xs text-muted-foreground mt-0.5">{stat.Label}</div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Skills tags */}
      {data.SkillsTags && data.SkillsTags.length > 0 && (
        <div>
          <h3 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider mb-2">
            Skills
          </h3>
          <div className="flex flex-wrap gap-1.5">
            {data.SkillsTags.map((skill) => (
              <span
                key={skill}
                className="px-2 py-0.5 rounded-full border text-xs text-muted-foreground"
              >
                {skill}
              </span>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}

'use client';

import { useTranslations } from 'next-intl';

interface SkillCategory {
  Name: string;
  Skills: string[];
}

interface SkillsFicheProps {
  // Peut être un tableau de SkillCategory (list_skills) ou un objet ExperienceData (get_experience)
  data: SkillCategory[] | Record<string, unknown>;
}

// Fiche skills — affichée quand tool_result list_skills ou get_experience est reçu
export function SkillsFiche({ data }: SkillsFicheProps) {
  // Détecter si c'est un tableau de catégories (list_skills) ou une ExperienceData
  if (Array.isArray(data)) {
    return <SkillsCategoriesFiche categories={data} />;
  }

  // Sinon c'est ExperienceData (get_experience)
  return <ExperienceFiche data={data as unknown as ExperienceData} />;
}

interface SkillsCategoriesProps {
  categories: SkillCategory[];
}

function SkillsCategoriesFiche({ categories }: SkillsCategoriesProps) {
  const t = useTranslations('chat');
  return (
    <div className="p-6 space-y-5">
      <h2 className="text-lg font-bold text-foreground">{t('skillsTitle')}</h2>
      {categories.map((cat) => (
        <div key={cat.Name}>
          <h3 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider mb-2">
            {cat.Name}
          </h3>
          <div className="flex flex-wrap gap-1.5">
            {cat.Skills.map((skill) => (
              <span
                key={skill}
                className="px-2.5 py-1 rounded-lg bg-muted text-foreground text-xs font-medium"
              >
                {skill}
              </span>
            ))}
          </div>
        </div>
      ))}
    </div>
  );
}

// Un poste dans le parcours pro (rendu dans la fiche expérience).
interface ExperienceItem {
  Role: string;
  Company: string;
  Period: string;
  Summary: string;
}

interface ExperienceData {
  Bio: string;
  BioFull: string;
  Headline: string;
  TJM: string;
  Dispo: string;
  ExperienceYears: number;
  Domains: string[];
  Experience: ExperienceItem[];
}

function ExperienceFiche({ data }: { data: ExperienceData }) {
  const t = useTranslations('chat');
  return (
    <div className="p-6 space-y-5">
      {/* Headline + dispo + années */}
      <div>
        <h2 className="text-lg font-bold text-foreground">{data.Headline}</h2>
        <div className="flex items-center gap-3 mt-1.5 flex-wrap">
          {data.ExperienceYears > 0 && (
            <span className="text-sm font-medium text-foreground">
              {t('yearsExperience', { years: data.ExperienceYears })}
            </span>
          )}
          <span className="text-sm text-muted-foreground">{data.TJM}</span>
          <span className="inline-flex items-center gap-1 text-xs font-medium text-green-600 dark:text-green-400">
            <span className="w-1.5 h-1.5 rounded-full bg-green-500 inline-block" />
            {data.Dispo}
          </span>
        </div>
      </div>

      {/* Bio */}
      <p className="text-sm text-muted-foreground leading-relaxed">
        {data.BioFull || data.Bio}
      </p>

      {/* Domaines */}
      {data.Domains && data.Domains.length > 0 && (
        <div>
          <h3 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider mb-2">
            {t('domains')}
          </h3>
          <div className="flex flex-wrap gap-1.5">
            {data.Domains.map((d) => (
              <span
                key={d}
                className="px-2.5 py-1 rounded-lg bg-muted text-foreground text-xs font-medium"
              >
                {d}
              </span>
            ))}
          </div>
        </div>
      )}

      {/* Expériences pro */}
      {data.Experience && data.Experience.length > 0 && (
        <div>
          <h3 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider mb-3">
            {t('experiences')}
          </h3>
          <div className="space-y-4">
            {data.Experience.map((exp, i) => (
              <div key={i} className="border-l-2 border-primary/30 pl-3">
                <div className="flex items-start justify-between gap-2">
                  <div>
                    <p className="text-sm font-semibold text-foreground">{exp.Role}</p>
                    <p className="text-xs text-muted-foreground">{exp.Company}</p>
                  </div>
                  <span className="text-xs text-muted-foreground shrink-0">{exp.Period}</span>
                </div>
                <p className="mt-1 text-xs text-muted-foreground leading-relaxed">{exp.Summary}</p>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}

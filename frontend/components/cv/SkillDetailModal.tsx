'use client';

import { useEffect, useCallback, useRef } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { useTranslations, useLocale } from 'next-intl';
import { X, Code2, Briefcase, Rocket, Github, ExternalLink } from 'lucide-react';
import { cn } from '@/lib/utils';
import { Skill, Project, Experience, LangStatsResponse } from '@/lib/types';
import { projectsForSkill, experiencesForSkill, locForSkill } from '@/lib/skillAliases';

// Fiche détail d'un skill, ouverte au clic sur une pastille.
//
// QUOI : affiche les stats d'un skill — LOC (si langage), projets qui l'utilisent (+ noms/liens),
// expériences où il apparaît. ADAPTATIF : le bloc LOC n'apparaît que pour les skills-langages
// (Go, TS, Python…) ; pour React/Docker/AWS, on montre projets + XP seuls. "En fonction de la
// situation" = on n'invente pas une stat absente, on masque la section.
// POURQUOI nouveau composant (vs DetailModal) : DetailModal est taillé projet/expérience (descriptions
// fonctionnelle/technique, carrousel images) — forme incompatible avec une fiche skill. Surgical :
// on ne tord pas l'existant.
// COMMENT : toute la donnée est dérivée côté client (skillAliases) à partir des props déjà sur la page
// (projets/expériences) + le map LOC fetché server-side. Zéro appel réseau ici.

interface SkillDetailModalProps {
  isOpen: boolean;
  onClose: () => void;
  skill: Skill | null;
  projects: Project[];
  experiences: Experience[];
  langStats?: LangStatsResponse | null;
}

const backdropVariants = {
  hidden: { opacity: 0 },
  visible: { opacity: 1 },
};

const modalVariants = {
  hidden: { opacity: 0, scale: 0.95, y: 20 },
  visible: { opacity: 1, scale: 1, y: 0, transition: { type: 'spring', damping: 25, stiffness: 300 } },
  exit: { opacity: 0, scale: 0.95, y: 20, transition: { duration: 0.2 } },
};

// Petite carte stat (LOC, nb projets, nb expériences) — chiffre + libellé. Chiffre en ivoire pour
// les trois : ce sont trois mesures de même rang, une couleur par case n'y ajoutait rien.
function StatBox({ value, label }: { value: string; label: string }) {
  return (
    <div className="rounded-lg border border-border bg-background p-4 text-center">
      <div className="text-2xl font-semibold tabular-nums text-foreground">{value}</div>
      <div className="mt-1 text-xs text-muted-foreground">{label}</div>
    </div>
  );
}

export default function SkillDetailModal({
  isOpen,
  onClose,
  skill,
  projects,
  experiences,
  langStats,
}: SkillDetailModalProps) {
  const t = useTranslations('cv.skillDetail');
  const locale = useLocale();
  const closeButtonRef = useRef<HTMLButtonElement>(null);

  // Escape pour fermer + lock scroll body, comme DetailModal.
  const handleKeyDown = useCallback(
    (event: KeyboardEvent) => {
      if (event.key === 'Escape') onClose();
    },
    [onClose]
  );

  useEffect(() => {
    if (!isOpen) return;
    const previouslyFocused = document.activeElement as HTMLElement;
    closeButtonRef.current?.focus();
    document.addEventListener('keydown', handleKeyDown);
    document.body.style.overflow = 'hidden';
    return () => {
      document.removeEventListener('keydown', handleKeyDown);
      document.body.style.overflow = '';
      previouslyFocused?.focus();
    };
  }, [isOpen, handleKeyDown]);

  // Rien à afficher tant qu'aucun skill n'est sélectionné.
  if (!skill) return null;

  // Données dérivées (pures) — recalculées à chaque rendu, négligeable sur ces volumes.
  const loc = locForSkill(skill, langStats);
  const skillProjects = projectsForSkill(skill, projects);
  const skillExperiences = experiencesForSkill(skill, experiences);
  const hasAnything = !!loc || skillProjects.length > 0 || skillExperiences.length > 0;

  const handleBackdropClick = (event: React.MouseEvent) => {
    if (event.target === event.currentTarget) onClose();
  };

  // Premier lien externe d'un projet (github prioritaire, sinon demo) — pour rendre le nom cliquable.
  const projectLink = (p: Project): { url: string; kind: 'github' | 'demo' } | null => {
    if (p.githubUrl) return { url: p.githubUrl, kind: 'github' };
    if (p.demoUrl) return { url: p.demoUrl, kind: 'demo' };
    return null;
  };

  return (
    <AnimatePresence>
      {isOpen && (
        <motion.div
          className="fixed inset-0 z-50 flex items-center justify-center p-4 md:p-6"
          variants={backdropVariants}
          initial="hidden"
          animate="visible"
          exit="hidden"
          onClick={handleBackdropClick}
          aria-modal="true"
          role="dialog"
          aria-labelledby="skill-modal-title"
          data-testid="skill-detail-modal"
        >
          {/* Voile : le fond de page assombri, comme DetailModal. */}
          <div className="absolute inset-0 bg-background/80 backdrop-blur-sm" />

          <motion.div
            className={cn(
              'relative w-full max-h-[90vh] overflow-y-auto rounded-xl',
              'border border-border bg-card text-card-foreground shadow-2xl',
              'md:max-w-2xl'
            )}
            variants={modalVariants}
            initial="hidden"
            animate="visible"
            exit="exit"
          >
            {/* Header : nom + catégorie + niveau + années */}
            <div className="sticky top-0 z-10 flex items-start justify-between gap-4 border-b border-border bg-card/95 p-4 backdrop-blur-sm md:p-6">
              <div className="flex-1">
                <h2
                  id="skill-modal-title"
                  className="text-2xl font-medium text-foreground md:text-3xl"
                >
                  {skill.name}
                </h2>
                <p className="mt-1 text-sm font-medium capitalize text-foreground/80">
                  {skill.category} · {t(`levels.${skill.level}`)}
                </p>
                <p className="mt-0.5 text-sm text-muted-foreground">
                  {t('years', { count: skill.yearsExperience })}
                </p>
              </div>
              <button
                ref={closeButtonRef}
                onClick={onClose}
                className={cn(
                  'flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-full',
                  'text-muted-foreground transition-colors hover:bg-secondary hover:text-foreground',
                  'focus:outline-none focus-visible:ring-2 focus-visible:ring-ring'
                )}
                aria-label={t('close')}
              >
                <X className="h-5 w-5" />
              </button>
            </div>

            <div className="space-y-6 p-4 md:p-6">
              {/* Bloc stats chiffrées — LOC seulement si le skill est un langage */}
              <div className={cn('grid gap-3', loc ? 'grid-cols-3' : 'grid-cols-2')}>
                {loc && (
                  <StatBox
                    value={`~${loc.loc.toLocaleString(locale)}`}
                    label={t('locLabel')}
                  />
                )}
                <StatBox
                  value={skillProjects.length.toLocaleString(locale)}
                  label={t('projectsLabel')}
                />
                <StatBox
                  value={skillExperiences.length.toLocaleString(locale)}
                  label={t('experiencesLabel')}
                />
              </div>

              {/* Liste des projets utilisant le skill */}
              {skillProjects.length > 0 && (
                <section>
                  <div className="mb-3 flex items-center gap-2">
                    <Rocket className="h-4 w-4 text-muted-foreground" aria-hidden="true" />
                    <h3 className="text-xl font-medium text-foreground">
                      {t('projectsTitle')}
                    </h3>
                  </div>
                  <ul className="space-y-2" data-testid="skill-detail-projects">
                    {skillProjects.map((p) => {
                      const link = projectLink(p);
                      return (
                        <li
                          key={p.id}
                          className="flex items-center justify-between gap-3 rounded-lg border border-border bg-background px-3 py-2"
                        >
                          <span className="truncate font-medium text-foreground">
                            {p.title}
                          </span>
                          {link && (
                            <a
                              href={link.url}
                              target="_blank"
                              rel="noopener noreferrer"
                              className="flex flex-shrink-0 items-center gap-1 rounded text-sm text-muted-foreground hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                              aria-label={p.title}
                            >
                              {link.kind === 'github' ? (
                                <Github className="h-4 w-4" />
                              ) : (
                                <ExternalLink className="h-4 w-4" />
                              )}
                            </a>
                          )}
                        </li>
                      );
                    })}
                  </ul>
                </section>
              )}

              {/* Liste des expériences où le skill apparaît */}
              {skillExperiences.length > 0 && (
                <section>
                  <div className="mb-3 flex items-center gap-2">
                    <Briefcase className="h-4 w-4 text-muted-foreground" aria-hidden="true" />
                    <h3 className="text-xl font-medium text-foreground">
                      {t('experiencesTitle')}
                    </h3>
                  </div>
                  <ul className="space-y-2">
                    {skillExperiences.map((e) => (
                      <li
                        key={e.id}
                        className="rounded-lg border border-border bg-background px-3 py-2 text-sm"
                      >
                        <span className="font-medium text-foreground">{e.title}</span>
                        <span className="text-muted-foreground"> · {e.company}</span>
                      </li>
                    ))}
                  </ul>
                </section>
              )}

              {/* État vide honnête : aucune stat rattachée à ce skill */}
              {!hasAnything && (
                <div className="flex flex-col items-center gap-2 py-8 text-center text-muted-foreground">
                  <Code2 className="h-8 w-8 opacity-50" />
                  <p>{t('noStats')}</p>
                </div>
              )}
            </div>
          </motion.div>
        </motion.div>
      )}
    </AnimatePresence>
  );
}

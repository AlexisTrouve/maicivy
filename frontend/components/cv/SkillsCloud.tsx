'use client';

import { motion, AnimatePresence } from 'framer-motion';
import { useState, useEffect, useMemo } from 'react';
import { useTranslations } from 'next-intl';
import { Skill, Project, Experience, LangStatsResponse } from '@/lib/types';
import SkillDetailModal from '@/components/cv/SkillDetailModal';

interface SkillsCloudProps {
  skills: Skill[];
  // Optionnels : alimentent la fiche détail ouverte au clic sur une pastille. Absents (ex: tests),
  // la pastille s'ouvre quand même mais la fiche affiche 0 projet / 0 LOC.
  projects?: Project[];
  experiences?: Experience[];
  langStats?: LangStatsResponse | null;
}

// Pastilles et filtres : une seule matière neutre (carte + filet), plus de couleur par catégorie.
// POURQUOI : les six teintes (bleu, violet, vert, orange, rose…) codaient une catégorie qu'aucune
// légende n'expliquait, et le filtre juste au-dessus dit déjà quelle catégorie on regarde. La
// TAILLE reste le seul encodage (niveau × pertinence), comme le dit la légende.
const filterBase =
  'rounded-full border px-4 py-1.5 text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring';
// Filtre actif = « où l'on est » : filet d'or (doctrine), texte ivoire. Inactif : gris, filet neutre.
const filterActive = 'border-gold/70 bg-secondary text-foreground';
const filterIdle = 'border-border text-muted-foreground hover:border-muted-foreground/60 hover:text-foreground';

export default function SkillsCloud({ skills, projects, experiences, langStats }: SkillsCloudProps) {
  const [selectedCategory, setSelectedCategory] = useState<string | null>(null);
  // Skill dont la fiche détail est ouverte (null = fermée).
  const [selectedSkill, setSelectedSkill] = useState<Skill | null>(null);
  const t = useTranslations('cv.skills');

  // Get categories - memoized for performance
  const categories = useMemo(
    () => Array.from(new Set(skills.map((s) => s.category))),
    [skills]
  );

  // Reset category filter when skills change (theme switch)
  // Also validate that selected category still exists in new skills
  useEffect(() => {
    if (selectedCategory && !categories.includes(selectedCategory)) {
      setSelectedCategory(null);
    }
  }, [categories, selectedCategory]);

  // Convert skill level string to numeric value for sizing
  const getLevelValue = (level: string): number => {
    switch (level) {
      case 'expert': return 4;
      case 'advanced': return 3;
      case 'intermediate': return 2;
      case 'beginner': return 1;
      default: return 1;
    }
  };

  // Calculate font size based on level and score
  const getFontSize = (skill: Skill) => {
    const baseSize = 14;
    const levelValue = getLevelValue(skill.level);
    const levelMultiplier = levelValue * 2;
    const scoreMultiplier = skill.score ? skill.score * 4 : 0;
    return baseSize + levelMultiplier + scoreMultiplier;
  };

  // Filter skills by category - memoized
  const filteredSkills = useMemo(
    () => selectedCategory
      ? skills.filter((s) => s.category === selectedCategory)
      : skills,
    [skills, selectedCategory]
  );

  return (
    <div className="space-y-6">
      {/* Category Filter */}
      <div className="flex flex-wrap gap-2" role="group" aria-label={t('filterLabel')}>
        <button
          type="button"
          onClick={() => setSelectedCategory(null)}
          aria-pressed={selectedCategory === null}
          className={`${filterBase} ${selectedCategory === null ? filterActive : filterIdle}`}
        >
          {t('all')}
        </button>
        {categories.map((category) => (
          <button
            key={category}
            type="button"
            onClick={() => setSelectedCategory(category)}
            aria-pressed={selectedCategory === category}
            className={`${filterBase} capitalize ${selectedCategory === category ? filterActive : filterIdle}`}
          >
            {category}
          </button>
        ))}
      </div>

      {/* Skills Cloud */}
      <motion.div
        className="flex min-h-[300px] flex-wrap items-center justify-center gap-3 rounded-xl border border-border bg-card p-6 md:p-10"
        layout
      >
        <AnimatePresence mode="popLayout">
          {filteredSkills.map((skill) => (
            <motion.button
              key={skill.id}
              type="button"
              layout
              initial={{ opacity: 0, scale: 0 }}
              animate={{ opacity: 1, scale: 1 }}
              exit={{ opacity: 0, scale: 0 }}
              transition={{ duration: 0.3 }}
              // Pastille = bouton : clic ouvre la fiche détail. <button> nous donne clavier (Enter/Space)
              // et focus gratuitement, sans gérer role/tabIndex/onKeyDown à la main.
              onClick={() => setSelectedSkill(skill)}
              data-testid={`skill-badge-${skill.id}`}
              className="cursor-pointer rounded-full border border-border bg-background px-4 py-1.5 font-medium text-foreground transition-colors hover:border-muted-foreground/60 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
              style={{
                fontSize: `${getFontSize(skill)}px`,
              }}
              title={t('badgeTitle', { name: skill.name })}
              aria-label={t('badgeTitle', { name: skill.name })}
            >
              {skill.name}
            </motion.button>
          ))}
        </AnimatePresence>
      </motion.div>

      {/* Legend */}
      <p className="text-center text-sm text-muted-foreground">{t('legend')}</p>

      {/* Fiche détail — ouverte au clic sur une pastille */}
      <SkillDetailModal
        isOpen={selectedSkill !== null}
        onClose={() => setSelectedSkill(null)}
        skill={selectedSkill}
        projects={projects ?? []}
        experiences={experiences ?? []}
        langStats={langStats}
      />
    </div>
  );
}

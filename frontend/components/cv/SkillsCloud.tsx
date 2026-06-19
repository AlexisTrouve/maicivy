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

const categoryColors: Record<string, string> = {
  backend: 'bg-blue-100 dark:bg-blue-900 text-blue-800 dark:text-blue-200 border-blue-300 dark:border-blue-700',
  frontend: 'bg-purple-100 dark:bg-purple-900 text-purple-800 dark:text-purple-200 border-purple-300 dark:border-purple-700',
  devops: 'bg-green-100 dark:bg-green-900 text-green-800 dark:text-green-200 border-green-300 dark:border-green-700',
  database: 'bg-orange-100 dark:bg-orange-900 text-orange-800 dark:text-orange-200 border-orange-300 dark:border-orange-700',
  tools: 'bg-gray-100 dark:bg-gray-700 text-gray-800 dark:text-gray-200 border-gray-300 dark:border-gray-600',
  other: 'bg-pink-100 dark:bg-pink-900 text-pink-800 dark:text-pink-200 border-pink-300 dark:border-pink-700',
};

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
      <div className="flex flex-wrap gap-2 justify-center">
        <button
          onClick={() => setSelectedCategory(null)}
          className={`px-4 py-2 rounded-full font-medium transition-all ${
            selectedCategory === null
              ? 'bg-blue-600 text-white shadow-lg scale-105'
              : 'bg-gray-200 dark:bg-gray-700 text-gray-700 dark:text-gray-300 hover:bg-gray-300 dark:hover:bg-gray-600'
          }`}
        >
          {t('all')}
        </button>
        {categories.map((category) => (
          <button
            key={category}
            onClick={() => setSelectedCategory(category)}
            className={`px-4 py-2 rounded-full font-medium transition-all capitalize ${
              selectedCategory === category
                ? 'bg-blue-600 text-white shadow-lg scale-105'
                : 'bg-gray-200 dark:bg-gray-700 text-gray-700 dark:text-gray-300 hover:bg-gray-300 dark:hover:bg-gray-600'
            }`}
          >
            {category}
          </button>
        ))}
      </div>

      {/* Skills Cloud */}
      <motion.div
        className="flex flex-wrap gap-3 justify-center items-center p-8 bg-gray-50 dark:bg-gray-900 rounded-lg min-h-[300px]"
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
              whileHover={{ scale: 1.15, rotate: 2 }}
              transition={{ duration: 0.3 }}
              // Pastille = bouton : clic ouvre la fiche détail. <button> nous donne clavier (Enter/Space)
              // et focus gratuitement, sans gérer role/tabIndex/onKeyDown à la main.
              onClick={() => setSelectedSkill(skill)}
              data-testid={`skill-badge-${skill.id}`}
              className={`px-4 py-2 rounded-full font-semibold border-2 cursor-pointer focus:outline-none focus:ring-2 focus:ring-offset-1 focus:ring-purple-500 ${
                categoryColors[skill.category] || categoryColors.other
              }`}
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
      <div className="text-center text-sm text-gray-600 dark:text-gray-400">
        <p>{t('legend')}</p>
      </div>

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

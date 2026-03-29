'use client';

import { useState } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import ExperienceTimeline from '@/components/cv/ExperienceTimeline';
import SkillsCloud from '@/components/cv/SkillsCloud';
import ProjectsGrid from '@/components/cv/ProjectsGrid';
import GitStatsPanel from '@/components/cv/GitStatsPanel';
import { Experience, Skill, Project } from '@/lib/types';

interface CVTabsProps {
  experiences: Experience[];
  skills: Skill[];
  projects: Project[];
  labels: {
    experiences: string;
    skills: string;
    projects: string;
  };
}

const tabs = [
  { id: 'experiences', icon: '💼', color: 'blue' },
  { id: 'skills', icon: '🎯', color: 'purple' },
  { id: 'projects', icon: '🚀', color: 'green' },
  { id: 'gitstats', icon: '📊', color: 'orange' },
] as const;

type TabId = (typeof tabs)[number]['id'];

// Couleurs Tailwind par tab
const colorMap: Record<string, { active: string; hover: string; border: string }> = {
  blue:   { active: 'bg-blue-600 text-white', hover: 'hover:bg-blue-50 dark:hover:bg-blue-900/20', border: 'border-blue-600' },
  purple: { active: 'bg-purple-600 text-white', hover: 'hover:bg-purple-50 dark:hover:bg-purple-900/20', border: 'border-purple-600' },
  green:  { active: 'bg-green-600 text-white', hover: 'hover:bg-green-50 dark:hover:bg-green-900/20', border: 'border-green-600' },
  orange: { active: 'bg-orange-500 text-white', hover: 'hover:bg-orange-50 dark:hover:bg-orange-900/20', border: 'border-orange-500' },
};

export default function CVTabs({ experiences, skills, projects, labels }: CVTabsProps) {
  const [activeTab, setActiveTab] = useState<TabId>('experiences');

  const getLabel = (id: TabId) => {
    if (id === 'gitstats') return 'Git Stats';
    return labels[id as keyof typeof labels];
  };

  return (
    <div>
      {/* Tab bar */}
      <div className="flex flex-wrap gap-2 mb-8 justify-center">
        {tabs.map((tab) => {
          const isActive = activeTab === tab.id;
          const colors = colorMap[tab.color];
          return (
            <button
              key={tab.id}
              onClick={() => setActiveTab(tab.id)}
              className={`
                flex items-center gap-2 px-5 py-2.5 rounded-full text-sm font-medium
                transition-all duration-200 border-2
                ${isActive
                  ? `${colors.active} ${colors.border} shadow-md`
                  : `bg-white dark:bg-gray-800 text-gray-700 dark:text-gray-300 border-gray-200 dark:border-gray-600 ${colors.hover}`
                }
              `}
            >
              <span>{tab.icon}</span>
              {getLabel(tab.id)}
            </button>
          );
        })}
      </div>

      {/* Tab content */}
      <AnimatePresence mode="wait">
        <motion.div
          key={activeTab}
          initial={{ opacity: 0, y: 12 }}
          animate={{ opacity: 1, y: 0 }}
          exit={{ opacity: 0, y: -12 }}
          transition={{ duration: 0.2 }}
        >
          {activeTab === 'experiences' && <ExperienceTimeline experiences={experiences} />}
          {activeTab === 'skills' && <SkillsCloud skills={skills} />}
          {activeTab === 'projects' && <ProjectsGrid projects={projects} />}
          {activeTab === 'gitstats' && <GitStatsPanel />}
        </motion.div>
      </AnimatePresence>
    </div>
  );
}

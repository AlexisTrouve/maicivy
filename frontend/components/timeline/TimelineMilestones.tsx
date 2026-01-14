'use client';

import React from 'react';
import { motion } from 'framer-motion';
import { TimelineMilestone } from '@/lib/types';
import { Award } from 'lucide-react';

interface TimelineMilestonesProps {
  milestones: TimelineMilestone[];
}

const TimelineMilestones: React.FC<TimelineMilestonesProps> = ({
  milestones,
}) => {
  if (milestones.length === 0) {
    return null;
  }

  return (
    <div className="mb-8">
      <div className="flex items-center gap-2 mb-4">
        <Award className="w-5 h-5 text-yellow-600 dark:text-yellow-400" />
        <h3 className="text-lg font-semibold text-gray-900 dark:text-white">
          Jalons Importants
        </h3>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {milestones.map((milestone, index) => (
          <MilestoneCard key={milestone.id} milestone={milestone} index={index} />
        ))}
      </div>
    </div>
  );
};

interface MilestoneCardProps {
  milestone: TimelineMilestone;
  index: number;
}

const MilestoneCard: React.FC<MilestoneCardProps> = ({ milestone, index }) => {
  // Couleur selon le type
  const getTypeColor = () => {
    switch (milestone.type) {
      case 'achievement':
        return {
          bg: 'bg-yellow-50 dark:bg-yellow-900/20',
          border: 'border-yellow-500',
          text: 'text-yellow-700 dark:text-yellow-400',
          icon: 'bg-yellow-500',
        };
      case 'career':
        return {
          bg: 'bg-blue-50 dark:bg-blue-900/20',
          border: 'border-blue-500',
          text: 'text-blue-700 dark:text-blue-400',
          icon: 'bg-blue-500',
        };
      case 'education':
        return {
          bg: 'bg-green-50 dark:bg-green-900/20',
          border: 'border-green-500',
          text: 'text-green-700 dark:text-green-400',
          icon: 'bg-green-500',
        };
      case 'project':
        return {
          bg: 'bg-purple-50 dark:bg-purple-900/20',
          border: 'border-purple-500',
          text: 'text-purple-700 dark:text-purple-400',
          icon: 'bg-purple-500',
        };
      default:
        return {
          bg: 'bg-gray-50 dark:bg-gray-900/20',
          border: 'border-gray-500',
          text: 'text-gray-700 dark:text-gray-400',
          icon: 'bg-gray-500',
        };
    }
  };

  const colors = getTypeColor();

  // Format date
  const formatDate = (dateString: string) => {
    const date = new Date(dateString);
    return date.toLocaleDateString('fr-FR', {
      day: 'numeric',
      month: 'long',
      year: 'numeric',
    });
  };

  return (
    <motion.div
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.5, delay: index * 0.1 }}
      className={`relative p-4 rounded-lg border-l-4 ${colors.bg} ${colors.border} hover:shadow-md transition-shadow`}
    >
      {/* Icône avec animation pulse */}
      <div className="flex items-start gap-3">
        <motion.div
          animate={{
            scale: [1, 1.1, 1],
          }}
          transition={{
            duration: 2,
            repeat: Infinity,
            ease: 'easeInOut',
          }}
          className={`flex items-center justify-center w-10 h-10 ${colors.icon} rounded-full flex-shrink-0 text-white text-xl`}
        >
          {milestone.icon}
        </motion.div>

        <div className="flex-1">
          <h4 className={`font-semibold ${colors.text} mb-1`}>
            {milestone.title}
          </h4>
          <p className="text-sm text-gray-600 dark:text-gray-400 mb-2">
            {milestone.description}
          </p>
          <p className="text-xs text-gray-500 dark:text-gray-500">
            {formatDate(milestone.date)}
          </p>
        </div>
      </div>

      {/* Badge type */}
      <div className="absolute top-2 right-2">
        <span className="text-xs px-2 py-1 bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-full">
          {milestone.type}
        </span>
      </div>
    </motion.div>
  );
};

export default TimelineMilestones;

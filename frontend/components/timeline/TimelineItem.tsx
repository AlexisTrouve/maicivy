'use client';

import React from 'react';
import { motion } from 'framer-motion';
import { useInView } from 'react-intersection-observer';
import { TimelineEvent } from '@/lib/types';
import { Calendar, Briefcase, Code, Trophy, Clock } from 'lucide-react';

interface TimelineItemProps {
  event: TimelineEvent;
  index: number;
  isSelected: boolean;
  onSelect: () => void;
}

// Variants d'animation
const itemVariants = {
  hidden: { opacity: 0, y: 50 },
  visible: {
    opacity: 1,
    y: 0,
    transition: {
      duration: 0.6,
      ease: 'easeOut',
    },
  },
};

const TimelineItem: React.FC<TimelineItemProps> = ({
  event,
  index,
  isSelected,
  onSelect,
}) => {
  const { ref, inView } = useInView({
    threshold: 0.3,
    triggerOnce: true,
  });

  // Alternance gauche/droite sur desktop (pair/impair)
  const isEven = index % 2 === 0;

  // Icône selon le type
  const getIcon = () => {
    switch (event.type) {
      case 'experience':
        return <Briefcase className="w-5 h-5" />;
      case 'project':
        return <Code className="w-5 h-5" />;
      default:
        return <Trophy className="w-5 h-5" />;
    }
  };

  // Couleur selon le type
  const getColors = () => {
    switch (event.type) {
      case 'experience':
        return {
          bg: 'bg-blue-50 dark:bg-blue-900/20',
          border: 'border-l-4 border-blue-500',
          dot: 'bg-blue-500',
          text: 'text-blue-600 dark:text-blue-400',
        };
      case 'project':
        return {
          bg: 'bg-purple-50 dark:bg-purple-900/20',
          border: 'border-l-4 border-purple-500',
          dot: 'bg-purple-500',
          text: 'text-purple-600 dark:text-purple-400',
        };
      default:
        return {
          bg: 'bg-gray-50 dark:bg-gray-900/20',
          border: 'border-l-4 border-gray-500',
          dot: 'bg-gray-500',
          text: 'text-gray-600 dark:text-gray-400',
        };
    }
  };

  const colors = getColors();

  // Format date
  const formatDate = (dateString: string) => {
    const date = new Date(dateString);
    return date.toLocaleDateString('fr-FR', {
      month: 'short',
      year: 'numeric',
    });
  };

  return (
    <motion.div
      ref={ref}
      variants={itemVariants}
      initial="hidden"
      animate={inView ? 'visible' : 'hidden'}
      className={`relative flex flex-col md:flex-row ${
        isEven ? 'md:flex-row' : 'md:flex-row-reverse'
      }`}
    >
      {/* Contenu principal */}
      <div
        className={`w-full md:w-1/2 ${
          isEven ? 'md:pr-12 md:text-right' : 'md:pl-12 md:text-left'
        }`}
      >
        <motion.div
          onClick={onSelect}
          whileHover={{ scale: 1.02 }}
          whileTap={{ scale: 0.98 }}
          className={`p-6 rounded-lg cursor-pointer transition-all ${
            colors.bg
          } ${colors.border} ${
            isSelected ? 'ring-2 ring-blue-500 shadow-lg' : 'hover:shadow-md'
          }`}
        >
          {/* Header */}
          <div
            className={`flex items-start justify-between mb-3 ${
              isEven ? 'md:flex-row-reverse' : ''
            }`}
          >
            <div className={isEven ? 'md:text-right' : ''}>
              <h3 className="text-lg md:text-xl font-bold text-gray-900 dark:text-white mb-1">
                {event.title}
              </h3>
              <p className="text-sm md:text-base text-gray-600 dark:text-gray-300">
                {event.subtitle}
              </p>
            </div>

            <span
              className={`flex items-center justify-center w-10 h-10 rounded-full ${colors.bg} ${colors.text} flex-shrink-0`}
            >
              {getIcon()}
            </span>
          </div>

          {/* Dates */}
          <div
            className={`flex items-center gap-2 text-xs md:text-sm text-gray-500 dark:text-gray-400 mb-3 ${
              isEven ? 'md:justify-end' : ''
            }`}
          >
            <Calendar className="w-4 h-4" />
            <span>
              {formatDate(event.startDate)}
              {event.endDate ? (
                <> → {formatDate(event.endDate)}</>
              ) : (
                <span className="ml-2 px-2 py-0.5 bg-green-100 text-green-700 dark:bg-green-900 dark:text-green-300 rounded-full text-xs font-medium">
                  En cours
                </span>
              )}
            </span>
          </div>

          {/* Durée */}
          {event.duration && (
            <div
              className={`flex items-center gap-2 text-xs text-gray-500 dark:text-gray-400 mb-3 ${
                isEven ? 'md:justify-end' : ''
              }`}
            >
              <Clock className="w-4 h-4" />
              <span>{event.duration}</span>
            </div>
          )}

          {/* Tags */}
          <div
            className={`flex flex-wrap gap-2 ${
              isEven ? 'md:justify-end' : ''
            }`}
          >
            {event.tags.slice(0, 5).map((tag) => (
              <span
                key={tag}
                className="inline-block px-2 py-1 text-xs bg-white dark:bg-gray-800 text-gray-700 dark:text-gray-300 border border-gray-200 dark:border-gray-700 rounded-md"
              >
                {tag}
              </span>
            ))}
            {event.tags.length > 5 && (
              <span className="inline-block px-2 py-1 text-xs text-gray-500">
                +{event.tags.length - 5}
              </span>
            )}
          </div>

          {/* Badge catégorie */}
          <div className={`mt-3 ${isEven ? 'md:text-right' : ''}`}>
            <span className="inline-block px-3 py-1 text-xs font-medium bg-gray-200 dark:bg-gray-700 text-gray-700 dark:text-gray-300 rounded-full">
              {event.category}
            </span>
          </div>
        </motion.div>
      </div>

      {/* Point central (desktop uniquement) */}
      <motion.div
        animate={isSelected ? { scale: 1.3 } : { scale: 1 }}
        transition={{ duration: 0.3 }}
        className="hidden md:block absolute left-1/2 top-8 transform -translate-x-1/2 -translate-y-1/2 z-10"
      >
        <div
          className={`w-5 h-5 ${colors.dot} rounded-full border-4 border-white dark:border-gray-900 shadow-lg`}
        />
        {event.isCurrent && (
          <motion.div
            className={`absolute inset-0 ${colors.dot} rounded-full`}
            animate={{
              scale: [1, 1.5, 1],
              opacity: [0.7, 0, 0.7],
            }}
            transition={{
              duration: 2,
              repeat: Infinity,
              ease: 'easeInOut',
            }}
          />
        )}
      </motion.div>

      {/* Spacer pour équilibrer la grille (desktop) */}
      <div className="hidden md:block w-1/2" />

      {/* Année (mobile uniquement) */}
      <div className="md:hidden mt-2 text-center">
        <span className="text-sm font-semibold text-gray-500 dark:text-gray-400">
          {new Date(event.startDate).getFullYear()}
        </span>
      </div>
    </motion.div>
  );
};

export default TimelineItem;

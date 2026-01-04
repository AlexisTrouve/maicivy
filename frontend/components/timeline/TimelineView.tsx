'use client';

import React, { useState } from 'react';
import { motion } from 'framer-motion';
import { TimelineEvent, TimelineMilestone } from '@/lib/types';
import TimelineItem from './TimelineItem';
import TimelineFilters from './TimelineFilters';
import TimelineMilestones from './TimelineMilestones';
import TimelineNavigation from './TimelineNavigation';
import TimelineModal from './TimelineModal';

interface TimelineViewProps {
  events: TimelineEvent[];
  categories: string[];
  milestones?: TimelineMilestone[];
}

// Framer Motion variants pour l'animation stagger
const containerVariants = {
  hidden: { opacity: 0 },
  visible: {
    opacity: 1,
    transition: {
      staggerChildren: 0.1,
      delayChildren: 0.2,
    },
  },
};

const TimelineView: React.FC<TimelineViewProps> = ({
  events: initialEvents,
  categories,
  milestones = [],
}) => {
  const [selectedCategory, setSelectedCategory] = useState<string>('');
  const [selectedType, setSelectedType] = useState<string>('all');
  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  const [selectedPeriod, setSelectedPeriod] = useState<{
    from: string;
    to: string;
  } | null>(null);
  const [selectedEvent, setSelectedEvent] = useState<TimelineEvent | null>(
    null
  );
  const [filteredEvents, setFilteredEvents] =
    useState<TimelineEvent[]>(initialEvents);

  // Gestionnaire de filtres
  const handleFiltersChange = (filters: {
    category: string;
    type: string;
    period: { from: string; to: string } | null;
  }) => {
    setSelectedCategory(filters.category);
    setSelectedType(filters.type);
    setSelectedPeriod(filters.period);

    // Appliquer les filtres
    let filtered = initialEvents;

    // Filtrer par catégorie
    if (filters.category) {
      filtered = filtered.filter((e) => e.category === filters.category);
    }

    // Filtrer par type
    if (filters.type !== 'all') {
      filtered = filtered.filter((e) => e.type === filters.type);
    }

    // Filtrer par période
    if (filters.period) {
      const fromDate = new Date(filters.period.from);
      const toDate = new Date(filters.period.to);

      filtered = filtered.filter((e) => {
        const eventDate = new Date(e.startDate);
        return eventDate >= fromDate && eventDate <= toDate;
      });
    }

    setFilteredEvents(filtered);
  };

  // Réinitialiser les filtres
  const handleReset = () => {
    setSelectedCategory('');
    setSelectedType('all');
    setSelectedPeriod(null);
    setFilteredEvents(initialEvents);
  };

  // Extraire les années uniques pour la navigation
  const years = Array.from(
    new Set(
      filteredEvents.map((e) => new Date(e.startDate).getFullYear())
    )
  ).sort((a, b) => b - a);

  return (
    <div className="timeline-container relative">
      {/* Navigation années */}
      <TimelineNavigation years={years} />

      {/* Filtres */}
      <TimelineFilters
        categories={categories}
        selectedCategory={selectedCategory}
        selectedType={selectedType}
        onFiltersChange={handleFiltersChange}
        onReset={handleReset}
      />

      {/* Milestones */}
      {milestones.length > 0 && <TimelineMilestones milestones={milestones} />}

      {/* Timeline principale */}
      <motion.div
        variants={containerVariants}
        initial="hidden"
        animate="visible"
        className="relative mt-12"
      >
        {/* Ligne verticale centrale */}
        <div className="absolute left-1/2 transform -translate-x-1/2 w-1 h-full bg-gradient-to-b from-blue-500 via-purple-500 to-pink-500 hidden md:block" />

        {/* Ligne horizontale mobile */}
        <div className="absolute top-8 left-0 w-full h-1 bg-gradient-to-r from-blue-500 via-purple-500 to-pink-500 md:hidden" />

        {/* Événements */}
        <div className="space-y-16 md:space-y-12">
          {filteredEvents.length === 0 ? (
            <div className="text-center py-12">
              <p className="text-gray-500 text-lg">
                Aucun événement trouvé pour les filtres sélectionnés.
              </p>
              <button
                onClick={handleReset}
                className="mt-4 px-6 py-2 bg-blue-500 text-white rounded-lg hover:bg-blue-600 transition-colors"
              >
                Réinitialiser les filtres
              </button>
            </div>
          ) : (
            filteredEvents.map((event, index) => (
              <TimelineItem
                key={event.id}
                event={event}
                index={index}
                isSelected={selectedEvent?.id === event.id}
                onSelect={() => setSelectedEvent(event)}
              />
            ))
          )}
        </div>
      </motion.div>

      {/* Modal détails */}
      {selectedEvent && (
        <TimelineModal
          event={selectedEvent}
          onClose={() => setSelectedEvent(null)}
        />
      )}

      {/* Stats résumé (sticky footer) */}
      <div className="mt-16 p-6 bg-gradient-to-r from-blue-50 to-purple-50 rounded-xl">
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4 text-center">
          <div>
            <p className="text-3xl font-bold text-blue-600">
              {filteredEvents.length}
            </p>
            <p className="text-sm text-gray-600">Événements</p>
          </div>
          <div>
            <p className="text-3xl font-bold text-purple-600">
              {filteredEvents.filter((e) => e.type === 'experience').length}
            </p>
            <p className="text-sm text-gray-600">Expériences</p>
          </div>
          <div>
            <p className="text-3xl font-bold text-pink-600">
              {filteredEvents.filter((e) => e.type === 'project').length}
            </p>
            <p className="text-sm text-gray-600">Projets</p>
          </div>
          <div>
            <p className="text-3xl font-bold text-indigo-600">
              {categories.length}
            </p>
            <p className="text-sm text-gray-600">Catégories</p>
          </div>
        </div>
      </div>
    </div>
  );
};

export default TimelineView;

'use client';

import React, { useState, useEffect } from 'react';
import { motion } from 'framer-motion';
import { ChevronUp } from 'lucide-react';

interface TimelineNavigationProps {
  years: number[];
}

const TimelineNavigation: React.FC<TimelineNavigationProps> = ({ years }) => {
  const [activeYear, setActiveYear] = useState<number | null>(null);
  const [scrollProgress, setScrollProgress] = useState(0);

  // Calculer la progression du scroll
  useEffect(() => {
    const handleScroll = () => {
      const windowHeight = window.innerHeight;
      const documentHeight = document.documentElement.scrollHeight;
      const scrollTop = window.scrollY;

      const totalScroll = documentHeight - windowHeight;
      const progress = (scrollTop / totalScroll) * 100;

      setScrollProgress(Math.min(progress, 100));
    };

    window.addEventListener('scroll', handleScroll);
    return () => window.removeEventListener('scroll', handleScroll);
  }, []);

  // Smooth scroll vers une année
  const scrollToYear = (year: number) => {
    setActiveYear(year);

    // Trouver le premier élément de cette année
    const yearElements = document.querySelectorAll('[data-year]');
    const targetElement = Array.from(yearElements).find(
      (el) => el.getAttribute('data-year') === String(year)
    );

    if (targetElement) {
      targetElement.scrollIntoView({
        behavior: 'smooth',
        block: 'start',
      });
    }
  };

  // Scroll to top
  const scrollToTop = () => {
    window.scrollTo({
      top: 0,
      behavior: 'smooth',
    });
  };

  // Afficher seulement si on a des années
  if (years.length === 0) {
    return null;
  }

  return (
    <>
      {/* Barre de navigation sticky */}
      <div className="sticky top-0 z-40 bg-white dark:bg-gray-900 border-b border-gray-200 dark:border-gray-700 mb-8 shadow-sm">
        <div className="container mx-auto px-4">
          <div className="flex items-center gap-2 py-3 overflow-x-auto scrollbar-hide">
            <span className="text-sm font-medium text-gray-600 dark:text-gray-400 mr-2 whitespace-nowrap">
              Années :
            </span>
            {years.map((year) => (
              <motion.button
                key={year}
                onClick={() => scrollToYear(year)}
                whileHover={{ scale: 1.05 }}
                whileTap={{ scale: 0.95 }}
                className={`px-4 py-1.5 rounded-full text-sm font-medium whitespace-nowrap transition-all ${
                  activeYear === year
                    ? 'bg-blue-500 text-white shadow-md'
                    : 'bg-gray-100 dark:bg-gray-800 text-gray-700 dark:text-gray-300 hover:bg-gray-200 dark:hover:bg-gray-700'
                }`}
              >
                {year}
              </motion.button>
            ))}
          </div>

          {/* Barre de progression */}
          <div className="h-1 bg-gray-200 dark:bg-gray-700">
            <motion.div
              className="h-full bg-gradient-to-r from-blue-500 via-purple-500 to-pink-500"
              style={{ width: `${scrollProgress}%` }}
              transition={{ duration: 0.1 }}
            />
          </div>
        </div>
      </div>

      {/* Bouton scroll to top (apparaît après scroll) */}
      {scrollProgress > 20 && (
        <motion.button
          initial={{ opacity: 0, scale: 0 }}
          animate={{ opacity: 1, scale: 1 }}
          exit={{ opacity: 0, scale: 0 }}
          onClick={scrollToTop}
          className="fixed bottom-8 right-8 z-50 p-4 bg-blue-500 text-white rounded-full shadow-lg hover:bg-blue-600 transition-colors"
          whileHover={{ scale: 1.1 }}
          whileTap={{ scale: 0.9 }}
        >
          <ChevronUp className="w-6 h-6" />
        </motion.button>
      )}

      {/* Indicateur de position actuelle */}
      <div className="fixed bottom-8 left-8 z-40 hidden md:block">
        <div className="bg-white dark:bg-gray-800 rounded-lg shadow-lg p-3 border border-gray-200 dark:border-gray-700">
          <p className="text-xs text-gray-500 dark:text-gray-400 mb-1">
            Progression
          </p>
          <div className="flex items-center gap-2">
            <div className="w-24 h-2 bg-gray-200 dark:bg-gray-700 rounded-full overflow-hidden">
              <motion.div
                className="h-full bg-blue-500"
                style={{ width: `${scrollProgress}%` }}
                transition={{ duration: 0.1 }}
              />
            </div>
            <span className="text-sm font-semibold text-gray-900 dark:text-white">
              {Math.round(scrollProgress)}%
            </span>
          </div>
        </div>
      </div>
    </>
  );
};

export default TimelineNavigation;

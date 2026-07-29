'use client';

import React, { useEffect } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { TimelineEvent } from '@/lib/types';
import { X, Calendar, MapPin, ExternalLink, Github } from 'lucide-react';

interface TimelineModalProps {
  event: TimelineEvent;
  onClose: () => void;
}

const TimelineModal: React.FC<TimelineModalProps> = ({ event, onClose }) => {
  // Fermer avec Escape
  useEffect(() => {
    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        onClose();
      }
    };

    document.addEventListener('keydown', handleEscape);
    // Bloquer le scroll du body
    document.body.style.overflow = 'hidden';

    return () => {
      document.removeEventListener('keydown', handleEscape);
      document.body.style.overflow = 'unset';
    };
  }, [onClose]);

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
    <AnimatePresence>
      <motion.div
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        exit={{ opacity: 0 }}
        onClick={onClose}
        className="fixed inset-0 bg-black/60 backdrop-blur-sm flex items-center justify-center z-50 p-4"
      >
        <motion.div
          initial={{ scale: 0.9, opacity: 0, y: 20 }}
          animate={{ scale: 1, opacity: 1, y: 0 }}
          exit={{ scale: 0.9, opacity: 0, y: 20 }}
          onClick={(e) => e.stopPropagation()}
          className="bg-white dark:bg-gray-900 rounded-2xl shadow-2xl max-w-3xl w-full max-h-[90vh] overflow-y-auto relative"
        >
          {/* Header */}
          <div className="sticky top-0 bg-white dark:bg-gray-900 border-b border-gray-200 dark:border-gray-700 p-6 flex items-start justify-between">
            <div className="flex-1">
              <div className="flex items-center gap-2 mb-2">
                <span
                  className={`px-3 py-1 rounded-full text-xs font-medium ${
                    event.type === 'experience'
                      ? 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400'
                      : 'bg-purple-100 text-purple-700 dark:bg-purple-900/30 dark:text-purple-400'
                  }`}
                >
                  {event.type === 'experience' ? 'Expérience' : 'Projet'}
                </span>
                <span className="px-3 py-1 bg-gray-100 dark:bg-gray-800 text-gray-700 dark:text-gray-300 rounded-full text-xs font-medium">
                  {event.category}
                </span>
                {event.isCurrent && (
                  <span className="px-3 py-1 bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400 rounded-full text-xs font-medium">
                    En cours
                  </span>
                )}
              </div>

              <h2 className="text-2xl md:text-3xl font-bold text-gray-900 dark:text-white mb-1">
                {event.title}
              </h2>
              <p className="text-lg text-gray-600 dark:text-gray-300">
                {event.subtitle}
              </p>
            </div>

            <button
              onClick={onClose}
              className="flex-shrink-0 p-2 rounded-full hover:bg-gray-100 dark:hover:bg-gray-800 transition-colors"
            >
              <X className="w-6 h-6 text-gray-500 dark:text-gray-400" />
            </button>
          </div>

          {/* Body */}
          <div className="p-6 space-y-6">
            {/* Dates et durée */}
            <div className="flex flex-wrap gap-4">
              <div className="flex items-center gap-2 text-gray-600 dark:text-gray-400">
                <Calendar className="w-5 h-5" />
                <span>
                  {formatDate(event.startDate)}
                  {event.endDate ? (
                    <> → {formatDate(event.endDate)}</>
                  ) : (
                    <span className="ml-2 text-green-600 dark:text-green-400 font-medium">
                      En cours
                    </span>
                  )}
                </span>
              </div>
              {event.duration && (
                <div className="flex items-center gap-2 text-gray-600 dark:text-gray-400">
                  <MapPin className="w-5 h-5" />
                  <span>{event.duration}</span>
                </div>
              )}
            </div>

            {/* Image */}
            {event.image && (
              <div className="rounded-lg overflow-hidden">
                <img
                  src={event.image}
                  alt={event.title}
                  className="w-full h-64 object-cover"
                />
              </div>
            )}

            {/* Description */}
            <div>
              <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-3">
                Description
              </h3>
              <div className="bg-gray-50 dark:bg-gray-800 p-4 rounded-lg">
                <p className="text-gray-700 dark:text-gray-300 whitespace-pre-wrap leading-relaxed">
                  {event.content}
                </p>
              </div>
            </div>

            {/* Technologies */}
            {event.tags && event.tags.length > 0 && (
              <div>
                <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-3">
                  Technologies & Compétences
                </h3>
                <div className="flex flex-wrap gap-2">
                  {event.tags.map((tag) => (
                    <motion.span
                      key={tag}
                      whileHover={{ scale: 1.05 }}
                      className="px-4 py-2 bg-blue-500 text-white text-sm font-medium rounded-full shadow-sm"
                    >
                      {tag}
                    </motion.span>
                  ))}
                </div>
              </div>
            )}

            {/* Stats supplémentaires (si projet) */}
            {event.type === 'project' && event.stats && (
              <div>
                <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-3">
                  Statistiques
                </h3>
                <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                  {event.stats.stars !== undefined && (
                    <div className="bg-gray-50 dark:bg-gray-800 p-4 rounded-lg text-center">
                      <p className="text-2xl font-bold text-yellow-600 dark:text-yellow-400">
                        {event.stats.stars}
                      </p>
                      <p className="text-sm text-gray-600 dark:text-gray-400">
                        Stars
                      </p>
                    </div>
                  )}
                  {event.stats.forks !== undefined && (
                    <div className="bg-gray-50 dark:bg-gray-800 p-4 rounded-lg text-center">
                      <p className="text-2xl font-bold text-blue-600 dark:text-blue-400">
                        {event.stats.forks}
                      </p>
                      <p className="text-sm text-gray-600 dark:text-gray-400">
                        Forks
                      </p>
                    </div>
                  )}
                </div>
              </div>
            )}

            {/* Liens */}
            {(event.githubUrl || event.demoUrl) && (
              <div>
                <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-3">
                  Liens
                </h3>
                <div className="flex flex-wrap gap-3">
                  {event.githubUrl && (
                    <a
                      href={event.githubUrl}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="flex items-center gap-2 px-4 py-2 bg-gray-900 dark:bg-gray-700 text-white rounded-lg hover:bg-gray-800 dark:hover:bg-gray-600 transition-colors"
                    >
                      <Github className="w-5 h-5" />
                      Voir sur GitHub
                    </a>
                  )}
                  {event.demoUrl && (
                    <a
                      href={event.demoUrl}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="flex items-center gap-2 px-4 py-2 bg-blue-500 text-white rounded-lg hover:bg-blue-600 transition-colors"
                    >
                      <ExternalLink className="w-5 h-5" />
                      Voir la démo
                    </a>
                  )}
                </div>
              </div>
            )}
          </div>

          {/* Footer */}
          <div className="sticky bottom-0 bg-gray-50 dark:bg-gray-800 border-t border-gray-200 dark:border-gray-700 p-4 flex justify-end">
            <button
              onClick={onClose}
              className="px-6 py-2 bg-gray-200 dark:bg-gray-700 text-gray-700 dark:text-gray-300 rounded-lg hover:bg-gray-300 dark:hover:bg-gray-600 transition-colors font-medium"
            >
              Fermer
            </button>
          </div>
        </motion.div>
      </motion.div>
    </AnimatePresence>
  );
};

export default TimelineModal;

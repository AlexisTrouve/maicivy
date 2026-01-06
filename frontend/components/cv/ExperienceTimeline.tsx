'use client';

import { useState } from 'react';
import { motion } from 'framer-motion';
import { format } from 'date-fns';
import { fr, enUS } from 'date-fns/locale';
import { useLocale, useTranslations } from 'next-intl';
import { Briefcase, Calendar, ArrowRight } from 'lucide-react';
import { Experience, DetailLink } from '@/lib/types';
import DetailModal, { DetailModalData } from './DetailModal';

const localeMap = {
  fr: fr,
  en: enUS
};

interface ExperienceTimelineProps {
  experiences: Experience[];
}

const containerVariants = {
  hidden: { opacity: 0 },
  visible: {
    opacity: 1,
    transition: {
      staggerChildren: 0.2,
    },
  },
};

const itemVariants = {
  hidden: { opacity: 0, x: -50 },
  visible: {
    opacity: 1,
    x: 0,
    transition: {
      duration: 0.6,
      ease: 'easeOut',
    },
  },
};

export default function ExperienceTimeline({ experiences }: ExperienceTimelineProps) {
  const locale = useLocale();
  const t = useTranslations('cv.duration');
  const tExp = useTranslations('cv.experience');
  const dateLocale = localeMap[locale as keyof typeof localeMap];

  const [isModalOpen, setIsModalOpen] = useState(false);
  const [selectedExperience, setSelectedExperience] = useState<DetailModalData | null>(null);

  const formatDate = (dateString: string | undefined | null) => {
    // Vérifier les cas null/undefined/empty
    if (!dateString || dateString === '') return t('unknownDate');

    try {
      // Essayer de parser la date (supporte ISO, SQL, et formats courants)
      const date = new Date(dateString);

      // Vérifier si la date est valide AVANT d'appeler format
      if (!date || isNaN(date.getTime()) || date.getTime() === 0) {
        console.warn('Invalid date:', dateString);
        return t('invalidDate');
      }

      return format(date, 'MMM yyyy', { locale: dateLocale });
    } catch (error) {
      console.error('Error formatting date:', dateString, error);
      return t('invalidDate');
    }
  };

  const calculateDuration = (start: string | undefined | null, end?: string | null) => {
    // Vérifier les cas null/undefined/empty pour start
    if (!start || start === '') {
      return t('unknownDuration');
    }

    try {
      const startDate = new Date(start);
      const endDate = end && end !== '' ? new Date(end) : new Date();

      // Vérifier que les dates sont valides
      if (!startDate || !endDate || isNaN(startDate.getTime()) || isNaN(endDate.getTime())) {
        return t('unknownDuration');
      }

      const months = Math.round(
        (endDate.getTime() - startDate.getTime()) / (1000 * 60 * 60 * 24 * 30)
      );

      // Gérer les durées négatives
      if (months < 0) {
        return t('unknownDuration');
      }

      const years = Math.floor(months / 12);
      const remainingMonths = months % 12;

      if (years > 0 && remainingMonths > 0) {
        const yearText = years === 1 ? t('year') : t('years');
        const monthText = t('months');
        return `${years} ${yearText} ${remainingMonths} ${monthText}`;
      } else if (years > 0) {
        const yearText = years === 1 ? t('year') : t('years');
        return `${years} ${yearText}`;
      } else {
        const monthText = months === 1 ? t('month') : t('months');
        return `${months} ${monthText}`;
      }
    } catch (error) {
      console.error('Error calculating duration:', error);
      return t('unknownDuration');
    }
  };

  // Map backend link icon to frontend link type
  const mapIconToType = (icon: string): DetailLink['type'] => {
    const iconMap: Record<string, DetailLink['type']> = {
      github: 'github',
      globe: 'website',
      linkedin: 'linkedin',
      demo: 'demo',
      link: 'other',
      npm: 'other',
      book: 'other',
      download: 'other',
    };
    return iconMap[icon?.toLowerCase()] || 'other';
  };

  const openExperienceModal = (exp: Experience) => {
    try {
      // Map backend link format to frontend format
      let mappedLinks: DetailLink[] | undefined;
      if (exp.links && Array.isArray(exp.links)) {
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        mappedLinks = exp.links.map((link: any) => ({
          type: mapIconToType(link.icon || link.type),
          url: link.url,
          label: link.name || link.label,
        })).filter((link: DetailLink) => link.url);
      }

      setSelectedExperience({
        title: exp.title,
        subtitle: exp.company,
        period: `${formatDate(exp.startDate)} - ${exp.endDate ? formatDate(exp.endDate) : t('present')}`,
        functionalDescription: exp.functionalDescription || exp.description || '',
        technicalDescription: exp.technicalDescription || '',
        skills: exp.technologies || [],
        images: exp.images || [],
        links: mappedLinks,
      });
      setIsModalOpen(true);
    } catch (error) {
      console.error('Error opening experience modal:', error);
    }
  };

  const closeModal = () => {
    setIsModalOpen(false);
    setSelectedExperience(null);
  };

  return (
    <motion.div
      className="relative"
      variants={containerVariants}
      initial="hidden"
      whileInView="visible"
      viewport={{ once: true, margin: '-100px' }}
    >
      {/* Vertical Line */}
      <div className="absolute left-8 md:left-1/2 top-0 bottom-0 w-0.5 bg-gradient-to-b from-blue-600 via-purple-600 to-pink-600" />

      {/* Timeline Items */}
      <div className="space-y-12">
        {experiences.map((exp, index) => (
          <motion.div
            key={exp.id}
            variants={itemVariants}
            className={`relative flex items-center ${
              index % 2 === 0 ? 'md:flex-row' : 'md:flex-row-reverse'
            }`}
          >
            {/* Timeline Dot */}
            <div className="absolute left-8 md:left-1/2 w-4 h-4 rounded-full bg-blue-600 border-4 border-white dark:border-gray-900 transform -translate-x-1/2 z-10" />

            {/* Content Card */}
            <div
              className={`ml-20 md:ml-0 md:w-5/12 ${
                index % 2 === 0 ? 'md:mr-auto md:pr-12' : 'md:ml-auto md:pl-12'
              }`}
            >
              <motion.div
                className="bg-white dark:bg-gray-800 rounded-lg shadow-lg p-6 border border-gray-200 dark:border-gray-700 hover:shadow-xl transition-all cursor-pointer group"
                whileHover={{ scale: 1.02 }}
                transition={{ duration: 0.2 }}
                onClick={() => openExperienceModal(exp)}
                role="button"
                tabIndex={0}
                onKeyDown={(e) => {
                  if (e.key === 'Enter' || e.key === ' ') {
                    e.preventDefault();
                    openExperienceModal(exp);
                  }
                }}
                aria-label={`${tExp('viewDetails')}: ${exp.title} - ${exp.company}`}
              >
                {/* Header */}
                <div className="flex items-start justify-between mb-3">
                  <div className="flex-1">
                    <h3 className="text-xl font-bold text-gray-900 dark:text-white mb-1">
                      {exp.title}
                    </h3>
                    <div className="flex items-center gap-2 text-blue-600 dark:text-blue-400 font-medium">
                      <Briefcase className="w-4 h-4" />
                      <span>{exp.company}</span>
                    </div>
                  </div>
                  {exp.score && (
                    <div className="ml-4 px-3 py-1 bg-blue-100 dark:bg-blue-900 text-blue-800 dark:text-blue-200 rounded-full text-sm font-semibold">
                      {Math.round(exp.score * 100)}%
                    </div>
                  )}
                </div>

                {/* Date & Duration */}
                <div className="flex items-center gap-2 text-sm text-gray-600 dark:text-gray-400 mb-4">
                  <Calendar className="w-4 h-4" />
                  <span>
                    {formatDate(exp.startDate)} - {exp.endDate ? formatDate(exp.endDate) : t('present')}
                  </span>
                  <span className="text-gray-400">•</span>
                  <span>{calculateDuration(exp.startDate, exp.endDate)}</span>
                </div>

                {/* Short Description / Catchphrase */}
                <p className="text-gray-700 dark:text-gray-300 mb-4 leading-relaxed line-clamp-2">
                  {exp.description}
                </p>

                {/* Technologies - show only first 4 */}
                <div className="flex flex-wrap gap-2 mb-4">
                  {exp.technologies.slice(0, 4).map((tech) => (
                    <span
                      key={tech}
                      className="px-3 py-1 bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300 rounded-full text-sm font-medium"
                    >
                      {tech}
                    </span>
                  ))}
                  {exp.technologies.length > 4 && (
                    <span className="px-3 py-1 bg-gray-100 dark:bg-gray-700 text-gray-500 dark:text-gray-400 rounded-full text-sm font-medium">
                      +{exp.technologies.length - 4}
                    </span>
                  )}
                </div>

                {/* View Details Button */}
                <div className="flex items-center gap-2 text-blue-600 dark:text-blue-400 font-medium group-hover:text-blue-700 dark:group-hover:text-blue-300 transition-colors">
                  <span>{tExp('viewDetails')}</span>
                  <ArrowRight className="w-4 h-4 transform group-hover:translate-x-1 transition-transform" />
                </div>
              </motion.div>
            </div>
          </motion.div>
        ))}
      </div>

      {/* Detail Modal */}
      {selectedExperience && (
        <DetailModal
          isOpen={isModalOpen}
          onClose={closeModal}
          type="experience"
          data={selectedExperience}
        />
      )}
    </motion.div>
  );
}

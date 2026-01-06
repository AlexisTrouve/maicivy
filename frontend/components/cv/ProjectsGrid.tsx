'use client';

import { useState } from 'react';
import { motion } from 'framer-motion';
import { useTranslations } from 'next-intl';
import { Star, Clock } from 'lucide-react';
import { Project, DetailLink } from '@/lib/types';
import DetailModal, { DetailModalData } from './DetailModal';

interface ProjectsGridProps {
  projects: Project[];
}

const containerVariants = {
  hidden: { opacity: 0 },
  visible: {
    opacity: 1,
    transition: {
      staggerChildren: 0.1,
    },
  },
};

const itemVariants = {
  hidden: { opacity: 0, y: 20 },
  visible: {
    opacity: 1,
    y: 0,
    transition: {
      duration: 0.5,
    },
  },
};

export default function ProjectsGrid({ projects }: ProjectsGridProps) {
  const t = useTranslations('cv.projects');
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [selectedProject, setSelectedProject] = useState<DetailModalData | null>(null);

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

  const openProjectModal = (project: Project) => {
    try {
      // Build links array from project URLs
      const links: DetailLink[] = [];
      if (project.githubUrl) {
        links.push({ type: 'github', url: project.githubUrl });
      }
      if (project.demoUrl) {
        links.push({ type: 'demo', url: project.demoUrl });
      }
      // Add any additional links from project.links (map backend format to frontend format)
      if (project.links && Array.isArray(project.links)) {
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        project.links.forEach((link: any) => {
          // Backend format: {name, url, icon} -> Frontend format: {type, url, label}
          if (link.url) {
            links.push({
              type: mapIconToType(link.icon || link.type),
              url: link.url,
              label: link.name || link.label,
            });
          }
        });
      }

      setSelectedProject({
        title: project.title,
        subtitle: project.category || t('defaultCategory'),
        functionalDescription: project.functionalDescription || project.description || '',
        technicalDescription: project.technicalDescription || '',
        skills: project.technologies || [],
        images: project.images || [],
        links: links.length > 0 ? links : undefined,
      });
      setIsModalOpen(true);
    } catch (error) {
      console.error('Error opening project modal:', error);
    }
  };

  const closeModal = () => {
    setIsModalOpen(false);
    setSelectedProject(null);
  };

  return (
    <>
      <motion.div
        className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6"
        variants={containerVariants}
        initial="hidden"
        whileInView="visible"
        viewport={{ once: true }}
      >
        {projects.map((project) => (
          <motion.div
            key={project.id}
            variants={itemVariants}
            whileHover={{ y: -8, scale: 1.02 }}
            onClick={() => openProjectModal(project)}
            className={`relative bg-white dark:bg-gray-800 rounded-lg shadow-lg p-6 border-2 transition-all cursor-pointer ${
              project.featured
                ? 'border-yellow-400 dark:border-yellow-600'
                : 'border-gray-200 dark:border-gray-700 hover:border-blue-400 dark:hover:border-blue-500'
            }`}
            role="button"
            tabIndex={0}
            onKeyDown={(e) => {
              if (e.key === 'Enter' || e.key === ' ') {
                e.preventDefault();
                openProjectModal(project);
              }
            }}
            aria-label={t('openDetails', { title: project.title })}
          >
            {/* Badges Container - Top Right */}
            <div className="absolute top-3 right-3 flex flex-col gap-2 items-end">
              {/* Category Badge */}
              {project.category && (
                <span className="px-2 py-1 bg-blue-100 dark:bg-blue-900/50 text-blue-700 dark:text-blue-300 rounded-full text-xs font-medium">
                  {project.category}
                </span>
              )}
              {/* In Progress Badge */}
              {project.inProgress && (
                <span className="flex items-center gap-1 px-2 py-1 bg-amber-100 dark:bg-amber-900/50 text-amber-700 dark:text-amber-300 rounded-full text-xs font-medium">
                  <Clock className="w-3 h-3" />
                  {t('inProgress')}
                </span>
              )}
            </div>

            {/* Featured Badge */}
            {project.featured && (
              <div className="flex items-center gap-1 text-yellow-600 dark:text-yellow-400 mb-2 font-semibold text-sm">
                <Star className="w-4 h-4 fill-current" />
                <span>{t('featured')}</span>
              </div>
            )}

            {/* Title */}
            <h3 className="text-xl font-bold text-gray-900 dark:text-white mb-2 pr-20">
              {project.title}
            </h3>

            {/* Catchphrase / Short Description */}
            <p className="text-gray-600 dark:text-gray-400 text-sm line-clamp-2 mb-4">
              {project.catchphrase || project.description}
            </p>

            {/* Technologies Preview */}
            <div className="flex flex-wrap gap-1.5">
              {project.technologies.slice(0, 3).map((tech) => (
                <span
                  key={tech}
                  className="px-2 py-0.5 bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-400 rounded text-xs"
                >
                  {tech}
                </span>
              ))}
              {project.technologies.length > 3 && (
                <span className="px-2 py-0.5 text-gray-500 dark:text-gray-400 text-xs">
                  +{project.technologies.length - 3}
                </span>
              )}
            </div>

            {/* Click indicator */}
            <div className="mt-4 text-xs text-gray-400 dark:text-gray-500 text-center">
              {t('clickForDetails')}
            </div>
          </motion.div>
        ))}
      </motion.div>

      {/* Detail Modal */}
      {selectedProject && (
        <DetailModal
          isOpen={isModalOpen}
          onClose={closeModal}
          type="project"
          data={selectedProject}
        />
      )}
    </>
  );
}

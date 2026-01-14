'use client';

import { useEffect, useCallback, useState, useRef } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { useTranslations } from 'next-intl';
import {
  X,
  Github,
  ExternalLink,
  Globe,
  Linkedin,
  Link as LinkIcon,
  ChevronLeft,
  ChevronRight,
  Wrench,
  Code2,
} from 'lucide-react';
import { cn } from '@/lib/utils';
import { DetailLink } from '@/lib/types';

export interface DetailModalData {
  title: string;
  subtitle?: string; // Company name for jobs, category for projects
  period?: string; // Date range for jobs
  images?: string[]; // URLs for images
  functionalDescription: string;
  technicalDescription: string;
  skills: string[];
  links?: DetailLink[];
}

export interface DetailModalProps {
  isOpen: boolean;
  onClose: () => void;
  type: 'project' | 'experience';
  data: DetailModalData;
}

const backdropVariants = {
  hidden: { opacity: 0 },
  visible: { opacity: 1 },
};

const modalVariants = {
  hidden: {
    opacity: 0,
    scale: 0.95,
    y: 20,
  },
  visible: {
    opacity: 1,
    scale: 1,
    y: 0,
    transition: {
      type: 'spring',
      damping: 25,
      stiffness: 300,
    },
  },
  exit: {
    opacity: 0,
    scale: 0.95,
    y: 20,
    transition: {
      duration: 0.2,
    },
  },
};

const linkIcons: Record<DetailLink['type'], React.ComponentType<{ className?: string }>> = {
  github: Github,
  demo: ExternalLink,
  website: Globe,
  linkedin: Linkedin,
  other: LinkIcon,
};

const linkColors: Record<DetailLink['type'], string> = {
  github: 'hover:bg-gray-800 hover:text-white dark:hover:bg-gray-700',
  demo: 'hover:bg-blue-600 hover:text-white',
  website: 'hover:bg-green-600 hover:text-white',
  linkedin: 'hover:bg-blue-700 hover:text-white',
  other: 'hover:bg-purple-600 hover:text-white',
};

export default function DetailModal({ isOpen, onClose, type, data }: DetailModalProps) {
  const t = useTranslations('cv.detailModal');
  const [currentImageIndex, setCurrentImageIndex] = useState(0);
  const modalRef = useRef<HTMLDivElement>(null);
  const closeButtonRef = useRef<HTMLButtonElement>(null);

  // Handle escape key
  const handleKeyDown = useCallback(
    (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        onClose();
      }
    },
    [onClose]
  );

  // Focus trap
  useEffect(() => {
    if (isOpen) {
      // Store the currently focused element
      const previouslyFocused = document.activeElement as HTMLElement;

      // Focus the close button when modal opens
      closeButtonRef.current?.focus();

      // Add escape key listener
      document.addEventListener('keydown', handleKeyDown);

      // Prevent body scroll
      document.body.style.overflow = 'hidden';

      return () => {
        document.removeEventListener('keydown', handleKeyDown);
        document.body.style.overflow = '';
        // Restore focus when modal closes
        previouslyFocused?.focus();
      };
    }
  }, [isOpen, handleKeyDown]);

  // Reset image index when modal opens with new data
  useEffect(() => {
    if (isOpen) {
      setCurrentImageIndex(0);
    }
  }, [isOpen, data.title]);

  // Handle backdrop click
  const handleBackdropClick = (event: React.MouseEvent) => {
    if (event.target === event.currentTarget) {
      onClose();
    }
  };

  // Image carousel navigation
  const nextImage = () => {
    if (data.images && data.images.length > 1) {
      setCurrentImageIndex((prev) => (prev + 1) % data.images!.length);
    }
  };

  const prevImage = () => {
    if (data.images && data.images.length > 1) {
      setCurrentImageIndex((prev) => (prev - 1 + data.images!.length) % data.images!.length);
    }
  };

  // Get default link label
  const getLinkLabel = (link: DetailLink): string => {
    if (link.label) return link.label;
    switch (link.type) {
      case 'github':
        return t('links.github');
      case 'demo':
        return t('links.demo');
      case 'website':
        return t('links.website');
      case 'linkedin':
        return t('links.linkedin');
      default:
        return t('links.other');
    }
  };

  return (
    <AnimatePresence>
      {isOpen && (
        <motion.div
          className="fixed inset-0 z-50 flex items-center justify-center p-4 md:p-6"
          variants={backdropVariants}
          initial="hidden"
          animate="visible"
          exit="hidden"
          onClick={handleBackdropClick}
          aria-modal="true"
          role="dialog"
          aria-labelledby="modal-title"
        >
          {/* Backdrop */}
          <div className="absolute inset-0 bg-black/60 backdrop-blur-sm" />

          {/* Modal */}
          <motion.div
            ref={modalRef}
            className={cn(
              'relative w-full max-h-[90vh] overflow-y-auto rounded-xl',
              'bg-white dark:bg-gray-900',
              'shadow-2xl',
              'md:max-w-2xl lg:max-w-3xl'
            )}
            variants={modalVariants}
            initial="hidden"
            animate="visible"
            exit="exit"
          >
            {/* Header */}
            <div className="sticky top-0 z-10 flex items-start justify-between gap-4 border-b border-gray-200 bg-white/95 p-4 backdrop-blur-sm dark:border-gray-700 dark:bg-gray-900/95 md:p-6">
              <div className="flex-1">
                <h2
                  id="modal-title"
                  className="text-xl font-bold text-gray-900 dark:text-white md:text-2xl"
                >
                  {data.title}
                </h2>
                {data.subtitle && (
                  <p className="mt-1 text-sm font-medium text-blue-600 dark:text-blue-400">
                    {data.subtitle}
                  </p>
                )}
                {data.period && (
                  <p className="mt-0.5 text-sm text-gray-500 dark:text-gray-400">{data.period}</p>
                )}
              </div>
              <button
                ref={closeButtonRef}
                onClick={onClose}
                className={cn(
                  'flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-full',
                  'text-gray-500 transition-colors',
                  'hover:bg-gray-100 hover:text-gray-700',
                  'dark:text-gray-400 dark:hover:bg-gray-800 dark:hover:text-gray-200',
                  'focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2',
                  'dark:focus:ring-offset-gray-900'
                )}
                aria-label={t('close')}
              >
                <X className="h-5 w-5" />
              </button>
            </div>

            {/* Content */}
            <div className="p-4 md:p-6">
              {/* Image Section */}
              {data.images && data.images.length > 0 && (
                <div className="mb-6">
                  <div className="relative aspect-video overflow-hidden rounded-lg bg-gray-100 dark:bg-gray-800">
                    <motion.img
                      key={currentImageIndex}
                      src={data.images[currentImageIndex]}
                      alt={`${data.title} - ${t('image')} ${currentImageIndex + 1}`}
                      className="h-full w-full object-cover"
                      initial={{ opacity: 0 }}
                      animate={{ opacity: 1 }}
                      transition={{ duration: 0.3 }}
                    />

                    {/* Carousel Navigation */}
                    {data.images.length > 1 && (
                      <>
                        <button
                          onClick={prevImage}
                          className={cn(
                            'absolute left-2 top-1/2 -translate-y-1/2',
                            'flex h-10 w-10 items-center justify-center rounded-full',
                            'bg-black/50 text-white backdrop-blur-sm',
                            'transition-all hover:bg-black/70',
                            'focus:outline-none focus:ring-2 focus:ring-white'
                          )}
                          aria-label={t('prevImage')}
                        >
                          <ChevronLeft className="h-6 w-6" />
                        </button>
                        <button
                          onClick={nextImage}
                          className={cn(
                            'absolute right-2 top-1/2 -translate-y-1/2',
                            'flex h-10 w-10 items-center justify-center rounded-full',
                            'bg-black/50 text-white backdrop-blur-sm',
                            'transition-all hover:bg-black/70',
                            'focus:outline-none focus:ring-2 focus:ring-white'
                          )}
                          aria-label={t('nextImage')}
                        >
                          <ChevronRight className="h-6 w-6" />
                        </button>

                        {/* Dots indicator */}
                        <div className="absolute bottom-3 left-1/2 flex -translate-x-1/2 gap-2">
                          {data.images.map((_, index) => (
                            <button
                              key={index}
                              onClick={() => setCurrentImageIndex(index)}
                              className={cn(
                                'h-2 w-2 rounded-full transition-all',
                                index === currentImageIndex
                                  ? 'w-4 bg-white'
                                  : 'bg-white/50 hover:bg-white/75'
                              )}
                              aria-label={`${t('goToImage')} ${index + 1}`}
                            />
                          ))}
                        </div>
                      </>
                    )}
                  </div>
                </div>
              )}

              {/* Functional Description */}
              <section className="mb-6">
                <div className="mb-3 flex items-center gap-2">
                  <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-green-100 text-green-600 dark:bg-green-900/30 dark:text-green-400">
                    <Wrench className="h-4 w-4" />
                  </div>
                  <h3 className="text-lg font-semibold text-gray-900 dark:text-white">
                    {t('functional.title')}
                  </h3>
                </div>
                <p className="whitespace-pre-wrap text-gray-700 dark:text-gray-300 leading-relaxed">
                  {data.functionalDescription}
                </p>
              </section>

              {/* Technical Description */}
              <section className="mb-6">
                <div className="mb-3 flex items-center gap-2">
                  <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-blue-100 text-blue-600 dark:bg-blue-900/30 dark:text-blue-400">
                    <Code2 className="h-4 w-4" />
                  </div>
                  <h3 className="text-lg font-semibold text-gray-900 dark:text-white">
                    {t('technical.title')}
                  </h3>
                </div>
                <p className="whitespace-pre-wrap text-gray-700 dark:text-gray-300 leading-relaxed">
                  {data.technicalDescription}
                </p>
              </section>

              {/* Skills/Technologies */}
              {data.skills.length > 0 && (
                <section className="mb-6">
                  <h3 className="mb-3 text-lg font-semibold text-gray-900 dark:text-white">
                    {type === 'project' ? t('technologies') : t('skills')}
                  </h3>
                  <div className="flex flex-wrap gap-2">
                    {data.skills.map((skill) => (
                      <span
                        key={skill}
                        className={cn(
                          'inline-flex items-center rounded-full px-3 py-1',
                          'text-sm font-medium',
                          'bg-gray-100 text-gray-700',
                          'dark:bg-gray-800 dark:text-gray-300',
                          'transition-colors hover:bg-gray-200 dark:hover:bg-gray-700'
                        )}
                      >
                        {skill}
                      </span>
                    ))}
                  </div>
                </section>
              )}

              {/* Links */}
              {data.links && data.links.length > 0 && (
                <section>
                  <h3 className="mb-3 text-lg font-semibold text-gray-900 dark:text-white">
                    {t('links.title')}
                  </h3>
                  <div className="flex flex-wrap gap-3">
                    {data.links.map((link, index) => {
                      const Icon = linkIcons[link.type];
                      const colorClass = linkColors[link.type];

                      return (
                        <a
                          key={index}
                          href={link.url}
                          target="_blank"
                          rel="noopener noreferrer"
                          className={cn(
                            'inline-flex items-center gap-2 rounded-lg px-4 py-2',
                            'border border-gray-200 dark:border-gray-700',
                            'text-sm font-medium text-gray-700 dark:text-gray-300',
                            'transition-all duration-200',
                            colorClass,
                            'focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2',
                            'dark:focus:ring-offset-gray-900'
                          )}
                        >
                          <Icon className="h-4 w-4" />
                          {getLinkLabel(link)}
                        </a>
                      );
                    })}
                  </div>
                </section>
              )}
            </div>
          </motion.div>
        </motion.div>
      )}
    </AnimatePresence>
  );
}

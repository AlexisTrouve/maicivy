'use client';

import { useState } from 'react';
import { motion } from 'framer-motion';
import { format } from 'date-fns';
// Imports par CHEMIN DIRECT, et non `from 'date-fns/locale'`.
//
// POURQUOI : `date-fns/locale` est un barrel file qui réexporte TOUTES les locales. Webpack ne sait
// pas élaguer ce qui n'est pas utilisé ici (les locales sont des objets construits à l'exécution),
// et embarque donc la centaine de langues du paquet. Mesuré sur le bundle de /cv : un chunk de
// 393 kB dont le contenu était intégralement des traductions de dates — basque, néerlandais,
// allemand… pour une page qui n'en formate que deux. Le chemin direct ne tire que le fichier visé.
//
// ⚠️ Ne pas « simplifier » en revenant à l'import groupé : c'est exactement le piège d'origine, et
// rien ne le signale (ni erreur, ni warning) — seul le poids du bundle change.
import { fr } from 'date-fns/locale/fr';
import { enUS } from 'date-fns/locale/en-US';
// Les trois autres langues du site, par le même chemin direct : sans elles, /de, /it et /zh
// affichaient les mois en anglais (« Jan 2020 ») au milieu d'une interface traduite.
import { de } from 'date-fns/locale/de';
import { it } from 'date-fns/locale/it';
import { zhCN } from 'date-fns/locale/zh-CN';
import { useLocale, useTranslations } from 'next-intl';
import { Briefcase, Calendar, ArrowRight } from 'lucide-react';
import { Experience, DetailLink } from '@/lib/types';
import DetailModal, { DetailModalData } from './DetailModal';

const localeMap = {
  fr: fr,
  en: enUS,
  de: de,
  it: it,
  zh: zhCN,
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
      {/* Ligne du temps : un filet neutre (plus de dégradé bleu→violet). Collée à gauche sur
          téléphone pour laisser la largeur aux cartes, centrée à partir de md. */}
      <div className="absolute left-3 top-0 bottom-0 w-px bg-border md:left-1/2" data-testid="timeline-line" />

      {/* Timeline Items */}
      <div className="space-y-10 md:space-y-12">
        {experiences.map((exp, index) => {
          // Poste en cours (sans date de fin) = « où l'on est » : seul point marqué d'or (doctrine
          // de l'or : la position courante). Les autres points restent des repères neutres.
          const isCurrent = !exp.endDate;
          return (
          <motion.div
            key={exp.id}
            variants={itemVariants}
            data-testid="timeline-item"
            className={`relative flex items-center ${
              index % 2 === 0 ? 'md:flex-row' : 'md:flex-row-reverse'
            }`}
          >
            {/* Timeline Dot */}
            <div
              data-testid="timeline-dot"
              data-current={isCurrent || undefined}
              className={`absolute left-3 top-8 z-10 h-3 w-3 -translate-x-1/2 rounded-full border-2 border-background md:left-1/2 md:top-1/2 md:-translate-y-1/2 ${
                isCurrent ? 'bg-gold' : 'bg-muted-foreground'
              }`}
            />

            {/* Content Card */}
            <div
              className={`ml-9 w-full md:ml-0 md:w-5/12 ${
                index % 2 === 0 ? 'md:mr-auto md:pr-12' : 'md:ml-auto md:pl-12'
              }`}
            >
              <div
                className="group cursor-pointer rounded-lg border border-border bg-card p-5 transition-colors hover:border-muted-foreground/50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring md:p-6"
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
                <div className="mb-3 flex items-start justify-between gap-4">
                  <div className="min-w-0 flex-1">
                    <h3 className="mb-1 text-xl font-medium leading-snug text-foreground">
                      {exp.title}
                    </h3>
                    <div className="flex items-center gap-2 text-sm font-medium text-foreground/80">
                      <Briefcase className="h-4 w-4 shrink-0 text-muted-foreground" aria-hidden="true" />
                      <span>{exp.company}</span>
                    </div>
                  </div>
                  {exp.score && (
                    <div className="shrink-0 rounded-full border border-border px-2.5 py-0.5 text-xs font-medium tabular-nums text-muted-foreground">
                      {Math.round(exp.score * 100)}%
                    </div>
                  )}
                </div>

                {/* Date & Duration */}
                <div className="mb-4 flex flex-wrap items-center gap-x-2 gap-y-1 text-sm text-muted-foreground">
                  <Calendar className="h-4 w-4" aria-hidden="true" />
                  <span>
                    {formatDate(exp.startDate)} - {exp.endDate ? formatDate(exp.endDate) : t('present')}
                  </span>
                  <span aria-hidden="true">·</span>
                  <span>{calculateDuration(exp.startDate, exp.endDate)}</span>
                </div>

                {/* Short Description / Catchphrase */}
                <p className="mb-4 line-clamp-2 leading-relaxed text-foreground/85">
                  {exp.description}
                </p>

                {/* Technologies - show only first 4 (garde || [] : data possiblement sans technologies) */}
                <div className="mb-4 flex flex-wrap gap-2">
                  {(exp.technologies || []).slice(0, 4).map((tech) => (
                    <span
                      key={tech}
                      data-testid="tech-chip"
                      className="rounded-full bg-secondary px-3 py-1 text-xs font-medium text-muted-foreground"
                    >
                      {tech}
                    </span>
                  ))}
                  {(exp.technologies || []).length > 4 && (
                    <span className="rounded-full px-2 py-1 text-xs font-medium text-muted-foreground">
                      +{(exp.technologies || []).length - 4}
                    </span>
                  )}
                </div>

                {/* View Details */}
                <div className="flex items-center gap-2 text-sm font-medium text-foreground">
                  <span className="underline-offset-4 group-hover:underline">{tExp('viewDetails')}</span>
                  <ArrowRight className="h-4 w-4 transition-transform group-hover:translate-x-1" aria-hidden="true" />
                </div>
              </div>
            </div>
          </motion.div>
          );
        })}
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

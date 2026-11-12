'use client';

import { useState, useEffect, useRef } from 'react';
import { motion } from 'framer-motion';
import { useTranslations } from 'next-intl';
import { Star, Clock } from 'lucide-react';
import { Project, DetailLink } from '@/lib/types';
import DetailModal, { DetailModalData } from './DetailModal';

interface ProjectsGridProps {
  projects: Project[];
}

// Nombre de cartes montées d'emblée, puis ajoutées à chaque passage de la sentinelle.
//
// POURQUOI un lazy-load ici : le CV rend 79 projets d'un bloc. Chacun est une carte animée
// (framer-motion : hover, variants, stagger), donc 79 souscriptions d'animation montées au premier
// rendu alors que 6 à 9 cartes seulement tiennent dans l'écran. C'est ce montage massif qui coûte
// à l'hydratation, pas le HTML lui-même.
//
// ⚠️ On ne retire RIEN du HTML serveur : /cv est dans le sitemap et les projets sont sa matière
// indexable. Tronquer le rendu serveur ferait disparaître 67 projets pour Google. Le découpage est
// donc purement CLIENT (cf. le commentaire du composant).
const CARTES_INITIALES = 12;
const CARTES_PAR_PALIER = 12;

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

  // --- Montage progressif des cartes -------------------------------------------------------
  //
  // QUOI : `nbAffichees` borne le nombre de cartes RENDUES PAR REACT côté client. Il démarre à
  // `projects.length` (tout), puis retombe à CARTES_INITIALES une fois le composant monté, et
  // remonte par paliers quand la sentinelle entre dans le viewport.
  //
  // POURQUOI démarrer à « tout » plutôt qu'à 12 : le rendu SERVEUR doit contenir les 79 projets
  // (SEO — /cv est dans le sitemap). Si l'état initial valait 12, le premier rendu client
  // divergerait du HTML serveur et React signalerait un mismatch d'hydratation, avec à la clé un
  // re-rendu complet de la section — soit l'inverse du but recherché. On rend donc exactement le
  // même arbre que le serveur, PUIS on élague dans un effet (qui ne s'exécute qu'après
  // l'hydratation, donc sans divergence possible).
  //
  // CONSÉQUENCE ASSUMÉE : sans JS, les 79 cartes restent visibles — c'est voulu, c'est ce qui
  // garde le contenu indexable et accessible.
  const [nbAffichees, setNbAffichees] = useState(projects.length);
  const sentinelleRef = useRef<HTMLDivElement | null>(null);

  // Élagage post-hydratation : on retombe au premier palier une fois que React a repris la main.
  // Dépend de projects.length pour se réinitialiser si la liste change (changement de thème).
  useEffect(() => {
    setNbAffichees((n) => Math.min(n, CARTES_INITIALES));
  }, [projects.length]);

  // Révélation par paliers via IntersectionObserver.
  //
  // POURQUOI un observer plutôt qu'un bouton « voir plus » : le comportement actuel (tout est là,
  // on scrolle) est préservé — personne ne perd un clic. L'observer est aussi ce qui permet de ne
  // rien monter tant que le visiteur n'est pas descendu.
  //
  // COMMENT : la sentinelle est le div placé APRÈS la grille. `rootMargin` de 600 px déclenche le
  // palier avant qu'elle soit visible, pour que les cartes soient prêtes quand on arrive dessus.
  // L'observer est recréé quand `nbAffichees` change : sinon la closure capturerait une valeur
  // périmée et n'ajouterait qu'un seul palier.
  useEffect(() => {
    const cible = sentinelleRef.current;
    // Plus rien à révéler, ou environnement sans IntersectionObserver (jsdom, très vieux
    // navigateur) → on ne câble rien. Le repli est géré juste en dessous.
    if (!cible || nbAffichees >= projects.length) return;
    if (typeof IntersectionObserver === 'undefined') {
      // Pas d'observer disponible : on affiche tout plutôt que de bloquer l'accès aux projets.
      setNbAffichees(projects.length);
      return;
    }

    const observer = new IntersectionObserver(
      (entries) => {
        if (entries.some((e) => e.isIntersecting)) {
          setNbAffichees((n) => Math.min(n + CARTES_PAR_PALIER, projects.length));
        }
      },
      { rootMargin: '600px' }
    );
    observer.observe(cible);
    return () => observer.disconnect();
  }, [nbAffichees, projects.length]);

  const projetsVisibles = projects.slice(0, nbAffichees);
  const resteAAfficher = projects.length - projetsVisibles.length;

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
        {projetsVisibles.map((project) => (
          <motion.div
            key={project.id}
            data-testid="project-card"
            data-featured={project.featured || undefined}
            variants={itemVariants}
            onClick={() => openProjectModal(project)}
            // Projet mis en avant = projet « retenu » : filet d'or en tête de carte, comme les
            // projets retenus du dossier vivant. Jamais un aplat : la carte reste de la même matière.
            className={`group relative flex cursor-pointer flex-col rounded-lg border bg-card p-5 transition-colors hover:border-muted-foreground/50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring md:p-6 ${
              project.featured ? 'border-border border-t-gold' : 'border-border'
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
            {/* Repères de la carte : catégorie, statut, mise en avant — en tête, sur une ligne
                qui passe à la ligne au besoin (au lieu d'une pile en surimpression du titre). */}
            <div className="mb-3 flex flex-wrap items-center gap-2 text-xs">
              {project.featured && (
                <span className="inline-flex items-center gap-1 font-medium text-foreground">
                  <Star className="h-3.5 w-3.5 fill-current text-gold" aria-hidden="true" />
                  {t('featured')}
                </span>
              )}
              {project.category && (
                <span data-testid="project-category" className="rounded-full bg-secondary px-2 py-0.5 font-medium text-muted-foreground">
                  {project.category}
                </span>
              )}
              {project.inProgress && (
                <span className="inline-flex items-center gap-1 rounded-full border border-border px-2 py-0.5 font-medium text-muted-foreground">
                  <Clock className="h-3 w-3" aria-hidden="true" />
                  {t('inProgress')}
                </span>
              )}
            </div>

            {/* Title */}
            <h3 className="mb-2 text-xl font-medium leading-snug text-foreground">
              {project.title}
            </h3>

            {/* Catchphrase / Short Description */}
            <p className="mb-4 line-clamp-2 text-sm text-muted-foreground">
              {project.catchphrase || project.description}
            </p>

            {/* Technologies Preview — garde || [] : la data peut arriver sans technologies */}
            <div className="flex flex-wrap gap-1.5">
              {(project.technologies || []).slice(0, 3).map((tech) => (
                <span
                  key={tech}
                  data-testid="tech-chip"
                  className="rounded bg-secondary px-2 py-0.5 text-xs text-muted-foreground"
                >
                  {tech}
                </span>
              ))}
              {(project.technologies || []).length > 3 && (
                <span className="px-1 py-0.5 text-xs text-muted-foreground">
                  +{(project.technologies || []).length - 3}
                </span>
              )}
            </div>

            {/* Invitation à ouvrir la fiche, calée en bas de carte */}
            <div className="mt-auto pt-4 text-xs font-medium text-muted-foreground transition-colors group-hover:text-foreground">
              {t('clickForDetails')}
            </div>
          </motion.div>
        ))}
      </motion.div>

      {/* Sentinelle du lazy-load : entrée dans le viewport (600 px d'avance) → palier suivant.
          Rendue uniquement s'il reste des projets à révéler, et porte un testid pour que l'E2E
          puisse constater la révélation progressive. */}
      {resteAAfficher > 0 && (
        <div
          ref={sentinelleRef}
          data-testid="projects-sentinel"
          data-remaining={resteAAfficher}
          aria-hidden="true"
          className="h-px w-full"
        />
      )}

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

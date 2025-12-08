'use client';

import { motion } from 'framer-motion';
import { ExternalLink, Github, Star } from 'lucide-react';
import Link from 'next/link';
import { Project } from '@/lib/types';

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

const languageColors: Record<string, string> = {
  TypeScript: 'bg-blue-500',
  JavaScript: 'bg-yellow-500',
  Go: 'bg-cyan-500',
  Python: 'bg-green-500',
  Rust: 'bg-orange-500',
  Java: 'bg-red-500',
  'C++': 'bg-purple-500',
};

export default function ProjectsGrid({ projects }: ProjectsGridProps) {
  return (
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
          whileHover={{ y: -8 }}
          className={`bg-white dark:bg-gray-800 rounded-lg shadow-lg p-6 border-2 transition-all ${
            project.featured
              ? 'border-yellow-400 dark:border-yellow-600'
              : 'border-gray-200 dark:border-gray-700'
          }`}
        >
          {/* Featured Badge */}
          {project.featured && (
            <div className="flex items-center gap-1 text-yellow-600 dark:text-yellow-400 mb-2 font-semibold text-sm">
              <Star className="w-4 h-4 fill-current" />
              <span>Projet Vedette</span>
            </div>
          )}

          {/* Header */}
          <div className="mb-3">
            <h3 className="text-xl font-bold text-gray-900 dark:text-white mb-2">
              {project.title}
            </h3>

            {/* Language & Stars */}
            <div className="flex items-center gap-3 text-sm">
              {project.language && (
                <div className="flex items-center gap-1">
                  <div
                    className={`w-3 h-3 rounded-full ${
                      languageColors[project.language] || 'bg-gray-500'
                    }`}
                  />
                  <span className="text-gray-600 dark:text-gray-400">
                    {project.language}
                  </span>
                </div>
              )}
              {project.stars !== undefined && (
                <div className="flex items-center gap-1 text-gray-600 dark:text-gray-400">
                  <Star className="w-4 h-4" />
                  <span>{project.stars}</span>
                </div>
              )}
            </div>
          </div>

          {/* Description */}
          <p className="text-gray-700 dark:text-gray-300 mb-4 line-clamp-3">
            {project.description}
          </p>

          {/* Technologies */}
          <div className="flex flex-wrap gap-2 mb-4">
            {project.technologies.slice(0, 4).map((tech) => (
              <span
                key={tech}
                className="px-2 py-1 bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300 rounded text-xs font-medium"
              >
                {tech}
              </span>
            ))}
            {project.technologies.length > 4 && (
              <span className="px-2 py-1 text-gray-500 dark:text-gray-400 text-xs">
                +{project.technologies.length - 4}
              </span>
            )}
          </div>

          {/* Links */}
          <div className="flex gap-3">
            {project.githubUrl && (
              <Link
                href={project.githubUrl}
                target="_blank"
                rel="noopener noreferrer"
                className="flex items-center gap-1 text-gray-700 dark:text-gray-300 hover:text-blue-600 dark:hover:text-blue-400 transition-colors"
              >
                <Github className="w-4 h-4" />
                <span className="text-sm font-medium">Code</span>
              </Link>
            )}
            {project.demoUrl && (
              <Link
                href={project.demoUrl}
                target="_blank"
                rel="noopener noreferrer"
                className="flex items-center gap-1 text-gray-700 dark:text-gray-300 hover:text-blue-600 dark:hover:text-blue-400 transition-colors"
              >
                <ExternalLink className="w-4 h-4" />
                <span className="text-sm font-medium">Demo</span>
              </Link>
            )}
          </div>

          {/* Score Badge */}
          {project.score && (
            <div className="mt-4 pt-4 border-t border-gray-200 dark:border-gray-700">
              <div className="flex items-center justify-between text-sm">
                <span className="text-gray-600 dark:text-gray-400">Pertinence</span>
                <div className="flex items-center gap-2">
                  <div className="w-24 h-2 bg-gray-200 dark:bg-gray-700 rounded-full overflow-hidden">
                    <div
                      className="h-full bg-blue-600 rounded-full transition-all"
                      style={{ width: `${project.score * 100}%` }}
                    />
                  </div>
                  <span className="font-semibold text-blue-600">
                    {Math.round(project.score * 100)}%
                  </span>
                </div>
              </div>
            </div>
          )}
        </motion.div>
      ))}
    </motion.div>
  );
}

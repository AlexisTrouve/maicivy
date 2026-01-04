'use client';

import React from 'react';
import { ChevronLeft, ChevronRight, X, ExternalLink, Github } from 'lucide-react';
import type { Portfolio3DProject } from '@/lib/types';

interface PortfolioNavigationProps {
  projects: Portfolio3DProject[];
  currentIndex: number | null;
  onNavigate: (index: number | null) => void;
  onClose: () => void;
}

export function PortfolioNavigation({
  projects,
  currentIndex,
  onNavigate,
  onClose
}: PortfolioNavigationProps) {
  const hasSelection = currentIndex !== null;
  const currentProject = hasSelection ? projects[currentIndex] : null;

  const handlePrev = () => {
    if (currentIndex === null) return;
    const newIndex = currentIndex > 0 ? currentIndex - 1 : projects.length - 1;
    onNavigate(newIndex);
  };

  const handleNext = () => {
    if (currentIndex === null) return;
    const newIndex = currentIndex < projects.length - 1 ? currentIndex + 1 : 0;
    onNavigate(newIndex);
  };

  return (
    <>
      {/* Navigation arrows - always visible */}
      <div className="absolute inset-x-0 top-1/2 -translate-y-1/2 flex justify-between px-4 pointer-events-none z-10">
        <button
          onClick={handlePrev}
          disabled={!hasSelection}
          className="pointer-events-auto p-3 rounded-full bg-white/10 backdrop-blur-md border border-white/20 text-white hover:bg-white/20 transition-all disabled:opacity-30 disabled:cursor-not-allowed"
          aria-label="Previous project"
        >
          <ChevronLeft className="w-6 h-6" />
        </button>
        <button
          onClick={handleNext}
          disabled={!hasSelection}
          className="pointer-events-auto p-3 rounded-full bg-white/10 backdrop-blur-md border border-white/20 text-white hover:bg-white/20 transition-all disabled:opacity-30 disabled:cursor-not-allowed"
          aria-label="Next project"
        >
          <ChevronRight className="w-6 h-6" />
        </button>
      </div>

      {/* Navigation indicator - dots for few projects, counter for many */}
      <div className="absolute bottom-6 left-1/2 -translate-x-1/2 z-10">
        {projects.length <= 12 ? (
          // Dots for 12 or fewer projects
          <div className="flex gap-2">
            {projects.map((_, index) => (
              <button
                key={index}
                onClick={() => onNavigate(index)}
                className={`w-2 h-2 rounded-full transition-all ${
                  index === currentIndex
                    ? 'bg-white w-6'
                    : 'bg-white/40 hover:bg-white/60'
                }`}
                aria-label={`Go to project ${index + 1}`}
              />
            ))}
          </div>
        ) : (
          // Counter + slider for many projects
          <div className="bg-black/60 backdrop-blur-md rounded-full px-4 py-2 flex items-center gap-4">
            <span className="text-white text-sm font-medium min-w-[60px] text-center">
              {currentIndex !== null ? currentIndex + 1 : '-'} / {projects.length}
            </span>
            <input
              type="range"
              min={0}
              max={projects.length - 1}
              value={currentIndex ?? 0}
              onChange={(e) => onNavigate(parseInt(e.target.value))}
              className="w-32 h-1 bg-white/20 rounded-full appearance-none cursor-pointer
                [&::-webkit-slider-thumb]:appearance-none [&::-webkit-slider-thumb]:w-3 [&::-webkit-slider-thumb]:h-3
                [&::-webkit-slider-thumb]:rounded-full [&::-webkit-slider-thumb]:bg-white"
            />
          </div>
        )}
      </div>

      {/* Close button - when project selected */}
      {hasSelection && (
        <button
          onClick={onClose}
          className="absolute top-4 right-4 p-2 rounded-full bg-white/10 backdrop-blur-md border border-white/20 text-white hover:bg-white/20 transition-all z-10"
          aria-label="Close"
        >
          <X className="w-5 h-5" />
        </button>
      )}

      {/* Project details panel - when selected */}
      {hasSelection && currentProject && (
        <div className="absolute bottom-20 left-1/2 -translate-x-1/2 w-full max-w-lg px-4 z-10">
          <div className="bg-black/60 backdrop-blur-xl rounded-2xl border border-white/10 p-6 text-white">
            <h2 className="text-2xl font-bold mb-2">{currentProject.title}</h2>
            <p className="text-gray-300 text-sm mb-4">{currentProject.description}</p>

            {/* Tech stack */}
            <div className="flex flex-wrap gap-2 mb-4">
              {currentProject.technologies.map((tech) => (
                <span
                  key={tech}
                  className="px-2 py-1 text-xs rounded-full bg-white/10 border border-white/20"
                >
                  {tech}
                </span>
              ))}
            </div>

            {/* Action buttons */}
            <div className="flex gap-3">
              {currentProject.demoUrl && (
                <a
                  href={currentProject.demoUrl}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="flex items-center gap-2 px-4 py-2 rounded-lg bg-white text-black font-medium hover:bg-gray-100 transition-colors"
                >
                  <ExternalLink className="w-4 h-4" />
                  Demo
                </a>
              )}
              {currentProject.githubUrl && (
                <a
                  href={currentProject.githubUrl}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="flex items-center gap-2 px-4 py-2 rounded-lg bg-white/10 border border-white/20 hover:bg-white/20 transition-colors"
                >
                  <Github className="w-4 h-4" />
                  GitHub
                </a>
              )}
            </div>
          </div>
        </div>
      )}

      {/* Instructions - when no selection */}
      {!hasSelection && (
        <div className="absolute bottom-20 left-1/2 -translate-x-1/2 text-center z-10">
          <p className="text-white/60 text-sm">
            Click on a card to see details
          </p>
        </div>
      )}
    </>
  );
}

export default PortfolioNavigation;

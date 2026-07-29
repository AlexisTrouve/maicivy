'use client';

import { useState, useEffect, useCallback, useRef } from 'react';

interface UseTimelineScrollOptions {
  threshold?: number;
  offset?: number;
}

interface UseTimelineScrollReturn {
  scrollProgress: number;
  activeSection: string | null;
  isNearTop: boolean;
  isNearBottom: boolean;
  scrollToYear: (year: number) => void;
  scrollToTop: () => void;
  scrollToElement: (elementId: string) => void;
}

export const useTimelineScroll = (
  options: UseTimelineScrollOptions = {}
): UseTimelineScrollReturn => {
  const { threshold = 0.3, offset = 100 } = options;

  const [scrollProgress, setScrollProgress] = useState<number>(0);
  const [activeSection, setActiveSection] = useState<string | null>(null);
  const [isNearTop, setIsNearTop] = useState<boolean>(true);
  const [isNearBottom, setIsNearBottom] = useState<boolean>(false);

  const observerRef = useRef<IntersectionObserver | null>(null);
  const sectionsRef = useRef<Map<Element, string>>(new Map());

  // Calculer la progression du scroll
  useEffect(() => {
    const handleScroll = () => {
      const windowHeight = window.innerHeight;
      const documentHeight = document.documentElement.scrollHeight;
      const scrollTop = window.scrollY;

      // Progression (0-100)
      const totalScroll = documentHeight - windowHeight;
      const progress = totalScroll > 0 ? (scrollTop / totalScroll) * 100 : 0;
      setScrollProgress(Math.min(Math.max(progress, 0), 100));

      // Détecter si on est près du haut ou du bas
      setIsNearTop(scrollTop < offset);
      setIsNearBottom(scrollTop + windowHeight > documentHeight - offset);
    };

    // Initial call
    handleScroll();

    window.addEventListener('scroll', handleScroll, { passive: true });
    return () => window.removeEventListener('scroll', handleScroll);
  }, [offset]);

  // Observer les sections pour déterminer la section active
  useEffect(() => {
    // Créer l'observer
    observerRef.current = new IntersectionObserver(
      (entries) => {
        entries.forEach((entry) => {
          if (entry.isIntersecting) {
            const sectionId = sectionsRef.current.get(entry.target);
            if (sectionId) {
              setActiveSection(sectionId);
            }
          }
        });
      },
      {
        threshold,
        rootMargin: '-20% 0px -20% 0px',
      }
    );

    // Observer les éléments avec data-section
    const sections = document.querySelectorAll('[data-section]');
    sections.forEach((section) => {
      const sectionId = section.getAttribute('data-section');
      if (sectionId) {
        sectionsRef.current.set(section, sectionId);
        observerRef.current?.observe(section);
      }
    });

    // Observer les années (data-year)
    const yearElements = document.querySelectorAll('[data-year]');
    yearElements.forEach((element) => {
      const year = element.getAttribute('data-year');
      if (year) {
        sectionsRef.current.set(element, `year-${year}`);
        observerRef.current?.observe(element);
      }
    });

    return () => {
      observerRef.current?.disconnect();
      sectionsRef.current.clear();
    };
  }, [threshold]);

  // Scroll vers une année spécifique
  const scrollToYear = useCallback((year: number) => {
    const yearElements = document.querySelectorAll('[data-year]');
    const targetElement = Array.from(yearElements).find(
      (el) => el.getAttribute('data-year') === String(year)
    );

    if (targetElement) {
      const elementPosition =
        targetElement.getBoundingClientRect().top + window.scrollY;
      const offsetPosition = elementPosition - offset;

      window.scrollTo({
        top: offsetPosition,
        behavior: 'smooth',
      });

      setActiveSection(`year-${year}`);
    }
  }, [offset]);

  // Scroll vers le haut
  const scrollToTop = useCallback(() => {
    window.scrollTo({
      top: 0,
      behavior: 'smooth',
    });
  }, []);

  // Scroll vers un élément spécifique par ID
  const scrollToElement = useCallback((elementId: string) => {
    const element = document.getElementById(elementId);

    if (element) {
      const elementPosition = element.getBoundingClientRect().top + window.scrollY;
      const offsetPosition = elementPosition - offset;

      window.scrollTo({
        top: offsetPosition,
        behavior: 'smooth',
      });

      setActiveSection(elementId);
    }
  }, [offset]);

  return {
    scrollProgress,
    activeSection,
    isNearTop,
    isNearBottom,
    scrollToYear,
    scrollToTop,
    scrollToElement,
  };
};

// Hook pour détecter la direction du scroll
export const useScrollDirection = () => {
  const [scrollDirection, setScrollDirection] = useState<'up' | 'down' | null>(null);
  const lastScrollY = useRef(0);

  useEffect(() => {
    const handleScroll = () => {
      const currentScrollY = window.scrollY;

      if (currentScrollY > lastScrollY.current) {
        setScrollDirection('down');
      } else if (currentScrollY < lastScrollY.current) {
        setScrollDirection('up');
      }

      lastScrollY.current = currentScrollY;
    };

    window.addEventListener('scroll', handleScroll, { passive: true });
    return () => window.removeEventListener('scroll', handleScroll);
  }, []);

  return scrollDirection;
};

// Hook pour le snap scroll
export const useScrollSnap = (snapThreshold = 100) => {
  const [isSnapping, setIsSnapping] = useState(false);
  const snapTimeoutRef = useRef<NodeJS.Timeout | null>(null);

  useEffect(() => {
    const handleScroll = () => {
      setIsSnapping(true);

      // Délai avant de considérer le scroll comme terminé
      if (snapTimeoutRef.current) {
        clearTimeout(snapTimeoutRef.current);
      }

      snapTimeoutRef.current = setTimeout(() => {
        setIsSnapping(false);

        // Trouver l'élément le plus proche
        const snapElements = document.querySelectorAll('[data-snap]');
        let closestElement: Element | null = null;
        let closestDistance = Infinity;

        snapElements.forEach((element) => {
          const rect = element.getBoundingClientRect();
          const distance = Math.abs(rect.top);

          if (distance < closestDistance && distance < snapThreshold) {
            closestDistance = distance;
            closestElement = element;
          }
        });

        // Snap vers l'élément le plus proche
        if (closestElement) {
          (closestElement as Element).scrollIntoView({
            behavior: 'smooth',
            block: 'start',
          });
        }
      }, 150);
    };

    window.addEventListener('scroll', handleScroll, { passive: true });

    return () => {
      window.removeEventListener('scroll', handleScroll);
      if (snapTimeoutRef.current) {
        clearTimeout(snapTimeoutRef.current);
      }
    };
  }, [snapThreshold]);

  return { isSnapping };
};

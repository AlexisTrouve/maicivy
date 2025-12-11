'use client';

import { useState, useEffect, useCallback } from 'react';
import { TimelineEvent, TimelineMilestone } from '@/lib/types';
import { timelineApi } from '@/lib/api';

interface UseTimelineDataOptions {
  category?: string;
  type?: string;
  from?: string;
  to?: string;
  autoFetch?: boolean;
}

interface UseTimelineDataReturn {
  events: TimelineEvent[];
  categories: string[];
  milestones: TimelineMilestone[];
  stats: TimelineStats | null;
  isLoading: boolean;
  error: Error | null;
  refetch: () => Promise<void>;
  filter: (filters: TimelineFilters) => void;
  reset: () => void;
}

interface TimelineFilters {
  category?: string;
  type?: string;
  from?: string;
  to?: string;
}

interface TimelineStats {
  totalExperiences: number;
  totalProjects: number;
  categoriesBreakdown: Record<string, number>;
  yearsOfExperience: number;
  topTechnologies: Array<{ name: string; count: number }>;
}

export const useTimelineData = (
  options: UseTimelineDataOptions = {}
): UseTimelineDataReturn => {
  const { autoFetch = true } = options;

  const [events, setEvents] = useState<TimelineEvent[]>([]);
  const [allEvents, setAllEvents] = useState<TimelineEvent[]>([]);
  const [categories, setCategories] = useState<string[]>([]);
  const [milestones, setMilestones] = useState<TimelineMilestone[]>([]);
  const [stats, setStats] = useState<TimelineStats | null>(null);
  const [isLoading, setIsLoading] = useState<boolean>(false);
  const [error, setError] = useState<Error | null>(null);
  const [activeFilters, setActiveFilters] = useState<TimelineFilters>({
    category: options.category,
    type: options.type,
    from: options.from,
    to: options.to,
  });

  // Fetch timeline data
  const fetchTimeline = useCallback(async () => {
    setIsLoading(true);
    setError(null);

    try {
      // Fetch events avec filtres
      const params = new URLSearchParams();
      if (activeFilters.category) params.append('category', activeFilters.category);
      if (activeFilters.from) params.append('from', activeFilters.from);
      if (activeFilters.to) params.append('to', activeFilters.to);

      const eventsData = await timelineApi.getTimeline(
        activeFilters.category,
        activeFilters.from,
        activeFilters.to
      );

      setAllEvents(eventsData.events);

      // Appliquer le filtre de type localement
      let filteredEvents = eventsData.events;
      if (activeFilters.type && activeFilters.type !== 'all') {
        filteredEvents = eventsData.events.filter(
          (e: TimelineEvent) => e.type === activeFilters.type
        );
      }

      setEvents(filteredEvents);
      setStats(eventsData.stats);

      // Fetch categories (uniquement au premier chargement)
      if (categories.length === 0) {
        const categoriesData = await timelineApi.getCategories();
        setCategories(categoriesData);
      }

      // Fetch milestones (uniquement au premier chargement)
      if (milestones.length === 0) {
        const milestonesData = await timelineApi.getMilestones();
        setMilestones(milestonesData);
      }
    } catch (err) {
      const error = err as Error;
      setError(error);
      setEvents([]);
      setAllEvents([]);
      console.error('Error fetching timeline data:', err);
    } finally {
      setIsLoading(false);
    }
  }, [activeFilters, categories.length, milestones.length]);

  // Initial fetch
  useEffect(() => {
    if (autoFetch) {
      fetchTimeline();
    }
  }, [autoFetch, fetchTimeline]);

  // Filter events
  const filter = useCallback(
    (filters: TimelineFilters) => {
      setActiveFilters(filters);

      // Filtrage local pour le type
      let filtered = allEvents;

      if (filters.type && filters.type !== 'all') {
        filtered = filtered.filter((e) => e.type === filters.type);
      }

      if (filters.category) {
        filtered = filtered.filter((e) => e.category === filters.category);
      }

      if (filters.from || filters.to) {
        filtered = filtered.filter((e) => {
          const eventDate = new Date(e.startDate);
          const fromDate = filters.from ? new Date(filters.from) : null;
          const toDate = filters.to ? new Date(filters.to) : null;

          if (fromDate && eventDate < fromDate) return false;
          if (toDate && eventDate > toDate) return false;
          return true;
        });
      }

      setEvents(filtered);
    },
    [allEvents]
  );

  // Reset filters
  const reset = useCallback(() => {
    setActiveFilters({});
    setEvents(allEvents);
  }, [allEvents]);

  // Refetch data
  const refetch = useCallback(async () => {
    await fetchTimeline();
  }, [fetchTimeline]);

  return {
    events,
    categories,
    milestones,
    stats,
    isLoading,
    error,
    refetch,
    filter,
    reset,
  };
};

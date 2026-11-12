// frontend/hooks/useProfileDetection.ts
'use client';

import { useState, useEffect } from 'react';
import { profileApi } from '@/lib/api';

export type ProfileType = 'recruiter' | 'cto' | 'tech_lead' | 'ceo' | 'developer' | 'other';

export interface ProfileDetection {
  profileType: ProfileType;
  confidence: number;
  enrichmentData?: Record<string, any>;
  deviceInfo?: {
    browser: string;
    os: string;
    deviceType: string;
    isBot: boolean;
  };
  detectionSources?: string[];
  bypassEnabled: boolean;
  isDetected: boolean;
}

/**
 * Hook pour récupérer les informations de détection de profil
 */
export function useProfileDetection() {
  const [profileData, setProfileData] = useState<ProfileDetection>({
    profileType: 'other',
    confidence: 0,
    bypassEnabled: false,
    isDetected: false,
  });

  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    async function fetchProfile() {
      try {
        // Essayer de récupérer le profil actuel (depuis middleware/cache)
        const data = await profileApi.getCurrent();

        setProfileData({
          profileType: data.profile_type as ProfileType,
          confidence: data.confidence || 0,
          enrichmentData: data.enrichment_data,
          deviceInfo: data.device_info,
          detectionSources: data.detection_sources,
          bypassEnabled: data.bypass_enabled || false,
          isDetected: data.profile_type !== 'other' && (data.confidence || 0) > 0,
        });
      } catch (err) {
        console.error('Failed to fetch profile detection:', err);
        setError('Failed to load profile');
      } finally {
        setLoading(false);
      }
    }

    fetchProfile();
  }, []);

  return {
    ...profileData,
    loading,
    error,
  };
}

/**
 * Hook pour détecter manuellement le profil (force detection)
 */
export function useProfileDetectionManual() {
  const [profileData, setProfileData] = useState<ProfileDetection | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const detect = async () => {
    setLoading(true);
    setError(null);

    try {
      const data = await profileApi.detect();

      setProfileData({
        profileType: data.profile_type as ProfileType,
        confidence: data.confidence || 0,
        enrichmentData: data.enrichment_data,
        deviceInfo: data.device_info,
        detectionSources: data.detection_sources,
        bypassEnabled: data.bypass_enabled || false,
        isDetected: data.profile_type !== 'other' && (data.confidence || 0) > 0,
      });

      return data;
    } catch (err) {
      console.error('Profile detection error:', err);
      setError('Detection failed');
      throw err;
    } finally {
      setLoading(false);
    }
  };

  return {
    profileData,
    loading,
    error,
    detect,
  };
}

/**
 * Hook pour vérifier le statut de bypass
 */
export function useBypassStatus() {
  const [bypassed, setBypassed] = useState(false);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    async function checkBypass() {
      try {
        const data = await profileApi.getBypassStatus();
        setBypassed(data.bypass || false);
      } catch (err) {
        console.error('Failed to check bypass status:', err);
      } finally {
        setLoading(false);
      }
    }

    checkBypass();
  }, []);

  const refresh = async () => {
    setLoading(true);
    try {
      const data = await profileApi.getBypassStatus();
      setBypassed(data.bypass || false);
    } catch (err) {
      console.error('Failed to refresh bypass status:', err);
    } finally {
      setLoading(false);
    }
  };

  return {
    bypassed,
    loading,
    refresh,
  };
}

/**
 * Hook pour les stats de profils (admin/analytics)
 */
export function useProfileStats() {
  const [stats, setStats] = useState<any>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    async function fetchStats() {
      try {
        const data = await profileApi.getStats();
        setStats(data);
      } catch (err) {
        console.error('Failed to fetch profile stats:', err);
        setError('Failed to load stats');
      } finally {
        setLoading(false);
      }
    }

    fetchStats();
  }, []);

  return {
    stats,
    loading,
    error,
  };
}

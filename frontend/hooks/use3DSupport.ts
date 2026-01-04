/**
 * Hook pour détecter le support WebGL et les performances du device
 * Permet d'adapter l'expérience 3D selon les capacités
 */

import { useState, useEffect } from 'react';

export type PerformanceLevel = 'high' | 'medium' | 'low' | 'none';

export interface Device3DSupport {
  isSupported: boolean;
  performanceLevel: PerformanceLevel;
  webGLVersion: number | null;
  isMobile: boolean;
  reason?: string;
}

/**
 * Hook pour vérifier le support des effets 3D
 */
export function use3DSupport(): Device3DSupport {
  const [support, setSupport] = useState<Device3DSupport>({
    isSupported: false,
    performanceLevel: 'none',
    webGLVersion: null,
    isMobile: false,
  });

  useEffect(() => {
    const detectSupport = (): Device3DSupport => {
      // 1. Détecter mobile
      const isMobile = /Android|webOS|iPhone|iPad|iPod|BlackBerry|IEMobile|Opera Mini/i.test(
        navigator.userAgent
      );

      // 2. Tester WebGL
      const canvas = document.createElement('canvas');
      let gl: WebGLRenderingContext | WebGL2RenderingContext | null = null;
      let webGLVersion: number | null = null;

      try {
        // Tester WebGL 2.0
        gl = canvas.getContext('webgl2') as WebGL2RenderingContext | null;
        if (!gl) {
          gl = canvas.getContext('experimental-webgl2') as WebGL2RenderingContext | null;
        }
        if (gl) {
          webGLVersion = 2;
        } else {
          // Fallback WebGL 1.0
          gl = canvas.getContext('webgl') as WebGLRenderingContext | null;
          if (!gl) {
            gl = canvas.getContext('experimental-webgl') as WebGLRenderingContext | null;
          }
          if (gl) {
            webGLVersion = 1;
          }
        }
      } catch (e) {
        return {
          isSupported: false,
          performanceLevel: 'none',
          webGLVersion: null,
          isMobile,
          reason: 'WebGL not available',
        };
      }

      if (!gl) {
        return {
          isSupported: false,
          performanceLevel: 'none',
          webGLVersion: null,
          isMobile,
          reason: 'WebGL context creation failed',
        };
      }

      // 3. Détecter niveau de performance
      let performanceLevel: PerformanceLevel = 'low';

      // Vérifier renderer
      const debugInfo = gl.getExtension('WEBGL_debug_renderer_info');
      let renderer = 'unknown';
      if (debugInfo) {
        renderer = gl.getParameter(debugInfo.UNMASKED_RENDERER_WEBGL) || 'unknown';
      }

      // Heuristique de performance
      const hasHighPerformanceGPU = /nvidia|geforce|radeon|intel iris|apple m[1-9]/i.test(
        renderer.toLowerCase()
      );

      const hasLowEndGPU = /intel hd|intel uhd graphics [4-6]/i.test(
        renderer.toLowerCase()
      );

      // Calculer niveau selon device + GPU
      if (isMobile) {
        // Mobile : généralement low/medium
        performanceLevel = hasHighPerformanceGPU ? 'medium' : 'low';
      } else {
        // Desktop
        if (hasLowEndGPU) {
          performanceLevel = 'medium';
        } else if (hasHighPerformanceGPU) {
          performanceLevel = 'high';
        } else {
          performanceLevel = 'medium';
        }
      }

      // 4. Vérifier RAM disponible (si l'API est disponible)
      if ('deviceMemory' in navigator) {
        const memory = (navigator as any).deviceMemory; // GB
        if (memory < 4) {
          performanceLevel = 'low';
        }
      }

      // 5. Désactiver 3D sur mobile low-end
      const isSupported = !(isMobile && performanceLevel === 'low');

      return {
        isSupported,
        performanceLevel,
        webGLVersion,
        isMobile,
        reason: isSupported ? undefined : 'Low-end mobile device',
      };
    };

    setSupport(detectSupport());
  }, []);

  return support;
}

/**
 * Hook pour obtenir uniquement le booléen de support
 */
export function useHas3DSupport(): boolean {
  const { isSupported } = use3DSupport();
  return isSupported;
}

/**
 * Hook pour obtenir les options de qualité 3D selon performance
 */
export function use3DQualitySettings() {
  const { performanceLevel } = use3DSupport();

  const getQualitySettings = () => {
    switch (performanceLevel) {
      case 'high':
        return {
          antialias: true,
          shadows: true,
          particleCount: 1000,
          maxFPS: 60,
          pixelRatio: Math.min(window.devicePixelRatio, 2),
        };

      case 'medium':
        return {
          antialias: true,
          shadows: false,
          particleCount: 500,
          maxFPS: 45,
          pixelRatio: 1,
        };

      case 'low':
        return {
          antialias: false,
          shadows: false,
          particleCount: 200,
          maxFPS: 30,
          pixelRatio: 1,
        };

      default:
        return {
          antialias: false,
          shadows: false,
          particleCount: 0,
          maxFPS: 30,
          pixelRatio: 1,
        };
    }
  };

  return getQualitySettings();
}

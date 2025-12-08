/**
 * Hook pour gérer les controls 3D (souris, touch)
 * Wrapper autour d'OrbitControls avec smooth damping
 */

import { useEffect, useRef } from 'react';
import type { OrbitControls as OrbitControlsType } from 'three-stdlib';

export interface ControlsConfig {
  enableDamping?: boolean;
  dampingFactor?: number;
  enableZoom?: boolean;
  enablePan?: boolean;
  autoRotate?: boolean;
  autoRotateSpeed?: number;
  minDistance?: number;
  maxDistance?: number;
  minPolarAngle?: number;
  maxPolarAngle?: number;
}

/**
 * Hook pour configurer les controls 3D avec des defaults optimisés
 */
export function use3DControls(
  controlsRef: React.RefObject<OrbitControlsType>,
  config: ControlsConfig = {}
) {
  const {
    enableDamping = true,
    dampingFactor = 0.05,
    enableZoom = true,
    enablePan = false,
    autoRotate = false,
    autoRotateSpeed = 1.0,
    minDistance = 2,
    maxDistance = 10,
    minPolarAngle = Math.PI / 4,
    maxPolarAngle = Math.PI / 1.5,
  } = config;

  useEffect(() => {
    if (!controlsRef.current) return;

    const controls = controlsRef.current;

    // Configuration
    controls.enableDamping = enableDamping;
    controls.dampingFactor = dampingFactor;
    controls.enableZoom = enableZoom;
    controls.enablePan = enablePan;
    controls.autoRotate = autoRotate;
    controls.autoRotateSpeed = autoRotateSpeed;
    controls.minDistance = minDistance;
    controls.maxDistance = maxDistance;
    controls.minPolarAngle = minPolarAngle;
    controls.maxPolarAngle = maxPolarAngle;

    // Touch controls optimization
    controls.touches = {
      ONE: 2, // TOUCH_ROTATE
      TWO: 1, // TOUCH_DOLLY_PAN
    };

    return () => {
      // Cleanup si nécessaire
    };
  }, [
    controlsRef,
    enableDamping,
    dampingFactor,
    enableZoom,
    enablePan,
    autoRotate,
    autoRotateSpeed,
    minDistance,
    maxDistance,
    minPolarAngle,
    maxPolarAngle,
  ]);
}

/**
 * Hook pour gérer les interactions hover (rotation douce)
 */
export function use3DHoverRotation(
  enabled: boolean = true,
  sensitivity: number = 0.002
) {
  const rotationRef = useRef({ x: 0, y: 0 });
  const targetRef = useRef({ x: 0, y: 0 });

  useEffect(() => {
    if (!enabled) return;

    const handleMouseMove = (e: MouseEvent) => {
      // Normaliser coordonnées (-1 à 1)
      const x = (e.clientX / window.innerWidth) * 2 - 1;
      const y = -(e.clientY / window.innerHeight) * 2 + 1;

      targetRef.current = {
        x: y * sensitivity * 10,
        y: x * sensitivity * 10,
      };
    };

    window.addEventListener('mousemove', handleMouseMove);

    return () => {
      window.removeEventListener('mousemove', handleMouseMove);
    };
  }, [enabled, sensitivity]);

  // Lerp pour smooth rotation
  const updateRotation = (delta: number) => {
    const lerpFactor = 1 - Math.pow(0.001, delta);

    rotationRef.current.x +=
      (targetRef.current.x - rotationRef.current.x) * lerpFactor;
    rotationRef.current.y +=
      (targetRef.current.y - rotationRef.current.y) * lerpFactor;

    return rotationRef.current;
  };

  return { updateRotation, rotation: rotationRef.current };
}

/**
 * Hook pour smooth camera transitions
 */
export function useSmoothCamera() {
  const targetPosition = useRef({ x: 0, y: 0, z: 5 });
  const currentPosition = useRef({ x: 0, y: 0, z: 5 });

  const setTarget = (x: number, y: number, z: number) => {
    targetPosition.current = { x, y, z };
  };

  const updateCamera = (camera: any, delta: number) => {
    const lerpFactor = 1 - Math.pow(0.001, delta);

    currentPosition.current.x +=
      (targetPosition.current.x - currentPosition.current.x) * lerpFactor;
    currentPosition.current.y +=
      (targetPosition.current.y - currentPosition.current.y) * lerpFactor;
    currentPosition.current.z +=
      (targetPosition.current.z - currentPosition.current.z) * lerpFactor;

    camera.position.set(
      currentPosition.current.x,
      currentPosition.current.y,
      currentPosition.current.z
    );
  };

  return { setTarget, updateCamera };
}

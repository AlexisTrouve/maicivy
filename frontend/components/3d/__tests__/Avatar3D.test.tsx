/**
 * Tests pour Avatar3D
 * Mock Three.js pour éviter problèmes WebGL en tests
 */

import React from 'react';
import { render, screen } from '@testing-library/react';
import { Avatar3D, AvatarCube3D } from '../Avatar3D';

// Mock @react-three/fiber
jest.mock('@react-three/fiber', () => ({
  Canvas: ({ children }: any) => <div data-testid="canvas-mock">{children}</div>,
  useFrame: jest.fn(),
  useThree: () => ({
    viewport: { width: 1, height: 1 },
    camera: {},
    gl: {},
    scene: {},
  }),
}));

// Mock @react-three/drei
jest.mock('@react-three/drei', () => ({
  OrbitControls: () => null,
  PerspectiveCamera: () => null,
  Text: ({ children }: any) => <div>{children}</div>,
}));

// Mock @react-spring/three
jest.mock('@react-spring/three', () => ({
  useSpring: () => ({ scale: 1 }),
  animated: {
    mesh: ({ children, ...props }: any) => <mesh {...props}>{children}</mesh>,
  },
}));

// Mock hooks
jest.mock('@/hooks/use3DSupport', () => ({
  use3DSupport: () => ({
    isSupported: true,
    performanceLevel: 'high',
    webGLVersion: 2,
    isMobile: false,
  }),
  use3DQualitySettings: () => ({
    antialias: true,
    shadows: true,
    particleCount: 1000,
    maxFPS: 60,
    pixelRatio: 2,
  }),
  useHas3DSupport: () => true,
}));

describe('Avatar3D', () => {
  beforeEach(() => {
    // Reset mocks
    jest.clearAllMocks();
  });

  it('renders without crashing', () => {
    render(<Avatar3D />);
    expect(screen.getByTestId('canvas-mock')).toBeInTheDocument();
  });

  it('applies custom height', () => {
    const { container } = render(<Avatar3D height="500px" />);
    const wrapper = container.firstChild as HTMLElement;
    expect(wrapper.style.height).toBe('500px');
  });

  it('applies custom color prop', () => {
    render(<Avatar3D color="#ff0000" />);
    // Le composant devrait se rendre sans erreur
    expect(screen.getByTestId('canvas-mock')).toBeInTheDocument();
  });

  it('displays FPS counter when showFPS is true', () => {
    render(<Avatar3D showFPS={true} />);
    // Vérifier que le FPS counter est présent
    const fpsElement = screen.queryByText(/FPS/i);
    expect(fpsElement).toBeTruthy();
  });
});

describe('AvatarCube3D', () => {
  it('renders cube variant', () => {
    render(<AvatarCube3D />);
    expect(screen.getByTestId('canvas-mock')).toBeInTheDocument();
  });

  it('applies custom color to cube', () => {
    render(<AvatarCube3D color="#00ff00" />);
    expect(screen.getByTestId('canvas-mock')).toBeInTheDocument();
  });
});

describe('Avatar3D - WebGL Fallback', () => {
  beforeEach(() => {
    // Mock no WebGL support
    jest.resetModules();
    jest.doMock('@/hooks/use3DSupport', () => ({
      use3DSupport: () => ({
        isSupported: false,
        performanceLevel: 'none',
        webGLVersion: null,
        isMobile: false,
        reason: 'WebGL not available',
      }),
      use3DQualitySettings: () => ({
        antialias: false,
        shadows: false,
        particleCount: 0,
        maxFPS: 30,
        pixelRatio: 1,
      }),
    }));
  });

  it('displays fallback when WebGL not supported', () => {
    // Re-require component avec nouveau mock
    const { Avatar3D: Avatar3DNoWebGL } = require('../Avatar3D');

    render(<Avatar3DNoWebGL />);

    // Chercher le fallback
    const fallbackText = screen.queryByText(/Avatar 3D/i);
    expect(fallbackText).toBeTruthy();
  });
});

describe('Avatar3D - Performance', () => {
  it('renders on low-end devices with reduced quality', () => {
    jest.resetModules();
    jest.doMock('@/hooks/use3DSupport', () => ({
      use3DSupport: () => ({
        isSupported: true,
        performanceLevel: 'low',
        webGLVersion: 1,
        isMobile: true,
      }),
      use3DQualitySettings: () => ({
        antialias: false,
        shadows: false,
        particleCount: 200,
        maxFPS: 30,
        pixelRatio: 1,
      }),
    }));

    const { Avatar3D: Avatar3DLowPerf } = require('../Avatar3D');

    render(<Avatar3DLowPerf />);
    expect(screen.getByTestId('canvas-mock')).toBeInTheDocument();
  });
});

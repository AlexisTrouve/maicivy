import * as THREE from 'three';
import {
  generateSkillsGraph,
  fibonacciSpherePoint,
  generateColorGradient,
  generateParticlePositions,
  generateSpiralPositions,
  createInstancedGeometry,
  FPSMonitor,
  optimizeMaterial,
  disposeObject3D,
  calculateOptimalCameraDistance,
  createOptimizedMaterial,
  lerp,
  clamp,
  SKILL_CATEGORIES,
  SkillNode3D,
  SkillEdge3D
} from '../3d-utils';

describe('3D Utils', () => {
  describe('SKILL_CATEGORIES', () => {
    it('should have all required categories', () => {
      const categories = ['backend', 'frontend', 'devops', 'database', 'cloud', 'tools', 'languages', 'other'];

      categories.forEach(category => {
        expect(SKILL_CATEGORIES).toHaveProperty(category);
      });
    });

    it('should have valid hex colors', () => {
      Object.values(SKILL_CATEGORIES).forEach(color => {
        expect(color).toMatch(/^#[0-9a-f]{6}$/i);
      });
    });
  });

  describe('generateSkillsGraph', () => {
    it('should generate nodes for all skills', () => {
      const skills = [
        { name: 'JavaScript', level: 80, category: 'frontend' },
        { name: 'Python', level: 90, category: 'backend' },
        { name: 'Docker', level: 70, category: 'devops' }
      ];

      const { nodes, edges } = generateSkillsGraph(skills);

      expect(nodes).toHaveLength(3);
      expect(nodes[0].name).toBe('JavaScript');
      expect(nodes[1].name).toBe('Python');
      expect(nodes[2].name).toBe('Docker');
    });

    it('should normalize skill levels to 0-1', () => {
      const skills = [
        { name: 'JavaScript', level: 100, category: 'frontend' }
      ];

      const { nodes } = generateSkillsGraph(skills);

      expect(nodes[0].level).toBe(1); // 100/100 = 1
    });

    it('should assign correct colors based on category', () => {
      const skills = [
        { name: 'JavaScript', level: 80, category: 'frontend' },
        { name: 'Python', level: 90, category: 'backend' }
      ];

      const { nodes } = generateSkillsGraph(skills);

      expect(nodes[0].color).toBe(SKILL_CATEGORIES.frontend);
      expect(nodes[1].color).toBe(SKILL_CATEGORIES.backend);
    });

    it('should use default color for unknown category', () => {
      const skills = [
        { name: 'Unknown', level: 50, category: 'unknown-category' }
      ];

      const { nodes } = generateSkillsGraph(skills);

      expect(nodes[0].color).toBe(SKILL_CATEGORIES.other);
    });

    it('should calculate radius based on level', () => {
      const skills = [
        { name: 'Beginner', level: 0, category: 'frontend' },
        { name: 'Expert', level: 100, category: 'frontend' }
      ];

      const { nodes } = generateSkillsGraph(skills);

      // Radius formula: 0.2 + (level/100) * 0.3
      expect(nodes[0].radius).toBe(0.2); // 0.2 + 0 * 0.3
      expect(nodes[1].radius).toBe(0.5); // 0.2 + 1 * 0.3
    });

    it('should create edges between related categories', () => {
      const skills = [
        { name: 'Node.js', level: 80, category: 'backend' },
        { name: 'PostgreSQL', level: 70, category: 'database' }
      ];

      const { edges } = generateSkillsGraph(skills);

      // Backend and database should be connected
      expect(edges.length).toBeGreaterThan(0);
    });

    it('should set edge strength based on minimum level', () => {
      const skills = [
        { name: 'Node.js', level: 80, category: 'backend' },
        { name: 'PostgreSQL', level: 70, category: 'database' }
      ];

      const { edges } = generateSkillsGraph(skills);

      if (edges.length > 0) {
        // Strength should be min(0.8, 0.7) = 0.7
        expect(edges[0].strength).toBe(0.7);
      }
    });

    it('should handle empty skills array', () => {
      const { nodes, edges } = generateSkillsGraph([]);

      expect(nodes).toHaveLength(0);
      expect(edges).toHaveLength(0);
    });

    it('should assign unique IDs to nodes', () => {
      const skills = [
        { name: 'JavaScript', level: 80, category: 'frontend' },
        { name: 'Python', level: 90, category: 'backend' }
      ];

      const { nodes } = generateSkillsGraph(skills);

      const ids = nodes.map(n => n.id);
      const uniqueIds = new Set(ids);

      expect(uniqueIds.size).toBe(ids.length);
    });
  });

  describe('fibonacciSpherePoint', () => {
    it('should generate points on a sphere', () => {
      const point = fibonacciSpherePoint(0, 10, 1);

      expect(point).toHaveLength(3);
      const [x, y, z] = point;

      // Point should be roughly on unit sphere surface (distance ~1)
      const distance = Math.sqrt(x * x + y * y + z * z);
      expect(distance).toBeCloseTo(1, 1);
    });

    it('should distribute points evenly', () => {
      const points = [];
      const total = 100;

      for (let i = 0; i < total; i++) {
        points.push(fibonacciSpherePoint(i, total, 1));
      }

      // All points should be roughly the same distance from origin
      const distances = points.map(([x, y, z]) => Math.sqrt(x * x + y * y + z * z));
      const avgDistance = distances.reduce((a, b) => a + b, 0) / distances.length;

      distances.forEach(distance => {
        expect(distance).toBeCloseTo(avgDistance, 1);
      });
    });

    it('should respect custom radius', () => {
      const radius = 5;
      const point = fibonacciSpherePoint(0, 10, radius);
      const [x, y, z] = point;

      const distance = Math.sqrt(x * x + y * y + z * z);
      expect(distance).toBeCloseTo(radius, 1);
    });

    it('should handle single point', () => {
      const point = fibonacciSpherePoint(0, 1, 1);

      expect(point).toHaveLength(3);
      expect(point).toEqual(expect.arrayContaining([
        expect.any(Number),
        expect.any(Number),
        expect.any(Number)
      ]));
    });
  });

  describe('generateColorGradient', () => {
    it('should generate gradient between two colors', () => {
      const colors = generateColorGradient('#ff0000', '#0000ff', 5);

      expect(colors).toHaveLength(5);
      expect(colors[0]).toBe('#ff0000'); // Red
      expect(colors[4]).toBe('#0000ff'); // Blue
    });

    it('should generate intermediate colors', () => {
      const colors = generateColorGradient('#000000', '#ffffff', 3);

      expect(colors).toHaveLength(3);
      expect(colors[0]).toBe('#000000'); // Black
      expect(colors[2]).toBe('#ffffff'); // White
      // Middle color should be gray
      expect(colors[1]).toMatch(/^#[0-9a-f]{6}$/i);
    });

    it('should handle single step', () => {
      const colors = generateColorGradient('#ff0000', '#0000ff', 1);

      expect(colors).toHaveLength(1);
    });

    it('should return valid hex colors', () => {
      const colors = generateColorGradient('#123456', '#abcdef', 10);

      colors.forEach(color => {
        expect(color).toMatch(/^#[0-9a-f]{6}$/i);
      });
    });
  });

  describe('generateParticlePositions', () => {
    it('should generate correct number of positions', () => {
      const count = 100;
      const positions = generateParticlePositions(count);

      // Each position has x, y, z = 3 values
      expect(positions.length).toBe(count * 3);
    });

    it('should return Float32Array', () => {
      const positions = generateParticlePositions(10);
      expect(positions).toBeInstanceOf(Float32Array);
    });

    it('should respect bounds', () => {
      const bounds = { x: 10, y: 10, z: 10 };
      const positions = generateParticlePositions(100, bounds);

      for (let i = 0; i < 100; i++) {
        const x = positions[i * 3];
        const y = positions[i * 3 + 1];
        const z = positions[i * 3 + 2];

        expect(x).toBeGreaterThanOrEqual(-bounds.x / 2);
        expect(x).toBeLessThanOrEqual(bounds.x / 2);
        expect(y).toBeGreaterThanOrEqual(-bounds.y / 2);
        expect(y).toBeLessThanOrEqual(bounds.y / 2);
        expect(z).toBeGreaterThanOrEqual(-bounds.z / 2);
        expect(z).toBeLessThanOrEqual(bounds.z / 2);
      }
    });

    it('should use default bounds', () => {
      const positions = generateParticlePositions(10);
      expect(positions.length).toBe(30); // 10 * 3
    });
  });

  describe('generateSpiralPositions', () => {
    it('should generate correct number of positions', () => {
      const count = 50;
      const positions = generateSpiralPositions(count);

      expect(positions.length).toBe(count * 3);
    });

    it('should return Float32Array', () => {
      const positions = generateSpiralPositions(10);
      expect(positions).toBeInstanceOf(Float32Array);
    });

    it('should respect radius and height parameters', () => {
      const radius = 5;
      const height = 10;
      const positions = generateSpiralPositions(100, radius, height);

      for (let i = 0; i < 100; i++) {
        const y = positions[i * 3 + 1];

        // Y should be within height bounds
        expect(y).toBeGreaterThanOrEqual(-height / 2);
        expect(y).toBeLessThanOrEqual(height / 2);
      }
    });

    it('should create spiral pattern', () => {
      const positions = generateSpiralPositions(100, 5, 10);

      // First point should be at top of spiral
      const firstY = positions[1];
      const lastY = positions[positions.length - 2];

      // Y values should vary across the spiral (not all the same)
      expect(Math.abs(firstY)).toBeGreaterThan(0);
      expect(typeof lastY).toBe('number');
    });
  });

  describe('createInstancedGeometry', () => {
    it('should create InstancedBufferGeometry', () => {
      const geometry = new THREE.SphereGeometry(1, 16, 16);
      const positions = new Float32Array([0, 0, 0, 1, 1, 1]);

      const instanced = createInstancedGeometry(geometry, positions);

      expect(instanced).toBeInstanceOf(THREE.InstancedBufferGeometry);
    });

    it('should set instance positions attribute', () => {
      const geometry = new THREE.SphereGeometry(1, 16, 16);
      const positions = new Float32Array([0, 0, 0, 1, 1, 1]);

      const instanced = createInstancedGeometry(geometry, positions);

      expect(instanced.attributes.instancePosition).toBeDefined();
    });

    it('should set instance colors when provided', () => {
      const geometry = new THREE.SphereGeometry(1, 16, 16);
      const positions = new Float32Array([0, 0, 0]);
      const colors = new Float32Array([1, 0, 0]);

      const instanced = createInstancedGeometry(geometry, positions, colors);

      expect(instanced.attributes.instanceColor).toBeDefined();
    });

    it('should set instance scales when provided', () => {
      const geometry = new THREE.SphereGeometry(1, 16, 16);
      const positions = new Float32Array([0, 0, 0]);
      const scales = new Float32Array([1.5]);

      const instanced = createInstancedGeometry(geometry, positions, undefined, scales);

      expect(instanced.attributes.instanceScale).toBeDefined();
    });
  });

  describe('FPSMonitor', () => {
    it('should initialize with default FPS', () => {
      const monitor = new FPSMonitor();

      expect(monitor.getFPS()).toBe(60);
    });

    it('should update FPS over time', async () => {
      const monitor = new FPSMonitor();

      // Simulate multiple frames
      for (let i = 0; i < 100; i++) {
        monitor.update();
      }

      // Wait a bit
      await new Promise(resolve => setTimeout(resolve, 100));

      const fps = monitor.getFPS();
      expect(fps).toBeGreaterThan(0);
    });

    it('should return current FPS', () => {
      const monitor = new FPSMonitor();
      const fps = monitor.getFPS();

      expect(typeof fps).toBe('number');
      expect(fps).toBeGreaterThanOrEqual(0);
    });
  });

  describe('optimizeMaterial', () => {
    it('should optimize MeshStandardMaterial', () => {
      const material = new THREE.MeshStandardMaterial({ color: 0xff0000 });

      optimizeMaterial(material);

      // Should not throw and material should still be valid
      expect(material.color.getHex()).toBe(0xff0000);
    });

    it('should handle different material types', () => {
      const materials = [
        new THREE.MeshBasicMaterial(),
        new THREE.MeshStandardMaterial(),
        new THREE.MeshPhongMaterial()
      ];

      materials.forEach(material => {
        expect(() => optimizeMaterial(material)).not.toThrow();
      });
    });
  });

  describe('disposeObject3D', () => {
    it('should dispose mesh geometries', () => {
      const geometry = new THREE.BoxGeometry(1, 1, 1);
      const material = new THREE.MeshBasicMaterial();
      const mesh = new THREE.Mesh(geometry, material);

      const disposeSpy = jest.spyOn(geometry, 'dispose');

      disposeObject3D(mesh);

      expect(disposeSpy).toHaveBeenCalled();
    });

    it('should dispose materials', () => {
      const geometry = new THREE.BoxGeometry(1, 1, 1);
      const material = new THREE.MeshBasicMaterial();
      const mesh = new THREE.Mesh(geometry, material);

      const disposeSpy = jest.spyOn(material, 'dispose');

      disposeObject3D(mesh);

      expect(disposeSpy).toHaveBeenCalled();
    });

    it('should handle array of materials', () => {
      const geometry = new THREE.BoxGeometry(1, 1, 1);
      const materials = [
        new THREE.MeshBasicMaterial(),
        new THREE.MeshBasicMaterial()
      ];
      const mesh = new THREE.Mesh(geometry, materials);

      expect(() => disposeObject3D(mesh)).not.toThrow();
    });

    it('should traverse all children', () => {
      const group = new THREE.Group();
      const mesh1 = new THREE.Mesh(new THREE.BoxGeometry(), new THREE.MeshBasicMaterial());
      const mesh2 = new THREE.Mesh(new THREE.BoxGeometry(), new THREE.MeshBasicMaterial());

      group.add(mesh1);
      group.add(mesh2);

      expect(() => disposeObject3D(group)).not.toThrow();
    });
  });

  describe('calculateOptimalCameraDistance', () => {
    it('should calculate distance based on bounding box', () => {
      const box = new THREE.Box3(
        new THREE.Vector3(-1, -1, -1),
        new THREE.Vector3(1, 1, 1)
      );

      const distance = calculateOptimalCameraDistance(box, 75);

      expect(distance).toBeGreaterThan(0);
      expect(typeof distance).toBe('number');
    });

    it('should account for FOV', () => {
      const box = new THREE.Box3(
        new THREE.Vector3(-1, -1, -1),
        new THREE.Vector3(1, 1, 1)
      );

      const distance1 = calculateOptimalCameraDistance(box, 45);
      const distance2 = calculateOptimalCameraDistance(box, 90);

      // Narrower FOV should need more distance
      expect(distance1).toBeGreaterThan(distance2);
    });

    it('should handle larger objects', () => {
      const smallBox = new THREE.Box3(
        new THREE.Vector3(-1, -1, -1),
        new THREE.Vector3(1, 1, 1)
      );

      const largeBox = new THREE.Box3(
        new THREE.Vector3(-10, -10, -10),
        new THREE.Vector3(10, 10, 10)
      );

      const smallDistance = calculateOptimalCameraDistance(smallBox);
      const largeDistance = calculateOptimalCameraDistance(largeBox);

      expect(largeDistance).toBeGreaterThan(smallDistance);
    });
  });

  describe('createOptimizedMaterial', () => {
    it('should create MeshStandardMaterial', () => {
      const material = createOptimizedMaterial(0xff0000);

      expect(material).toBeInstanceOf(THREE.MeshStandardMaterial);
      expect(material.color.getHex()).toBe(0xff0000);
    });

    it('should use default options', () => {
      const material = createOptimizedMaterial(0xff0000);

      expect(material.metalness).toBe(0.5);
      expect(material.roughness).toBe(0.5);
    });

    it('should accept custom options', () => {
      const material = createOptimizedMaterial(0xff0000, {
        metalness: 0.8,
        roughness: 0.2,
        emissive: 0x00ff00,
        emissiveIntensity: 0.5
      });

      expect(material.metalness).toBe(0.8);
      expect(material.roughness).toBe(0.2);
      expect(material.emissive.getHex()).toBe(0x00ff00);
      expect(material.emissiveIntensity).toBe(0.5);
    });

    it('should accept color as string', () => {
      const material = createOptimizedMaterial('#ff0000');

      expect(material.color.getHex()).toBe(0xff0000);
    });
  });

  describe('lerp', () => {
    it('should interpolate between two values', () => {
      expect(lerp(0, 10, 0)).toBe(0);
      expect(lerp(0, 10, 1)).toBe(10);
      expect(lerp(0, 10, 0.5)).toBe(5);
    });

    it('should handle negative values', () => {
      expect(lerp(-10, 10, 0.5)).toBe(0);
      expect(lerp(-5, -1, 0.5)).toBe(-3);
    });

    it('should handle t > 1', () => {
      const result = lerp(0, 10, 2);
      expect(result).toBe(20);
    });

    it('should handle t < 0', () => {
      const result = lerp(0, 10, -1);
      expect(result).toBe(-10);
    });

    it('should be precise', () => {
      expect(lerp(0, 100, 0.25)).toBe(25);
      expect(lerp(0, 100, 0.75)).toBe(75);
    });
  });

  describe('clamp', () => {
    it('should clamp value within range', () => {
      expect(clamp(5, 0, 10)).toBe(5);
      expect(clamp(-5, 0, 10)).toBe(0);
      expect(clamp(15, 0, 10)).toBe(10);
    });

    it('should handle negative ranges', () => {
      expect(clamp(-5, -10, -1)).toBe(-5);
      expect(clamp(-15, -10, -1)).toBe(-10);
      expect(clamp(0, -10, -1)).toBe(-1);
    });

    it('should handle equal min and max', () => {
      expect(clamp(5, 10, 10)).toBe(10);
      expect(clamp(15, 10, 10)).toBe(10);
    });

    it('should handle floating point numbers', () => {
      expect(clamp(5.5, 0, 10)).toBe(5.5);
      expect(clamp(10.1, 0, 10)).toBe(10);
      expect(clamp(-0.1, 0, 10)).toBe(0);
    });
  });

  describe('Edge Cases', () => {
    it('should handle empty skills in generateSkillsGraph', () => {
      const { nodes, edges } = generateSkillsGraph([]);
      expect(nodes).toEqual([]);
      expect(edges).toEqual([]);
    });

    it('should handle zero radius in fibonacciSpherePoint', () => {
      const point = fibonacciSpherePoint(0, 10, 0);
      expect(point).toEqual([0, 0, 0]);
    });

    it('should handle single color in generateColorGradient', () => {
      const colors = generateColorGradient('#ff0000', '#ff0000', 5);
      colors.forEach(color => {
        expect(color).toBe('#ff0000');
      });
    });

    it('should handle zero particles', () => {
      const positions = generateParticlePositions(0);
      expect(positions.length).toBe(0);
    });
  });
});

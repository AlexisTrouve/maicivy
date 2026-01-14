// Mock for three.js library
export class Vector3 {
  x: number;
  y: number;
  z: number;

  constructor(x = 0, y = 0, z = 0) {
    this.x = x;
    this.y = y;
    this.z = z;
  }

  set(x: number, y: number, z: number) {
    this.x = x;
    this.y = y;
    this.z = z;
    return this;
  }
}

export class Color {
  private r: number = 0;
  private g: number = 0;
  private b: number = 0;

  constructor(color?: string | number) {
    if (typeof color === 'string') {
      this.setHex(parseInt(color.replace('#', ''), 16));
    } else if (typeof color === 'number') {
      this.setHex(color);
    }
  }

  setHex(hex: number) {
    this.r = ((hex >> 16) & 255) / 255;
    this.g = ((hex >> 8) & 255) / 255;
    this.b = (hex & 255) / 255;
    return this;
  }

  getHex(): number {
    return (
      ((Math.round(this.r * 255) << 16) ^
      (Math.round(this.g * 255) << 8) ^
      Math.round(this.b * 255))
    );
  }

  getHexString(): string {
    return ('000000' + this.getHex().toString(16)).slice(-6);
  }

  lerpColors(color1: Color, color2: Color, alpha: number) {
    this.r = color1.r + (color2.r - color1.r) * alpha;
    this.g = color1.g + (color2.g - color1.g) * alpha;
    this.b = color1.b + (color2.b - color1.b) * alpha;
    return this;
  }
}

export class Box3 {
  min: Vector3;
  max: Vector3;

  constructor(min?: Vector3, max?: Vector3) {
    this.min = min || new Vector3();
    this.max = max || new Vector3();
  }

  getSize(target: Vector3): Vector3 {
    target.x = this.max.x - this.min.x;
    target.y = this.max.y - this.min.y;
    target.z = this.max.z - this.min.z;
    return target;
  }
}

export class BufferGeometry {
  index: any = null;
  attributes: any = {};

  dispose() {}
}

export class BoxGeometry extends BufferGeometry {
  constructor(width?: number, height?: number, depth?: number) {
    super();
  }
}

export class SphereGeometry extends BufferGeometry {
  constructor(radius?: number, widthSegments?: number, heightSegments?: number) {
    super();
  }
}

export class Material {
  dispose() {}
}

export class MeshBasicMaterial extends Material {
  color: Color;

  constructor(params?: any) {
    super();
    this.color = new Color(params?.color || 0xffffff);
  }
}

export class MeshStandardMaterial extends Material {
  color: Color;
  metalness: number;
  roughness: number;
  emissive: Color;
  emissiveIntensity: number;
  flatShading: boolean = false;

  constructor(params?: any) {
    super();
    this.color = new Color(params?.color || 0xffffff);
    this.metalness = params?.metalness ?? 0.5;
    this.roughness = params?.roughness ?? 0.5;
    this.emissive = new Color(params?.emissive || 0x000000);
    this.emissiveIntensity = params?.emissiveIntensity ?? 0;
  }
}

export class MeshPhongMaterial extends Material {
  constructor(params?: any) {
    super();
  }
}

export class Object3D {
  children: Object3D[] = [];

  add(object: Object3D) {
    this.children.push(object);
    return this;
  }

  traverse(callback: (object: Object3D) => void) {
    callback(this);
    this.children.forEach(child => child.traverse(callback));
  }
}

export class Mesh extends Object3D {
  geometry: BufferGeometry;
  material: Material | Material[];

  constructor(geometry: BufferGeometry, material: Material | Material[]) {
    super();
    this.geometry = geometry;
    this.material = material;
  }
}

export class Group extends Object3D {
  constructor() {
    super();
  }
}

export class InstancedBufferGeometry extends BufferGeometry {
  setAttribute(name: string, attribute: any) {
    this.attributes[name] = attribute;
  }
}

export class InstancedBufferAttribute {
  constructor(array: Float32Array, itemSize: number) {}
}

export const FrontSide = 0;

// Export default object
export default {
  Vector3,
  Color,
  Box3,
  BufferGeometry,
  BoxGeometry,
  SphereGeometry,
  Material,
  MeshBasicMaterial,
  MeshStandardMaterial,
  MeshPhongMaterial,
  Object3D,
  Mesh,
  Group,
  InstancedBufferGeometry,
  InstancedBufferAttribute,
  FrontSide
};

// Mock jest GLOBAL de framer-motion.
//
// POURQUOI : les composants utilisent motion.div, motion.button, motion.span… + AnimatePresence. Les
// mocks inline des tests ne fournissaient que motion.div → `Element type is invalid: undefined` dès
// qu'un motion.button (ou AnimatePresence) est rendu. Ce mock couvre TOUT via un Proxy, et strippe les
// props framer-only pour éviter les warnings DOM.
import React from 'react';

// eslint-disable-next-line @typescript-eslint/no-explicit-any
const FRAMER_PROPS = new Set([
  'initial', 'animate', 'exit', 'transition', 'variants', 'whileHover', 'whileTap', 'whileFocus',
  'whileInView', 'whileDrag', 'layout', 'layoutId', 'viewport', 'drag', 'dragConstraints',
  'onAnimationStart', 'onAnimationComplete', 'custom', 'style',
]);

function makeComponent(tag: string) {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  return React.forwardRef(function MotionMock({ children, ...props }: any, ref: any) {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const clean: any = {};
    for (const k of Object.keys(props)) {
      if (!FRAMER_PROPS.has(k)) clean[k] = props[k];
    }
    // `style` peut porter du legit (fontSize…) — on le garde s'il est présent et plain.
    if (props.style && typeof props.style === 'object') clean.style = props.style;
    return React.createElement(tag, { ...clean, ref }, children);
  });
}

// motion.<n'importe quel tag> → composant passthrough.
export const motion = new Proxy(
  {},
  {
    get: (_target, tag: string) => makeComponent(tag),
  }
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
) as any;

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export const AnimatePresence = ({ children }: any) => <>{children}</>;
export const useAnimation = () => ({ start: () => Promise.resolve(), stop: () => {}, set: () => {} });
export const useInView = () => true;
export const useReducedMotion = () => false;
export const useScroll = () => ({ scrollYProgress: { on: () => () => {}, get: () => 0 } });
// eslint-disable-next-line @typescript-eslint/no-explicit-any
export const useTransform = () => 0;
// eslint-disable-next-line @typescript-eslint/no-explicit-any
export const useMotionValue = (v: any) => ({ get: () => v, set: () => {}, on: () => () => {} });

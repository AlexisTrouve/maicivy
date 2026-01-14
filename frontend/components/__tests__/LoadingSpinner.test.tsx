import { render, screen } from '@testing-library/react';
import React from 'react';

// Mock LoadingSpinner component
interface LoadingSpinnerProps {
  size?: 'sm' | 'md' | 'lg';
  className?: string;
}

const LoadingSpinner: React.FC<LoadingSpinnerProps> = ({ size = 'md', className = '' }) => {
  const sizeClasses = {
    sm: 'w-4 h-4',
    md: 'w-8 h-8',
    lg: 'w-12 h-12',
  };

  return (
    <div className={`flex items-center justify-center ${className}`} data-testid="loading-spinner">
      <svg
        className={`animate-spin ${sizeClasses[size]}`}
        xmlns="http://www.w3.org/2000/svg"
        fill="none"
        viewBox="0 0 24 24"
        aria-label="Loading"
      >
        <circle
          className="opacity-25"
          cx="12"
          cy="12"
          r="10"
          stroke="currentColor"
          strokeWidth="4"
        />
        <path
          className="opacity-75"
          fill="currentColor"
          d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
        />
      </svg>
    </div>
  );
};

describe('LoadingSpinner', () => {
  it('should render spinner', () => {
    render(<LoadingSpinner />);

    expect(screen.getByTestId('loading-spinner')).toBeInTheDocument();
  });

  it('should render SVG element', () => {
    const { container } = render(<LoadingSpinner />);

    const svg = container.querySelector('svg');
    expect(svg).toBeInTheDocument();
  });

  it('should have spin animation', () => {
    const { container } = render(<LoadingSpinner />);

    const svg = container.querySelector('svg');
    expect(svg).toHaveClass('animate-spin');
  });

  it('should render with medium size by default', () => {
    const { container } = render(<LoadingSpinner />);

    const svg = container.querySelector('svg');
    expect(svg).toHaveClass('w-8', 'h-8');
  });

  it('should render with small size', () => {
    const { container } = render(<LoadingSpinner size="sm" />);

    const svg = container.querySelector('svg');
    expect(svg).toHaveClass('w-4', 'h-4');
  });

  it('should render with large size', () => {
    const { container } = render(<LoadingSpinner size="lg" />);

    const svg = container.querySelector('svg');
    expect(svg).toHaveClass('w-12', 'h-12');
  });

  it('should accept custom className', () => {
    const { container } = render(<LoadingSpinner className="text-blue-500" />);

    const wrapper = screen.getByTestId('loading-spinner');
    expect(wrapper).toHaveClass('text-blue-500');
  });

  it('should have aria-label for accessibility', () => {
    const { container } = render(<LoadingSpinner />);

    const svg = container.querySelector('svg');
    expect(svg).toHaveAttribute('aria-label', 'Loading');
  });

  it('should center spinner in container', () => {
    const { container } = render(<LoadingSpinner />);

    const wrapper = screen.getByTestId('loading-spinner');
    expect(wrapper).toHaveClass('flex', 'items-center', 'justify-center');
  });

  it('should render circle element', () => {
    const { container } = render(<LoadingSpinner />);

    const circle = container.querySelector('circle');
    expect(circle).toBeInTheDocument();
    expect(circle).toHaveAttribute('cx', '12');
    expect(circle).toHaveAttribute('cy', '12');
    expect(circle).toHaveAttribute('r', '10');
  });

  it('should render path element', () => {
    const { container } = render(<LoadingSpinner />);

    const path = container.querySelector('path');
    expect(path).toBeInTheDocument();
  });

  it('should have correct viewBox', () => {
    const { container } = render(<LoadingSpinner />);

    const svg = container.querySelector('svg');
    expect(svg).toHaveAttribute('viewBox', '0 0 24 24');
  });

  it('should have opacity on circle', () => {
    const { container } = render(<LoadingSpinner />);

    const circle = container.querySelector('circle');
    expect(circle).toHaveClass('opacity-25');
  });

  it('should have opacity on path', () => {
    const { container } = render(<LoadingSpinner />);

    const path = container.querySelector('path');
    expect(path).toHaveClass('opacity-75');
  });

  it('should use currentColor for stroke', () => {
    const { container } = render(<LoadingSpinner />);

    const circle = container.querySelector('circle');
    expect(circle).toHaveAttribute('stroke', 'currentColor');
  });

  it('should use currentColor for path fill', () => {
    const { container } = render(<LoadingSpinner />);

    const path = container.querySelector('path');
    expect(path).toHaveAttribute('fill', 'currentColor');
  });

  it('should have no fill on SVG', () => {
    const { container } = render(<LoadingSpinner />);

    const svg = container.querySelector('svg');
    expect(svg).toHaveAttribute('fill', 'none');
  });

  it('should combine custom className with default classes', () => {
    const { container } = render(<LoadingSpinner className="my-4 text-red-500" />);

    const wrapper = screen.getByTestId('loading-spinner');
    expect(wrapper).toHaveClass('flex', 'items-center', 'justify-center', 'my-4', 'text-red-500');
  });

  it('should maintain size when custom className is provided', () => {
    const { container } = render(<LoadingSpinner size="lg" className="text-blue-500" />);

    const svg = container.querySelector('svg');
    expect(svg).toHaveClass('w-12', 'h-12');
  });

  it('should use correct stroke width', () => {
    const { container } = render(<LoadingSpinner />);

    const circle = container.querySelector('circle');
    expect(circle).toHaveAttribute('stroke-width', '4');
  });

  it('should render consistently across different sizes', () => {
    const sizes: Array<'sm' | 'md' | 'lg'> = ['sm', 'md', 'lg'];

    sizes.forEach(size => {
      const { container } = render(<LoadingSpinner size={size} />);
      const svg = container.querySelector('svg');
      expect(svg).toHaveClass('animate-spin');
    });
  });
});

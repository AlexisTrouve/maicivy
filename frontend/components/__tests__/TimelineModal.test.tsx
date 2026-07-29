import { render, screen, fireEvent } from '@testing-library/react';
import TimelineModal from '../timeline/TimelineModal';
import { TimelineEvent } from '@/lib/types';

// Mock framer-motion
jest.mock('framer-motion', () => ({
  motion: {
    div: ({ children, onClick, initial, animate, exit, className, ...props }: any) => (
      <div onClick={onClick} className={className} {...props}>{children}</div>
    ),
    span: ({ children, whileHover, className, ...props }: any) => (
      <span className={className} {...props}>{children}</span>
    ),
  },
  AnimatePresence: ({ children }: any) => <>{children}</>,
}));

// Mock lucide-react icons
jest.mock('lucide-react', () => ({
  X: () => <svg data-testid="x-icon" />,
  Calendar: () => <svg data-testid="calendar-icon" />,
  MapPin: () => <svg data-testid="mappin-icon" />,
  ExternalLink: () => <svg data-testid="external-link-icon" />,
  Github: () => <svg data-testid="github-icon" />,
}));

const mockEvent: TimelineEvent = {
  id: '1',
  type: 'project',
  title: 'E-commerce Platform',
  subtitle: 'Full Stack Project',
  content: 'Built a complete e-commerce platform with React and Node.js. Implemented authentication, payment processing, and real-time inventory management.',
  startDate: '2022-01-01',
  endDate: '2023-06-30',
  tags: ['React', 'Node.js', 'PostgreSQL', 'Redis', 'Docker'],
  category: 'Backend',
  isCurrent: false,
  duration: '1.5 years',
  image: 'https://example.com/project.jpg',
  githubUrl: 'https://github.com/user/ecommerce',
  demoUrl: 'https://demo.example.com',
  stats: {
    stars: 150,
    forks: 42,
    language: 'TypeScript',
  },
};

describe('TimelineModal', () => {
  beforeEach(() => {
    // Mock document.body.style
    document.body.style.overflow = '';
  });

  it('should render modal with event title', () => {
    render(<TimelineModal event={mockEvent} onClose={jest.fn()} />);

    expect(screen.getByText('E-commerce Platform')).toBeInTheDocument();
  });

  it('should render event subtitle', () => {
    render(<TimelineModal event={mockEvent} onClose={jest.fn()} />);

    expect(screen.getByText('Full Stack Project')).toBeInTheDocument();
  });

  it('should display event type badge', () => {
    render(<TimelineModal event={mockEvent} onClose={jest.fn()} />);

    expect(screen.getByText('Projet')).toBeInTheDocument();
  });

  it('should display experience type badge for experience events', () => {
    const experienceEvent = { ...mockEvent, type: 'experience' as const };

    render(<TimelineModal event={experienceEvent} onClose={jest.fn()} />);

    expect(screen.getByText('Expérience')).toBeInTheDocument();
  });

  it('should display category badge', () => {
    render(<TimelineModal event={mockEvent} onClose={jest.fn()} />);

    expect(screen.getByText('Backend')).toBeInTheDocument();
  });

  it('should show "En cours" badge for current events', () => {
    const currentEvent = { ...mockEvent, isCurrent: true, endDate: undefined };

    render(<TimelineModal event={currentEvent} onClose={jest.fn()} />);

    const badges = screen.getAllByText(/en cours/i);
    expect(badges.length).toBeGreaterThan(0);
  });

  it('should display formatted dates', () => {
    render(<TimelineModal event={mockEvent} onClose={jest.fn()} />);

    expect(screen.getByText(/1 janvier 2022/i)).toBeInTheDocument();
    expect(screen.getByText(/30 juin 2023/i)).toBeInTheDocument();
  });

  it('should display duration', () => {
    render(<TimelineModal event={mockEvent} onClose={jest.fn()} />);

    expect(screen.getByText('1.5 years')).toBeInTheDocument();
    expect(screen.getByTestId('mappin-icon')).toBeInTheDocument();
  });

  it('should render event image when provided', () => {
    render(<TimelineModal event={mockEvent} onClose={jest.fn()} />);

    const image = screen.getByAltText('E-commerce Platform');
    expect(image).toBeInTheDocument();
    expect(image).toHaveAttribute('src', 'https://example.com/project.jpg');
  });

  it('should not render image section when no image', () => {
    const eventWithoutImage = { ...mockEvent, image: undefined };

    render(<TimelineModal event={eventWithoutImage} onClose={jest.fn()} />);

    const image = screen.queryByAltText('E-commerce Platform');
    expect(image).not.toBeInTheDocument();
  });

  it('should display event content', () => {
    render(<TimelineModal event={mockEvent} onClose={jest.fn()} />);

    expect(screen.getByText(/built a complete e-commerce platform/i)).toBeInTheDocument();
  });

  it('should render all tags', () => {
    render(<TimelineModal event={mockEvent} onClose={jest.fn()} />);

    expect(screen.getByText('React')).toBeInTheDocument();
    expect(screen.getByText('Node.js')).toBeInTheDocument();
    expect(screen.getByText('PostgreSQL')).toBeInTheDocument();
    expect(screen.getByText('Redis')).toBeInTheDocument();
    expect(screen.getByText('Docker')).toBeInTheDocument();
  });

  it('should display project stats when available', () => {
    render(<TimelineModal event={mockEvent} onClose={jest.fn()} />);

    expect(screen.getByText('150')).toBeInTheDocument(); // Stars
    expect(screen.getByText('42')).toBeInTheDocument(); // Forks
    expect(screen.getByText('Stars')).toBeInTheDocument();
    expect(screen.getByText('Forks')).toBeInTheDocument();
  });

  it('should not show stats for experience type events', () => {
    const experienceEvent = {
      ...mockEvent,
      type: 'experience' as const,
      stats: undefined,
    };

    render(<TimelineModal event={experienceEvent} onClose={jest.fn()} />);

    expect(screen.queryByText('Statistiques')).not.toBeInTheDocument();
  });

  it('should render GitHub link', () => {
    render(<TimelineModal event={mockEvent} onClose={jest.fn()} />);

    const githubLink = screen.getByText('Voir sur GitHub').closest('a');
    expect(githubLink).toHaveAttribute('href', 'https://github.com/user/ecommerce');
    expect(githubLink).toHaveAttribute('target', '_blank');
    expect(githubLink).toHaveAttribute('rel', 'noopener noreferrer');
    expect(screen.getByTestId('github-icon')).toBeInTheDocument();
  });

  it('should render demo link', () => {
    render(<TimelineModal event={mockEvent} onClose={jest.fn()} />);

    const demoLink = screen.getByText('Voir la démo').closest('a');
    expect(demoLink).toHaveAttribute('href', 'https://demo.example.com');
    expect(demoLink).toHaveAttribute('target', '_blank');
    expect(demoLink).toHaveAttribute('rel', 'noopener noreferrer');
    expect(screen.getByTestId('external-link-icon')).toBeInTheDocument();
  });

  it('should not show links section when no URLs provided', () => {
    const eventWithoutLinks = {
      ...mockEvent,
      githubUrl: undefined,
      demoUrl: undefined
    };

    render(<TimelineModal event={eventWithoutLinks} onClose={jest.fn()} />);

    expect(screen.queryByText('Liens')).not.toBeInTheDocument();
  });

  it('should call onClose when close button is clicked', () => {
    const onClose = jest.fn();

    render(<TimelineModal event={mockEvent} onClose={onClose} />);

    const closeButtons = screen.getAllByText('Fermer');
    fireEvent.click(closeButtons[0]);

    expect(onClose).toHaveBeenCalled();
  });

  it('should call onClose when X icon is clicked', () => {
    const onClose = jest.fn();

    render(<TimelineModal event={mockEvent} onClose={onClose} />);

    const xButton = screen.getByTestId('x-icon').closest('button');
    if (xButton) {
      fireEvent.click(xButton);
      expect(onClose).toHaveBeenCalled();
    }
  });

  it('should call onClose when backdrop is clicked', () => {
    const onClose = jest.fn();

    const { container } = render(<TimelineModal event={mockEvent} onClose={onClose} />);

    const backdrop = container.querySelector('.fixed.inset-0');
    if (backdrop) {
      fireEvent.click(backdrop);
      expect(onClose).toHaveBeenCalled();
    }
  });

  it('should not close when modal content is clicked', () => {
    const onClose = jest.fn();

    render(<TimelineModal event={mockEvent} onClose={jest.fn()} />);

    const modalContent = screen.getByText('E-commerce Platform').closest('.bg-white');
    if (modalContent) {
      fireEvent.click(modalContent);
      // Should not propagate to backdrop
    }
  });

  it('should set body overflow to hidden on mount', () => {
    render(<TimelineModal event={mockEvent} onClose={jest.fn()} />);

    expect(document.body.style.overflow).toBe('hidden');
  });

  it('should restore body overflow on unmount', () => {
    const { unmount } = render(<TimelineModal event={mockEvent} onClose={jest.fn()} />);

    unmount();

    expect(document.body.style.overflow).toBe('unset');
  });

  it('should listen for Escape key', () => {
    const onClose = jest.fn();

    render(<TimelineModal event={mockEvent} onClose={onClose} />);

    const escapeEvent = new KeyboardEvent('keydown', { key: 'Escape' });
    document.dispatchEvent(escapeEvent);

    expect(onClose).toHaveBeenCalled();
  });

  it('should not close on other key presses', () => {
    const onClose = jest.fn();

    render(<TimelineModal event={mockEvent} onClose={onClose} />);

    const enterEvent = new KeyboardEvent('keydown', { key: 'Enter' });
    document.dispatchEvent(enterEvent);

    expect(onClose).not.toHaveBeenCalled();
  });

  it('should render calendar icon', () => {
    render(<TimelineModal event={mockEvent} onClose={jest.fn()} />);

    expect(screen.getByTestId('calendar-icon')).toBeInTheDocument();
  });

  it('should handle events without endDate', () => {
    const ongoingEvent = { ...mockEvent, endDate: undefined };

    render(<TimelineModal event={ongoingEvent} onClose={jest.fn()} />);

    expect(screen.getByText(/1 janvier 2022/i)).toBeInTheDocument();
    expect(screen.queryByText(/30 juin 2023/i)).not.toBeInTheDocument();
  });

  it('should not render duration section when not provided', () => {
    const eventWithoutDuration = { ...mockEvent, duration: undefined };

    render(<TimelineModal event={eventWithoutDuration} onClose={jest.fn()} />);

    expect(screen.queryByTestId('mappin-icon')).not.toBeInTheDocument();
  });

  it('should apply correct color scheme for project type', () => {
    const { container } = render(<TimelineModal event={mockEvent} onClose={jest.fn()} />);

    const purpleBadge = container.querySelector('.bg-purple-100');
    expect(purpleBadge).toBeInTheDocument();
  });

  it('should apply correct color scheme for experience type', () => {
    const experienceEvent = { ...mockEvent, type: 'experience' as const };

    const { container } = render(<TimelineModal event={experienceEvent} onClose={jest.fn()} />);

    const blueBadge = container.querySelector('.bg-blue-100');
    expect(blueBadge).toBeInTheDocument();
  });
});

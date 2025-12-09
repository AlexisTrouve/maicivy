import { render, screen, fireEvent } from '@testing-library/react';
import TimelineItem from '../timeline/TimelineItem';
import { TimelineEvent } from '@/lib/types';

// Mock framer-motion
jest.mock('framer-motion', () => ({
  motion: {
    div: ({ children, animate, whileHover, whileTap, ...props }: any) => (
      <div {...props}>{children}</div>
    ),
  },
}));

// Mock react-intersection-observer
jest.mock('react-intersection-observer', () => ({
  useInView: () => ({
    ref: jest.fn(),
    inView: true,
  }),
}));

// Mock lucide-react icons
jest.mock('lucide-react', () => ({
  Calendar: () => <svg data-testid="calendar-icon" />,
  Briefcase: () => <svg data-testid="briefcase-icon" />,
  Code: () => <svg data-testid="code-icon" />,
  Trophy: () => <svg data-testid="trophy-icon" />,
  Clock: () => <svg data-testid="clock-icon" />,
}));

const mockEvent: TimelineEvent = {
  id: '1',
  type: 'experience',
  title: 'Senior Developer',
  subtitle: 'Tech Corp',
  content: 'Led development team',
  startDate: '2022-01-01',
  endDate: '2023-12-31',
  tags: ['React', 'TypeScript', 'Node.js', 'AWS'],
  category: 'Backend',
  isCurrent: false,
  duration: '2 years',
};

describe('TimelineItem', () => {
  it('should render event title and subtitle', () => {
    render(
      <TimelineItem
        event={mockEvent}
        index={0}
        isSelected={false}
        onSelect={jest.fn()}
      />
    );

    expect(screen.getByText('Senior Developer')).toBeInTheDocument();
    expect(screen.getByText('Tech Corp')).toBeInTheDocument();
  });

  it('should render Briefcase icon for experience type', () => {
    render(
      <TimelineItem
        event={mockEvent}
        index={0}
        isSelected={false}
        onSelect={jest.fn()}
      />
    );

    expect(screen.getByTestId('briefcase-icon')).toBeInTheDocument();
  });

  it('should render Code icon for project type', () => {
    const projectEvent = { ...mockEvent, type: 'project' as const };

    render(
      <TimelineItem
        event={projectEvent}
        index={0}
        isSelected={false}
        onSelect={jest.fn()}
      />
    );

    expect(screen.getByTestId('code-icon')).toBeInTheDocument();
  });

  it('should format dates correctly', () => {
    render(
      <TimelineItem
        event={mockEvent}
        index={0}
        isSelected={false}
        onSelect={jest.fn()}
      />
    );

    expect(screen.getByText(/janv\. 2022/i)).toBeInTheDocument();
    expect(screen.getByText(/déc\. 2023/i)).toBeInTheDocument();
  });

  it('should show "En cours" badge for ongoing events', () => {
    const ongoingEvent = { ...mockEvent, endDate: undefined, isCurrent: true };

    render(
      <TimelineItem
        event={ongoingEvent}
        index={0}
        isSelected={false}
        onSelect={jest.fn()}
      />
    );

    expect(screen.getByText(/en cours/i)).toBeInTheDocument();
  });

  it('should display duration when provided', () => {
    render(
      <TimelineItem
        event={mockEvent}
        index={0}
        isSelected={false}
        onSelect={jest.fn()}
      />
    );

    expect(screen.getByText('2 years')).toBeInTheDocument();
    expect(screen.getByTestId('clock-icon')).toBeInTheDocument();
  });

  it('should render first 5 tags', () => {
    render(
      <TimelineItem
        event={mockEvent}
        index={0}
        isSelected={false}
        onSelect={jest.fn()}
      />
    );

    expect(screen.getByText('React')).toBeInTheDocument();
    expect(screen.getByText('TypeScript')).toBeInTheDocument();
    expect(screen.getByText('Node.js')).toBeInTheDocument();
    expect(screen.getByText('AWS')).toBeInTheDocument();
  });

  it('should show "+X" indicator when more than 5 tags', () => {
    const eventWithManyTags = {
      ...mockEvent,
      tags: ['Tag1', 'Tag2', 'Tag3', 'Tag4', 'Tag5', 'Tag6', 'Tag7'],
    };

    render(
      <TimelineItem
        event={eventWithManyTags}
        index={0}
        isSelected={false}
        onSelect={jest.fn()}
      />
    );

    expect(screen.getByText('+2')).toBeInTheDocument();
  });

  it('should display category badge', () => {
    render(
      <TimelineItem
        event={mockEvent}
        index={0}
        isSelected={false}
        onSelect={jest.fn()}
      />
    );

    expect(screen.getByText('Backend')).toBeInTheDocument();
  });

  it('should call onSelect when clicked', () => {
    const onSelect = jest.fn();

    render(
      <TimelineItem
        event={mockEvent}
        index={0}
        isSelected={false}
        onSelect={onSelect}
      />
    );

    const card = screen.getByText('Senior Developer').closest('div');
    if (card) {
      fireEvent.click(card);
      expect(onSelect).toHaveBeenCalled();
    }
  });

  it('should apply selected styles when isSelected is true', () => {
    const { container } = render(
      <TimelineItem
        event={mockEvent}
        index={0}
        isSelected={true}
        onSelect={jest.fn()}
      />
    );

    const card = container.querySelector('.ring-2');
    expect(card).toBeInTheDocument();
  });

  it('should apply blue color scheme for experience type', () => {
    const { container } = render(
      <TimelineItem
        event={mockEvent}
        index={0}
        isSelected={false}
        onSelect={jest.fn()}
      />
    );

    const blueElement = container.querySelector('.border-blue-500');
    expect(blueElement).toBeInTheDocument();
  });

  it('should apply purple color scheme for project type', () => {
    const projectEvent = { ...mockEvent, type: 'project' as const };

    const { container } = render(
      <TimelineItem
        event={projectEvent}
        index={0}
        isSelected={false}
        onSelect={jest.fn()}
      />
    );

    const purpleElement = container.querySelector('.border-purple-500');
    expect(purpleElement).toBeInTheDocument();
  });

  it('should alternate layout for even/odd indices (desktop)', () => {
    const { container: container0 } = render(
      <TimelineItem
        event={mockEvent}
        index={0}
        isSelected={false}
        onSelect={jest.fn()}
      />
    );

    const { container: container1 } = render(
      <TimelineItem
        event={mockEvent}
        index={1}
        isSelected={false}
        onSelect={jest.fn()}
      />
    );

    // Even index (0) should have flex-row
    const evenContainer = container0.querySelector('.md\\:flex-row');
    expect(evenContainer).toBeInTheDocument();

    // Odd index (1) should have flex-row-reverse
    const oddContainer = container1.querySelector('.md\\:flex-row-reverse');
    expect(oddContainer).toBeInTheDocument();
  });

  it('should show pulsing animation for current events', () => {
    const currentEvent = { ...mockEvent, isCurrent: true, endDate: undefined };

    render(
      <TimelineItem
        event={currentEvent}
        index={0}
        isSelected={false}
        onSelect={jest.fn()}
      />
    );

    // Should render the event
    expect(screen.getByText('Senior Developer')).toBeInTheDocument();
  });

  it('should render calendar icon', () => {
    render(
      <TimelineItem
        event={mockEvent}
        index={0}
        isSelected={false}
        onSelect={jest.fn()}
      />
    );

    expect(screen.getByTestId('calendar-icon')).toBeInTheDocument();
  });

  it('should not show duration when not provided', () => {
    const eventWithoutDuration = { ...mockEvent, duration: undefined };

    render(
      <TimelineItem
        event={eventWithoutDuration}
        index={0}
        isSelected={false}
        onSelect={jest.fn()}
      />
    );

    expect(screen.queryByTestId('clock-icon')).not.toBeInTheDocument();
  });

  it('should handle events with no endDate', () => {
    const ongoingEvent = { ...mockEvent, endDate: undefined };

    render(
      <TimelineItem
        event={ongoingEvent}
        index={0}
        isSelected={false}
        onSelect={jest.fn()}
      />
    );

    expect(screen.getByText(/janv\. 2022/i)).toBeInTheDocument();
  });

  it('should show year in mobile view', () => {
    const { container } = render(
      <TimelineItem
        event={mockEvent}
        index={0}
        isSelected={false}
        onSelect={jest.fn()}
      />
    );

    const yearElement = container.querySelector('.md\\:hidden');
    expect(yearElement).toBeInTheDocument();
    expect(screen.getByText('2022')).toBeInTheDocument();
  });

  it('should apply correct text alignment based on index', () => {
    const { container: container0 } = render(
      <TimelineItem
        event={mockEvent}
        index={0}
        isSelected={false}
        onSelect={jest.fn()}
      />
    );

    const { container: container1 } = render(
      <TimelineItem
        event={mockEvent}
        index={1}
        isSelected={false}
        onSelect={jest.fn()}
      />
    );

    // Even indices should have md:text-right
    const rightAligned = container0.querySelector('.md\\:text-right');
    expect(rightAligned).toBeInTheDocument();

    // Odd indices should have md:text-left
    const leftAligned = container1.querySelector('.md\\:text-left');
    expect(leftAligned).toBeInTheDocument();
  });
});

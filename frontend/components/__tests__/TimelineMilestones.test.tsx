import { render, screen } from '@testing-library/react';
import TimelineMilestones from '../timeline/TimelineMilestones';
import { TimelineMilestone } from '@/lib/types';

// Mock framer-motion
jest.mock('framer-motion', () => ({
  motion: {
    div: ({ children, initial, animate, transition, ...props }: any) => (
      <div {...props}>{children}</div>
    ),
  },
}));

// Mock lucide-react icons
jest.mock('lucide-react', () => ({
  Award: () => <svg data-testid="award-icon" />,
}));

const mockMilestones: TimelineMilestone[] = [
  {
    id: '1',
    title: 'First Professional Role',
    description: 'Started career as a developer',
    date: '2020-01-15',
    icon: '🎉',
    type: 'career',
  },
  {
    id: '2',
    title: 'Master\'s Degree',
    description: 'Completed Computer Science degree',
    date: '2019-06-30',
    icon: '🎓',
    type: 'education',
  },
  {
    id: '3',
    title: 'Open Source Contribution',
    description: 'First major open source project',
    date: '2021-03-10',
    icon: '🚀',
    type: 'project',
  },
  {
    id: '4',
    title: 'Award Recognition',
    description: 'Developer of the year',
    date: '2022-12-01',
    icon: '🏆',
    type: 'achievement',
  },
];

describe('TimelineMilestones', () => {
  it('should not render anything when milestones array is empty', () => {
    const { container } = render(<TimelineMilestones milestones={[]} />);
    expect(container.firstChild).toBeNull();
  });

  it('should render title with award icon', () => {
    render(<TimelineMilestones milestones={mockMilestones} />);

    expect(screen.getByText('Jalons Importants')).toBeInTheDocument();
    expect(screen.getByTestId('award-icon')).toBeInTheDocument();
  });

  it('should render all milestones', () => {
    render(<TimelineMilestones milestones={mockMilestones} />);

    expect(screen.getByText('First Professional Role')).toBeInTheDocument();
    expect(screen.getByText('Master\'s Degree')).toBeInTheDocument();
    expect(screen.getByText('Open Source Contribution')).toBeInTheDocument();
    expect(screen.getByText('Award Recognition')).toBeInTheDocument();
  });

  it('should display milestone descriptions', () => {
    render(<TimelineMilestones milestones={mockMilestones} />);

    expect(screen.getByText('Started career as a developer')).toBeInTheDocument();
    expect(screen.getByText('Completed Computer Science degree')).toBeInTheDocument();
    expect(screen.getByText('First major open source project')).toBeInTheDocument();
    expect(screen.getByText('Developer of the year')).toBeInTheDocument();
  });

  it('should display milestone icons', () => {
    render(<TimelineMilestones milestones={mockMilestones} />);

    expect(screen.getByText('🎉')).toBeInTheDocument();
    expect(screen.getByText('🎓')).toBeInTheDocument();
    expect(screen.getByText('🚀')).toBeInTheDocument();
    expect(screen.getByText('🏆')).toBeInTheDocument();
  });

  it('should format dates correctly', () => {
    render(<TimelineMilestones milestones={mockMilestones} />);

    expect(screen.getByText(/15 janvier 2020/i)).toBeInTheDocument();
    expect(screen.getByText(/30 juin 2019/i)).toBeInTheDocument();
    expect(screen.getByText(/10 mars 2021/i)).toBeInTheDocument();
    expect(screen.getByText(/1 décembre 2022/i)).toBeInTheDocument();
  });

  it('should display type badges', () => {
    render(<TimelineMilestones milestones={mockMilestones} />);

    expect(screen.getByText('career')).toBeInTheDocument();
    expect(screen.getByText('education')).toBeInTheDocument();
    expect(screen.getByText('project')).toBeInTheDocument();
    expect(screen.getByText('achievement')).toBeInTheDocument();
  });

  it('should apply yellow color scheme for achievement type', () => {
    const achievements = mockMilestones.filter(m => m.type === 'achievement');
    const { container } = render(<TimelineMilestones milestones={achievements} />);

    const yellowElement = container.querySelector('.border-yellow-500');
    expect(yellowElement).toBeInTheDocument();
  });

  it('should apply blue color scheme for career type', () => {
    const careers = mockMilestones.filter(m => m.type === 'career');
    const { container } = render(<TimelineMilestones milestones={careers} />);

    const blueElement = container.querySelector('.border-blue-500');
    expect(blueElement).toBeInTheDocument();
  });

  it('should apply green color scheme for education type', () => {
    const education = mockMilestones.filter(m => m.type === 'education');
    const { container } = render(<TimelineMilestones milestones={education} />);

    const greenElement = container.querySelector('.border-green-500');
    expect(greenElement).toBeInTheDocument();
  });

  it('should apply purple color scheme for project type', () => {
    const projects = mockMilestones.filter(m => m.type === 'project');
    const { container } = render(<TimelineMilestones milestones={projects} />);

    const purpleElement = container.querySelector('.border-purple-500');
    expect(purpleElement).toBeInTheDocument();
  });

  it('should use grid layout for responsive design', () => {
    const { container } = render(<TimelineMilestones milestones={mockMilestones} />);

    const grid = container.querySelector('.grid');
    expect(grid).toBeInTheDocument();
    expect(grid).toHaveClass('grid-cols-1', 'md:grid-cols-2', 'lg:grid-cols-3');
  });

  it('should apply hover shadow effect', () => {
    const { container } = render(<TimelineMilestones milestones={mockMilestones} />);

    const cards = container.querySelectorAll('.hover\\:shadow-md');
    expect(cards.length).toBe(mockMilestones.length);
  });

  it('should render with left border accent', () => {
    const { container } = render(<TimelineMilestones milestones={mockMilestones} />);

    const borderedCards = container.querySelectorAll('.border-l-4');
    expect(borderedCards.length).toBe(mockMilestones.length);
  });

  it('should handle single milestone', () => {
    const singleMilestone = [mockMilestones[0]];

    render(<TimelineMilestones milestones={singleMilestone} />);

    expect(screen.getByText('First Professional Role')).toBeInTheDocument();
  });

  it('should handle milestone with unknown type', () => {
    const unknownTypeMilestone: TimelineMilestone[] = [
      {
        id: '5',
        title: 'Unknown Milestone',
        description: 'Some description',
        date: '2023-01-01',
        icon: '❓',
        type: 'other' as any, // Force unknown type
      },
    ];

    const { container } = render(<TimelineMilestones milestones={unknownTypeMilestone} />);

    expect(screen.getByText('Unknown Milestone')).toBeInTheDocument();

    // Should fallback to gray color scheme
    const grayElement = container.querySelector('.border-gray-500');
    expect(grayElement).toBeInTheDocument();
  });

  it('should render icon with pulsing animation wrapper', () => {
    const { container } = render(<TimelineMilestones milestones={[mockMilestones[0]]} />);

    const iconWrapper = container.querySelector('.rounded-full');
    expect(iconWrapper).toBeInTheDocument();
  });

  it('should position type badge in top-right corner', () => {
    const { container } = render(<TimelineMilestones milestones={[mockMilestones[0]]} />);

    const badge = container.querySelector('.absolute.top-2.right-2');
    expect(badge).toBeInTheDocument();
  });
});

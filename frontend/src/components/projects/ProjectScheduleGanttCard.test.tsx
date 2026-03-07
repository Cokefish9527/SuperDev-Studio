import { render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, describe, expect, it, vi } from 'vitest';
import ProjectScheduleGanttCard from './ProjectScheduleGanttCard';
import type { Task } from '../../types';

const buildTask = (index: number): Task => ({
  id: `task-${index}`,
  project_id: 'project-1',
  title: `任务 ${index}`,
  description: `描述 ${index}`,
  status: index % 3 === 0 ? 'done' : index % 2 === 0 ? 'in_progress' : 'todo',
  priority: index % 2 === 0 ? 'high' : 'medium',
  assignee: `owner-${index}`,
  start_date: `2026-03-${String(index).padStart(2, '0')}`,
  due_date: `2026-03-${String(index + 12).padStart(2, '0')}`,
  estimated_days: 13,
  created_at: '2026-03-01T00:00:00Z',
  updated_at: '2026-03-01T00:00:00Z',
});

describe('ProjectScheduleGanttCard', () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('shows empty state when no project is selected', () => {
    render(<ProjectScheduleGanttCard tasks={[]} projectSelected={false} />);

    expect(screen.getByText('请选择一个工作区以查看甘特图。')).toBeInTheDocument();
  });

  it('paginates rows, windows long date ranges, and supports back to top', async () => {
    const scrollToSpy = vi.fn();
    vi.stubGlobal('scrollTo', scrollToSpy);

    render(
      <ProjectScheduleGanttCard
        projectSelected
        tasks={Array.from({ length: 8 }, (_, index) => buildTask(index + 1))}
      />,
    );

    expect(screen.getByText('任务 1-6 / 8')).toBeInTheDocument();
    expect(screen.getByText('日期 1-14 / 20')).toBeInTheDocument();
    expect(screen.getByText('当前日期窗口')).toBeInTheDocument();
    expect(screen.getByText('2026-03-01 - 2026-03-14')).toBeInTheDocument();
    expect(screen.getByText('任务 1 / 2')).toBeInTheDocument();
    expect(screen.getByText('日期 1 / 2')).toBeInTheDocument();
    expect(screen.getByText('任务 6')).toBeInTheDocument();
    expect(screen.queryByText('任务 7')).not.toBeInTheDocument();

    await userEvent.click(screen.getByRole('button', { name: '下一组任务' }));

    expect(screen.getByText('任务 7')).toBeInTheDocument();
    expect(screen.getByText('任务 7-8 / 8')).toBeInTheDocument();
    expect(screen.getByText('任务 2 / 2')).toBeInTheDocument();

    await userEvent.click(screen.getByRole('button', { name: '下一段日期' }));

    expect(screen.getByText('日期 15-20 / 20')).toBeInTheDocument();
    expect(screen.getByText('2026-03-15 - 2026-03-20')).toBeInTheDocument();
    expect(screen.getByText('日期 2 / 2')).toBeInTheDocument();

    const summaryBar = screen.getByText('任务 7-8 / 8').closest('.ant-space');
    expect(summaryBar).not.toBeNull();
    await userEvent.click(within(summaryBar as HTMLElement).getByRole('button', { name: '回到顶部' }));

    expect(scrollToSpy).toHaveBeenCalledWith({ top: 0, behavior: 'smooth' });
  });
});

import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import type { ReactElement } from 'react';
import { ProjectStateProvider } from '../state/project-context';

export function renderWithProviders(ui: ReactElement) {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
      },
    },
  });

  return render(
    <QueryClientProvider client={queryClient}>
      <ProjectStateProvider>
        <BrowserRouter>{ui}</BrowserRouter>
      </ProjectStateProvider>
    </QueryClientProvider>,
  );
}

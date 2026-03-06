import { ConfigProvider, theme } from 'antd';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { BrowserRouter } from 'react-router-dom';
import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import 'antd/dist/reset.css';
import App from './App';
import './index.css';
import { ProjectStateProvider } from './state/project-context';

const queryClient = new QueryClient();

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <ConfigProvider
      theme={{
        algorithm: theme.defaultAlgorithm,
        token: {
          colorPrimary: '#0f766e',
          colorInfo: '#0ea5a4',
          colorSuccess: '#16a34a',
          colorWarning: '#f59e0b',
          borderRadius: 10,
          fontFamily: 'var(--body-font)',
        },
      }}
    >
      <QueryClientProvider client={queryClient}>
        <ProjectStateProvider>
          <BrowserRouter>
            <App />
          </BrowserRouter>
        </ProjectStateProvider>
      </QueryClientProvider>
    </ConfigProvider>
  </StrictMode>,
);

import { Spin } from 'antd';
import { lazy, Suspense, type ComponentType } from 'react';
import { Navigate, Route, Routes } from 'react-router-dom';
import AppShell from './components/AppShell';

const DashboardPage = lazy(() => import('./pages/DashboardPage'));
const ProjectsPage = lazy(() => import('./pages/ProjectsPage'));
const ChangeCenterPage = lazy(() => import('./pages/ChangeCenterPage'));
const PipelinePage = lazy(() => import('./pages/PipelinePage'));
const ProjectSettingsPage = lazy(() => import('./pages/ProjectSettingsPage'));
const ContextHubPage = lazy(() => import('./pages/ContextHubPage'));
const SimpleDeliveryPage = lazy(() => import('./pages/SimpleDeliveryPage'));

function PageFallback() {
  return (
    <div style={{ minHeight: 240, display: 'grid', placeItems: 'center' }}>
      <Spin size="large" description="页面加载中" />
    </div>
  );
}

function renderLazyPage(Component: ComponentType) {
  return (
    <Suspense fallback={<PageFallback />}>
      <Component />
    </Suspense>
  );
}

export default function App() {
  return (
    <Routes>
      <Route element={<AppShell />}>
        <Route path="/" element={renderLazyPage(DashboardPage)} />
        <Route path="/projects" element={renderLazyPage(ProjectsPage)} />
        <Route path="/changes" element={renderLazyPage(ChangeCenterPage)} />
        <Route path="/simple" element={renderLazyPage(SimpleDeliveryPage)} />
        <Route path="/pipeline" element={renderLazyPage(PipelinePage)} />
        <Route path="/settings" element={renderLazyPage(ProjectSettingsPage)} />
        <Route path="/context" element={renderLazyPage(ContextHubPage)} />
        <Route path="/memory" element={<Navigate to="/context?tab=memory" replace />} />
        <Route path="/knowledge" element={<Navigate to="/context?tab=knowledge" replace />} />
      </Route>
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  );
}

import { Navigate, Route, Routes } from 'react-router-dom';
import AppShell from './components/AppShell';
import ContextPage from './pages/ContextPage';
import DashboardPage from './pages/DashboardPage';
import KnowledgePage from './pages/KnowledgePage';
import MemoryPage from './pages/MemoryPage';
import PipelinePage from './pages/PipelinePage';
import ProjectsPage from './pages/ProjectsPage';

export default function App() {
  return (
    <Routes>
      <Route element={<AppShell />}>
        <Route path="/" element={<DashboardPage />} />
        <Route path="/projects" element={<ProjectsPage />} />
        <Route path="/pipeline" element={<PipelinePage />} />
        <Route path="/memory" element={<MemoryPage />} />
        <Route path="/knowledge" element={<KnowledgePage />} />
        <Route path="/context" element={<ContextPage />} />
      </Route>
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  );
}

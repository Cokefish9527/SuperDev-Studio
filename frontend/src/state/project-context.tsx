/* eslint-disable react-refresh/only-export-components */

import { createContext, useContext, useMemo, useState } from 'react';
import type { ReactNode } from 'react';

type ProjectStateContextValue = {
  activeProjectId: string;
  setActiveProjectId: (projectId: string) => void;
};

const ProjectStateContext = createContext<ProjectStateContextValue | undefined>(undefined);

const STORAGE_KEY = 'superdev-studio-active-project';

export function ProjectStateProvider({ children }: { children: ReactNode }) {
  const [activeProjectId, setActiveProjectIdState] = useState<string>(() => localStorage.getItem(STORAGE_KEY) ?? '');

  const setActiveProjectId = (projectId: string) => {
    setActiveProjectIdState(projectId);
    localStorage.setItem(STORAGE_KEY, projectId);
  };

  const value = useMemo(
    () => ({
      activeProjectId,
      setActiveProjectId,
    }),
    [activeProjectId],
  );

  return <ProjectStateContext.Provider value={value}>{children}</ProjectStateContext.Provider>;
}

export function useProjectState() {
  const ctx = useContext(ProjectStateContext);
  if (!ctx) {
    throw new Error('useProjectState must be used within ProjectStateProvider');
  }
  return ctx;
}

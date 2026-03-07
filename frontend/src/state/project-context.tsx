/* eslint-disable react-refresh/only-export-components */

import { createContext, useContext, useMemo, useState } from 'react';
import type { ReactNode } from 'react';

type ProjectStateContextValue = {
  activeProjectId: string;
  activeChangeBatchId: string;
  setActiveProjectId: (projectId: string) => void;
  setActiveChangeBatchId: (changeBatchId: string) => void;
};

const ProjectStateContext = createContext<ProjectStateContextValue | undefined>(undefined);

const STORAGE_KEY = 'superdev-studio-active-project';
const CHANGE_BATCH_STORAGE_KEY = 'superdev-studio-active-change-batch';

export function ProjectStateProvider({ children }: { children: ReactNode }) {
  const [activeProjectId, setActiveProjectIdState] = useState<string>(() => localStorage.getItem(STORAGE_KEY) ?? '');
  const [activeChangeBatchId, setActiveChangeBatchIdState] = useState<string>(
    () => localStorage.getItem(CHANGE_BATCH_STORAGE_KEY) ?? '',
  );

  const setActiveProjectId = (projectId: string) => {
    setActiveProjectIdState(projectId);
    localStorage.setItem(STORAGE_KEY, projectId);
  };

  const setActiveChangeBatchId = (changeBatchId: string) => {
    setActiveChangeBatchIdState(changeBatchId);
    localStorage.setItem(CHANGE_BATCH_STORAGE_KEY, changeBatchId);
  };

  const value = useMemo(
    () => ({
      activeProjectId,
      activeChangeBatchId,
      setActiveProjectId,
      setActiveChangeBatchId,
    }),
    [activeChangeBatchId, activeProjectId],
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

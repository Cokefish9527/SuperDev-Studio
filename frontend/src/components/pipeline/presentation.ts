import type { PipelineStage } from '../../types';

export function stageStatusColor(status?: string) {
  switch (status) {
    case 'completed':
      return 'green';
    case 'failed':
      return 'red';
    case 'missing':
      return 'orange';
    case 'pending':
      return 'default';
    default:
      return 'blue';
  }
}

export function stageBackground(stage: Pick<PipelineStage, 'key'>) {
  switch (stage.key) {
    case 'idea':
      return '#eff6ff';
    case 'design':
      return '#f5f3ff';
    case 'superdev':
      return '#ecfeff';
    case 'output':
      return '#f0fdf4';
    case 'rethink':
      return '#fff7ed';
    default:
      return '#fff';
  }
}

export function stageAccentColor(stageKey?: string) {
  switch (stageKey) {
    case 'idea':
      return '#3b82f6';
    case 'design':
      return '#8b5cf6';
    case 'superdev':
      return '#14b8a6';
    case 'output':
      return '#22c55e';
    case 'rethink':
      return '#f97316';
    default:
      return '#94a3b8';
  }
}

export function stageShadowColor(stageKey?: string) {
  switch (stageKey) {
    case 'idea':
      return 'rgba(59, 130, 246, 0.14)';
    case 'design':
      return 'rgba(139, 92, 246, 0.14)';
    case 'superdev':
      return 'rgba(20, 184, 166, 0.14)';
    case 'output':
      return 'rgba(34, 197, 94, 0.14)';
    case 'rethink':
      return 'rgba(249, 115, 22, 0.16)';
    default:
      return 'rgba(15, 23, 42, 0.08)';
  }
}

export function stageLabel(stage?: string) {
  switch (stage) {
    case 'idea':
      return '构思';
    case 'design':
      return '设计';
    case 'superdev':
      return 'super-dev';
    case 'output':
      return '产出';
    case 'rethink':
      return '再构思';
    default:
      return stage || '-';
  }
}

export function formatFileSize(size: number) {
  if (size < 1024) {
    return `${size} B`;
  }
  if (size < 1024 * 1024) {
    return `${(size / 1024).toFixed(1)} KB`;
  }
  return `${(size / 1024 / 1024).toFixed(1)} MB`;
}

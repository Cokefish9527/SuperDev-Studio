import { Card, Col, Row, Space, Tag, Typography } from 'antd';
import type { PipelineArtifact } from '../types';

const SECTION_METADATA = '文档元数据';
const SECTION_INPUT = '输入快照';
const SECTION_SUMMARY = '执行摘要';
const SECTION_RISKS = '风险与依赖';
const SECTION_ACCEPTANCE = '验收检查点';
const SECTION_NEXT = '下一步动作';
const SECTION_QUESTIONS = '待确认问题';
const SECTION_RAW = 'LLM 原始输出';
const LABEL_TEMPLATE_PREVIEW = '模板化预览';
const LABEL_EMPTY_CONTENT = '暂无内容';
const LABEL_EMPTY_TEXT = '暂无文本内容';
const LABEL_MISSING_SUMMARY = '当前文档未提供执行摘要。';

type TemplateMetadataEntry = {
  label: string;
  value: string;
};

type TemplateTextBlock = {
  paragraphs: string[];
  bullets: string[];
};

type TemplateSubsectionNode = TemplateTextBlock & {
  title: string;
};

type TemplateSectionNode = TemplateTextBlock & {
  title: string;
  subsections: TemplateSubsectionNode[];
  metadata: TemplateMetadataEntry[];
  code: string;
};

type ParsedTemplateArtifact = {
  title: string;
  metadata: TemplateMetadataEntry[];
  sections: TemplateSectionNode[];
};

type Props = {
  artifact: PipelineArtifact;
  content: string;
};

export default function PipelineArtifactTemplatePreview({ artifact, content }: Props) {
  const parsed = parseTemplateArtifact(content);
  if (!parsed) {
    return renderRawTextPreview(content);
  }

  const palette = stagePalette(artifact.stage);
  const summary = buildSummary(parsed.sections);
  const templateKind = metadataValue(parsed.metadata, 'template_kind');
  const visibleSections = parsed.sections.filter(
    (section) => section.title !== SECTION_METADATA && section.title !== SECTION_SUMMARY,
  );

  return (
    <Space orientation="vertical" size="large" style={{ width: '100%' }}>
      <Card
        size="small"
        style={{
          borderRadius: 18,
          borderColor: palette.border,
          background: `linear-gradient(135deg, ${palette.soft} 0%, #ffffff 100%)`,
          boxShadow: `0 18px 40px ${palette.shadow}`,
        }}
      >
        <Space orientation="vertical" size={12} style={{ width: '100%' }}>
          <Space wrap>
            <Tag color={palette.tag}>{artifactStageLabel(artifact.stage)}</Tag>
            <Tag color="purple">{LABEL_TEMPLATE_PREVIEW}</Tag>
            {templateKind ? <Tag>{templateKindLabel(templateKind)}</Tag> : null}
          </Space>
          <div>
            <Typography.Title level={4} style={{ margin: 0 }}>
              {parsed.title || artifact.name}
            </Typography.Title>
            <Typography.Paragraph style={{ margin: '8px 0 0', color: '#475569', fontSize: 15, lineHeight: 1.75 }}>
              {summary}
            </Typography.Paragraph>
          </div>
          <Space wrap>
            {visibleSections
              .filter((section) => section.title !== SECTION_RAW)
              .map((section) => (
                <Tag key={section.title} color="default" style={{ borderRadius: 999, paddingInline: 10 }}>
                  {section.title}
                </Tag>
              ))}
          </Space>
        </Space>
      </Card>

      {parsed.metadata.length > 0 ? (
        <Row gutter={[12, 12]}>
          {parsed.metadata.map((entry) => (
            <Col xs={12} lg={8} xl={6} key={entry.label}>
              <div
                style={{
                  height: '100%',
                  padding: '14px 16px',
                  borderRadius: 16,
                  border: `1px solid ${palette.border}`,
                  background: '#ffffff',
                }}
              >
                <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                  {metadataLabel(entry.label)}
                </Typography.Text>
                <Typography.Paragraph style={{ margin: '6px 0 0', color: '#0f172a', fontWeight: 600, wordBreak: 'break-word' }}>
                  {formatMetadataValue(entry.label, entry.value)}
                </Typography.Paragraph>
                <Typography.Text type="secondary" style={{ fontSize: 11 }}>
                  {entry.label}
                </Typography.Text>
              </div>
            </Col>
          ))}
        </Row>
      ) : null}

      {visibleSections.map((section) => {
        const surface = sectionSurface(section.title);
        return (
          <Card
            key={section.title}
            size="small"
            style={{
              borderRadius: 18,
              borderColor: surface.borderColor,
              background: surface.background,
            }}
          >
            <Space orientation="vertical" size="middle" style={{ width: '100%' }}>
              <Typography.Title level={5} style={{ margin: 0 }}>
                {section.title}
              </Typography.Title>

              {section.paragraphs.map((paragraph) => (
                <Typography.Paragraph key={`${section.title}-${paragraph}`} style={{ margin: 0, color: '#334155', lineHeight: 1.8 }}>
                  {paragraph}
                </Typography.Paragraph>
              ))}

              {section.subsections.length > 0 ? (
                <Row gutter={[12, 12]}>
                  {section.subsections.map((subsection) => (
                    <Col xs={24} lg={12} key={`${section.title}-${subsection.title}`}>
                      <div
                        style={{
                          height: '100%',
                          padding: '14px 16px',
                          borderRadius: 16,
                          border: '1px solid rgba(148, 163, 184, 0.2)',
                          background: '#ffffff',
                        }}
                      >
                        <Typography.Text strong>{subsection.title}</Typography.Text>
                        {subsection.paragraphs.map((paragraph) => (
                          <Typography.Paragraph key={`${subsection.title}-${paragraph}`} style={{ margin: '8px 0 0', color: '#475569' }}>
                            {paragraph}
                          </Typography.Paragraph>
                        ))}
                        {subsection.bullets.length > 0 ? renderBulletRows(subsection.bullets) : null}
                      </div>
                    </Col>
                  ))}
                </Row>
              ) : null}

              {section.bullets.length > 0 ? renderBulletRows(section.bullets) : null}

              {section.code ? (
                <pre
                  style={{
                    margin: 0,
                    whiteSpace: 'pre-wrap',
                    wordBreak: 'break-word',
                    padding: 18,
                    borderRadius: 14,
                    background: '#0f172a',
                    color: '#e2e8f0',
                    maxHeight: 360,
                    overflow: 'auto',
                    fontSize: 12,
                    lineHeight: 1.7,
                  }}
                >
                  {section.code}
                </pre>
              ) : null}

              {section.paragraphs.length === 0 &&
              section.subsections.length === 0 &&
              section.bullets.length === 0 &&
              !section.code ? (
                <Typography.Text type="secondary">{LABEL_EMPTY_CONTENT}</Typography.Text>
              ) : null}
            </Space>
          </Card>
        );
      })}
    </Space>
  );
}

function parseTemplateArtifact(content: string): ParsedTemplateArtifact | null {
  const lines = content.replaceAll('\r', '').split('\n');
  let title = '';
  const sections: TemplateSectionNode[] = [];
  let currentSection: TemplateSectionNode | null = null;
  let currentSubsection: TemplateSubsectionNode | null = null;

  for (const rawLine of lines) {
    const line = rawLine;
    const trimmed = line.trim();

    if (!title && trimmed.startsWith('# ')) {
      title = trimmed.slice(2).trim();
      continue;
    }

    if (trimmed.startsWith('## ')) {
      currentSection = createSection(trimmed.slice(3).trim());
      sections.push(currentSection);
      currentSubsection = null;
      continue;
    }

    if (!currentSection) {
      continue;
    }

    if (currentSection.title === SECTION_RAW) {
      if (trimmed.startsWith('```')) {
        continue;
      }
      if (!trimmed && !currentSection.code) {
        continue;
      }
      currentSection.code = currentSection.code ? `${currentSection.code}\n${line}` : line;
      continue;
    }

    if (currentSection.title === SECTION_METADATA && trimmed.startsWith('|')) {
      const cells = trimmed
        .split('|')
        .map((cell) => cell.trim())
        .filter(Boolean);
      if (cells.length >= 2 && cells[0] !== '字段' && !/^[-: ]+$/.test(cells[0])) {
        currentSection.metadata.push({ label: cells[0], value: cells[1] });
      }
      continue;
    }

    if (trimmed.startsWith('### ')) {
      currentSubsection = createSubsection(trimmed.slice(4).trim());
      currentSection.subsections.push(currentSubsection);
      continue;
    }

    if (trimmed.startsWith('- ')) {
      const target = currentSubsection ?? currentSection;
      target.bullets.push(trimmed.slice(2).trim());
      continue;
    }

    if (!trimmed) {
      continue;
    }

    const target = currentSubsection ?? currentSection;
    target.paragraphs.push(trimmed);
  }

  const metadataSection = sections.find((section) => section.title === SECTION_METADATA);
  const metadata = metadataSection?.metadata ?? [];
  const sectionTitles = new Set(sections.map((section) => section.title));
  if (metadata.length === 0 || !sectionTitles.has(SECTION_INPUT) || !sectionTitles.has(SECTION_SUMMARY)) {
    return null;
  }

  return {
    title,
    metadata,
    sections,
  };
}

function createSection(title: string): TemplateSectionNode {
  return {
    title,
    paragraphs: [],
    bullets: [],
    subsections: [],
    metadata: [],
    code: '',
  };
}

function createSubsection(title: string): TemplateSubsectionNode {
  return {
    title,
    paragraphs: [],
    bullets: [],
  };
}

function renderBulletRows(items: string[]) {
  return (
    <div style={{ display: 'grid', gap: 10 }}>
      {items.map((item) => (
        <div
          key={item}
          style={{
            padding: '12px 14px',
            borderRadius: 14,
            background: '#ffffff',
            border: '1px solid rgba(148, 163, 184, 0.2)',
            color: '#334155',
            lineHeight: 1.7,
          }}
        >
          ? {item}
        </div>
      ))}
    </div>
  );
}

function renderRawTextPreview(content: string) {
  return (
    <pre
      style={{
        margin: 0,
        whiteSpace: 'pre-wrap',
        wordBreak: 'break-word',
        padding: 18,
        borderRadius: 14,
        background: '#0f172a',
        color: '#e2e8f0',
        maxHeight: 520,
        overflow: 'auto',
        fontSize: 13,
        lineHeight: 1.65,
      }}
    >
      {content || LABEL_EMPTY_TEXT}
    </pre>
  );
}

function buildSummary(sections: TemplateSectionNode[]) {
  const summarySection = sections.find((section) => section.title === SECTION_SUMMARY);
  if (!summarySection) {
    return LABEL_MISSING_SUMMARY;
  }
  if (summarySection.paragraphs.length > 0) {
    return summarySection.paragraphs.join(' ');
  }
  if (summarySection.bullets.length > 0) {
    return summarySection.bullets.join('；');
  }
  return LABEL_MISSING_SUMMARY;
}

function metadataValue(metadata: TemplateMetadataEntry[], key: string) {
  return metadata.find((entry) => entry.label === key)?.value ?? '';
}

function metadataLabel(key: string) {
  switch (key) {
    case 'run_id':
      return '运行 ID';
    case 'stage':
      return '阶段';
    case 'template_kind':
      return '模板类型';
    case 'change_id':
      return '变更 ID';
    case 'generated_at':
      return '生成时间';
    case 'multimodal_assets':
      return '多模态素材';
    default:
      return key;
  }
}

function formatMetadataValue(key: string, value: string) {
  if (key === 'template_kind') {
    return templateKindLabel(value);
  }
  if (key === 'stage') {
    return artifactStageLabel(value);
  }
  if (key.endsWith('_at') && value.includes('T')) {
    return value.replace('T', ' ').replace('Z', ' UTC');
  }
  return value || '-';
}

function templateKindLabel(kind: string) {
  switch (kind) {
    case 'concept':
      return '构思模板';
    case 'design':
      return '设计模板';
    case 'reflection':
      return '复盘模板';
    default:
      return kind || '未标记';
  }
}

function artifactStageLabel(stage?: string) {
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
      return stage || '未分配阶段';
  }
}

function stagePalette(stage?: string) {
  switch (stage) {
    case 'idea':
      return { soft: '#eef6ff', border: '#bfdbfe', shadow: 'rgba(59, 130, 246, 0.12)', tag: 'blue' as const };
    case 'design':
      return { soft: '#f5f3ff', border: '#d8b4fe', shadow: 'rgba(139, 92, 246, 0.14)', tag: 'purple' as const };
    case 'superdev':
      return { soft: '#ecfeff', border: '#99f6e4', shadow: 'rgba(20, 184, 166, 0.12)', tag: 'cyan' as const };
    case 'output':
      return { soft: '#f0fdf4', border: '#86efac', shadow: 'rgba(34, 197, 94, 0.12)', tag: 'green' as const };
    case 'rethink':
      return { soft: '#fff7ed', border: '#fdba74', shadow: 'rgba(249, 115, 22, 0.12)', tag: 'orange' as const };
    default:
      return { soft: '#f8fafc', border: '#cbd5e1', shadow: 'rgba(15, 23, 42, 0.08)', tag: 'default' as const };
  }
}

function sectionSurface(title: string) {
  switch (title) {
    case SECTION_INPUT:
      return { background: '#eff6ff', borderColor: '#bfdbfe' };
    case SECTION_RISKS:
      return { background: '#fff7ed', borderColor: '#fdba74' };
    case SECTION_ACCEPTANCE:
      return { background: '#f0fdf4', borderColor: '#86efac' };
    case SECTION_NEXT:
      return { background: '#f5f3ff', borderColor: '#d8b4fe' };
    case SECTION_QUESTIONS:
      return { background: '#fefce8', borderColor: '#fde68a' };
    case SECTION_RAW:
      return { background: '#e2e8f0', borderColor: '#94a3b8' };
    default:
      return { background: '#ffffff', borderColor: '#e2e8f0' };
  }
}

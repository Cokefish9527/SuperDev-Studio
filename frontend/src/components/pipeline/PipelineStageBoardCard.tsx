import { Button, Card, Col, Empty, Row, Space, Tag, Typography } from 'antd';
import type { PipelineArtifact, PipelineCompletion, PipelineStage } from '../../types';
import { stageAccentColor, stageBackground, stageShadowColor, stageStatusColor } from './presentation';

type Props = {
  loading: boolean;
  completionData?: PipelineCompletion;
  stageBoard: PipelineStage[];
  selectedArtifact?: PipelineArtifact;
  previewVisible: boolean;
  onTogglePreview: () => void;
  onSelectArtifact: (path: string) => void;
};

export default function PipelineStageBoardCard({
  loading,
  completionData,
  stageBoard,
  selectedArtifact,
  previewVisible,
  onTogglePreview,
  onSelectArtifact,
}: Props) {
  return (
    <Card
      title="阶段看板"
      loading={loading}
      extra={completionData?.preview_url ? <Button onClick={onTogglePreview}>{previewVisible ? '隐藏预览' : '预览页面'}</Button> : null}
      style={{ borderRadius: 18 }}
    >
      {!completionData ? (
        <Empty description="运行开始后，阶段产物会在这里持续出现" />
      ) : (
        <Space orientation="vertical" size="large" style={{ width: '100%' }}>
          <Typography.Text type="secondary">输出目录：{completionData.output_dir}</Typography.Text>
          <Row gutter={[12, 12]}>
            {stageBoard.map((stage) => {
              const isActiveStage = !!selectedArtifact && stage.artifacts.some((artifact) => artifact.path === selectedArtifact.path);
              return (
                <Col xs={24} sm={12} key={stage.key}>
                  <Card
                    size="small"
                    style={{
                      borderRadius: 18,
                      background: isActiveStage ? `linear-gradient(135deg, ${stageBackground(stage)} 0%, #ffffff 100%)` : stageBackground(stage),
                      borderColor: isActiveStage ? stageAccentColor(stage.key) : '#dbeafe',
                      boxShadow: isActiveStage ? `0 16px 36px ${stageShadowColor(stage.key)}` : 'none',
                      transition: 'all 0.2s ease',
                    }}
                  >
                    <Space orientation="vertical" size={10} style={{ width: '100%' }}>
                      <Space wrap>
                        <Tag color={stageStatusColor(stage.status)}>{stage.title}</Tag>
                        <Tag>{stage.status}</Tag>
                        {isActiveStage ? <Tag color="gold">当前预览</Tag> : null}
                      </Space>
                      <Typography.Text type="secondary">
                        {stage.artifacts.length > 0 ? `共 ${stage.artifacts.length} 个产物` : '等待该阶段产物'}
                      </Typography.Text>
                      <Space wrap size={[8, 8]}>
                        {stage.artifacts.slice(0, 4).map((artifact) => {
                          const isSelectedArtifact = selectedArtifact?.path === artifact.path;
                          return (
                            <Button
                              key={artifact.path}
                              size="small"
                              type={isSelectedArtifact ? 'primary' : 'default'}
                              onClick={() => onSelectArtifact(artifact.path)}
                              style={{
                                borderRadius: 999,
                                borderColor: isSelectedArtifact ? stageAccentColor(stage.key) : undefined,
                              }}
                            >
                              {artifact.name}
                            </Button>
                          );
                        })}
                      </Space>
                    </Space>
                  </Card>
                </Col>
              );
            })}
          </Row>
        </Space>
      )}
    </Card>
  );
}

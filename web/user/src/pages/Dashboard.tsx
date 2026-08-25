import { useEffect, useState } from 'react';
import { Row, Col, Card, Statistic } from 'antd';
import { SendOutlined, CheckCircleOutlined, CloseCircleOutlined, RiseOutlined } from '@ant-design/icons';
import { request } from '../api/client';

interface Overview {
  total: number;
  success: number;
  failed: number;
  success_rate: number;
}

export default function Dashboard() {
  const [ov, setOv] = useState<Overview | null>(null);

  useEffect(() => {
    request<Overview>({ url: '/stats/overview', method: 'GET' })
      .then(setOv)
      .catch(() => {});
  }, []);

  return (
    <div>
      <Row gutter={16}>
        <Col span={6}>
          <Card>
            <Statistic title="24h 发送总量" value={ov?.total ?? '-'} prefix={<SendOutlined />} />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic title="送达成功" value={ov?.success ?? '-'} prefix={<CheckCircleOutlined />} valueStyle={{ color: '#16a34a' }} />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic title="发送失败" value={ov?.failed ?? '-'} prefix={<CloseCircleOutlined />} valueStyle={{ color: '#dc2626' }} />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic title="成功率" value={ov?.success_rate ?? '-'} suffix="%" prefix={<RiseOutlined />} valueStyle={{ color: '#1e40af' }} />
          </Card>
        </Col>
      </Row>
    </div>
  );
}

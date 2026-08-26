import { useEffect, useState } from 'react';
import { Row, Col, Card, Statistic } from 'antd';
import { UserOutlined, SendOutlined, FileProtectOutlined, CheckCircleOutlined, CloseCircleOutlined } from '@ant-design/icons';
import { request } from '../api/client';

interface AdminStats {
  messages: { total: number; success: number; failed: number; success_rate: number };
  users: number;
  templates: number;
}

export default function Dashboard() {
  const [stats, setStats] = useState<AdminStats | null>(null);

  useEffect(() => {
    request<AdminStats>({ url: '/admin/stats', method: 'GET' }).then(setStats).catch(() => {});
  }, []);

  return (
    <div>
      <Row gutter={16}>
        <Col span={6}>
          <Card><Statistic title="用户数" value={stats?.users ?? '-'} prefix={<UserOutlined />} /></Card>
        </Col>
        <Col span={6}>
          <Card><Statistic title="模板数" value={stats?.templates ?? '-'} prefix={<FileProtectOutlined />} /></Card>
        </Col>
      </Row>
      <Row gutter={16} style={{ marginTop: 16 }}>
        <Col span={8}>
          <Card><Statistic title="24h 全平台消息量" value={stats?.messages.total ?? '-'} prefix={<SendOutlined />} /></Card>
        </Col>
        <Col span={8}>
          <Card><Statistic title="送达成功" value={stats?.messages.success ?? '-'} prefix={<CheckCircleOutlined />} valueStyle={{ color: '#16a34a' }} /></Card>
        </Col>
        <Col span={8}>
          <Card><Statistic title="成功率" value={stats?.messages.success_rate ?? '-'} suffix="%" prefix={<CloseCircleOutlined />} valueStyle={{ color: '#1d4ed8' }} /></Card>
        </Col>
      </Row>
    </div>
  );
}

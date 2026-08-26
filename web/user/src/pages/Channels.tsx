import { useEffect, useState } from 'react';
import { Card, Row, Col, Tag, Button, Modal, Form, Input, Switch, message, Popconfirm, Descriptions, Typography, Progress, Statistic } from 'antd';
import { request } from '../api/client';

interface ChannelRow {
  type: string;
  name: string;
  desc: string;
  configured: boolean;
  status?: string;
  updated_at?: string;
}

interface ChannelHealth {
  type: string;
  name: string;
  breaker_state: string; // closed|open|half_open
  breaker_failures: number;
  msg_24h: number;
  success_24h: number;
  success_rate: number;
  last_success_at: string | null;
  last_failure_at: string | null;
}

const stateMap: Record<string, { color: string; text: string }> = {
  closed: { color: 'success', text: '正常' },
  open: { color: 'error', text: '熔断中' },
  half_open: { color: 'warning', text: '探测中' },
};

export default function Channels() {
  const [rows, setRows] = useState<ChannelRow[]>([]);
  const [health, setHealth] = useState<ChannelHealth[]>([]);
  const [editing, setEditing] = useState<ChannelRow | null>(null);
  const [form] = Form.useForm();

  const load = async () => {
    try {
      const [d, h] = await Promise.all([
        request<{ data: ChannelRow[] }>({ url: '/channels', method: 'GET' }),
        request<{ data: ChannelHealth[] }>({ url: '/channels/health', method: 'GET' }),
      ]);
      setRows(d.data);
      setHealth(h.data);
    } catch { /* 已提示 */ }
  };
  useEffect(() => { load(); }, []);

  const openEdit = (row: ChannelRow) => {
    setEditing(row);
    form.resetFields();
  };

  const save = async () => {
    const values = form.getFieldsValue();
    const clean: Record<string, any> = {};
    for (const [k, v] of Object.entries(values)) {
      if (v !== '' && v !== undefined) clean[k] = v;
    }
    await request({ url: `/channels/${editing!.type}`, method: 'PUT', data: { config: clean } });
    message.success('渠道配置已保存');
    setEditing(null);
    load();
  };

  const removeOverride = async (type: string) => {
    await request({ url: `/channels/${type}`, method: 'DELETE' });
    message.success('已删除租户配置（回到平台默认）');
    load();
  };

  return (
    <div>
      <Card title="渠道健康看板" style={{ marginBottom: 16 }}>
        <Row gutter={16}>
          {health.map((h) => {
            const s = stateMap[h.breaker_state] ?? { color: 'default', text: h.breaker_state };
            return (
              <Col span={4} key={h.type} style={{ textAlign: 'center' }}>
                <Statistic title={h.name} value={h.success_rate} suffix="%" precision={1} />
                <div style={{ marginTop: 8 }}>
                  <Tag color={s.color}>{s.text}</Tag>
                </div>
                <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                  24h: {h.success_24h} 成功 / {h.msg_24h} 总量
                </Typography.Text>
              </Col>
            );
          })}
        </Row>
      </Card>

      <Row gutter={16}>
        {rows.map((r) => (
          <Col span={12} key={r.type} style={{ marginBottom: 16 }}>
            <Card
              title={<>{r.name} <Tag color="blue" style={{ marginLeft: 8 }}>{r.type}</Tag></>}
              extra={<Tag color={r.configured ? 'success' : 'default'}>{r.configured ? '已配置' : '未配置'}</Tag>}
            >
              <Typography.Paragraph type="secondary" style={{ minHeight: 40 }}>{r.desc}</Typography.Paragraph>
              {r.configured && r.updated_at && (
                <Descriptions size="small" column={1} style={{ marginBottom: 12 }}>
                  <Descriptions.Item label="最后更新">{r.updated_at}</Descriptions.Item>
                </Descriptions>
              )}
              <Button type="primary" onClick={() => openEdit(r)} style={{ marginRight: 8 }}>
                {r.configured ? '修改配置' : '配置渠道'}
              </Button>
              {r.type !== 'webhook' && r.configured && (
                <Popconfirm title="删除租户配置，回到平台默认？" onConfirm={() => removeOverride(r.type)}>
                  <Button danger>重置为默认</Button>
                </Popconfirm>
              )}
            </Card>
          </Col>
        ))}
      </Row>

      <Modal
        title={`配置渠道: ${editing?.name}`}
        open={!!editing}
        onCancel={() => setEditing(null)}
        onOk={save}
        okText="保存"
        destroyOnClose
        width={520}
        footer={editing?.type === 'webhook' || editing?.type === 'inapp'
          ? [<Button key="close" onClick={() => setEditing(null)}>关 闭</Button>]
          : undefined}
      >
        {editing?.type === 'email' && (
          <Form form={form} layout="vertical" key="email">
            <Form.Item name="host" label="SMTP 主机" rules={[{ required: true, message: '必填' }]}><Input placeholder="smtp.example.com" /></Form.Item>
            <Form.Item name="port" label="端口（465 隐式 TLS / 587 STARTTLS）"><Input placeholder="465" /></Form.Item>
            <Form.Item name="user" label="用户名"><Input /></Form.Item>
            <Form.Item name="password" label="密码"><Input.Password /></Form.Item>
            <Form.Item name="from" label="发件人"><Input placeholder="no-reply@example.com" /></Form.Item>
          </Form>
        )}
        {editing?.type === 'apns' && (
          <Form form={form} layout="vertical" key="apns">
            <Form.Item name="team_id" label="Team ID" rules={[{ required: true }]}><Input placeholder="10 位 Team ID" /></Form.Item>
            <Form.Item name="key_id" label="Key ID" rules={[{ required: true }]}><Input placeholder="10 位 Key ID" /></Form.Item>
            <Form.Item name="bundle_id" label="Bundle ID" rules={[{ required: true }]}><Input placeholder="com.example.app" /></Form.Item>
            <Form.Item name="key_p8" label=".p8 私钥（PEM）" rules={[{ required: true }]}><Input.TextArea rows={5} placeholder="-----BEGIN PRIVATE KEY-----" /></Form.Item>
            <Form.Item name="sandbox" label="沙盒环境（开发）" valuePropName="checked"><Switch /></Form.Item>
          </Form>
        )}
        {editing?.type === 'fcm' && (
          <Form form={form} layout="vertical" key="fcm">
            <Form.Item name="project_id" label="Project ID" rules={[{ required: true }]}><Input /></Form.Item>
            <Form.Item name="client_email" label="服务账号邮箱" rules={[{ required: true }]}><Input placeholder="xxx@project.iam.gserviceaccount.com" /></Form.Item>
            <Form.Item name="private_key" label="RSA 私钥" rules={[{ required: true }]}><Input.TextArea rows={5} placeholder="-----BEGIN PRIVATE KEY-----" /></Form.Item>
          </Form>
        )}
        {editing?.type === 'webhook' && (
          <Typography.Text type="secondary">Webhook 渠道使用租户的"回调订阅"配置，无需单独配置。</Typography.Text>
        )}
        {editing?.type === 'inapp' && (
          <Typography.Text type="secondary">站内信渠道为平台内置收件箱，无需外部配置。</Typography.Text>
        )}
      </Modal>
    </div>
  );
}

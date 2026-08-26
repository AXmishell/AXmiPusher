import { useEffect, useState } from 'react';
import { Card, Row, Col, Tag, Button, Modal, Form, Input, InputNumber, Switch, message, Popconfirm, Descriptions, Typography, Progress, Statistic, List, Divider } from 'antd';
import { PlusOutlined } from '@ant-design/icons';
import { request, type Callback } from '../api/client';

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

// Webhook 通道的送达配置 = 回调订阅(内嵌于通道配置页, 替代原独立"回调订阅"页面)
function WebhookCallbacks() {
  const [callbacks, setCallbacks] = useState<Callback[]>([]);
  const [registerOpen, setRegisterOpen] = useState(false);
  const [form] = Form.useForm();

  const load = async () => {
    try {
      const d = await request<{ data: Callback[]; total: number }>({ url: '/callbacks', method: 'GET' });
      setCallbacks(d.data);
    } catch { /* 已提示 */ }
  };
  useEffect(() => { load(); }, []);

  const del = async (id: number) => {
    await request({ url: `/callbacks/${id}`, method: 'DELETE' });
    message.success('已删除');
    load();
  };

  const submit = async () => {
    const values = await form.validateFields();
    await request({
      url: '/callbacks',
      method: 'POST',
      data: { url: values.url, secret: values.secret, events: values.events?.split(',') ?? ['success', 'failed'] },
    });
    message.success('注册成功');
    setRegisterOpen(false);
    form.resetFields();
    load();
  };

  return (
    <>
      <Divider style={{ margin: '12px 0 8px' }} orientation="left">回调订阅（Webhook 送达地址）</Divider>
      {callbacks.length === 0 ? (
        <Typography.Text type="secondary">尚未注册回调地址，Webhook 消息将无处送达。</Typography.Text>
      ) : (
        <List
          size="small"
          dataSource={callbacks}
          renderItem={(r) => (
            <List.Item
              actions={[
                <Popconfirm key="del" title="确定删除该回调？" onConfirm={() => del(r.id)}>
                  <a style={{ color: '#dc2626' }}>删除</a>
                </Popconfirm>,
              ]}
            >
              <List.Item.Meta
                title={<code style={{ wordBreak: 'break-all' }}>{r.url}</code>}
                description={(r.events || '').split(',').filter(Boolean).map((e) => <Tag key={e} style={{ marginRight: 4 }}>{e}</Tag>)}
              />
            </List.Item>
          )}
        />
      )}
      <Button type="dashed" icon={<PlusOutlined />} block style={{ marginTop: 8 }} onClick={() => setRegisterOpen(true)}>
        注册回调
      </Button>
      <Modal
        title="注册回调地址"
        open={registerOpen}
        onCancel={() => setRegisterOpen(false)}
        onOk={submit}
        okText="注册"
        destroyOnClose
        width={520}
      >
        <Form form={form} layout="vertical" preserve={false}>
          <Form.Item name="url" label="回调 URL" rules={[{ required: true, type: 'url', message: '请输入有效 URL' }]}>
            <Input placeholder="https://your-server.com/hook" />
          </Form.Item>
          <Form.Item name="secret" label="签名密钥" tooltip="回调带 X-MP-Signature 头 (HMAC-SHA256)，留空则不签名">
            <Input placeholder="用于 HMAC 签名校验" />
          </Form.Item>
          <Form.Item name="events" label="订阅事件（逗号分隔）" initialValue="success,failed" tooltip="success,failed,retry,dead">
            <Input />
          </Form.Item>
        </Form>
      </Modal>
    </>
  );
}

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
    message.success('通道配置已保存');
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
              {r.type === 'webhook' ? (
                // Webhook 无独立配置, 其送达配置即回调订阅(内嵌管理)
                <WebhookCallbacks />
              ) : (
                <>
                  <Button type="primary" onClick={() => openEdit(r)} style={{ marginRight: 8 }}>
                    {r.configured ? '修改配置' : '配置通道'}
                  </Button>
                  {r.configured && (
                    <Popconfirm title="删除租户配置，回到平台默认？" onConfirm={() => removeOverride(r.type)}>
                      <Button danger>重置为默认</Button>
                    </Popconfirm>
                  )}
                </>
              )}
            </Card>
          </Col>
        ))}
      </Row>

      <Modal
        title={`配置通道: ${editing?.name}`}
        open={!!editing}
        onCancel={() => setEditing(null)}
        onOk={save}
        okText="保存"
        destroyOnClose
        width={520}
        footer={editing?.type === 'inapp'
          ? [<Button key="close" onClick={() => setEditing(null)}>关 闭</Button>]
          : undefined}
      >
        {editing?.type === 'email' && (
          <Form form={form} layout="vertical" key="email">
            <Form.Item name="host" label="SMTP 主机" rules={[{ required: true, message: '必填' }]}><Input placeholder="smtp.example.com" /></Form.Item>
            <Form.Item name="port" label="端口（465 隐式 TLS / 587 STARTTLS）"><InputNumber style={{ width: '100%' }} placeholder="465" min={1} max={65535} /></Form.Item>
            <Form.Item name="user" label="用户名"><Input /></Form.Item>
            <Form.Item name="password" label="密码"><Input.Password /></Form.Item>
            <Form.Item name="from" label="发件人"><Input placeholder="no-reply@example.com" /></Form.Item>
            <Form.Item name="recipient" label="默认收件人" tooltip="发送消息时未指定收件人, 则邮件发到此地址"><Input placeholder="收件人邮箱" /></Form.Item>
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
        {editing?.type === 'inapp' && (
          <Typography.Text type="secondary">站内信渠道为平台内置收件箱，无需外部配置。</Typography.Text>
        )}
      </Modal>
    </div>
  );
}

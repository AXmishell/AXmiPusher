import { useEffect, useState } from 'react';
import { Card, Row, Col, Form, Input, InputNumber, Divider, Typography, Button, message, Alert, Popconfirm } from 'antd';
import { MailOutlined, PayCircleOutlined, SettingOutlined, SafetyCertificateOutlined } from '@ant-design/icons';
import { request } from '../api/client';

interface SettingsData {
  smtp: { host?: string; port?: number; user?: string; password?: string; from?: string };
  epay: { gateway?: string; pid?: string; key?: string };
  retention_days: string;
  rate_limit_per_minute: string;
  admin_path: string;
}

export default function Settings() {
  const [data, setData] = useState<SettingsData | null>(null);
  const [smtpForm] = Form.useForm();
  const [epayForm] = Form.useForm();
  const [sysForm] = Form.useForm();
  const [pathInput, setPathInput] = useState('');

  useEffect(() => {
    request<SettingsData>({ url: '/admin/settings', method: 'GET' }).then((d) => {
      setData(d);
      setPathInput(d.admin_path || '');
      smtpForm.setFieldsValue(d.smtp);
      epayForm.setFieldsValue(d.epay);
      sysForm.setFieldsValue({
        retention_days: parseInt(d.retention_days) || 90,
        rate_limit_per_minute: parseInt(d.rate_limit_per_minute) || 600,
      });
    }).catch(() => {});
  }, []);

  const saveSmtp = async (values: any) => {
    await request({ url: '/admin/settings', method: 'PUT', data: { smtp: { ...values } } });
    message.success('SMTP 配置已保存');
  };

  const saveEpay = async (values: any) => {
    await request({ url: '/admin/settings', method: 'PUT', data: { epay: { ...values } } });
    message.success('易支付配置已保存');
  };

  const saveSys = async (values: any) => {
    await request({
      url: '/admin/settings',
      method: 'PUT',
      data: { retention_days: values.retention_days, rate_limit_per_minute: values.rate_limit_per_minute },
    });
    message.success('系统设置已保存');
  };

  const rotatePath = async () => {
    const d = await request<{ admin_path: string }>({ url: '/admin/settings/rotate-admin-path', method: 'POST', data: {} });
    message.success(`已轮换, 正在跳转到新路径 /${d.admin_path}/ ...`, 2);
    // 立即重定向到新后台地址(旧路径已由后端直接废除, 404 不再提供内容)。
    window.location.href = `/${d.admin_path}/`;
  };

  const applyPath = async () => {
    const custom = pathInput.trim();
    if (!custom) {
      message.warning('请输入自定义路径, 或使用随机轮换');
      return;
    }
    const d = await request<{ admin_path: string }>({ url: '/admin/settings/rotate-admin-path', method: 'POST', data: { admin_path: custom } });
    message.success(`已应用, 正在跳转到新路径 /${d.admin_path}/ ...`, 2);
    // 旧路径立即废除, 跳转到新后台地址。
    window.location.href = `/${d.admin_path}/`;
  };

  return (
    <div>
      <Row gutter={16}>
        <Col span={12}>
          <Card title={<><MailOutlined /> 邮件渠道 SMTP</>}>
            <Alert style={{ marginBottom: 16 }} type="info" showIcon message="系统邮件 — 仅用于发送注册/登录验证码等系统邮件; 用户中心邮件渠道需在用户中心自行配置 SMTP" />
            <Form form={smtpForm} layout="vertical" onFinish={saveSmtp}>
              <Form.Item name="host" label="SMTP 主机" rules={[{ required: true }]}>
                <Input placeholder="smtp.example.com" />
              </Form.Item>
              <Row gutter={12}>
                <Col span={12}>
                  <Form.Item name="port" label="端口" rules={[{ required: true }]}>
                    <InputNumber style={{ width: '100%' }} placeholder="465" />
                  </Form.Item>
                </Col>
                <Col span={12}>
                  <Form.Item name="from" label="发件人">
                    <Input placeholder="no-reply@example.com" />
                  </Form.Item>
                </Col>
              </Row>
              <Form.Item name="user" label="用户名">
                <Input placeholder="smtp 账号" />
              </Form.Item>
              <Form.Item name="password" label="密码（留空则不修改）">
                <Input.Password placeholder="smtp 密码" />
              </Form.Item>
              <Button type="primary" htmlType="submit">保存 SMTP</Button>
            </Form>
          </Card>
        </Col>
        <Col span={12}>
          <Card title={<><PayCircleOutlined /> 易支付（套餐订阅）</>}>
            <Form form={epayForm} layout="vertical" onFinish={saveEpay}>
              <Form.Item name="gateway" label="支付网关地址">
                <Input placeholder="https://your-epay.com" />
              </Form.Item>
              <Form.Item name="pid" label="商户 ID (pid)">
                <Input placeholder="1000" />
              </Form.Item>
              <Form.Item name="key" label="商户密钥 (key)（留空则不修改）">
                <Input.Password placeholder="epay key" />
              </Form.Item>
              <Button type="primary" htmlType="submit">保存易支付</Button>
            </Form>
          </Card>
        </Col>
      </Row>
      <Card style={{ marginTop: 16 }} title={<><SettingOutlined /> 系统设置</>}>
        <Form form={sysForm} layout="inline" onFinish={saveSys}>
          <Form.Item name="retention_days" label="消息保留天数">
            <InputNumber min={1} max={3650} style={{ width: 120 }} />
          </Form.Item>
          <Form.Item name="rate_limit_per_minute" label="每租户每分钟限流">
            <InputNumber min={1} style={{ width: 120 }} />
          </Form.Item>
          <Button type="primary" htmlType="submit">保存</Button>
        </Form>
        <Divider />
        <Typography.Text strong>
          <SafetyCertificateOutlined /> 管理员后台路径（随机隐藏路由，全局唯一）
        </Typography.Text>
        <div style={{ marginTop: 12, display: 'flex', alignItems: 'center', gap: 12, flexWrap: 'wrap' }}>
          <code style={{ background: '#f1f5f9', padding: '6px 12px', borderRadius: 6, fontSize: 14 }}>
            {location.origin}/{data?.admin_path || '-'}/
          </code>
          <Input
            value={pathInput}
            onChange={(e) => setPathInput(e.target.value)}
            style={{ width: 200 }}
            placeholder="自定义路径(8-32位字母数字)"
            maxLength={32}
            allowClear
          />
          <Popconfirm title="应用后旧路径立即废除(404)且不可再访问，确定？" onConfirm={applyPath}>
            <Button type="primary">应用自定义路径</Button>
          </Popconfirm>
          <Popconfirm title="轮换后旧路径立即废除并生成随机新路径，确定？" onConfirm={rotatePath}>
            <Button danger>随机轮换</Button>
          </Popconfirm>
        </div>
        <Alert style={{ marginTop: 12 }} type="info" showIcon message="路径混淆仅作为安全纵深，真正的防护是平台管理员 JWT + 角色校验。更换路径后旧路径立即失效(404)，后台路径保持唯一。" />
      </Card>
    </div>
  );
}

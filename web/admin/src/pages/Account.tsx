import { useEffect, useState } from 'react';
import { Card, Descriptions, Form, Input, Button, message, Alert, Typography } from 'antd';
import { MailOutlined, LockOutlined, IdcardOutlined, UserOutlined, SafetyCertificateOutlined } from '@ant-design/icons';
import { request, type Admin } from '../api/client';
import ChangePasswordModal from '../components/ChangePasswordModal';

interface TotpSetupData {
  secret: string;
  otpauth_url: string;
  qr_data_url: string;
}

// 账户设置页: 账户信息展示 + 用户名/QQ + 邮箱 + 密码 + 两步验证(镜像用户中心)。
export default function Account() {
  const [admin, setAdmin] = useState<Admin | null>(null);
  const [profileForm] = Form.useForm();
  const [emailForm] = Form.useForm();
  const [saving, setSaving] = useState(false);
  const [savingEmail, setSavingEmail] = useState(false);
  const [pwdOpen, setPwdOpen] = useState(false);
  // TOTP 绑定状态。
  const [totpSetup, setTotpSetup] = useState<TotpSetupData | null>(null);
  const [totpBusy, setTotpBusy] = useState(false);

  const loadAdmin = () => {
    request<{ admin: Admin }>({ url: '/admin/auth/me', method: 'GET' })
      .then((d) => {
        setAdmin(d.admin);
        profileForm.setFieldsValue({ nickname: d.admin.nickname, qq: d.admin.qq });
        emailForm.setFieldsValue({ email: d.admin.email });
        if (d.admin.totp_enabled) setTotpSetup(null);
      })
      .catch(() => {});
  };

  useEffect(loadAdmin, [profileForm, emailForm]);

  // 保存昵称/QQ。
  const saveProfile = async (values: { nickname: string; qq?: string }) => {
    setSaving(true);
    try {
      const d = await request<{ admin: Admin }>({ url: '/admin/auth/profile', method: 'PUT', data: values });
      setAdmin(d.admin);
      message.success('账户资料已更新');
    } catch {
      /* 拦截器已提示 */
    } finally {
      setSaving(false);
    }
  };

  // 修改登录邮箱。
  const saveEmail = async (values: { email: string }) => {
    setSavingEmail(true);
    try {
      const d = await request<{ admin: Admin }>({ url: '/admin/auth/email', method: 'PUT', data: values });
      setAdmin(d.admin);
      emailForm.setFieldsValue({ email: d.admin.email });
      message.success('邮箱已修改, 下次登录请使用新邮箱');
    } catch {
      /* 拦截器已提示 */
    } finally {
      setSavingEmail(false);
    }
  };

  // 开启两步验证: 获取密钥与二维码。
  const startTotpSetup = async () => {
    setTotpBusy(true);
    try {
      const d = await request<TotpSetupData>({ url: '/admin/auth/totp/setup', method: 'POST' });
      setTotpSetup(d);
    } catch {
      /* 拦截器已提示 */
    } finally {
      setTotpBusy(false);
    }
  };

  // 确认启用。
  const confirmTotp = async (values: { code: string }) => {
    setTotpBusy(true);
    try {
      await request({ url: '/admin/auth/totp/confirm', method: 'POST', data: { code: values.code } });
      message.success('两步验证已启用');
      setTotpSetup(null);
      loadAdmin();
    } catch {
      /* 拦截器已提示 */
    } finally {
      setTotpBusy(false);
    }
  };

  // 关闭两步验证。
  const disableTotp = async (values: { code: string }) => {
    setTotpBusy(true);
    try {
      await request({ url: '/admin/auth/totp/disable', method: 'POST', data: { code: values.code } });
      message.success('两步验证已关闭');
      loadAdmin();
    } catch {
      /* 拦截器已提示 */
    } finally {
      setTotpBusy(false);
    }
  };

  return (
    <div>
      <Card title="账户信息" style={{ marginBottom: 16 }}>
        <Descriptions column={2} size="middle">
          <Descriptions.Item label="账号 ID">{admin?.id ?? '-'}</Descriptions.Item>
          <Descriptions.Item label="用户名">{admin?.nickname || '-'}</Descriptions.Item>
          <Descriptions.Item label="角色">{admin?.role === 'super_admin' ? '超管' : '管理员'}</Descriptions.Item>
          <Descriptions.Item label="邮箱">{admin?.email || '-'}</Descriptions.Item>
          <Descriptions.Item label="注册时间">
            {admin?.created_at ? new Date(admin.created_at).toLocaleString() : '-'}
          </Descriptions.Item>
          <Descriptions.Item label="最近登录 IP">{admin?.last_login_ip || '-'}</Descriptions.Item>
        </Descriptions>
      </Card>

      <Card title="账户资料" style={{ marginBottom: 16 }}>
        <Form form={profileForm} layout="vertical" style={{ maxWidth: 480 }} onFinish={saveProfile}>
          <Form.Item name="nickname" label="用户名" rules={[{ required: true, message: '请输入用户名' }, { max: 64, message: '最多 64 字' }]}>
            <Input placeholder="用户名" prefix={<UserOutlined />} />
          </Form.Item>
          <Form.Item name="qq" label="QQ 号码" rules={[{ max: 32, message: '最多 32 位' }]}>
            <Input placeholder="QQ 号码(可空)" prefix={<IdcardOutlined />} />
          </Form.Item>
          <Button type="primary" htmlType="submit" loading={saving}>
            保存资料
          </Button>
        </Form>
      </Card>

      <Card title="邮箱修改" style={{ marginBottom: 16 }}>
        <Form form={emailForm} layout="vertical" style={{ maxWidth: 480 }} onFinish={saveEmail}>
          <Form.Item name="email" label="登录邮箱" rules={[{ required: true, message: '请输入邮箱' }, { type: 'email', message: '邮箱格式不正确' }]}>
            <Input placeholder="新邮箱" prefix={<MailOutlined />} />
          </Form.Item>
          <Button type="primary" htmlType="submit" loading={savingEmail}>
            修改邮箱
          </Button>
          <div style={{ color: 'rgba(0,0,0,.45)', fontSize: 12, marginTop: 8 }}>
            修改后旧邮箱立即失效, 下次登录请使用新邮箱。
          </div>
        </Form>
      </Card>

      <Card title="密码修改">
        <Button icon={<LockOutlined />} onClick={() => setPwdOpen(true)}>
          修改密码
        </Button>
      </Card>

      <Card title="两步验证(TOTP)" style={{ marginTop: 16 }}>
        {admin?.totp_enabled ? (
          <div style={{ maxWidth: 480 }}>
            <Alert type="success" showIcon message="两步验证已启用" description="登录时需要输入动态验证码, 增强账号安全。" style={{ marginBottom: 16 }} />
            <Form layout="vertical" onFinish={disableTotp}>
              <Form.Item name="code" label="输入当前验证码以关闭" rules={[{ required: true, len: 6, message: '请输入 6 位验证码' }]}>
                <Input placeholder="6 位验证码" maxLength={6} prefix={<SafetyCertificateOutlined />} />
              </Form.Item>
              <Button danger htmlType="submit" loading={totpBusy}>
                关闭两步验证
              </Button>
            </Form>
          </div>
        ) : totpSetup ? (
          <div style={{ maxWidth: 480 }}>
            <Typography.Paragraph type="secondary">
              使用 Authenticator 类应用(如 Google Authenticator / Microsoft Authenticator)扫描二维码, 或手动输入密钥。
            </Typography.Paragraph>
            <div style={{ textAlign: 'center', marginBottom: 16 }}>
              <img src={totpSetup.qr_data_url} alt="TOTP 二维码" width={200} height={200} />
            </div>
            <Typography.Paragraph copyable={{ text: totpSetup.secret }} style={{ textAlign: 'center', marginBottom: 16 }}>
              密钥: <Typography.Text code>{totpSetup.secret}</Typography.Text>
            </Typography.Paragraph>
            <Form layout="vertical" onFinish={confirmTotp}>
              <Form.Item name="code" label="输入应用中的 6 位验证码确认" rules={[{ required: true, len: 6, message: '请输入 6 位验证码' }]}>
                <Input placeholder="6 位验证码" maxLength={6} prefix={<SafetyCertificateOutlined />} />
              </Form.Item>
              <Button type="primary" htmlType="submit" loading={totpBusy}>
                确认并启用
              </Button>
              <Button style={{ marginLeft: 8 }} onClick={() => setTotpSetup(null)}>
                取消
              </Button>
            </Form>
          </div>
        ) : (
          <div>
            <Alert type="info" showIcon message="尚未启用" description="开启后, 登录时除密码外还需输入动态验证码。" style={{ marginBottom: 16 }} />
            <Button icon={<SafetyCertificateOutlined />} onClick={startTotpSetup} loading={totpBusy}>
              开启两步验证
            </Button>
          </div>
        )}
      </Card>

      <ChangePasswordModal open={pwdOpen} onClose={() => setPwdOpen(false)} />
    </div>
  );
}

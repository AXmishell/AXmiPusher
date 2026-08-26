import { useEffect, useState } from 'react';
import { Card, Descriptions, Form, Input, Button, message } from 'antd';
import { MailOutlined, LockOutlined, IdcardOutlined, UserOutlined } from '@ant-design/icons';
import { request, type Admin } from '../api/client';
import ChangePasswordModal from '../components/ChangePasswordModal';

// 账户设置页: 账户信息展示 + 昵称/QQ + 邮箱 + 密码修改(镜像用户中心)。
export default function Account() {
  const [admin, setAdmin] = useState<Admin | null>(null);
  const [profileForm] = Form.useForm();
  const [emailForm] = Form.useForm();
  const [saving, setSaving] = useState(false);
  const [savingEmail, setSavingEmail] = useState(false);
  const [pwdOpen, setPwdOpen] = useState(false);

  useEffect(() => {
    request<{ admin: Admin }>({ url: '/admin/auth/me', method: 'GET' })
      .then((d) => {
        setAdmin(d.admin);
        profileForm.setFieldsValue({ nickname: d.admin.nickname, qq: d.admin.qq });
        emailForm.setFieldsValue({ email: d.admin.email });
      })
      .catch(() => {});
  }, [profileForm, emailForm]);

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

  return (
    <div>
      <Card title="账户信息" style={{ marginBottom: 16 }}>
        <Descriptions column={2} size="middle">
          <Descriptions.Item label="账号 ID">{admin?.id ?? '-'}</Descriptions.Item>
          <Descriptions.Item label="账户昵称">{admin?.nickname || '-'}</Descriptions.Item>
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
          <Form.Item name="nickname" label="账户昵称" rules={[{ required: true, message: '请输入昵称' }, { max: 64, message: '最多 64 字' }]}>
            <Input placeholder="昵称" prefix={<UserOutlined />} />
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

      <ChangePasswordModal open={pwdOpen} onClose={() => setPwdOpen(false)} />
    </div>
  );
}

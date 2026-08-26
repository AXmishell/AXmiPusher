import { useEffect, useState } from 'react';
import { ProTable } from '@ant-design/pro-components';
import { Button, Form, Input, message, Modal, Popconfirm, Tag } from 'antd';
import { PlusOutlined } from '@ant-design/icons';
import dayjs from 'dayjs';
import { request, type Admin } from '../api/client';

export default function Admins() {
  const [createOpen, setCreateOpen] = useState(false);
  const [createForm] = Form.useForm();
  const [pwdOpen, setPwdOpen] = useState(false);
  const [pwdForm] = Form.useForm();
  const [pwdTarget, setPwdTarget] = useState<Admin | null>(null);
  const [currentId, setCurrentId] = useState<number | null>(null);

  // 取当前登录管理员(用于自身那行禁用"禁用"按钮)。
  useEffect(() => {
    request<{ admin: Admin }>({ url: '/admin/auth/me', method: 'GET' })
      .then((d) => setCurrentId(d.admin.id))
      .catch(() => { /* 拦截器已提示 */ });
  }, []);

  const setStatus = async (id: number, status: string) => {
    await request({ url: `/admin/admins/${id}/status`, method: 'PUT', data: { status } });
    message.success(status === 'disabled' ? '已禁用' : '已启用');
    return true;
  };

  const onCreate = async (values: { email: string; password: string; nickname?: string }) => {
    await request({ url: '/admin/admins', method: 'POST', data: values });
    message.success('新增管理员成功');
    createForm.resetFields();
    setCreateOpen(false);
    return true;
  };

  const onResetPassword = async (values: { password: string }) => {
    if (!pwdTarget) return;
    await request({ url: `/admin/admins/${pwdTarget.id}/password`, method: 'PUT', data: values });
    message.success('密码已重置');
    pwdForm.resetFields();
    setPwdTarget(null);
    setPwdOpen(false);
    return true;
  };

  return (
    <>
      <ProTable<Admin>
        headerTitle="管理员管理"
        rowKey="id"
        search={false}
        toolBarRender={() => [
          <Button key="create" type="primary" icon={<PlusOutlined />} onClick={() => setCreateOpen(true)}>
            新增管理员
          </Button>,
        ]}
        request={async (params) => {
          const { current = 1, pageSize = 20 } = params as any;
          const d = await request<{ data: Admin[]; total: number }>({
            url: '/admin/admins',
            method: 'GET',
            params: { current, pageSize },
          });
          return { data: d.data, total: d.total, success: true };
        }}
        columns={[
          { title: 'ID', dataIndex: 'id', width: 70 },
          { title: '邮箱', dataIndex: 'email', width: 220, copyable: true },
          { title: '昵称', dataIndex: 'nickname', width: 120, render: (_, r) => r.nickname || '-' },
          {
            title: '角色',
            dataIndex: 'role',
            width: 110,
            render: (_, r) => (r.role === 'super_admin' ? <Tag color="gold">超管</Tag> : <Tag color="blue">管理员</Tag>),
          },
          {
            title: '状态',
            dataIndex: 'status',
            width: 90,
            render: (_, r) => <Tag color={r.status === 'active' ? 'success' : 'error'}>{r.status === 'active' ? '启用' : '禁用'}</Tag>,
          },
          {
            title: '最近登录',
            dataIndex: 'last_login_at',
            width: 170,
            render: (_, r) => (r.last_login_at ? dayjs(r.last_login_at).format('YYYY-MM-DD HH:mm:ss') : '-'),
          },
          { title: '创建时间', dataIndex: 'created_at', valueType: 'dateTime', width: 170 },
          {
            title: '操作',
            valueType: 'option',
            width: 180,
            render: (_, r) => {
              const actions: React.ReactNode[] = [];
              if (r.status === 'active') {
                if (r.id !== currentId) {
                  actions.push(
                    <Popconfirm key="d" title="禁用后该管理员无法登录，确定？" onConfirm={() => setStatus(r.id, 'disabled').then(() => location.reload())}>
                      <a style={{ color: '#dc2626' }}>禁用</a>
                    </Popconfirm>,
                  );
                }
              } else {
                actions.push(<a key="e" onClick={() => setStatus(r.id, 'active').then(() => location.reload())}>启用</a>);
              }
              actions.push(
                <a key="p" onClick={() => { setPwdTarget(r); setPwdOpen(true); }}>
                  重置密码
                </a>,
              );
              return actions;
            },
          },
        ]}
      />
      <Modal
        title="新增管理员"
        open={createOpen}
        onCancel={() => { createForm.resetFields(); setCreateOpen(false); }}
        onOk={() => createForm.submit()}
        destroyOnClose
        width={420}
      >
        <Form form={createForm} layout="vertical" onFinish={onCreate} style={{ marginTop: 8 }}>
          <Form.Item
            name="email"
            label="邮箱"
            rules={[
              { required: true, message: '请输入邮箱' },
              { type: 'email', message: '邮箱格式不正确' },
            ]}
          >
            <Input placeholder="admin@example.com" />
          </Form.Item>
          <Form.Item name="password" label="密码" rules={[{ required: true, min: 8, message: '密码至少 8 位' }]}>
            <Input.Password placeholder="至少 8 位" />
          </Form.Item>
          <Form.Item name="nickname" label="昵称" rules={[{ max: 32, message: '昵称最长 32 个字符' }]}>
            <Input placeholder="可选" />
          </Form.Item>
        </Form>
      </Modal>
      <Modal
        title="重置密码"
        open={pwdOpen}
        onCancel={() => { pwdForm.resetFields(); setPwdOpen(false); }}
        onOk={() => pwdForm.submit()}
        confirmLoading={false}
        destroyOnClose
        width={420}
      >
        <Form form={pwdForm} layout="vertical" onFinish={onResetPassword} style={{ marginTop: 8 }}>
          <Form.Item name="password" label="新密码" rules={[{ required: true, min: 8, message: '密码至少 8 位' }]}>
            <Input.Password placeholder="至少 8 位" />
          </Form.Item>
        </Form>
      </Modal>
    </>
  );
}

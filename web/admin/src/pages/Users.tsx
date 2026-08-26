import { useState } from 'react';
import { ProTable, ModalForm, ProFormText } from '@ant-design/pro-components';
import { Button, message, Popconfirm, Tag } from 'antd';
import { PlusOutlined } from '@ant-design/icons';
import { request, type User } from '../api/client';

const roleMap: Record<string, { color: string; text: string }> = {
  tenant_admin: { color: 'blue', text: '租户管理员' },
  tenant_user: { color: 'default', text: '租户用户' },
  platform_admin: { color: 'purple', text: '平台管理员' },
};

export default function Users() {
  const [editUser, setEditUser] = useState<User | null>(null);
  const [createOpen, setCreateOpen] = useState(false);

  const setStatus = async (id: number, status: string) => {
    await request({ url: `/admin/users/${id}/status`, method: 'PUT', data: { status } });
    message.success('操作成功');
    return true;
  };

  const saveEdit = async (values: { email: string; nickname: string; password?: string }) => {
    await request({ url: `/admin/users/${editUser!.id}`, method: 'PUT', data: values });
    message.success('已保存');
    setEditUser(null);
    location.reload();
    return true;
  };

  const createUser = async (values: { email: string; nickname?: string; password: string }) => {
    await request({ url: '/admin/users', method: 'POST', data: values });
    message.success('用户已创建');
    setCreateOpen(false);
    location.reload();
    return true;
  };

  return (
    <>
      <ProTable<User>
        headerTitle="用户管理"
        rowKey="id"
        search={{ labelWidth: 'auto' }}
        toolBarRender={() => [
          <Button key="add" type="primary" icon={<PlusOutlined />} onClick={() => setCreateOpen(true)}>
            新增用户
          </Button>,
        ]}
        request={async (params) => {
          const { current = 1, pageSize = 20, email, ...rest } = params as any;
          const d = await request<{ data: User[]; total: number }>({
            url: '/admin/users',
            method: 'GET',
            params: { current, pageSize, ...rest },
          });
          return { data: d.data, total: d.total, success: true };
        }}
        columns={[
          { title: 'ID', dataIndex: 'id', width: 70 },
          { title: '邮箱', dataIndex: 'email', width: 220, copyable: true },
          { title: '用户名', dataIndex: 'nickname', width: 120 },
          {
            title: '角色',
            dataIndex: 'role',
            width: 120,
            render: (_, r) => {
              const m = roleMap[r.role] ?? { color: 'default', text: r.role };
              return <Tag color={m.color}>{m.text}</Tag>;
            },
          },
          {
            title: '状态',
            dataIndex: 'status',
            width: 90,
            render: (_, r) => <Tag color={r.status === 'active' ? 'success' : 'default'}>{r.status === 'active' ? '启用' : '禁用'}</Tag>,
          },
          { title: '注册时间', dataIndex: 'created_at', valueType: 'dateTime', width: 170 },
          {
            title: '操作',
            valueType: 'option',
            width: 160,
            render: (_, r) => {
              if (r.role === 'platform_admin') return [];
              const ops: any[] = [
                <a key="edit" onClick={() => setEditUser(r)}>
                  编辑
                </a>,
              ];
              if (r.status === 'active') {
                ops.push(
                  <Popconfirm key="d" title="禁用后该用户无法登录，确定？" onConfirm={() => setStatus(r.id, 'disabled').then(() => location.reload())}>
                    <a style={{ color: '#dc2626' }}>禁用</a>
                  </Popconfirm>,
                );
              } else {
                ops.push(
                  <a key="e" onClick={() => setStatus(r.id, 'active').then(() => location.reload())}>
                    启用
                  </a>,
                );
              }
              return ops;
            },
          },
        ]}
      />
      {/* 新增用户: 邮箱 / 用户名 / 密码 */}
      <ModalForm<{ email: string; nickname?: string; password: string }>
        key="create-user"
        title="新增用户"
        open={createOpen}
        onOpenChange={(o) => !o && setCreateOpen(false)}
        onFinish={createUser}
        modalProps={{ destroyOnClose: true }}
      >
        <ProFormText name="email" label="邮箱" rules={[{ required: true, type: 'email' }]} />
        <ProFormText name="nickname" label="用户名" placeholder="留空默认使用邮箱" />
        <ProFormText.Password name="password" label="密码" rules={[{ required: true, min: 8 }]} placeholder="至少 8 位" />
      </ModalForm>
      {/* 编辑用户: 邮箱 / 用户名 / 重置密码 */}
      <ModalForm<User>
        key="edit-user"
        title={editUser ? `编辑用户 #${editUser.id}（${editUser.email}）` : '编辑用户'}
        open={!!editUser}
        onOpenChange={(o) => !o && setEditUser(null)}
        initialValues={{ email: editUser?.email, nickname: editUser?.nickname }}
        onFinish={saveEdit}
        modalProps={{ destroyOnClose: true }}
      >
        <ProFormText name="email" label="邮箱" rules={[{ required: true, type: 'email' }]} />
        <ProFormText name="nickname" label="用户名" rules={[{ required: true }]} />
        <ProFormText.Password name="password" label="重置密码" placeholder="留空则不修改（至少 8 位）" />
      </ModalForm>
    </>
  );
}

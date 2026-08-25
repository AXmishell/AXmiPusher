import { ProTable } from '@ant-design/pro-components';
import { message, Popconfirm, Tag } from 'antd';
import { request, type User } from '../api/client';

const roleMap: Record<string, { color: string; text: string }> = {
  tenant_admin: { color: 'blue', text: '租户管理员' },
  tenant_user: { color: 'default', text: '租户用户' },
  platform_admin: { color: 'purple', text: '平台管理员' },
};

export default function Users() {
  const setStatus = async (id: number, status: string) => {
    await request({ url: `/admin/users/${id}/status`, method: 'PUT', data: { status } });
    message.success('操作成功');
    return true;
  };

  return (
    <ProTable<User>
      headerTitle="用户管理"
      rowKey="id"
      search={{ labelWidth: 'auto' }}
      request={async (params) => {
        const { current = 1, pageSize = 20, tenant_id, email, ...rest } = params as any;
        const d = await request<{ data: User[]; total: number }>({
          url: '/admin/users',
          method: 'GET',
          params: { current, pageSize, tenant_id: tenant_id || undefined, ...rest },
        });
        return { data: d.data, total: d.total, success: true };
      }}
      columns={[
        { title: 'ID', dataIndex: 'id', width: 70 },
        { title: '邮箱', dataIndex: 'email', width: 220, copyable: true },
        { title: '昵称', dataIndex: 'nickname', width: 120 },
        { title: '租户 ID', dataIndex: 'tenant_id', width: 90 },
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
          width: 120,
          render: (_, r) =>
            r.role === 'platform_admin'
              ? []
              : r.status === 'active'
                ? [
                    <Popconfirm key="d" title="禁用后该用户无法登录，确定？" onConfirm={() => setStatus(r.id, 'disabled').then(() => location.reload())}>
                      <a style={{ color: '#dc2626' }}>禁用</a>
                    </Popconfirm>,
                  ]
                : [<a key="e" onClick={() => setStatus(r.id, 'active').then(() => location.reload())}>启用</a>],
        },
      ]}
    />
  );
}

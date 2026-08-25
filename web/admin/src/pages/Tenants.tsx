import { ProTable } from '@ant-design/pro-components';
import { message, Popconfirm, Tag } from 'antd';
import { request, type TenantRow } from '../api/client';

export default function Tenants() {
  const setStatus = async (id: number, status: string) => {
    await request({ url: `/admin/tenants/${id}/status`, method: 'PUT', data: { status } });
    message.success(status === 'disabled' ? '已禁用' : '已启用');
    return true;
  };

  return (
    <ProTable<TenantRow>
      headerTitle="租户管理"
      rowKey="id"
      search={false}
      request={async (params) => {
        const { current = 1, pageSize = 20 } = params as any;
        const d = await request<{ data: TenantRow[]; total: number }>({
          url: '/admin/tenants',
          method: 'GET',
          params: { current, pageSize },
        });
        return { data: d.data, total: d.total, success: true };
      }}
      columns={[
        { title: 'ID', dataIndex: 'id', width: 70 },
        { title: '租户名称', dataIndex: 'name', width: 180 },
        { title: '用户数', dataIndex: 'user_count', width: 90 },
        { title: '24h 消息量', dataIndex: 'msg_24h', width: 110 },
        {
          title: '状态',
          dataIndex: 'status',
          width: 90,
          render: (_, r) => <Tag color={r.status === 'active' ? 'success' : 'default'}>{r.status === 'active' ? '启用' : '禁用'}</Tag>,
        },
        { title: '创建时间', dataIndex: 'created_at', valueType: 'dateTime', width: 170 },
        {
          title: '操作',
          valueType: 'option',
          width: 120,
          render: (_, r) =>
            r.status === 'active'
              ? [
                  <Popconfirm key="d" title="禁用后该租户全部 API 将不可用，确定？" onConfirm={() => setStatus(r.id, 'disabled').then(() => location.reload())}>
                    <a style={{ color: '#dc2626' }}>禁用</a>
                  </Popconfirm>,
                ]
              : [
                  <a key="e" onClick={() => setStatus(r.id, 'active').then(() => location.reload())}>启用</a>,
                ],
        },
      ]}
    />
  );
}

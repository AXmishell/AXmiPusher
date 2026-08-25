import { ProTable } from '@ant-design/pro-components';
import { Tag, Tooltip } from 'antd';
import { request, type AuditLog } from '../api/client';

const actionColor: Record<string, string> = {
  'template.create': 'blue',
  'template.update': 'blue',
  'template.delete': 'blue',
  'review.approve': 'green',
  'review.reject': 'red',
  'tenant.set_status': 'orange',
  'user.set_status': 'orange',
  'plan.create': 'purple',
  'plan.update': 'purple',
  'plan.delete': 'purple',
  'settings.update': 'cyan',
  'settings.rotate_admin_path': 'magenta',
};

export default function AuditLogs() {
  return (
    <ProTable<AuditLog>
      headerTitle="审计日志"
      rowKey="id"
      search={false}
      request={async (params) => {
        const { current = 1, pageSize = 20 } = params as any;
        const d = await request<{ data: AuditLog[]; total: number }>({
          url: '/admin/audit-logs',
          method: 'GET',
          params: { current, pageSize },
        });
        return { data: d.data, total: d.total, success: true };
      }}
      columns={[
        { title: 'ID', dataIndex: 'id', width: 70 },
        { title: '操作人', dataIndex: 'actor_email', width: 200 },
        {
          title: '动作',
          dataIndex: 'action',
          width: 220,
          render: (_, r) => <Tag color={actionColor[r.action] ?? 'default'}>{r.action}</Tag>,
        },
        {
          title: '详情',
          dataIndex: 'detail',
          ellipsis: true,
          render: (_, r) => (
            <Tooltip title={r.detail}>
              <span style={{ fontFamily: 'monospace', fontSize: 12 }}>{r.detail || '-'}</span>
            </Tooltip>
          ),
        },
        { title: 'IP', dataIndex: 'ip', width: 140 },
        { title: '时间', dataIndex: 'created_at', valueType: 'dateTime', width: 170 },
      ]}
      pagination={{ defaultPageSize: 10 }}
    />
  );
}

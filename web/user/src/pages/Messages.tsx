import { ProTable } from '@ant-design/pro-components';
import { Tag, Typography, Tooltip } from 'antd';
import { request, type MessageRecord } from '../api/client';

const statusMap: Record<string, { color: string; text: string }> = {
  PENDING: { color: 'default', text: '排队中' },
  SENDING: { color: 'processing', text: '发送中' },
  SUCCESS: { color: 'success', text: '送达' },
  FAILED: { color: 'error', text: '失败' },
  RETRYING: { color: 'warning', text: '重试中' },
  DEAD: { color: 'volcano', text: '死信' },
  CANCELLED: { color: 'default', text: '已取消' },
};

export default function Messages() {
  return (
    <ProTable<MessageRecord>
      headerTitle="消息记录"
      rowKey="message_id"
      search={{ labelWidth: 'auto' }}
      request={async (params) => {
        const { current = 1, pageSize = 20, ...rest } = params as any;
        const d = await request<{ data: MessageRecord[]; total: number }>({
          url: '/messages',
          method: 'GET',
          params: { current, pageSize, ...rest },
        });
        return { data: d.data, total: d.total, success: true };
      }}
      columns={[
        { title: 'ID', dataIndex: 'message_id', width: 80 },
        { title: '标题', dataIndex: 'title', width: 160, ellipsis: true },
        {
          title: '内容',
          dataIndex: 'content',
          ellipsis: true,
          render: (_, r) => (
            <Tooltip title={r.content}>
              <Typography.Text ellipsis style={{ maxWidth: 260, display: 'block' }}>{r.content}</Typography.Text>
            </Tooltip>
          ),
        },
        { title: '渠道', dataIndex: 'channel', width: 100, valueType: 'select', valueEnum: { webhook: { text: 'Webhook' }, email: { text: '邮件' }, apns: { text: 'APNs' }, inapp: { text: '站内信' } } },
        {
          title: '状态',
          dataIndex: 'status',
          width: 100,
          valueType: 'select',
          valueEnum: Object.fromEntries(Object.entries(statusMap).map(([k, v]) => [k, { text: v.text }])),
          render: (_, r) => {
            const s = statusMap[r.status] ?? { color: 'default', text: r.status };
            return <Tag color={s.color}>{s.text}</Tag>;
          },
        },
        { title: '错误', dataIndex: 'error', width: 140, ellipsis: true },
        { title: '时间', dataIndex: 'created_at', width: 170, valueType: 'dateTime' },
      ]}
      pagination={{ defaultPageSize: 10 }}
    />
  );
}

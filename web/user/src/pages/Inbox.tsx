import { ProTable } from '@ant-design/pro-components';
import { Tag, Button, message, Typography, Tooltip } from 'antd';
import { CheckOutlined } from '@ant-design/icons';
import { request } from '../api/client';

export interface InboxMessage {
  id: number;
  title: string;
  content: string;
  is_read: boolean;
  created_at: string;
}

export default function Inbox() {
  const markRead = async (id: number) => {
    await request({ url: `/inbox/${id}/read`, method: 'PUT', data: {} });
    message.success('已标记已读');
    return true;
  };

  const markAll = async () => {
    await request({ url: '/inbox/read-all', method: 'PUT', data: {} });
    message.success('已全部标记已读');
    return true;
  };

  return (
    <ProTable<InboxMessage>
      headerTitle="站内信"
      rowKey="id"
      search={false}
      toolBarRender={() => [
        <Button key="all" icon={<CheckOutlined />} onClick={() => markAll().then(() => location.reload())}>
          全部已读
        </Button>,
      ]}
      request={async (params) => {
        const { current = 1, pageSize = 20 } = params as any;
        const d = await request<{ data: InboxMessage[]; total: number }>({
          url: '/inbox',
          method: 'GET',
          params: { current, pageSize },
        });
        return { data: d.data, total: d.total, success: true };
      }}
      columns={[
        {
          title: '状态',
          dataIndex: 'is_read',
          width: 90,
          render: (_, r) =>
            r.is_read ? <Tag>已读</Tag> : <Tag color="processing" style={{ fontWeight: 600 }}>未读</Tag>,
        },
        { title: '标题', dataIndex: 'title', width: 180, render: (_, r) => <Typography.Text strong={!r.is_read}>{r.title}</Typography.Text> },
        {
          title: '内容',
          dataIndex: 'content',
          ellipsis: true,
          render: (_, r) => (
            <Tooltip title={r.content}>
              <Typography.Text ellipsis style={{ maxWidth: 320, display: 'block' }}>{r.content}</Typography.Text>
            </Tooltip>
          ),
        },
        { title: '时间', dataIndex: 'created_at', valueType: 'dateTime', width: 170 },
        {
          title: '操作',
          valueType: 'option',
          width: 110,
          render: (_, r) =>
            r.is_read
              ? []
              : [<a key="read" onClick={() => markRead(r.id).then(() => location.reload())}>标记已读</a>],
        },
      ]}
    />
  );
}

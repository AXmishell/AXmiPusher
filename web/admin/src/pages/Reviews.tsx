import { ProTable, ModalForm, ProFormTextArea } from '@ant-design/pro-components';
import { Button, message, Popconfirm, Tag, Typography, Space } from 'antd';
import { CheckOutlined, CloseOutlined } from '@ant-design/icons';
import { request, type ReviewItem } from '../api/client';

export default function Reviews() {
  const approve = async (r: ReviewItem) => {
    await request({
      url: `/admin/templates/${r.template_id}/versions/${r.version_id}/approve`,
      method: 'POST',
      data: { note: '审核通过' },
    });
    message.success('已批准');
    return true;
  };

  return (
    <ProTable<ReviewItem>
      headerTitle="模板审核"
      rowKey="version_id"
      search={false}
      params={{}}
      request={async (params) => {
        const { current = 1, pageSize = 20 } = params as any;
        const d = await request<{ data: ReviewItem[]; total: number }>({
          url: '/admin/templates/reviews',
          method: 'GET',
          params: { current, pageSize, review_status: 'pending' },
        });
        return { data: d.data, total: d.total, success: true };
      }}
      columns={[
        { title: 'ID', dataIndex: 'template_id', width: 70 },
        { title: '租户', dataIndex: 'tenant_name', width: 130 },
        { title: '编码', dataIndex: 'code', width: 130, render: (_, r) => <Tag>{r.code}</Tag> },
        { title: '名称', dataIndex: 'name', width: 120 },
        {
          title: '内容',
          dataIndex: 'content',
          ellipsis: true,
          render: (_, r) => (
            <Typography.Text ellipsis style={{ maxWidth: 300, display: 'block', fontFamily: 'monospace' }}>
              {r.content}
            </Typography.Text>
          ),
        },
        { title: '版本', dataIndex: 'version', width: 70, render: (_, r) => <Tag color="blue">v{r.version}</Tag> },
        { title: '提交时间', dataIndex: 'created_at', valueType: 'dateTime', width: 170 },
        {
          title: '操作',
          valueType: 'option',
          width: 200,
          render: (_, r) => [
            <Popconfirm key="a" title="批准后将立即生效，确定？" onConfirm={() => approve(r).then(() => location.reload())}>
              <Button key="a2" type="link" size="small" icon={<CheckOutlined />} style={{ color: '#16a34a' }}>
                批准
              </Button>
            </Popconfirm>,
            <ModalForm<any>
              key="r"
              title="驳回模板"
              width={480}
              trigger={
                <Button type="link" size="small" danger icon={<CloseOutlined />}>
                  驳回
                </Button>
              }
              onFinish={async (values) => {
                await request({
                  url: `/admin/templates/${r.template_id}/versions/${r.version_id}/reject`,
                  method: 'POST',
                  data: { note: values.note },
                });
                message.success('已驳回');
                return true;
              }}
              modalProps={{ destroyOnClose: true }}
            >
              <ProFormTextArea name="note" label="驳回原因" rules={[{ required: true, message: '请填写驳回原因' }]} fieldProps={{ rows: 3 }} />
            </ModalForm>,
          ],
        },
      ]}
    />
  );
}

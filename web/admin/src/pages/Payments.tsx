import { ProTable } from '@ant-design/pro-components';
import { Tag } from 'antd';
import { request, type PaymentOrder } from '../api/client';

const statusMap: Record<string, { color: string; text: string }> = {
  pending: { color: 'warning', text: '待支付' },
  paid: { color: 'success', text: '已支付' },
  closed: { color: 'default', text: '已关闭' },
};

export default function Payments() {
  return (
    <ProTable<PaymentOrder>
      headerTitle="支付订单"
      rowKey="id"
      search={{ labelWidth: 'auto' }}
      request={async (params) => {
        const { current = 1, pageSize = 20, status, ...rest } = params as any;
        const d = await request<{ data: PaymentOrder[]; total: number }>({
          url: '/admin/payment-orders',
          method: 'GET',
          params: { current, pageSize, status: status || undefined, ...rest },
        });
        return { data: d.data, total: d.total, success: true };
      }}
      columns={[
        { title: 'ID', dataIndex: 'id', width: 70 },
        { title: '租户 ID', dataIndex: 'tenant_id', width: 90 },
        { title: '平台订单号', dataIndex: 'out_trade_no', width: 190, copyable: true },
        { title: '易支付单号', dataIndex: 'epay_trade_no', width: 170, ellipsis: true },
        { title: '金额', dataIndex: 'amount', width: 100, render: (_, r) => `¥${r.amount}` },
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
        { title: '创建时间', dataIndex: 'created_at', valueType: 'dateTime', width: 170 },
      ]}
    />
  );
}

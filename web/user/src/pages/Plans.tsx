import { useEffect, useState } from 'react';
import { Card, Row, Col, Tag, Button, message, Statistic, Descriptions, Modal, Typography, Alert } from 'antd';
import { CheckCircleOutlined, CrownOutlined } from '@ant-design/icons';
import dayjs from 'dayjs';
import { request, type Plan } from '../api/client';

interface Subscription {
  id: number;
  plan_id: number;
  start_at: string;
  end_at: string;
  status: string;
}

const channelMap: Record<string, string> = { webhook: 'Webhook', email: '邮件', apns: 'APNs', fcm: 'FCM', inapp: '站内信' };

export default function Plans() {
  const [plans, setPlans] = useState<Plan[]>([]);
  const [sub, setSub] = useState<{ subscription: Subscription | null; plan: Plan | null } | null>(null);
  const [paying, setPaying] = useState<Plan | null>(null);
  const [payInfo, setPayInfo] = useState<{ order: any; pay_url: string } | null>(null);
  const [loading, setLoading] = useState(false);

  const load = async () => {
    try {
      const [p, s] = await Promise.all([
        request<{ data: Plan[] }>({ url: '/pay/plans', method: 'GET' }),
        request<{ subscription: Subscription | null; plan: Plan | null }>({ url: '/pay/subscription', method: 'GET' }),
      ]);
      setPlans(p.data);
      setSub(s);
    } catch { /* 拦截器已提示 */ }
  };

  useEffect(() => { load(); }, []);

  const createOrder = async (plan: Plan) => {
    setPaying(plan);
    try {
      const d = await request<{ order: any; pay_url: string }>({
        url: '/pay/orders',
        method: 'POST',
        data: { plan_id: plan.id, type: 'alipay' },
      });
      // 免费套餐: 后端已直接激活订阅(订单 status=paid 且无支付链接), 无需易支付。
      if (d.order?.status === 'paid' && !d.pay_url) {
        message.success(`「${plan.name}」已免费开通`);
        setPaying(null);
        await load();
        return;
      }
      setPayInfo(d);
    } catch { setPaying(null); }
  };

  // 本地调试: 模拟易支付回调。
  const simulatePay = async () => {
    setLoading(true);
    try {
      await request({ url: `/pay/orders/${payInfo!.order.id}/simulate`, method: 'POST', data: {} });
      message.success('模拟支付成功，套餐已生效');
      setPayInfo(null);
      setPaying(null);
      await load();
    } catch { /* 已提示 */ } finally { setLoading(false); }
  };

  return (
    <div>
      <Card style={{ marginBottom: 16 }}>
        <Descriptions title="当前订阅" column={2} size="middle">
          <Descriptions.Item label="套餐">
            {sub?.subscription ? <Tag color="gold" icon={<CrownOutlined />}>{sub.plan?.name}</Tag> : <Tag>未订阅（免费版）</Tag>}
          </Descriptions.Item>
          <Descriptions.Item label="有效期">
            {sub?.subscription
              ? `${dayjs(sub.subscription.start_at).format('YYYY-MM-DD')} ~ ${dayjs(sub.subscription.end_at).format('YYYY-MM-DD')}`
              : '-'}
          </Descriptions.Item>
          <Descriptions.Item label="月消息额度">
            {sub?.plan ? (() => { try { return `${Number(JSON.parse(sub.plan!.quota).monthly_messages).toLocaleString()} 条`; } catch { return '-'; } })() : '1,000 条'}
          </Descriptions.Item>
          <Descriptions.Item label="可用渠道">
            {sub?.plan ? (() => {
              try { return (JSON.parse(sub.plan!.quota).channels as string[]).map((c) => <Tag key={c} color="blue">{channelMap[c] || c}</Tag>); } catch { return '-'; }
            })() : <Tag color="blue">Webhook</Tag>}
          </Descriptions.Item>
        </Descriptions>
      </Card>

      <Row gutter={16}>
        {plans.map((p) => (
          <Col span={8} key={p.id}>
            <Card
              title={p.name}
              extra={p.price === 0 ? <Tag color="green">免费</Tag> : undefined}
              style={{ textAlign: 'center' }}
            >
              <Statistic
                value={p.price}
                prefix="¥"
                precision={p.price === 0 ? 0 : 2}
                valueStyle={{ fontSize: 32, fontWeight: 600 }}
              />
              <Typography.Text type="secondary">
                {p.duration_days} 天有效期 ·{' '}
                {(() => { try { return `${Number(JSON.parse(p.quota).monthly_messages).toLocaleString()} 条/月`; } catch { return '-'; } })()}
              </Typography.Text>
              <div style={{ marginTop: 8 }}>
                {(() => {
                  try { return (JSON.parse(p.quota).channels as string[]).map((c) => <Tag key={c}>{channelMap[c] || c}</Tag>); } catch { return null; }
                })()}
              </div>
              <Typography.Paragraph type="secondary" style={{ minHeight: 40, marginTop: 8 }}>{p.description}</Typography.Paragraph>
              <Button
                type={p.price === 0 ? 'default' : 'primary'}
                block
                onClick={() => createOrder(p)}
                disabled={sub?.subscription?.plan_id === p.id}
              >
                {sub?.subscription?.plan_id === p.id ? '当前套餐' : p.price === 0 ? '免费开通' : '立即购买'}
              </Button>
            </Card>
          </Col>
        ))}
      </Row>

      <Modal
        title="确认支付"
        open={!!payInfo}
        onCancel={() => { setPayInfo(null); setPaying(null); }}
        footer={[
          <Button key="local" type="primary" loading={loading} onClick={simulatePay} style={{ marginRight: 8 }}>
            模拟支付（本地测试）
          </Button>,
          <Button key="real" onClick={() => window.open(payInfo?.pay_url, '_blank')}>
            跳转易支付
          </Button>,
        ]}
      >
        <Alert
          type="info"
          showIcon
          message={`订单 ${payInfo?.order?.out_trade_no}`}
          description={`金额 ¥${payInfo?.order?.amount} · 请在 30 分钟内完成支付`}
          style={{ marginBottom: 12 }}
        />
        <Typography.Paragraph type="secondary" style={{ fontSize: 12 }}>
          本地环境无真实网关，可用"模拟支付"触发回调（内部走完整验签链路）。
        </Typography.Paragraph>
      </Modal>
    </div>
  );
}

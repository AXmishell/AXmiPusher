import { Outlet, useLocation, useNavigate } from 'react-router-dom';
import { ProLayout } from '@ant-design/pro-components';
import {
  DashboardOutlined,
  SendOutlined,
  ProfileOutlined,
  KeyOutlined,
  LinkOutlined,
  LogoutOutlined,
  UserOutlined,
  CrownOutlined,
  ThunderboltOutlined,
  MailOutlined,
  CarryOutOutlined,
  LockOutlined,
} from '@ant-design/icons';
import { Dropdown, Space, Avatar, Badge } from 'antd';
import { useEffect, useState } from 'react';
import { request, type User, type Tenant } from '../api/client';
import ChangePasswordModal from '../components/ChangePasswordModal';

export default function MainLayout() {
  const navigate = useNavigate();
  const location = useLocation();
  const [user, setUser] = useState<User | null>(null);
  const [tenant, setTenant] = useState<Tenant | null>(null);
  const [unread, setUnread] = useState(0);
  const [pwdOpen, setPwdOpen] = useState(false);

  useEffect(() => {
    const token = localStorage.getItem('mp_token');
    if (!token) {
      navigate('/login', { replace: true });
      return;
    }
    request<{ user: User; tenant: Tenant }>({ url: '/auth/me', method: 'GET' })
      .then((d) => {
        setUser(d.user);
        setTenant(d.tenant);
        // 拉取未读站内信数(菜单角标)。
        request<{ unread: number }>({ url: '/inbox/unread-count', method: 'GET' })
          .then((u) => setUnread(u.unread))
          .catch(() => {});
      })
      .catch(() => {
        localStorage.removeItem('mp_token');
        navigate('/login', { replace: true });
      });
  }, [navigate]);

  const logout = () => {
    localStorage.removeItem('mp_token');
    navigate('/login', { replace: true });
  };

  return (
    <ProLayout
      title="MessagePusher"
      logo={<SendOutlined style={{ fontSize: 22, color: '#1e40af' }} />}
      location={{ pathname: location.pathname }}
      route={{
        path: '/',
        routes: [
          { path: '/', name: '概览', icon: <DashboardOutlined /> },
          { path: '/send', name: '发送消息', icon: <SendOutlined /> },
          { path: '/messages', name: '消息记录', icon: <ProfileOutlined /> },
          { path: '/api-keys', name: 'API Key', icon: <KeyOutlined /> },
          { path: '/callbacks', name: '回调订阅', icon: <LinkOutlined /> },
          { path: '/plans', name: '套餐订阅', icon: <CrownOutlined /> },
          { path: '/channels', name: '渠道配置', icon: <ThunderboltOutlined /> },
          { path: '/inbox', name: '站内信', icon: <MailOutlined /> },
          { path: '/batch-tasks', name: '批量任务', icon: <CarryOutOutlined /> },
        ],
      }}
      menuItemRender={(item, dom) => (
        <a onClick={() => navigate(item.path!)}>
          {item.path === '/inbox' && unread > 0 ? (
            <Badge count={unread} size="small" offset={[6, 0]}>
              {dom}
            </Badge>
          ) : (
            dom
          )}
        </a>
      )}
      avatarProps={{
        icon: <UserOutlined />,
        title: user?.email || '未登录',
        render: (_: any, dom: React.ReactNode) => (
          <Dropdown
            menu={{
              items: [
                { key: 'password', icon: <LockOutlined />, label: '修改密码' },
                { type: 'divider' },
                { key: 'logout', icon: <LogoutOutlined />, label: '退出登录' },
              ],
              onClick: ({ key }) => {
                if (key === 'password') setPwdOpen(true);
                if (key === 'logout') logout();
              },
            }}
          >
            {dom}
          </Dropdown>
        ),
      }}
      actionsRender={() => [
        <Space key="tenant" style={{ marginRight: 16, color: 'rgba(0,0,0,.65)' }}>
          <Avatar size="small" style={{ background: '#1e40af' }}>
            {tenant?.name?.charAt(0) || 'T'}
          </Avatar>
          租户: {tenant?.name || '-'}
        </Space>,
      ]}
      layout="mix"
    >
      <Outlet />
      <ChangePasswordModal open={pwdOpen} onClose={() => setPwdOpen(false)} />
    </ProLayout>
  );
}

import { Outlet, useLocation, useNavigate } from 'react-router-dom';
import { ProLayout } from '@ant-design/pro-components';
import {
  DashboardOutlined,
  TeamOutlined,
  UserOutlined,
  AuditOutlined,
  GiftOutlined,
  PayCircleOutlined,
  FileSearchOutlined,
  SettingOutlined,
  LogoutOutlined,
  SafetyCertificateOutlined,
  LockOutlined,
} from '@ant-design/icons';
import { Dropdown, Space, Avatar } from 'antd';
import { useEffect, useState } from 'react';
import { request, type User } from '../api/client';
import ChangePasswordModal from '../components/ChangePasswordModal';

export default function MainLayout() {
  const navigate = useNavigate();
  const location = useLocation();
  const [user, setUser] = useState<User | null>(null);
  const [pwdOpen, setPwdOpen] = useState(false);

  useEffect(() => {
    const token = localStorage.getItem('mp_admin_token');
    if (!token) {
      navigate('/login', { replace: true });
      return;
    }
    request<{ user: User; is_admin: boolean }>({ url: '/auth/me', method: 'GET' })
      .then((d) => {
        if (!d.is_admin) {
          localStorage.removeItem('mp_admin_token');
          navigate('/login', { replace: true });
          return;
        }
        setUser(d.user);
      })
      .catch(() => {
        localStorage.removeItem('mp_admin_token');
        navigate('/login', { replace: true });
      });
  }, [navigate]);

  const logout = () => {
    localStorage.removeItem('mp_admin_token');
    navigate('/login', { replace: true });
  };

  return (
    <ProLayout
      title="MessagePusher 管理后台"
      logo={<SafetyCertificateOutlined style={{ fontSize: 22, color: '#7c3aed' }} />}
      location={{ pathname: location.pathname }}
      route={{
        path: '/',
        routes: [
          { path: '/', name: '平台概览', icon: <DashboardOutlined /> },
          { path: '/tenants', name: '租户管理', icon: <TeamOutlined /> },
          { path: '/users', name: '用户管理', icon: <UserOutlined /> },
          { path: '/reviews', name: '模板审核', icon: <AuditOutlined /> },
          { path: '/plans', name: '套餐管理', icon: <GiftOutlined /> },
          { path: '/payments', name: '支付订单', icon: <PayCircleOutlined /> },
          { path: '/audit-logs', name: '审计日志', icon: <FileSearchOutlined /> },
          { path: '/settings', name: '系统设置', icon: <SettingOutlined /> },
        ],
      }}
      menuItemRender={(item, dom) => <a onClick={() => navigate(item.path!)}>{dom}</a>}
      avatarProps={{
        icon: <UserOutlined />,
        title: user?.email || '管理员',
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
        <Space key="role" style={{ marginRight: 16, color: 'rgba(0,0,0,.65)' }}>
          <Avatar size="small" style={{ background: '#7c3aed' }}>P</Avatar>
          平台管理员
        </Space>,
      ]}
      layout="mix"
    >
      <Outlet />
      <ChangePasswordModal open={pwdOpen} onClose={() => setPwdOpen(false)} />
    </ProLayout>
  );
}

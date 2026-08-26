import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import MainLayout from './layouts/MainLayout';
import Login from './pages/Login';
import Register from './pages/Register';
import Dashboard from './pages/Dashboard';
import Messages from './pages/Messages';
import SendMessage from './pages/SendMessage';
import ApiKeys from './pages/ApiKeys';
import Callbacks from './pages/Callbacks';
import Plans from './pages/Plans';
import Channels from './pages/Channels';
import Inbox from './pages/Inbox';
import BatchTasks from './pages/BatchTasks';

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/login" element={<Login />} />
        <Route path="/register" element={<Register />} />
        <Route path="/" element={<MainLayout />}>
          <Route index element={<Dashboard />} />
          <Route path="messages" element={<Messages />} />
          <Route path="send" element={<SendMessage />} />
          <Route path="api-keys" element={<ApiKeys />} />
          <Route path="compat-keys" element={<Navigate to="/api-keys" replace />} />
          <Route path="callbacks" element={<Callbacks />} />
          <Route path="plans" element={<Plans />} />
          <Route path="channels" element={<Channels />} />
          <Route path="inbox" element={<Inbox />} />
          <Route path="batch-tasks" element={<BatchTasks />} />
        </Route>
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </BrowserRouter>
  );
}

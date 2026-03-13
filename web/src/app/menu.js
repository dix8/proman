import {
  ApiOutlined,
  AppstoreOutlined,
  FileSearchOutlined,
  NotificationOutlined,
} from "@ant-design/icons";

export const adminMenuItems = [
  {
    key: "/projects",
    icon: AppstoreOutlined,
    label: "项目管理",
    path: "/projects",
  },
  {
    key: "/versions/compare",
    icon: FileSearchOutlined,
    label: "版本对比",
    path: "/versions/compare",
  },
  {
    key: "/announcements",
    icon: NotificationOutlined,
    label: "公告管理",
    path: "/announcements",
  },
  {
    key: "/integration",
    icon: ApiOutlined,
    label: "接口接入",
    path: "/integration",
  },
];

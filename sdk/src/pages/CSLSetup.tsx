import { useState } from 'react';
import {
  Box,
  Button,
  Card,
  CardContent,
  Link,
  List,
  ListItem,
  ListItemText,
  TextField,
  Typography,
} from '@mui/material';

/**
 * CustomSkinLoader 接入配置页（HASkinProxy SDK 路由模块）。
 *
 * 功能与原 /customskinloader 静态页一致：生成 CustomSkinAPI
 * config.json 并复制 / 下载。CustomSkinAPI 根地址默认取
 * window.__BACKEND_URL__（HRPAuth 注入的主服务回源地址）+ /csl/
 * （HASkinProxy 的 relay dest），与 CSL 客户端的实际访问路径一致。
 *
 * 注意：本组件被 SDKHandler 编译进 HRPAuth-Web，只能使用 HRPAuth-Web
 * 已有的依赖（React 19 / MUI 7），且不得引用 window.location 以外的
 * 运行时环境（iframe 内嵌场景由外层 Dashboard 决定）。
 */
const BACKEND_URL = typeof window !== 'undefined' ? window.__BACKEND_URL__ : '';

export default function CSLSetup() {
  const [root, setRoot] = useState(() => {
    const base = (BACKEND_URL || '').replace(/\/+$/, '');
    return `${base || window.location.origin}/csl/`;
  });
  const [copied, setCopied] = useState(false);

  const configJson = buildConfig(root);

  const handleCopy = async () => {
    await navigator.clipboard.writeText(configJson);
    setCopied(true);
    setTimeout(() => setCopied(false), 1500);
  };

  const handleDownload = () => {
    const blob = new Blob([configJson], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = 'config.json';
    a.click();
    URL.revokeObjectURL(url);
  };

  return (
    <Box sx={{ maxWidth: 720, mx: 'auto', p: 3 }}>
      <Typography variant="h5" component="h1" sx={{ mb: 2 }}>
        CustomSkinLoader 接入配置
      </Typography>

      <Card sx={{ mb: 3 }}>
        <CardContent>
          <Typography variant="h6" sx={{ mb: 1 }}>
            1. 生成配置文件
          </Typography>
          <TextField
            fullWidth
            label="CustomSkinAPI 根地址（必须以 / 结尾）"
            value={root}
            onChange={(e) => setRoot(e.target.value)}
            sx={{ mb: 2 }}
          />
          <Box
            component="pre"
            sx={{
              bgcolor: 'rgba(127,127,127,.12)',
              borderRadius: 1,
              p: 1.5,
              overflowX: 'auto',
              fontSize: '0.8rem',
              mb: 2,
            }}
          >
            {configJson}
          </Box>
          <Box sx={{ display: 'flex', gap: 1 }}>
            <Button variant="contained" onClick={handleCopy}>
              {copied ? '已复制!' : '复制 config.json'}
            </Button>
            <Button variant="outlined" onClick={handleDownload}>
              下载 config.json
            </Button>
          </Box>
        </CardContent>
      </Card>

      <Card>
        <CardContent>
          <Typography variant="h6" sx={{ mb: 1 }}>
            2. 使用说明
          </Typography>
          <List dense>
            <ListItem>
              <ListItemText>
                安装{' '}
                <Link
                  href="https://github.com/xfl03/MCCustomSkinLoader"
                  target="_blank"
                  rel="noopener"
                >
                  CustomSkinLoader
                </Link>{' '}
                到你的 Minecraft 版本目录（建议使用官方或兼容的加载器）。
              </ListItemText>
            </ListItem>
            <ListItem>
              <ListItemText>
                将生成的 <code>config.json</code> 放入{' '}
                <code>.minecraft/customskinloader/</code> 目录（没有则新建）。
              </ListItemText>
            </ListItem>
            <ListItem>
              <ListItemText>
                启动游戏，登录后皮肤与披风将通过本服务自动加载。
              </ListItemText>
            </ListItem>
            <ListItem>
              <ListItemText>
                若其他玩家的皮肤加载异常，请确认其客户端可以访问上面的根地址。
              </ListItemText>
            </ListItem>
          </List>
        </CardContent>
      </Card>
    </Box>
  );
}

function buildConfig(root: string): string {
  const trimmed = root.trim();
  const normalized = trimmed && !trimmed.endsWith('/') ? `${trimmed}/` : trimmed;
  return JSON.stringify(
    {
      enable: true,
      loadlist: [
        {
          name: 'HASkinProxy',
          type: 'CustomSkinAPI',
          root: normalized,
        },
      ],
    },
    null,
    2,
  );
}

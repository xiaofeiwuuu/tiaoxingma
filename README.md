# 热敏打印服务

一个基于 Go 语言开发的 Windows LPT 端口热敏打印机控制服务，提供 HTTP API 接口供网页调用。

## 功能特点

- ✅ 支持 LPT 并行端口打印机
- ✅ 提供 HTTP API 接口
- ✅ 支持 ESC/POS 指令集
- ✅ 内置测试页面
- ✅ 支持跨域访问（CORS）
- ✅ 单文件可执行程序，无需安装

## 下载使用

### 方法一：直接下载

1. 访问 [Releases](https://github.com/xiaofeiwuuu/tiaoxingma/releases) 页面
2. 下载对应版本：
   - `打印服务-win64.exe` - Windows 64位系统
   - `打印服务-win32.exe` - Windows 32位系统
3. 双击运行即可

### 方法二：自动构建

每次提交代码后，GitHub Actions 会自动构建最新版本，可在 [Actions](https://github.com/xiaofeiwuuu/tiaoxingma/actions) 页面下载。

## 使用说明

### 1. 启动服务

双击运行 exe 文件，服务会自动启动并打开测试页面。

```
=======================================
    热敏打印机服务已启动
=======================================
服务地址: http://localhost:9100
测试页面: http://localhost:9100/test
API接口: http://localhost:9100/api/print

按 Ctrl+C 停止服务
=======================================
```

### 2. API 接口

#### 打印接口
- **URL**: `POST http://localhost:9100/api/print`
- **Content-Type**: `application/json`
- **请求示例**:

```json
{
  "content": "要打印的内容",
  "cut": true,      // 是否切纸
  "bold": false,    // 是否加粗
  "center": false,  // 是否居中
  "fontSize": 3     // 字体大小 (1-5)
}
```

- **响应示例**:

```json
{
  "status": "success",
  "message": "打印成功"
}
```

#### 状态检查
- **URL**: `GET http://localhost:9100/api/status`
- **响应示例**:

```json
{
  "status": "running",
  "version": "1.0.0",
  "port": "LPT1"
}
```

### 3. 网页调用示例

```javascript
// JavaScript 调用示例
async function printReceipt() {
    const printData = {
        content: "订单号：2024001\n商品：可乐\n价格：5元",
        cut: true,
        bold: false,
        center: true,
        fontSize: 3
    };

    try {
        const response = await fetch('http://localhost:9100/api/print', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(printData)
        });

        const result = await response.json();
        console.log(result.message);
    } catch (error) {
        console.error('打印失败:', error);
    }
}
```

## 支持的打印机

支持所有兼容 ESC/POS 指令集的热敏打印机，包括但不限于：
- EPSON 系列
- 佳博（Gprinter）系列
- 芯烨（Xprinter）系列
- 其他支持 ESC/POS 的热敏打印机

## 常见问题

### 1. 无法连接打印机
- 确保打印机连接到 LPT1 端口
- 检查打印机电源是否开启
- 尝试以管理员权限运行程序

### 2. 打印乱码
- 确保打印内容编码正确
- 检查打印机是否支持中文

### 3. 无法访问服务
- 检查防火墙设置
- 确保 9100 端口未被占用

## 开发说明

### 环境要求
- Go 1.21 或更高版本

### 本地编译
```bash
# Windows 64位
GOOS=windows GOARCH=amd64 go build -o 打印服务.exe print-server.go

# Windows 32位  
GOOS=windows GOARCH=386 go build -o 打印服务32.exe print-server.go
```

## 许可证

MIT License
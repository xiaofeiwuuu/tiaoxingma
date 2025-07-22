# 条形码打印服务 API 文档

## 概述

条形码打印服务是一个基于 HTTP 的本地服务，用于通过 LPT 端口控制热敏打印机打印条形码。服务默认运行在 `http://localhost:9100`。

## API 端点

### 1. 打印条形码

打印指定类型和内容的条形码。

- **URL**: `/api/print`
- **方法**: `POST`
- **Content-Type**: `application/json`

#### 请求参数

| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|---------|------|
| `barcodeType` | string | 否 | "CODE128" | 条形码类型，可选值见下表 |
| `barcodeData` | string | 是 | - | 要编码的数据内容 |
| `showText` | boolean | 否 | false | 是否在条形码下方显示文字 |
| `center` | boolean | 否 | false | 是否居中打印 |
| `cut` | boolean | 否 | false | 打印完成后是否切纸 |
| `barcodeWidth` | integer | 否 | 3 | 条形码线条宽度 (2-6) |
| `barcodeHeight` | integer | 否 | 100 | 条形码高度 (1-255) |

#### 支持的条形码类型

| 类型 | 说明 | 支持的字符 | 示例 |
|------|------|------------|------|
| `CODE128` | 推荐，支持所有字符 | 所有 ASCII 字符（大小写字母、数字、符号） | UUID、订单号等 |
| `CODE128A` | 备选 CODE128 格式 | 同上 | 当主格式无法扫描时使用 |
| `CODE39` | 传统格式 | 大写字母、数字、部分符号（- . $ / + % 空格） | 简单编号 |
| `EAN13` | 商品条码 | 12或13位数字（自动计算校验码） | 6901234567890 |
| `EAN8` | 短条码 | 8位数字 | 12345678 |

#### 请求示例

**打印 UUID：**
```json
{
  "barcodeType": "CODE128",
  "barcodeData": "550e8400-e29b-41d4-a716-446655440000",
  "showText": true,
  "center": true,
  "cut": true,
  "barcodeWidth": 3,
  "barcodeHeight": 80
}
```

**打印商品条码（EAN-13）：**
```json
{
  "barcodeType": "EAN13",
  "barcodeData": "690123456789",  // 12位，自动计算第13位
  "showText": true,
  "center": true,
  "cut": true
}
```

**打印订单号：**
```json
{
  "barcodeType": "CODE128",
  "barcodeData": "ORD-2024-001234",
  "showText": true,
  "center": false,
  "cut": true,
  "barcodeHeight": 100
}
```

#### 响应格式

**成功响应：**
```json
{
  "status": "success",
  "message": "打印成功"
}
```

**错误响应：**
```json
{
  "status": "error",
  "message": "错误描述信息"
}
```

#### 常见错误

| 错误信息 | 原因 | 解决方法 |
|----------|------|----------|
| 无法打开打印机端口 | LPT1 端口不存在或无权限 | 检查打印机连接和权限 |
| EAN13条形码必须是12或13位数字 | EAN-13 数据格式错误 | 提供12或13位数字 |
| EAN8条形码必须是8位数字 | EAN-8 数据格式错误 | 提供8位数字 |

---

### 2. 服务状态检查

检查打印服务是否正常运行。

- **URL**: `/api/status`
- **方法**: `GET`

#### 响应示例

```json
{
  "status": "running",
  "version": "3.0.0",
  "port": "LPT1",
  "features": ["barcode"]
}
```

#### 响应字段说明

| 字段 | 类型 | 说明 |
|------|------|------|
| `status` | string | 服务状态，固定为 "running" |
| `version` | string | 服务版本号 |
| `port` | string | 使用的打印机端口 |
| `features` | array | 支持的功能列表 |

---

### 3. 测试页面

提供可视化的测试界面。

- **URL**: `/test`
- **方法**: `GET`
- **说明**: 在浏览器中访问此地址可打开测试页面

---

## 使用示例

### JavaScript (浏览器)

```javascript
async function printBarcode(data) {
    try {
        const response = await fetch('http://localhost:9100/api/print', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                barcodeType: 'CODE128',
                barcodeData: data,
                showText: true,
                center: true,
                cut: true
            })
        });
        
        const result = await response.json();
        if (result.status === 'success') {
            console.log('打印成功');
        } else {
            console.error('打印失败:', result.message);
        }
    } catch (error) {
        console.error('请求失败:', error);
    }
}

// 使用示例
printBarcode('550e8400-e29b-41d4-a716-446655440000');
```

### Python

```python
import requests
import json

def print_barcode(data):
    url = 'http://localhost:9100/api/print'
    payload = {
        'barcodeType': 'CODE128',
        'barcodeData': data,
        'showText': True,
        'center': True,
        'cut': True
    }
    
    try:
        response = requests.post(url, json=payload)
        result = response.json()
        
        if result['status'] == 'success':
            print('打印成功')
        else:
            print(f'打印失败: {result["message"]}')
    except Exception as e:
        print(f'请求失败: {e}')

# 使用示例
print_barcode('550e8400-e29b-41d4-a716-446655440000')
```

### cURL

```bash
# 打印 UUID
curl -X POST http://localhost:9100/api/print \
  -H "Content-Type: application/json" \
  -d '{
    "barcodeType": "CODE128",
    "barcodeData": "550e8400-e29b-41d4-a716-446655440000",
    "showText": true,
    "center": true,
    "cut": true
  }'

# 检查服务状态
curl http://localhost:9100/api/status
```

---

## 注意事项

1. **字符编码**：
   - CODE128 支持所有 ASCII 字符，适合 UUID、订单号等
   - CODE39 仅支持大写字母和数字，输入小写字母会导致打印失败
   - EAN 系列仅支持数字

2. **EAN-13 校验码**：
   - 输入 12 位数字时，系统会自动计算第 13 位校验码
   - 输入 13 位数字时，系统会重新计算校验码以确保正确

3. **打印机兼容性**：
   - 需要支持 ESC/POS 指令集的热敏打印机
   - 打印机必须连接到 LPT1 端口

4. **跨域访问**：
   - 服务已启用 CORS，支持任意来源的跨域请求
   - 在生产环境中建议限制允许的来源

5. **错误处理**：
   - 始终检查响应的 `status` 字段
   - 错误信息会在 `message` 字段中提供详细说明

---

## 故障排查

### 打印机无响应
1. 检查打印机是否开机
2. 确认打印机连接到 LPT1 端口
3. 检查 Windows 设备管理器中 LPT1 端口是否正常

### 条形码无法扫描
1. 检查条形码类型是否正确
2. 调整 `barcodeWidth` 和 `barcodeHeight` 参数
3. 确保打印机墨带充足

### 扫码结果错误
1. CODE128 扫出全是数字：使用主格式而非备选格式
2. 字符缺失：检查输入数据是否包含不支持的字符

---

## 版本历史

- **v3.0.0** - 专注条形码打印，移除文本打印功能
- **v2.0.0** - 添加条形码支持和中文编码
- **v1.0.0** - 初始版本

---

## 联系支持

项目地址：https://github.com/xiaofeiwuuu/tiaoxingma
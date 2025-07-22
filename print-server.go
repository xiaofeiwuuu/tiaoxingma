package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// PrintRequest 打印请求结构
type PrintRequest struct {
	BarcodeType   string `json:"barcodeType"`   // 条形码类型：CODE128, CODE39, EAN13 等
	BarcodeData   string `json:"barcodeData"`   // 条形码数据
	ShowText      bool   `json:"showText"`      // 是否显示条形码文字
	Cut           bool   `json:"cut"`           // 是否切纸
	Center        bool   `json:"center"`        // 是否居中
	BarcodeWidth  int    `json:"barcodeWidth"`  // 条形码宽度 (2-6)
	BarcodeHeight int    `json:"barcodeHeight"` // 条形码高度 (1-255)
}

// PrintResponse 打印响应结构
type PrintResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

func main() {
	// 设置服务端口
	port := ":9100"
	
	// 注册路由
	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/api/print", printHandler)
	http.HandleFunc("/api/status", statusHandler)
	http.HandleFunc("/test", testPageHandler)

	// 启动服务
	fmt.Println("=======================================")
	fmt.Println("    条形码打印服务已启动")
	fmt.Println("=======================================")
	fmt.Printf("服务地址: http://localhost%s\n", port)
	fmt.Println("测试页面: http://localhost" + port + "/test")
	fmt.Println("API接口: http://localhost" + port + "/api/print")
	fmt.Println("\n按 Ctrl+C 停止服务")
	fmt.Println("=======================================")

	// 自动打开测试页面
	go func() {
		openBrowser("http://localhost" + port + "/test")
	}()

	// 启动HTTP服务
	log.Fatal(http.ListenAndServe(port, nil))
}

// 主页处理器
func homeHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	html := `
	<!DOCTYPE html>
	<html>
	<head>
		<title>条形码打印服务</title>
		<meta charset="UTF-8">
	</head>
	<body style="font-family: Arial; text-align: center; margin-top: 50px;">
		<h1>条形码打印服务运行中</h1>
		<p>访问 <a href="/test">测试页面</a> 进行打印测试</p>
		<p>API文档：POST /api/print</p>
	</body>
	</html>
	`
	w.Write([]byte(html))
}

// 状态检查处理器
func statusHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "running",
		"version": "3.0.0",
		"port": "LPT1",
		"features": []string{"barcode"},
	})
}

// 打印处理器
func printHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	
	if r.Method == "OPTIONS" {
		return
	}

	if r.Method != "POST" {
		sendError(w, "仅支持POST请求")
		return
	}

	// 解析请求
	var req PrintRequest
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		sendError(w, "读取请求失败")
		return
	}

	if err := json.Unmarshal(body, &req); err != nil {
		sendError(w, "解析JSON失败")
		return
	}

	// 设置默认值
	if req.BarcodeWidth == 0 {
		req.BarcodeWidth = 3
	}
	if req.BarcodeHeight == 0 {
		req.BarcodeHeight = 100
	}
	if req.BarcodeType == "" {
		req.BarcodeType = "CODE128"
	}

	// 执行打印
	if err := printBarcode(&req); err != nil {
		sendError(w, err.Error())
		return
	}

	// 返回成功
	sendSuccess(w, "打印成功")
}

// 打印条形码
func printBarcode(req *PrintRequest) error {
	// 打开LPT1端口
	printer, err := os.OpenFile(`\\.\LPT1`, os.O_WRONLY, 0644)
	if err != nil {
		// 尝试其他格式
		printer, err = os.OpenFile("LPT1", os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("无法打开打印机端口: %v", err)
		}
	}
	defer printer.Close()

	// 初始化打印机 (ESC @)
	printer.Write([]byte("\x1B\x40"))

	// 设置打印浓度（有助于条形码清晰）
	// ESC 7 n (n=0-7, 数值越大越浓)
	printer.Write([]byte{0x1B, 0x37, 0x07})

	// 设置条形码高度
	// GS h n
	printer.Write([]byte{0x1D, 0x68, byte(req.BarcodeHeight)})

	// 设置条形码宽度
	// GS w n (n = 2-6)
	if req.BarcodeWidth >= 2 && req.BarcodeWidth <= 6 {
		printer.Write([]byte{0x1D, 0x77, byte(req.BarcodeWidth)})
	}

	// 设置是否打印条形码下方的文字
	// GS H n (0=不打印, 1=上方, 2=下方, 3=上下都打印)
	if req.ShowText {
		printer.Write([]byte{0x1D, 0x48, 0x02}) // 下方打印
		// 设置字体
		printer.Write([]byte{0x1D, 0x66, 0x00}) // 字体A
	} else {
		printer.Write([]byte{0x1D, 0x48, 0x00}) // 不打印
	}

	// 设置居中
	if req.Center {
		printer.Write([]byte("\x1B\x61\x01"))
	}

	// 添加空行（确保条形码上方有空间）
	printer.Write([]byte("\n"))

	// 选择条形码类型并打印
	barcodeType := strings.ToUpper(req.BarcodeType)
	data := []byte(req.BarcodeData)
	
	switch barcodeType {
	case "CODE39":
		// GS k 4 n d1...dn
		printer.Write([]byte{0x1D, 0x6B, 0x04})
		printer.Write([]byte{byte(len(data))})
		printer.Write(data)
		
	case "EAN13":
		// GS k 67 13 d1...d13
		if len(req.BarcodeData) == 13 {
			printer.Write([]byte{0x1D, 0x6B, 0x43, 0x0D})
			printer.Write(data)
		} else {
			return fmt.Errorf("EAN13条形码必须是13位数字")
		}
		
	case "EAN8":
		// GS k 68 8 d1...d8
		if len(req.BarcodeData) == 8 {
			printer.Write([]byte{0x1D, 0x6B, 0x44, 0x08})
			printer.Write(data)
		} else {
			return fmt.Errorf("EAN8条形码必须是8位数字")
		}
		
	case "CODE128":
		fallthrough
	default:
		// GS k 73 n d1...dn (CODE128)
		// 需要添加CODE128的起始码
		printer.Write([]byte{0x1D, 0x6B, 0x49})
		printer.Write([]byte{byte(len(data) + 2)})
		printer.Write([]byte{0x7B, 0x42}) // {B 表示 CODE128 B型
		printer.Write(data)
	}

	// 添加足够的换行确保条形码完整打印
	printer.Write([]byte("\n\n\n"))

	// 取消居中
	if req.Center {
		printer.Write([]byte("\x1B\x61\x00"))
	}

	// 切纸
	if req.Cut {
		// 走纸一段距离
		printer.Write([]byte("\n\n\n\n"))
		// GS V m (切纸)
		printer.Write([]byte{0x1D, 0x56, 0x00}) // 全切
	}

	return nil
}

// 测试页面处理器
func testPageHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(testHTML))
}

// 启用CORS
func enableCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
}

// 发送错误响应
func sendError(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(w).Encode(PrintResponse{
		Status:  "error",
		Message: message,
	})
}

// 发送成功响应
func sendSuccess(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(PrintResponse{
		Status:  "success",
		Message: message,
	})
}

// 打开浏览器
func openBrowser(url string) {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "rundll32"
		args = []string{"url.dll,FileProtocolHandler", url}
	case "darwin":
		cmd = "open"
		args = []string{url}
	default:
		cmd = "xdg-open"
		args = []string{url}
	}

	exec.Command(cmd, args...).Start()
}

// 测试页面HTML
const testHTML = `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>条形码打印测试</title>
    <style>
        body {
            font-family: Arial, sans-serif;
            max-width: 600px;
            margin: 0 auto;
            padding: 20px;
            background-color: #f5f5f5;
        }
        .container {
            background: white;
            padding: 30px;
            border-radius: 10px;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
        }
        h1 {
            color: #333;
            text-align: center;
        }
        input[type="text"] {
            width: 100%;
            padding: 10px;
            border: 1px solid #ddd;
            border-radius: 5px;
            font-size: 16px;
            box-sizing: border-box;
            margin: 10px 0;
        }
        .controls {
            margin: 20px 0;
            padding: 20px;
            background: #f9f9f9;
            border-radius: 5px;
        }
        .control-group {
            margin: 15px 0;
            display: flex;
            align-items: center;
            gap: 20px;
            flex-wrap: wrap;
        }
        label {
            display: flex;
            align-items: center;
            gap: 5px;
            cursor: pointer;
            font-weight: bold;
        }
        input[type="checkbox"] {
            width: 18px;
            height: 18px;
            cursor: pointer;
        }
        select {
            padding: 8px 12px;
            border: 1px solid #ddd;
            border-radius: 3px;
            font-size: 14px;
            background: white;
        }
        input[type="number"] {
            width: 80px;
            padding: 8px;
            border: 1px solid #ddd;
            border-radius: 3px;
        }
        button {
            background: #007bff;
            color: white;
            padding: 12px 30px;
            border: none;
            border-radius: 5px;
            font-size: 18px;
            cursor: pointer;
            transition: background 0.3s;
            width: 100%;
            margin-top: 20px;
        }
        button:hover {
            background: #0056b3;
        }
        button:active {
            transform: translateY(1px);
        }
        .status {
            margin-top: 20px;
            padding: 15px;
            border-radius: 5px;
            text-align: center;
            display: none;
            font-weight: bold;
        }
        .status.success {
            background: #d4edda;
            color: #155724;
            border: 1px solid #c3e6cb;
        }
        .status.error {
            background: #f8d7da;
            color: #721c24;
            border: 1px solid #f5c6cb;
        }
        .info {
            background: #d1ecf1;
            color: #0c5460;
            padding: 12px;
            border-radius: 5px;
            margin: 10px 0;
            font-size: 14px;
        }
        .examples {
            margin-top: 15px;
        }
        .example-btn {
            background: #6c757d;
            color: white;
            padding: 6px 12px;
            border: none;
            border-radius: 3px;
            font-size: 14px;
            cursor: pointer;
            margin: 5px;
        }
        .example-btn:hover {
            background: #5a6268;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>🏷️ 条形码打印测试</h1>
        
        <div class="controls">
            <div class="control-group">
                <label>条形码类型：</label>
                <select id="barcode-type" onchange="updateBarcodeInfo()">
                    <option value="CODE128">CODE128（推荐）</option>
                    <option value="CODE39">CODE39</option>
                    <option value="EAN13">EAN-13（商品条码）</option>
                    <option value="EAN8">EAN-8（短条码）</option>
                </select>
            </div>
            
            <div class="info" id="barcode-info">
                CODE128：支持所有ASCII字符，包括字母、数字和符号
            </div>

            <label>条形码数据：</label>
            <input type="text" id="barcode-data" placeholder="输入条形码数据" value="1234567890">
            
            <div class="examples">
                <button class="example-btn" onclick="setExample('product')">商品编号</button>
                <button class="example-btn" onclick="setExample('order')">订单号</button>
                <button class="example-btn" onclick="setExample('ean13')">EAN-13示例</button>
            </div>
            
            <div class="control-group">
                <label>
                    <input type="checkbox" id="barcode-showText" checked>
                    <span>显示条形码数字</span>
                </label>
                <label>
                    <input type="checkbox" id="barcode-center" checked>
                    <span>居中打印</span>
                </label>
                <label>
                    <input type="checkbox" id="barcode-cut" checked>
                    <span>打印后切纸</span>
                </label>
            </div>
            
            <div class="control-group">
                <label>
                    条形码宽度：
                    <select id="barcode-width">
                        <option value="2">细</option>
                        <option value="3" selected>正常</option>
                        <option value="4">粗</option>
                        <option value="5">很粗</option>
                        <option value="6">最粗</option>
                    </select>
                </label>
                <label>
                    条形码高度：
                    <input type="number" id="barcode-height" value="80" min="30" max="255">
                </label>
            </div>
        </div>

        <button onclick="printBarcode()">🖨️ 打印条形码</button>

        <div id="status" class="status"></div>
    </div>

    <script>
    // 更新条形码信息
    function updateBarcodeInfo() {
        const type = document.getElementById('barcode-type').value;
        const info = document.getElementById('barcode-info');
        const dataInput = document.getElementById('barcode-data');
        
        switch(type) {
            case 'CODE128':
                info.textContent = 'CODE128：支持所有ASCII字符，包括字母、数字和符号';
                dataInput.placeholder = '输入条形码数据';
                break;
            case 'CODE39':
                info.textContent = 'CODE39：支持大写字母、数字和部分符号（- . $ / + % 空格）';
                dataInput.placeholder = '输入大写字母和数字';
                break;
            case 'EAN13':
                info.textContent = 'EAN-13：必须是13位数字（通常用于商品条码）';
                dataInput.placeholder = '输入13位数字';
                break;
            case 'EAN8':
                info.textContent = 'EAN-8：必须是8位数字（用于小型商品）';
                dataInput.placeholder = '输入8位数字';
                break;
        }
    }

    // 设置示例
    function setExample(type) {
        const dataInput = document.getElementById('barcode-data');
        const typeSelect = document.getElementById('barcode-type');
        
        switch(type) {
            case 'product':
                typeSelect.value = 'CODE128';
                dataInput.value = 'PROD-2024-001';
                break;
            case 'order':
                typeSelect.value = 'CODE128';
                dataInput.value = 'ORD20240115001';
                break;
            case 'ean13':
                typeSelect.value = 'EAN13';
                dataInput.value = '6901234567890';
                break;
        }
        updateBarcodeInfo();
    }

    // 打印条形码
    async function printBarcode() {
        const barcodeType = document.getElementById('barcode-type').value;
        const barcodeData = document.getElementById('barcode-data').value;
        const showText = document.getElementById('barcode-showText').checked;
        const center = document.getElementById('barcode-center').checked;
        const cut = document.getElementById('barcode-cut').checked;
        const width = parseInt(document.getElementById('barcode-width').value);
        const height = parseInt(document.getElementById('barcode-height').value);

        if (!barcodeData.trim()) {
            showStatus('请输入条形码数据', 'error');
            return;
        }

        // 验证条形码数据
        if (barcodeType === 'EAN13' && barcodeData.length !== 13) {
            showStatus('EAN-13条形码必须是13位数字', 'error');
            return;
        }
        if (barcodeType === 'EAN8' && barcodeData.length !== 8) {
            showStatus('EAN-8条形码必须是8位数字', 'error');
            return;
        }
        if (barcodeType === 'CODE39' && !/^[A-Z0-9\-\.\$\/\+\%\s]*$/.test(barcodeData)) {
            showStatus('CODE39只支持大写字母、数字和部分符号', 'error');
            return;
        }

        const printData = {
            barcodeType: barcodeType,
            barcodeData: barcodeData,
            showText: showText,
            center: center,
            cut: cut,
            barcodeWidth: width,
            barcodeHeight: height
        };

        showStatus('正在打印...', 'success');

        try {
            const response = await fetch('http://localhost:9100/api/print', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify(printData)
            });

            const result = await response.json();
            
            if (result.status === 'success') {
                showStatus('✅ 打印成功！', 'success');
            } else {
                showStatus('❌ 打印失败：' + result.message, 'error');
            }
        } catch (error) {
            showStatus('❌ 连接失败：' + error.message, 'error');
        }
    }

    // 显示状态信息
    function showStatus(message, type) {
        const statusDiv = document.getElementById('status');
        statusDiv.textContent = message;
        statusDiv.className = 'status ' + type;
        statusDiv.style.display = 'block';

        if (type === 'success' && message.includes('✅')) {
            setTimeout(() => {
                statusDiv.style.display = 'none';
            }, 3000);
        }
    }

    // 检查服务状态
    async function checkStatus() {
        try {
            const response = await fetch('http://localhost:9100/api/status');
            const data = await response.json();
            console.log('服务状态:', data);
        } catch (error) {
            console.error('服务未启动');
        }
    }

    // 页面加载时检查状态
    checkStatus();
    </script>
</body>
</html>
`
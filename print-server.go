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

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

// PrintRequest 打印请求结构
type PrintRequest struct {
	Content      string `json:"content"`      // 打印内容
	Type         string `json:"type"`         // 打印类型：text 或 barcode
	BarcodeType  string `json:"barcodeType"`  // 条形码类型：CODE128, CODE39, EAN13 等
	BarcodeData  string `json:"barcodeData"`  // 条形码数据
	ShowText     bool   `json:"showText"`     // 是否显示条形码文字
	Cut          bool   `json:"cut"`          // 是否切纸
	Bold         bool   `json:"bold"`         // 是否加粗
	Center       bool   `json:"center"`       // 是否居中
	FontSize     int    `json:"fontSize"`     // 字体大小 (1-8)
	BarcodeWidth int    `json:"barcodeWidth"` // 条形码宽度 (2-6)
	BarcodeHeight int   `json:"barcodeHeight"`// 条形码高度 (1-255)
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
	fmt.Println("    热敏打印机服务已启动")
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

// 将UTF-8转换为GBK
func utf8ToGBK(s string) ([]byte, error) {
	reader := transform.NewReader(strings.NewReader(s), simplifiedchinese.GBK.NewEncoder())
	d, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	return d, nil
}

// 主页处理器
func homeHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	html := `
	<!DOCTYPE html>
	<html>
	<head>
		<title>热敏打印机服务</title>
		<meta charset="UTF-8">
	</head>
	<body style="font-family: Arial; text-align: center; margin-top: 50px;">
		<h1>热敏打印机服务运行中</h1>
		<p>访问 <a href="/test">测试页面</a> 进行打印测试</p>
		<p>支持文本和条形码打印</p>
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
		"version": "2.0.0",
		"port": "LPT1",
		"features": []string{"text", "barcode", "gbk-encoding"},
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
	if req.Type == "" {
		req.Type = "text"
	}
	if req.BarcodeWidth == 0 {
		req.BarcodeWidth = 3
	}
	if req.BarcodeHeight == 0 {
		req.BarcodeHeight = 100
	}

	// 执行打印
	if err := printToLPT(&req); err != nil {
		sendError(w, err.Error())
		return
	}

	// 返回成功
	sendSuccess(w, "打印成功")
}

// 打印到LPT端口
func printToLPT(req *PrintRequest) error {
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

	// 根据类型打印
	if req.Type == "barcode" && req.BarcodeData != "" {
		// 打印条形码
		if err := printBarcode(printer, req); err != nil {
			return err
		}
	} else {
		// 打印文本
		if err := printText(printer, req); err != nil {
			return err
		}
	}

	// 切纸
	if req.Cut {
		// GS V m (切纸)
		printer.Write([]byte("\x1D\x56\x41\x03"))
	}

	return nil
}

// 打印文本
func printText(printer *os.File, req *PrintRequest) error {
	// 设置字体大小
	if req.FontSize > 0 && req.FontSize <= 8 {
		// ESC ! n 设置打印模式
		fontSize := byte(req.FontSize - 1)
		printer.Write([]byte{0x1B, 0x21, fontSize})
	}

	// 设置居中打印
	if req.Center {
		// ESC a 1 (居中)
		printer.Write([]byte("\x1B\x61\x01"))
	}

	// 设置加粗
	if req.Bold {
		// ESC E 1 (加粗开启)
		printer.Write([]byte("\x1B\x45\x01"))
	}

	// 转换编码并写入内容
	content := strings.ReplaceAll(req.Content, "\r\n", "\n")
	gbkContent, err := utf8ToGBK(content)
	if err != nil {
		// 如果转换失败，尝试直接打印
		printer.Write([]byte(content))
	} else {
		printer.Write(gbkContent)
	}
	
	// 添加换行
	printer.Write([]byte("\n\n"))

	// 取消加粗
	if req.Bold {
		// ESC E 0 (加粗关闭)
		printer.Write([]byte("\x1B\x45\x00"))
	}

	// 取消居中
	if req.Center {
		// ESC a 0 (左对齐)
		printer.Write([]byte("\x1B\x61\x00"))
	}

	return nil
}

// 打印条形码
func printBarcode(printer *os.File, req *PrintRequest) error {
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
	} else {
		printer.Write([]byte{0x1D, 0x48, 0x00}) // 不打印
	}

	// 设置居中
	if req.Center {
		printer.Write([]byte("\x1B\x61\x01"))
	}

	// 选择条形码类型并打印
	switch strings.ToUpper(req.BarcodeType) {
	case "CODE39":
		// GS k 4 n d1...dn
		data := []byte(req.BarcodeData)
		printer.Write([]byte{0x1D, 0x6B, 0x04, byte(len(data))})
		printer.Write(data)
	case "EAN13":
		// GS k 2 d1...d13
		if len(req.BarcodeData) == 13 {
			printer.Write([]byte{0x1D, 0x6B, 0x02})
			printer.Write([]byte(req.BarcodeData))
		}
	case "EAN8":
		// GS k 3 d1...d8
		if len(req.BarcodeData) == 8 {
			printer.Write([]byte{0x1D, 0x6B, 0x03})
			printer.Write([]byte(req.BarcodeData))
		}
	case "CODE128":
		fallthrough
	default:
		// GS k 73 n d1...dn (CODE128)
		data := []byte(req.BarcodeData)
		printer.Write([]byte{0x1D, 0x6B, 0x49, byte(len(data))})
		printer.Write(data)
	}

	// 换行
	printer.Write([]byte("\n\n"))

	// 取消居中
	if req.Center {
		printer.Write([]byte("\x1B\x61\x00"))
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
    <title>热敏打印机测试 - 支持条形码</title>
    <style>
        body {
            font-family: Arial, sans-serif;
            max-width: 800px;
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
        .tabs {
            display: flex;
            margin-bottom: 20px;
            border-bottom: 2px solid #ddd;
        }
        .tab {
            padding: 10px 20px;
            cursor: pointer;
            background: #f0f0f0;
            border: 1px solid #ddd;
            border-bottom: none;
            margin-right: 5px;
            border-radius: 5px 5px 0 0;
        }
        .tab.active {
            background: white;
            border-bottom: 2px solid white;
            margin-bottom: -2px;
        }
        .tab-content {
            display: none;
        }
        .tab-content.active {
            display: block;
        }
        textarea {
            width: 100%;
            height: 300px;
            padding: 10px;
            border: 1px solid #ddd;
            border-radius: 5px;
            font-family: monospace;
            font-size: 14px;
            box-sizing: border-box;
        }
        input[type="text"] {
            width: 100%;
            padding: 10px;
            border: 1px solid #ddd;
            border-radius: 5px;
            font-size: 14px;
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
            margin: 10px 0;
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
        }
        input[type="checkbox"] {
            width: 18px;
            height: 18px;
            cursor: pointer;
        }
        select {
            padding: 5px 10px;
            border: 1px solid #ddd;
            border-radius: 3px;
            font-size: 14px;
        }
        button {
            background: #007bff;
            color: white;
            padding: 12px 30px;
            border: none;
            border-radius: 5px;
            font-size: 16px;
            cursor: pointer;
            transition: background 0.3s;
        }
        button:hover {
            background: #0056b3;
        }
        button:active {
            transform: translateY(1px);
        }
        .button-group {
            text-align: center;
            margin-top: 20px;
        }
        .status {
            margin-top: 20px;
            padding: 10px;
            border-radius: 5px;
            text-align: center;
            display: none;
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
        .barcode-controls {
            background: #e9ecef;
            padding: 15px;
            border-radius: 5px;
            margin: 10px 0;
        }
        .info {
            background: #d1ecf1;
            color: #0c5460;
            padding: 10px;
            border-radius: 5px;
            margin: 10px 0;
            font-size: 14px;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>🖨️ 热敏打印机测试 - 支持条形码</h1>
        
        <div class="tabs">
            <div class="tab active" onclick="switchTab('text')">文本打印</div>
            <div class="tab" onclick="switchTab('barcode')">条形码打印</div>
        </div>

        <!-- 文本打印标签页 -->
        <div id="text-tab" class="tab-content active">
            <h3>打印内容：</h3>
            <textarea id="text-content" placeholder="请输入要打印的内容...">
================================
        跳星马小店
================================
订单号：2024001234
时间：2024-01-15 14:30:25
--------------------------------
商品名称        数量    单价    小计
--------------------------------
可口可乐        2      ¥6.00   ¥12.00
乐事薯片        1      ¥8.00   ¥8.00
德芙巧克力      1      ¥12.00  ¥12.00
--------------------------------
合计：                        ¥32.00
实付：                        ¥32.00
--------------------------------
感谢您的光临，欢迎下次再来！
================================</textarea>

            <div class="controls">
                <h3>打印选项：</h3>
                <div class="control-group">
                    <label>
                        <input type="checkbox" id="text-cut" checked>
                        <span>自动切纸</span>
                    </label>
                    <label>
                        <input type="checkbox" id="text-bold">
                        <span>加粗打印</span>
                    </label>
                    <label>
                        <input type="checkbox" id="text-center">
                        <span>居中打印</span>
                    </label>
                </div>
                <div class="control-group">
                    <label>
                        字体大小：
                        <select id="text-fontSize">
                            <option value="1">最小</option>
                            <option value="2">较小</option>
                            <option value="3" selected>正常</option>
                            <option value="4">较大</option>
                            <option value="5">最大</option>
                        </select>
                    </label>
                </div>
            </div>

            <div class="button-group">
                <button onclick="printText()">🖨️ 打印文本</button>
            </div>
        </div>

        <!-- 条形码打印标签页 -->
        <div id="barcode-tab" class="tab-content">
            <h3>条形码设置：</h3>
            
            <div class="barcode-controls">
                <label>条形码类型：</label>
                <select id="barcode-type" onchange="updateBarcodeInfo()">
                    <option value="CODE128">CODE128（推荐）</option>
                    <option value="CODE39">CODE39</option>
                    <option value="EAN13">EAN-13（13位）</option>
                    <option value="EAN8">EAN-8（8位）</option>
                </select>
                
                <div class="info" id="barcode-info">
                    CODE128：支持所有ASCII字符，包括字母、数字和符号
                </div>

                <label>条形码数据：</label>
                <input type="text" id="barcode-data" placeholder="输入条形码数据" value="1234567890">
                
                <div class="control-group">
                    <label>
                        <input type="checkbox" id="barcode-showText" checked>
                        <span>显示条形码文字</span>
                    </label>
                    <label>
                        <input type="checkbox" id="barcode-center" checked>
                        <span>居中打印</span>
                    </label>
                    <label>
                        <input type="checkbox" id="barcode-cut" checked>
                        <span>自动切纸</span>
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
                        <input type="number" id="barcode-height" value="100" min="1" max="255" style="width: 80px;">
                    </label>
                </div>
            </div>

            <div class="button-group">
                <button onclick="printBarcode()">🖨️ 打印条形码</button>
            </div>
        </div>

        <div id="status" class="status"></div>
    </div>

    <script>
    // 切换标签页
    function switchTab(tab) {
        // 切换标签
        document.querySelectorAll('.tab').forEach(t => t.classList.remove('active'));
        document.querySelectorAll('.tab-content').forEach(c => c.classList.remove('active'));
        
        if (tab === 'text') {
            document.querySelector('.tab:nth-child(1)').classList.add('active');
            document.getElementById('text-tab').classList.add('active');
        } else {
            document.querySelector('.tab:nth-child(2)').classList.add('active');
            document.getElementById('barcode-tab').classList.add('active');
        }
    }

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
                info.textContent = 'EAN-13：必须是13位数字';
                dataInput.placeholder = '输入13位数字';
                dataInput.value = '6901234567890';
                break;
            case 'EAN8':
                info.textContent = 'EAN-8：必须是8位数字';
                dataInput.placeholder = '输入8位数字';
                dataInput.value = '12345678';
                break;
        }
    }

    // 打印文本
    async function printText() {
        const content = document.getElementById('text-content').value;
        const cut = document.getElementById('text-cut').checked;
        const bold = document.getElementById('text-bold').checked;
        const center = document.getElementById('text-center').checked;
        const fontSize = parseInt(document.getElementById('text-fontSize').value);

        if (!content.trim()) {
            showStatus('请输入打印内容', 'error');
            return;
        }

        const printData = {
            type: 'text',
            content: content,
            cut: cut,
            bold: bold,
            center: center,
            fontSize: fontSize
        };

        await sendPrintRequest(printData);
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

        const printData = {
            type: 'barcode',
            barcodeType: barcodeType,
            barcodeData: barcodeData,
            showText: showText,
            center: center,
            cut: cut,
            barcodeWidth: width,
            barcodeHeight: height
        };

        await sendPrintRequest(printData);
    }

    // 发送打印请求
    async function sendPrintRequest(data) {
        try {
            const response = await fetch('http://localhost:9100/api/print', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify(data)
            });

            const result = await response.json();
            
            if (result.status === 'success') {
                showStatus('打印成功！', 'success');
            } else {
                showStatus('打印失败：' + result.message, 'error');
            }
        } catch (error) {
            showStatus('连接失败：' + error.message, 'error');
        }
    }

    // 显示状态信息
    function showStatus(message, type) {
        const statusDiv = document.getElementById('status');
        statusDiv.textContent = message;
        statusDiv.className = 'status ' + type;
        statusDiv.style.display = 'block';

        setTimeout(() => {
            statusDiv.style.display = 'none';
        }, 3000);
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
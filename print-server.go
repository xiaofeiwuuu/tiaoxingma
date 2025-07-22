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
	Content  string `json:"content"`  // 打印内容
	Cut      bool   `json:"cut"`      // 是否切纸
	Bold     bool   `json:"bold"`     // 是否加粗
	Center   bool   `json:"center"`   // 是否居中
	FontSize int    `json:"fontSize"` // 字体大小 (1-8)
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
		"version": "1.0.0",
		"port": "LPT1",
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

	// 写入打印内容
	// 处理中文编码（转换为GBK）
	content := strings.ReplaceAll(req.Content, "\r\n", "\n")
	printer.Write([]byte(content))
	
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

	// 切纸
	if req.Cut {
		// GS V m (切纸)
		printer.Write([]byte("\x1D\x56\x41\x03"))
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
    <title>热敏打印机测试</title>
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
        .examples {
            margin-top: 20px;
        }
        .example-btn {
            background: #6c757d;
            font-size: 14px;
            padding: 8px 15px;
            margin: 5px;
        }
        .example-btn:hover {
            background: #5a6268;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>🖨️ 热敏打印机测试</h1>
        
        <div>
            <h3>打印内容：</h3>
            <textarea id="content" placeholder="请输入要打印的内容...">
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
        </div>

        <div class="controls">
            <h3>打印选项：</h3>
            <div class="control-group">
                <label>
                    <input type="checkbox" id="cut" checked>
                    <span>自动切纸</span>
                </label>
                <label>
                    <input type="checkbox" id="bold">
                    <span>加粗打印</span>
                </label>
                <label>
                    <input type="checkbox" id="center">
                    <span>居中打印</span>
                </label>
            </div>
            <div class="control-group">
                <label>
                    字体大小：
                    <select id="fontSize">
                        <option value="1">最小</option>
                        <option value="2">较小</option>
                        <option value="3" selected>正常</option>
                        <option value="4">较大</option>
                        <option value="5">最大</option>
                    </select>
                </label>
            </div>
        </div>

        <div class="examples">
            <h3>示例模板：</h3>
            <button class="example-btn" onclick="loadExample('receipt')">收银小票</button>
            <button class="example-btn" onclick="loadExample('order')">订单凭证</button>
            <button class="example-btn" onclick="loadExample('test')">测试打印</button>
        </div>

        <div class="button-group">
            <button onclick="testPrint()">🖨️ 打印测试</button>
        </div>

        <div id="status" class="status"></div>
    </div>

    <script>
    // 示例模板
    const examples = {
        receipt: '================================\\n        跳星马小店\\n================================\\n订单号：2024001234\\n时间：2024-01-15 14:30:25\\n--------------------------------\\n商品名称        数量    单价    小计\\n--------------------------------\\n可口可乐        2      ¥6.00   ¥12.00\\n乐事薯片        1      ¥8.00   ¥8.00\\n德芙巧克力      1      ¥12.00  ¥12.00\\n--------------------------------\\n合计：                        ¥32.00\\n实付：                        ¥32.00\\n--------------------------------\\n感谢您的光临，欢迎下次再来！\\n================================',
        order: '订单号：ORD-2024-001234\\n================================\\n客户信息\\n姓名：张三\\n电话：138****5678\\n地址：北京市朝阳区xxx街道xxx号\\n\\n订单明细\\n--------------------------------\\n1. 商品A x 2\\n2. 商品B x 1\\n3. 商品C x 3\\n\\n订单金额：¥128.00\\n配送费：¥5.00\\n总计：¥133.00\\n\\n订单状态：已支付\\n================================',
        test: '打印机测试页\\n================================\\n测试内容：\\n1. 这是第一行测试文字\\n2. 这是第二行测试文字\\n3. 1234567890\\n4. ABCDEFGHIJKLMNOPQRSTUVWXYZ\\n5. abcdefghijklmnopqrstuvwxyz\\n6. !@#$%^&*()_+-=[]{}|;:,.<>?\\n================================\\n测试完成！'
    };

    // 加载示例
    function loadExample(type) {
        document.getElementById('content').value = examples[type];
    }

    // 打印函数
    async function testPrint() {
        const content = document.getElementById('content').value;
        const cut = document.getElementById('cut').checked;
        const bold = document.getElementById('bold').checked;
        const center = document.getElementById('center').checked;
        const fontSize = parseInt(document.getElementById('fontSize').value);

        if (!content.trim()) {
            showStatus('请输入打印内容', 'error');
            return;
        }

        const printData = {
            content: content,
            cut: cut,
            bold: bold,
            center: center,
            fontSize: fontSize
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
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"image/png"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/skip2/go-qrcode"
)

var (
	debugMode = flag.Bool("debug", false, "Enable debug mode")
	forwarder *Forwarder
	storage   *Storage
	rules     []Rule
	templates []Template
)

func init() {
	// 初始化日志
	initLogger()

	// 创建必要的目录
	createDirs()
}

func initLogger() {
	// 设置日志文件路径为db目录下的log.txt
	logFile, err := os.OpenFile(filepath.Join(".", "db", "log.txt"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("Failed to open log file: %v", err)
		return
	}

	// 设置日志输出
	log.SetOutput(logFile)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
}

func createDirs() {
	// 创建 db 目录
	dbDir := filepath.Join(".", "db")
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		log.Printf("Failed to create db directory: %v", err)
	}
}

func main() {
	flag.Parse()

	log.Println("Starting port forwarder...")

	// 初始化 forwarder 和 storage
	forwarder = NewForwarder()
	storage = NewStorage()

	// 检查 WebView2 运行时
	if err := checkWebView2(); err != nil {
		log.Printf("WebView2 check failed: %v", err)
		fmt.Println("Error: WebView2 runtime not found. Please install WebView2 runtime.")
		os.Exit(1)
	}

	// 加载配置
	loadConfig()

	// 初始化 GUI
	initGUI()
}

func checkWebView2() error {
	// WebView2 检查逻辑
	// 在 Windows 上，WebView2 是必需的
	if runtime.GOOS == "windows" {
		// 简单检查，实际项目中可能需要更复杂的检查
		// 这里暂时返回 nil，假设 WebView2 已安装
		return nil
	}
	return nil
}

func loadConfig() {
	// 加载配置逻辑
	log.Println("Loading configuration...")

	// 加载规则
	var err error
	rules, err = storage.LoadRules()
	if err != nil {
		log.Printf("Failed to load rules: %v", err)
		rules = []Rule{}
	}

	// 加载模板
	templates, err = storage.LoadTemplates()
	if err != nil {
		log.Printf("Failed to load templates: %v", err)
		templates = []Template{}
	}
}

func initGUI() {
	// 注册HTTP处理函数
	http.HandleFunc("/", serveHTML)
	http.HandleFunc("/api/getLocalIPs", apiGetLocalIPs)
	http.HandleFunc("/api/getRules", apiGetRules)
	http.HandleFunc("/api/getTemplates", apiGetTemplates)
	http.HandleFunc("/api/addRule", apiAddRule)
	http.HandleFunc("/api/deleteRules", apiDeleteRules)
	http.HandleFunc("/api/updateRule", apiUpdateRule)
	http.HandleFunc("/api/saveAsTemplate", apiSaveAsTemplate)
	http.HandleFunc("/api/applyTemplate", apiApplyTemplate)
	http.HandleFunc("/api/startTCPForward", apiStartTCPForward)
	http.HandleFunc("/api/stopTCPForward", apiStopTCPForward)
	http.HandleFunc("/api/startUDPForward", apiStartUDPForward)
	http.HandleFunc("/api/stopUDPForward", apiStopUDPForward)
	http.HandleFunc("/api/isTCPRunning", apiIsTCPRunning)
	http.HandleFunc("/api/isUDPRunning", apiIsUDPRunning)
	http.HandleFunc("/api/startTemplateForward", apiStartTemplateForward)
	http.HandleFunc("/api/stopTemplateForward", apiStopTemplateForward)
	http.HandleFunc("/api/getQRCode", apiGetQRCode)
	http.HandleFunc("/api/deleteTemplate", apiDeleteTemplate)
	http.HandleFunc("/api/updateTemplate", apiUpdateTemplate)
	http.HandleFunc("/api/getLog", apiGetLog)

	// 启动HTTP服务器
	port := 8080
	for {
		log.Printf("Starting HTTP server on port %d...", port)
		log.Printf("Please open http://localhost:%d in your browser", port)

		// 在终端中显示端口信息
		fmt.Printf("Starting HTTP server on port %d...\n", port)
		fmt.Printf("Please open http://localhost:%d in your browser\n", port)

		if err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil); err != nil {
			log.Printf("Failed to start HTTP server on port %d: %v", port, err)
			fmt.Printf("Failed to start HTTP server on port %d: %v\n", port, err)
			// 端口被占用，尝试下一个端口
			port++
			continue
		}
		break
	}
}

func getHTMLContent() string {
	return `
<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Port Forwarder</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }

        body {
            font-family: Arial, sans-serif;
            background-color: #f5f5f5;
            color: #333;
        }

        .container {
            max-width: 1200px;
            margin: 20px auto;
            padding: 20px;
            background-color: white;
            border-radius: 8px;
            box-shadow: 0 0 10px rgba(0, 0, 0, 0.1);
        }

        h1 {
            text-align: center;
            margin-bottom: 20px;
            color: #2c3e50;
        }

        .header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 20px;
            padding-bottom: 10px;
            border-bottom: 1px solid #e0e0e0;
            flex-wrap: wrap;
            gap: 10px;
        }

        .header > div {
            display: flex;
            gap: 10px;
            flex-wrap: wrap;
            align-items: center;
        }

        .btn {
            padding: 8px 16px;
            border: none;
            border-radius: 4px;
            cursor: pointer;
            font-size: 14px;
            transition: background-color 0.3s;
        }

        .btn-primary {
            background-color: #3498db;
            color: white;
        }

        .btn-primary:hover {
            background-color: #2980b9;
        }

        .btn-danger {
            background-color: #e74c3c;
            color: white;
        }

        .btn-danger:hover {
            background-color: #c0392b;
        }

        .btn-success {
            background-color: #27ae60;
            color: white;
        }

        .btn-success:hover {
            background-color: #219a52;
        }

        .btn-warning {
            background-color: #f39c12;
            color: white;
        }

        .btn-warning:hover {
            background-color: #e67e22;
        }

        .rules-list {
            margin-bottom: 20px;
        }

        .rule-item {
            display: flex;
            align-items: center;
            padding: 15px;
            margin-bottom: 10px;
            background-color: #f9f9f9;
            border-radius: 4px;
            border: 1px solid #e0e0e0;
        }

        .rule-item:hover {
            background-color: #f0f0f0;
        }

        .rule-checkbox {
            margin-right: 15px;
        }

        .rule-config {
            flex: 1;
            display: grid;
            grid-template-columns: 200px 100px 200px 100px;
            gap: 10px;
        }

        .rule-config select,
        .rule-config input {
            padding: 6px 10px;
            border: 1px solid #ddd;
            border-radius: 4px;
            font-size: 14px;
        }

        .rule-actions {
            margin-left: 15px;
            display: flex;
            gap: 10px;
        }

        .rules-header {
            padding: 10px 15px;
            margin-bottom: 10px;
            background-color: #f0f0f0;
            border-radius: 4px;
            border: 1px solid #e0e0e0;
        }

        .rules-header .rule-config {
            font-weight: bold;
        }

        .rule-seq {
            width: 50px;
            text-align: center;
            font-weight: bold;
            margin-right: 10px;
        }

        .template-section {
            margin-top: 0px;
            padding-top: 0px;
            border-bottom: 0px solid #e0e0e0;
        }

        .template-header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 15px;
        }

        .template-actions {
            display: flex;
            gap: 10px;
            flex-wrap: wrap;
            align-items: center;
        }

        @media (max-width: 800px) {
            .template-actions {
                flex-direction: column;
            }
            .template-actions .btn {
                width: 100%;
            }
        }

        .template-select {
            padding: 6px 10px;
            border: 1px solid #ddd;
            border-radius: 4px;
            font-size: 14px;
        }

        .template-list {
            margin-top: 20px;
            padding: 15px;
            background-color: #f9f9f9;
            border-radius: 8px;
            border: 1px solid #e0e0e0;
        }

        .template-item {
            display: flex;
            justify-content: space-between;
            align-items: center;
            padding: 10px;
            margin-bottom: 8px;
            background-color: white;
            border-radius: 4px;
            border: 1px solid #e0e0e0;
        }

        .template-item:hover {
            background-color: #f0f0f0;
        }

        .template-info {
            flex: 1;
        }

        .template-name {
            font-weight: bold;
            margin-bottom: 5px;
        }

        .template-rules-count {
            color: #666;
            font-size: 14px;
        }

        .template-actions {
            display: flex;
            gap: 10px;
        }

        .status-message {
            position: fixed;
            top: 20px;
            right: 20px;
            padding: 12px 20px;
            border-radius: 4px;
            font-size: 14px;
            box-shadow: 0 2px 10px rgba(0, 0, 0, 0.2);
            z-index: 10000;
            animation: slideInRight 0.3s ease-out;
        }

        @keyframes slideInRight {
            from {
                transform: translateX(100%);
                opacity: 0;
            }
            to {
                transform: translateX(0);
                opacity: 1;
            }
        }

        .status-success {
            background-color: #d4edda;
            color: #155724;
            border: 1px solid #c3e6cb;
        }

        .status-error {
            background-color: #f8d7da;
            color: #721c24;
            border: 1px solid #f5c6cb;
        }

        .status-info {
            background-color: #d1ecf1;
            color: #0c5460;
            border: 1px solid #bee5eb;
        }

        .log-section {
            margin-top: 30px;
            padding-top: 20px;
            border-top: 1px solid #e0e0e0;
        }

        .log-section h3 {
            margin-bottom: 10px;
            color: #2c3e50;
        }

        .log-content {
            background-color: #f9f9f9;
            border: 1px solid #e0e0e0;
            border-radius: 4px;
            padding: 15px;
            height: 200px;
            overflow-y: auto;
            font-family: monospace;
            font-size: 12px;
            line-height: 1.4;
            white-space: pre-wrap;
        }

        .log-content p {
            margin: 0 0 5px 0;
            color: #333;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>端口转发工具</h1>

        <div class="header">
            <div>
                <button class="btn btn-primary" onclick="loadRules()">首页</button>
                <button class="btn btn-primary" onclick="addRule()">新增规则</button>
                <button class="btn btn-danger" onclick="deleteSelectedRules()">删除选中规则</button>
                <button class="btn btn-success" onclick="saveAsTemplate()">保存为模板</button>
                <button class="btn btn-warning" onclick="addToExistingTemplate()">加入已有模板</button>
                <button class="btn btn-warning" onclick="createNewTemplate()">新建模板</button>
            </div>
        </div>

        <div class="template-section">
            <div class="template-header">
          
                    <select class="template-select" id="templateSelect">
                        <option value="">选择模板</option>
                    </select>
                    <button class="btn btn-primary" onclick="applyTemplate()">切换到模板</button>
                    <button class="btn btn-success" onclick="startTemplateForward()">一键开启此模板所有转发</button>
                    <button class="btn btn-danger" onclick="stopTemplateForward()">一键关闭此模板所有转发</button>
                    <button class="btn btn-danger" onclick="deleteTemplate()">删除此模板</button>
                    <button class="btn btn-info" onclick="editTemplate()">编辑模板</button>
           
            </div>
           
        </div>
   

        <div class="rules-header">
            <div style="display: flex; align-items: center;">
                <div class="rule-seq"><strong>序号</strong></div>
                <div class="rule-config">
                    <div><strong>监听IP</strong></div>
                    <div><strong>监听端口</strong></div>
                    <div><strong>目标IP</strong></div>
                    <div><strong>目标端口</strong></div>
                </div>
            </div>
        </div>


        <div class="rules-list" id="rulesList">
            <!-- 规则列表将通过 JavaScript 动态生成 -->
        </div>
      

        

        <div class="status-message" id="statusMessage" style="display: none;"></div>

        <div class="log-section">
            <h3>运行日志</h3>
            <div class="log-content" id="logContent">
                <p>加载日志中...</p>
            </div>
        </div>
    </div>

    <script>
        // 初始化数据
        let rules = [];
        let templates = [];

        // 页面加载完成后初始化
        // window.onload = function() {
        //     initApp();
        // };

        // 初始化应用
        function initApp() {
            // 获取本地网卡IP地址
            getLocalIPs();
            
            // 加载规则
            loadRules();
            
            // 加载模板
            loadTemplates();
        }

        // 获取本地网卡IP地址
        function getLocalIPs() {
            fetch('/api/getLocalIPs')
                .then(response => response.json())
                .then(ips => {
                    console.log('Local IPs:', ips);
                    // 存储IP地址供后续使用
                    window.localIPs = ips;
                })
                .catch(error => {
                    console.error('Failed to get local IPs:', error);
                });
        }

        // 加载规则
        function loadRules() {
            fetch('/api/getRules')
                .then(response => response.json())
        .then(data => {
                    // 倒序显示规则列表
                    rules = data.slice().reverse();
                    renderRules();
                })
                .catch(error => {
                    console.error('Failed to load rules:', error);
                });
        }

        // 加载模板
        function loadTemplates() {
            fetch('/api/getTemplates')
                .then(response => response.json())
                .then(data => {
                    templates = data;
                    renderTemplates();
                })
                .catch(error => {
                    console.error('Failed to load templates:', error);
                });
        }

       // 渲染规则列表（倒序）
function renderRules(){
    const list = document.getElementById('rulesList');
    list.innerHTML = '';

    if(rules.length === 0){
        list.innerHTML = '<p style="text-align:center;color:#999;padding:20px">暂无规则，请点击“新增规则”按钮添加</p>';
        return;
    }

    /* 倒序遍历，同步插壳保证顺序 */
    for(let i = rules.length - 1; i >= 0; i--){
        const r = rules[i];

        const item = document.createElement('div');
        item.className = 'rule-item';
        item.dataset.id = r.id;

        /* 用字符串拼接代替 ${}，避开 Go 模板冲突 */
        item.innerHTML =
            '<input type="checkbox" class="rule-checkbox" data-id="'+ r.id +'">'+
            '<div style="display:flex;align-items:center">'+
              '<div class="rule-seq">'+ r.seq +'</div>'+
              '<div class="rule-config">'+
                '<select class="listen-addr" data-id="'+ r.id +'">'+ renderIPOptions(r.listenAddr) +'</select>'+
                '<input type="number" class="listen-port" data-id="'+ r.id +'" value="'+ r.listenPort +'" min="1" max="65535">'+
                '<select class="target-addr" data-id="'+ r.id +'">'+ renderTargetIPOptions(r.targetAddr) +'</select>'+
                '<input type="number" class="target-port" data-id="'+ r.id +'" value="'+ r.targetPort +'" min="1" max="65535">'+
              '</div>'+
            '</div>'+
            '<div class="rule-actions">'+
              '<button class="btn btn-default" data-role="tcpBtn">检测中…</button>'+
              '<button class="btn btn-default" data-role="udpBtn">检测中…</button>'+
              '<button class="btn btn-danger"  onclick="deleteRule(\''+ r.id +'\')">删除</button>'+
              '<button class="btn btn-primary" onclick="copyRule('+ i +')">复制</button>'+
              '<button class="btn btn-warning" onclick="showQRCode(\''+ r.listenAddr +'\','+ r.listenPort +')">二维码</button>'+
            '</div>';

        list.appendChild(item);          // 顺序固定
        addRuleEventListeners(item, r.id); // 你原来的绑定函数

        /* 异步只改按钮 */
        Promise.all([
            fetch('/api/isTCPRunning?listenAddr='+ r.listenAddr +'&listenPort='+ r.listenPort).then(res=>res.json()),
            fetch('/api/isUDPRunning?listenAddr='+ r.listenAddr +'&listenPort='+ r.listenPort).then(res=>res.json())
        ]).then(function(res){
            const tcpBtn = item.querySelector('[data-role=tcpBtn]');
            const udpBtn = item.querySelector('[data-role=udpBtn]');

            tcpBtn.className   = res[0].running ? 'btn btn-danger' : 'btn btn-success';
            tcpBtn.textContent = res[0].running ? '停止TCP转发' : '开启TCP转发';
            tcpBtn.onclick     = function(){ toggleTCPForward(i); };

            udpBtn.className   = res[1].running ? 'btn btn-danger' : 'btn btn-success';
            udpBtn.textContent = res[1].running ? '停止UDP转发' : '开启UDP转发';
            udpBtn.onclick     = function(){ toggleUDPForward(i); };
        });
    }
}
        // 渲染IP选项
        function renderIPOptions(selectedAddr) {
            let options = '<option value="">选择监听地址</option>';
            if (window.localIPs) {
                window.localIPs.forEach(function(ipInfo) {
                    const selected = ipInfo.ip === selectedAddr ? 'selected' : '';
                    options += '<option value="' + ipInfo.ip + '" ' + selected + '>' + ipInfo.ip + ' (' + ipInfo.name + ')</option>';
                });
            } else {
                options += '<option value="">正在加载网卡信息...</option>';
                // 尝试获取本地网卡IP地址
                if (!window.isGettingIPs) {
                    window.isGettingIPs = true;
                    fetch('/api/getLocalIPs')
                        .then(response => response.json())
                        .then(ips => {
                            console.log('Local IPs:', ips);
                            // 存储IP地址供后续使用
                            window.localIPs = ips;
                            // 重新渲染规则
                            loadRules();
                        })
                        .catch(error => {
                            console.error('Failed to get local IPs:', error);
                        })
                        .finally(() => {
                            window.isGettingIPs = false;
                        });
                }
            }
            // 检查是否是自定义IP
            const isCustom = selectedAddr && (!window.localIPs || !window.localIPs.some(function(ipInfo) { return ipInfo.ip === selectedAddr; }));
            if (isCustom) {
                options += '<option value="' + selectedAddr + '" selected>' + selectedAddr + '</option>';
            }
            return options;
        }

        // 添加规则事件监听器
        function addRuleEventListeners(ruleItem, ruleId) {
            // 监听地址变化
            const listenAddrSelect = ruleItem.querySelector('.listen-addr[data-id="' + ruleId + '"]');
            if (listenAddrSelect) {
                listenAddrSelect.addEventListener('change', function() {
                    updateRule(ruleId);
                });
            }

            // 监听端口变化
            const listenPortInput = ruleItem.querySelector('.listen-port[data-id="' + ruleId + '"]');
            if (listenPortInput) {
                listenPortInput.addEventListener('change', function() {
                    updateRule(ruleId);
                });
            }

            // 目标地址变化
            const targetAddrSelect = ruleItem.querySelector('.target-addr[data-id="' + ruleId + '"]');
            if (targetAddrSelect) {
                targetAddrSelect.addEventListener('change', function() {
                    if (this.value === 'custom') {
                        // 创建自定义输入框
                        const customInput = document.createElement('input');
                        customInput.type = 'text';
                        customInput.className = 'target-addr-custom';
                        customInput.placeholder = '请输入自定义IP地址';
                        customInput.style.padding = '6px 10px';
                        customInput.style.border = '1px solid #ddd';
                        customInput.style.borderRadius = '4px';
                        customInput.style.fontSize = '14px';

                        // 替换选择框为输入框
                        const parent = this.parentElement;
                        parent.replaceChild(customInput, this);

                        // 聚焦到输入框
                        customInput.focus();

                        // 监听输入框变化
                        customInput.addEventListener('change', function() {
                            if (this.value) {
                                // 更新规则
                                const ruleItem = this.closest('.rule-item');
                                const listenAddr = ruleItem.querySelector('.listen-addr').value;
                                const listenPort = ruleItem.querySelector('.listen-port').value;
                                const targetPort = ruleItem.querySelector('.target-port').value;

                                fetch('/api/updateRule', {
                                    method: 'POST',
                                    headers: {
                                        'Content-Type': 'application/json'
                                    },
                                    body: JSON.stringify({
                                        id: ruleId,
                                        listenAddr: listenAddr,
                                        listenPort: listenPort,
                                        targetAddr: this.value,
                                        targetPort: targetPort
                                    })
                                })
                                .then(response => response.json())
                                .then(data => {
                                    if (data.success) {
                                        loadRules();
                                    }
                                });
                            } else {
                                // 如果输入框为空，恢复选择框
                                parent.replaceChild(targetAddrSelect, this);
                            }
                        });
                    } else {
                        updateRule(ruleId);
                    }
                });
            }

            // 目标端口变化
            const targetPortInput = ruleItem.querySelector('.target-port[data-id="' + ruleId + '"]');
            if (targetPortInput) {
                targetPortInput.addEventListener('change', function() {
                    updateRule(ruleId);
                });
            }
        }

        // 更新规则
        function updateRule(ruleId) {
            const ruleItem = document.querySelector('.rule-item[data-id="' + ruleId + '"]');
            if (!ruleItem) {
                console.error('Rule item not found for id:', ruleId);
                return;
            }
            const listenAddr = ruleItem.querySelector('.listen-addr').value;
            const listenPort = ruleItem.querySelector('.listen-port').value;
            const targetAddr = ruleItem.querySelector('.target-addr') ? ruleItem.querySelector('.target-addr').value : ruleItem.querySelector('.target-addr-custom').value;
            const targetPort = ruleItem.querySelector('.target-port').value;

            fetch('/api/updateRule', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({
                    id: ruleId,
                    listenAddr: listenAddr,
                    listenPort: listenPort,
                    targetAddr: targetAddr,
                    targetPort: targetPort
                })
            })
            .then(response => response.json())
            .then(data => {
                if (data.success) {
                    loadRules();
                }
            })
            .catch(error => {
                console.error('Failed to update rule:', error);
            });
        }

        // 渲染模板列表
        function renderTemplates() {
            const templateSelect = document.getElementById('templateSelect');
            templateSelect.innerHTML = '';

            // 添加默认选择项
            const defaultOption = document.createElement('option');
            defaultOption.value = '';
            defaultOption.textContent = '选择模板';
            templateSelect.appendChild(defaultOption);

            // 添加模板选项
            templates.forEach(template => {
                const option = document.createElement('option');
                option.value = template.name;
                option.textContent = template.name;
                templateSelect.appendChild(option);
            });

            // 渲染模板列表
            renderTemplatesList();
        }

        // 渲染模板列表
        function renderTemplatesList() {
            const templateList = document.getElementById('templateList');
            templateList.innerHTML = '';

            if (templates.length === 0) {
                templateList.innerHTML = '<p style="text-align: center; color: #999; padding: 20px;">暂无模板，请点击"保存为模板"按钮创建</p>';
                return;
            }

            templates.forEach(template => {
                const templateItem = document.createElement('div');
                templateItem.className = 'template-item';
                templateItem.innerHTML = '<div class="template-info">' +
                    '<div class="template-name">' + template.name + '</div>' +
                    '<div class="template-rules-count">规则数量: ' + template.rules.length + '</div>' +
                    '<div class="template-sign" style="font-size:12px; color:#666; margin-top:4px;">创建时间: ' + (template.CreatedAt || '') + '</div>' +
                    '</div>' +
                    '<div class="template-actions">' +
                    '<button class="btn btn-primary" onclick="applyTemplateByName(\'' + template.name + '\')">切到模板</button>' +
                    '<button class="btn btn-success" onclick="startTemplateForwardByName(\'' + template.name + '\')">开启转发</button>' +
                    '<button class="btn btn-danger" onclick="stopTemplateForwardByName(\'' + template.name + '\')">关闭转发</button>' +
                    '<button class="btn btn-info" onclick="editTemplateByName(\'' + template.name + '\')">编辑</button>' +
                    '<button class="btn btn-danger" onclick="deleteTemplateByName(\'' + template.name + '\')">删除</button>' +
                    '</div>';
                templateList.appendChild(templateItem);
            });
        }

        // 按名称应用模板
        function applyTemplateByName(templateName) {
            const templateSelect = document.getElementById('templateSelect');
            templateSelect.value = templateName;
            applyTemplate();
        }

        // 按名称开启模板转发
        function startTemplateForwardByName(templateName) {
            const templateSelect = document.getElementById('templateSelect');
            templateSelect.value = templateName;
            startTemplateForward();
        }

        // 按名称关闭模板转发
        function stopTemplateForwardByName(templateName) {
            const templateSelect = document.getElementById('templateSelect');
            templateSelect.value = templateName;
            stopTemplateForward();
        }

        // 按名称编辑模板
        function editTemplateByName(templateName) {
            const templateSelect = document.getElementById('templateSelect');
            templateSelect.value = templateName;
            editTemplate();
        }

        // 按名称删除模板
        function deleteTemplateByName(templateName) {
            const templateSelect = document.getElementById('templateSelect');
            templateSelect.value = templateName;
            deleteTemplate();
        }

        // 编辑模板
        function editTemplate() {
            const templateSelect = document.getElementById('templateSelect');
            const templateName = templateSelect.value;
            if (!templateName) {
                showMessage('请先选择要编辑的模板', 'info');
                return;
            }

            // 创建编辑模板的对话框
            const overlay = document.createElement('div');
            overlay.style.position = 'fixed';
            overlay.style.top = '0';
            overlay.style.left = '0';
            overlay.style.width = '100%';
            overlay.style.height = '100%';
            overlay.style.backgroundColor = 'rgba(0, 0, 0, 0.5)';
            overlay.style.zIndex = '999';

            const dialog = document.createElement('div');
            dialog.style.position = 'fixed';
            dialog.style.top = '50%';
            dialog.style.left = '50%';
            dialog.style.transform = 'translate(-50%, -50%)';
            dialog.style.backgroundColor = 'white';
            dialog.style.padding = '20px';
            dialog.style.borderRadius = '8px';
            dialog.style.boxShadow = '0 0 20px rgba(0, 0, 0, 0.3)';
            dialog.style.zIndex = '1000';
            dialog.style.minWidth = '300px';

            dialog.innerHTML = '<h3 style="margin-top: 0;">编辑模板</h3>' +
                '<p>请输入新的模板名称：</p>' +
                '<div style="padding: 10px; margin: 15px 0;">' +
                '<input type="text" id="newTemplateName" value="' + templateName + '" style="width: 100%; padding: 8px; border: 1px solid #ddd; border-radius: 4px;">' +
                '</div>' +
                '<div style="display: flex; justify-content: flex-end; gap: 10px;">' +
                '<button id="cancelBtn" style="padding: 8px 16px; border: 1px solid #ddd; border-radius: 4px; background-color: #f5f5f5; cursor: pointer;">取消</button>' +
                '<button id="confirmBtn" style="padding: 8px 16px; border: none; border-radius: 4px; background-color: #3498db; color: white; cursor: pointer;">确定</button>' +
                '</div>';

            document.body.appendChild(overlay);
            document.body.appendChild(dialog);

            document.getElementById('cancelBtn').addEventListener('click', function() {
                document.body.removeChild(overlay);
                document.body.removeChild(dialog);
            });

            document.getElementById('confirmBtn').addEventListener('click', function() {
                const newTemplateName = document.getElementById('newTemplateName').value.trim();
                if (newTemplateName !== '') {
                    // 调用API更新模板名称
                    fetch('/api/updateTemplate', {
                        method: 'POST',
                        headers: {
                            'Content-Type': 'application/json'
                        },
                        body: JSON.stringify({
                            oldName: templateName,
                            newName: newTemplateName
                        })
                    })
                    .then(response => response.json())
                    .then(data => {
                        if (data.success) {
                            loadTemplates();
                            showMessage('模板编辑成功', 'success');
                        }
                    })
                    .catch(error => {
                        console.error('Failed to edit template:', error);
                        showMessage('模板编辑失败', 'error');
                    });
                }
                document.body.removeChild(overlay);
                document.body.removeChild(dialog);
            });

            overlay.addEventListener('click', function() {
                document.body.removeChild(overlay);
                document.body.removeChild(dialog);
            });
        }

        // 加入已有模板
        function addToExistingTemplate() {
            const selectedCheckboxes = document.querySelectorAll('.rule-checkbox:checked');
            if (selectedCheckboxes.length === 0) {
                showMessage('请先选择要加入模板的规则', 'info');
                return;
            }

            if (templates.length === 0) {
                showMessage('暂无模板，请先创建模板', 'info');
                return;
            }

            // 生成模板选择选项
            let templateList = '';
            templates.forEach((template, index) => {
                templateList += (index + 1) + '. ' + template.name + '\n';
            });

            // 让用户输入模板编号
            const templateIndex = prompt('请选择要加入的模板编号：\n' + templateList);
            if (templateIndex) {
                const index = parseInt(templateIndex) - 1;
                if (index >= 0 && index < templates.length) {
                    const templateName = templates[index].name;
                    const selectedIds = Array.from(selectedCheckboxes).map(cb => cb.dataset.id);
                    
                    // 调用API将规则加入已有模板
                    fetch('/api/saveAsTemplate', {
                        method: 'POST',
                        headers: {
                            'Content-Type': 'application/json'
                        },
                        body: JSON.stringify({
                            name: templateName,
                            ids: selectedIds
                        })
                    })
                    .then(response => response.json())
                    .then(data => {
                        if (data.success) {
                            loadTemplates();
                            showMessage('规则已成功加入模板', 'success');
                        }
                    })
                    .catch(error => {
                        console.error('Failed to add to existing template:', error);
                        showMessage('加入模板失败', 'error');
                    });
                } else {
                    showMessage('无效的模板编号', 'error');
                }
            }
        }

        // 创建模板选择对话框
        function createTemplateSelectDialog(selectedCheckboxes) {
            // 创建遮罩层
            const overlay = document.createElement('div');
            overlay.style.position = 'fixed';
            overlay.style.top = '0';
            overlay.style.left = '0';
            overlay.style.width = '100%';
            overlay.style.height = '100%';
            overlay.style.backgroundColor = 'rgba(0, 0, 0, 0.5)';
            overlay.style.zIndex = '999';

            // 创建对话框
            const dialog = document.createElement('div');
            dialog.style.position = 'fixed';
            dialog.style.top = '50%';
            dialog.style.left = '50%';
            dialog.style.transform = 'translate(-50%, -50%)';
            dialog.style.backgroundColor = 'white';
            dialog.style.padding = '20px';
            dialog.style.borderRadius = '8px';
            dialog.style.boxShadow = '0 0 20px rgba(0, 0, 0, 0.3)';
            dialog.style.zIndex = '1000';
            dialog.style.minWidth = '300px';

            // 创建对话框内容
            let templateOptions = '';
            templates.forEach(template => {
                templateOptions += '<option value="' + template.name + '">' + template.name + '</option>';
            });

            dialog.innerHTML = '<h3 style="margin-top: 0;">加入已有模板</h3>' +
                '<p>确定要将选中的规则加入模板吗？</p>' +
                '<div style="padding: 10px; margin: 15px 0;">' +
                '<label style="display: block; margin-bottom: 5px;">选择模板：</label>' +
                '<select id="existingTemplateSelect" style="width: 100%; padding: 8px;">' +
                templateOptions +
                '</select>' +
                '</div>' +
                '<div style="display: flex; justify-content: flex-end; gap: 10px;">' +
                '<button id="cancelBtn" style="padding: 8px 16px; border: 1px solid #ddd; border-radius: 4px; background-color: #f5f5f5; cursor: pointer;">取消</button>' +
                '<button id="confirmBtn" style="padding: 8px 16px; border: none; border-radius: 4px; background-color: #3498db; color: white; cursor: pointer;">确定</button>' +
                '</div>';

            // 添加到页面
            document.body.appendChild(overlay);
            document.body.appendChild(dialog);

            // 绑定事件
            document.getElementById('cancelBtn').addEventListener('click', function() {
                document.body.removeChild(overlay);
                document.body.removeChild(dialog);
            });

            document.getElementById('confirmBtn').addEventListener('click', function() {
                const templateSelect = document.getElementById('existingTemplateSelect');
                if (templateSelect) {
                    const templateName = templateSelect.value;
                    if (templateName) {
                        const selectedIds = Array.from(selectedCheckboxes).map(cb => cb.dataset.id);
                        // 调用API将规则加入已有模板
                        fetch('/api/saveAsTemplate', {
                            method: 'POST',
                            headers: {
                                'Content-Type': 'application/json'
                            },
                            body: JSON.stringify({
                                name: templateName,
                                ids: selectedIds
                            })
                        })
                        .then(response => response.json())
                        .then(data => {
                            if (data.success) {
                                loadTemplates();
                                showMessage('规则已成功加入模板', 'success');
                            }
                        })
                        .catch(error => {
                            console.error('Failed to add to existing template:', error);
                            showMessage('加入模板失败', 'error');
                        });
                    }
                }
                // 关闭对话框
                document.body.removeChild(overlay);
                document.body.removeChild(dialog);
            });

            // 点击遮罩层关闭对话框
            overlay.addEventListener('click', function() {
                document.body.removeChild(overlay);
                document.body.removeChild(dialog);
            });
        }

        // 新建模板
        function createNewTemplate() {

            // 创建新建模板的对话框
            const overlay = document.createElement('div');
            overlay.style.position = 'fixed';
            overlay.style.top = '0';
            overlay.style.left = '0';
            overlay.style.width = '100%';
            overlay.style.height = '100%';
            overlay.style.backgroundColor = 'rgba(0, 0, 0, 0.5)';
            overlay.style.zIndex = '999';

            const dialog = document.createElement('div');
            dialog.style.position = 'fixed';
            dialog.style.top = '50%';
            dialog.style.left = '50%';
            dialog.style.transform = 'translate(-50%, -50%)';
            dialog.style.backgroundColor = 'white';
            dialog.style.padding = '20px';
            dialog.style.borderRadius = '8px';
            dialog.style.boxShadow = '0 0 20px rgba(0, 0, 0, 0.3)';
            dialog.style.zIndex = '1000';
            dialog.style.minWidth = '300px';

            dialog.innerHTML = '<h3 style="margin-top: 0;">新建模板</h3>' +
                '<p>请输入模板名称：</p>' +
                '<div style="padding: 10px; margin: 15px 0;">' +
                '<input type="text" id="templateName" placeholder="请输入模板名称" style="width: 100%; padding: 8px; border: 1px solid #ddd; border-radius: 4px;">' +
                '</div>' +
                '<div style="display: flex; justify-content: flex-end; gap: 10px;">' +
                '<button id="cancelBtn" style="padding: 8px 16px; border: 1px solid #ddd; border-radius: 4px; background-color: #f5f5f5; cursor: pointer;">取消</button>' +
                '<button id="confirmBtn" style="padding: 8px 16px; border: none; border-radius: 4px; background-color: #3498db; color: white; cursor: pointer;">确定</button>' +
                '</div>';

            document.body.appendChild(overlay);
            document.body.appendChild(dialog);

            document.getElementById('cancelBtn').addEventListener('click', function() {
                document.body.removeChild(overlay);
                document.body.removeChild(dialog);
            });

            document.getElementById('confirmBtn').addEventListener('click', function() {
                const templateName = document.getElementById('templateName').value.trim();
                if (templateName !== '') {
                    // 直接创建模板，不强制要求必须选择规则
                    const selectedIds = Array.from(document.querySelectorAll('.rule-checkbox:checked')).map(cb => cb.dataset.id);
                    fetch('/api/saveAsTemplate', {
                        method: 'POST',
                        headers: {
                            'Content-Type': 'application/json'
                        },
                        body: JSON.stringify({
                            name: templateName,
                            ids: selectedIds
                        })
                    })
                    .then(response => response.json())
                    .then(data => {
                        if (data.success) {
                            loadTemplates();
                            showMessage('模板创建成功', 'success');
                        }
                    })
                    .catch(error => {
                        console.error('Failed to create template:', error);
                        showMessage('模板创建失败', 'error');
                    });
                }
                document.body.removeChild(overlay);
                document.body.removeChild(dialog);
            });

            overlay.addEventListener('click', function() {
                document.body.removeChild(overlay);
                document.body.removeChild(dialog);
            });
        }

        // 新增规则
        function addRule() {
            fetch('/api/addRule', {
                method: 'POST'
            })
            .then(response => response.json())
            .then(data => {
                if (data.success) {
                    loadRules();
                    showMessage('规则添加成功', 'success');
                }
            })
            .catch(error => {
                console.error('Failed to add rule:', error);
            });
        }

        // 删除选中规则
        function deleteSelectedRules() {
            const selectedCheckboxes = document.querySelectorAll('.rule-checkbox:checked');
            if (selectedCheckboxes.length === 0) {
                showMessage('请先选择要删除的规则', 'info');
                return;
            }

            const selectedIds = Array.from(selectedCheckboxes).map(cb => cb.dataset.id);
            fetch('/api/deleteRules', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({ ids: selectedIds })
            })
            .then(response => response.json())
            .then(data => {
                if (data.success) {
                    loadRules();
                    showMessage('规则删除成功', 'success');
                }
            })
            .catch(error => {
                console.error('Failed to delete rules:', error);
            });
        }

        // 删除单个规则
        function deleteRule(id) {
            if (confirm('确定要删除此规则吗？')) {
                fetch('/api/deleteRules', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json'
                    },
                    body: JSON.stringify({ ids: [id] })
                })
                .then(response => response.json())
                .then(data => {
                    if (data.success) {
                        const templateSelect = document.getElementById('templateSelect');
                        const templateName = templateSelect.value;
                        if (templateName && templateName !== 'default') {
                            // 当前在模板视图中，重新加载模板规则
                            fetch('/api/getTemplates')
                                .then(response => response.json())
                                .then(data => {
                                    const template = data.find(t => t.name === templateName);
                                    if (template) {
                                        renderTemplateRules(template);
                                    }
                                });
                        } else {
                            // 当前在所有记录视图中，加载所有规则
                            loadRules();
                        }
                        showMessage('规则删除成功', 'success');
                    }
                })
                .catch(error => {
                    console.error('Failed to delete rule:', error);
                });
            }
        }

        // 复制规则信息
        function copyRule(index) {
            const rule = rules[index];
            const info = rule.listenAddr + ':' + rule.listenPort;
            navigator.clipboard.writeText(info)
                .then(() => {
                    showMessage('已复制: ' + info, 'success');
                })
                .catch(err => {
                    console.error('复制失败:', err);
                    showMessage('复制失败', 'error');
                });
        }

        // 从模板复制规则信息
        function copyRuleFromTemplate(index, templateName) {
            fetch('/api/getTemplates')
                .then(response => response.json())
                .then(data => {
                    const template = data.find(t => t.name === templateName);
                    if (template) {
                        const rule = template.rules[index];
                        const info = rule.listenAddr + ':' + rule.listenPort;
                        navigator.clipboard.writeText(info)
                            .then(() => {
                                showMessage('已复制: ' + info, 'success');
                            })
                            .catch(err => {
                                console.error('复制失败:', err);
                                showMessage('复制失败', 'error');
                            });
                    }
                });
        }

        // 显示二维码
        function showQRCode(listenAddr, listenPort) {
            const info = listenAddr + ':' + listenPort;
            const qrCodeUrl = '/api/getQRCode?listenAddr=' + encodeURIComponent(listenAddr) + '&listenPort=' + encodeURIComponent(listenPort);
            
            // 创建弹窗
            const popupDiv = document.createElement('div');
            popupDiv.style.position = 'fixed';
            popupDiv.style.top = '50%';
            popupDiv.style.left = '50%';
            popupDiv.style.transform = 'translate(-50%, -50%)';
            popupDiv.style.backgroundColor = 'white';
            popupDiv.style.padding = '20px';
            popupDiv.style.borderRadius = '8px';
            popupDiv.style.boxShadow = '0 0 20px rgba(0, 0, 0, 0.3)';
            popupDiv.style.zIndex = '1000';
            popupDiv.style.textAlign = 'center';
            
            // 创建关闭按钮
            const closeBtn = document.createElement('button');
            closeBtn.textContent = '关闭';
            closeBtn.style.position = 'absolute';
            closeBtn.style.top = '10px';
            closeBtn.style.right = '10px';
            closeBtn.style.padding = '5px 10px';
            closeBtn.style.border = 'none';
            closeBtn.style.borderRadius = '4px';
            closeBtn.style.backgroundColor = '#666';
            closeBtn.style.color = 'white';
            closeBtn.style.cursor = 'pointer';
            closeBtn.onclick = function() {
                document.body.removeChild(popupDiv);
                document.body.removeChild(overlay);
            };
            
            // 创建内容
            const content = document.createElement('div');
            content.innerHTML = '<h3>访问地址</h3><p>' + info + '</p><img src="' + qrCodeUrl + '" alt="二维码"><p style="margin-top: 10px; font-size: 12px; color: #666;">扫码访问源IP:源端口</p>';
            
            // 组装弹窗
            popupDiv.appendChild(closeBtn);
            popupDiv.appendChild(content);
            
            // 创建遮罩层
            const overlay = document.createElement('div');
            overlay.style.position = 'fixed';
            overlay.style.top = '0';
            overlay.style.left = '0';
            overlay.style.width = '100%';
            overlay.style.height = '100%';
            overlay.style.backgroundColor = 'rgba(0, 0, 0, 0.5)';
            overlay.style.zIndex = '999';
            overlay.onclick = function() {
                document.body.removeChild(popupDiv);
                document.body.removeChild(overlay);
            };
            
            // 添加到页面
            document.body.appendChild(overlay);
            document.body.appendChild(popupDiv);
        }

        // 保存为模板
        function saveAsTemplate() {
            const selectedCheckboxes = document.querySelectorAll('.rule-checkbox:checked');
            if (selectedCheckboxes.length === 0) {
                showMessage('请先选择要保存为模板的规则', 'info');
                return;
            }

            // 创建保存模板的对话框
            const overlay = document.createElement('div');
            overlay.style.position = 'fixed';
            overlay.style.top = '0';
            overlay.style.left = '0';
            overlay.style.width = '100%';
            overlay.style.height = '100%';
            overlay.style.backgroundColor = 'rgba(0, 0, 0, 0.5)';
            overlay.style.zIndex = '999';

            const dialog = document.createElement('div');
            dialog.style.position = 'fixed';
            dialog.style.top = '50%';
            dialog.style.left = '50%';
            dialog.style.transform = 'translate(-50%, -50%)';
            dialog.style.backgroundColor = 'white';
            dialog.style.padding = '20px';
            dialog.style.borderRadius = '8px';
            dialog.style.boxShadow = '0 0 20px rgba(0, 0, 0, 0.3)';
            dialog.style.zIndex = '1000';
            dialog.style.minWidth = '300px';

            dialog.innerHTML = '<h3 style="margin-top: 0;">保存为模板</h3>' +
                '<p>请输入模板名称：</p>' +
                '<div style="padding: 10px; margin: 15px 0;">' +
                '<input type="text" id="templateName" placeholder="请输入模板名称" style="width: 100%; padding: 8px; border: 1px solid #ddd; border-radius: 4px;">' +
                '</div>' +
                '<div style="display: flex; justify-content: flex-end; gap: 10px;">' +
                '<button id="cancelBtn" style="padding: 8px 16px; border: 1px solid #ddd; border-radius: 4px; background-color: #f5f5f5; cursor: pointer;">取消</button>' +
                '<button id="confirmBtn" style="padding: 8px 16px; border: none; border-radius: 4px; background-color: #3498db; color: white; cursor: pointer;">确定</button>' +
                '</div>';

            document.body.appendChild(overlay);
            document.body.appendChild(dialog);

            document.getElementById('cancelBtn').addEventListener('click', function() {
                document.body.removeChild(overlay);
                document.body.removeChild(dialog);
            });

            document.getElementById('confirmBtn').addEventListener('click', function() {
                const templateName = document.getElementById('templateName').value.trim();
                if (templateName !== '') {
                    const selectedIds = Array.from(selectedCheckboxes).map(cb => cb.dataset.id);
                    fetch('/api/saveAsTemplate', {
                        method: 'POST',
                        headers: {
                            'Content-Type': 'application/json'
                        },
                        body: JSON.stringify({
                            name: templateName,
                            ids: selectedIds
                        })
                    })
                    .then(response => response.json())
                    .then(data => {
                        if (data.success) {
                            loadTemplates();
                            showMessage('模板保存成功', 'success');
                        }
                    })
                    .catch(error => {
                        console.error('Failed to save template:', error);
                        showMessage('模板保存失败', 'error');
                    });
                }
                document.body.removeChild(overlay);
                document.body.removeChild(dialog);
            });

            overlay.addEventListener('click', function() {
                document.body.removeChild(overlay);
                document.body.removeChild(dialog);
            });
        }

        // 应用模板
        function applyTemplate() {
            const templateSelect = document.getElementById('templateSelect');
            const templateName = templateSelect.value;
            if (!templateName) {
                showMessage('请先选择要应用的模板', 'info');
                return;
            }

            // 切换到模板记录
            fetch('/api/applyTemplate', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({ name: templateName })
            })
            .then(response => response.json())
            .then(data => {
                if (data.success) {
                    // 显示模板中的规则
                    renderTemplateRules({ name: templateName, rules: data.rules });
                    showMessage('已切换到模板：' + templateName, 'success');
                }
            })
            .catch(error => {
                console.error('Failed to get template:', error);
            });
        }

        // 渲染模板规则
        function renderTemplateRules(template) {
            const rulesList = document.getElementById('rulesList');
            rulesList.innerHTML = '';

            if (template.rules.length === 0) {
                rulesList.innerHTML = '<p style="text-align: center; color: #999; padding: 20px;">模板中暂无规则</p>';
                return;
            }

            // 按照规则的seq字段倒序排序
            const sortedRules = template.rules.sort((a, b) => {
                return (b.seq || 0) - (a.seq || 0);
            });

            sortedRules.forEach((rule, index) => {
                const ruleItem = document.createElement('div');
                ruleItem.className = 'rule-item';
                ruleItem.dataset.id = rule.id;

                // 检查TCP和UDP状态
                Promise.all([
                    fetch('/api/isTCPRunning?listenAddr=' + rule.listenAddr + '&listenPort=' + rule.listenPort).then(r => r.json()),
                    fetch('/api/isUDPRunning?listenAddr=' + rule.listenAddr + '&listenPort=' + rule.listenPort).then(r => r.json())
                ]).then(function(results) {
                    const tcpResult = results[0];
                    const udpResult = results[1];
                    const tcpRunning = tcpResult.running;
                    const udpRunning = udpResult.running;

                    // 确保seq字段存在
                    const seq = rule.seq || 0;
                    ruleItem.innerHTML = '<input type="checkbox" class="rule-checkbox" data-id="' + rule.id + '"><div style="display: flex; align-items: center;"><div class="rule-seq">' + seq + '</div><div class="rule-config"><select class="listen-addr" data-id="' + rule.id + '">' + renderIPOptions(rule.listenAddr) + '</select><input type="number" class="listen-port" data-id="' + rule.id + '" value="' + rule.listenPort + '" min="1" max="65535"><select class="target-addr" data-id="' + rule.id + '">' + renderTargetIPOptions(rule.targetAddr) + '</select><input type="number" class="target-port" data-id="' + rule.id + '" value="' + rule.targetPort + '" min="1" max="65535"></div></div><div class="rule-actions"><button class="btn ' + (tcpRunning ? 'btn-danger' : 'btn-success') + '" onclick="toggleTCPForwardFromTemplate(' + index + ', \'' + template.name + '\')">' + (tcpRunning ? '停止TCP转发' : '开启TCP转发') + '</button><button class="btn ' + (udpRunning ? 'btn-danger' : 'btn-success') + '" onclick="toggleUDPForwardFromTemplate(' + index + ', \'' + template.name + '\')">' + (udpRunning ? '停止UDP转发' : '开启UDP转发') + '</button><button class="btn btn-danger" onclick="deleteRule(\'' + rule.id + '\')">删除</button><button class="btn btn-primary" onclick="copyRuleFromTemplate(' + index + ', \'' + template.name + '\')">复制</button><button class="btn btn-warning" onclick="showQRCode(\'' + rule.listenAddr + '\', \'' + rule.listenPort + '\')">二维码</button></div>';


                    rulesList.appendChild(ruleItem);

                    // 添加事件监听器
                    addTemplateRuleEventListeners(ruleItem, rule.id);
                });
            });
        }

        // 添加模板规则事件监听器
        function addTemplateRuleEventListeners(ruleItem, ruleId) {
            // 监听地址变化
            const listenAddrSelect = ruleItem.querySelector('.listen-addr[data-id="' + ruleId + '"]');
            if (listenAddrSelect) {
                listenAddrSelect.addEventListener('change', function() {
                    // 监听地址没有自定义选项
                });
            }

            // 目标地址变化
            const targetAddrSelect = ruleItem.querySelector('.target-addr[data-id="' + ruleId + '"]');
            if (targetAddrSelect) {
                targetAddrSelect.addEventListener('change', function() {
                    if (this.value === 'custom') {
                        // 创建自定义输入框
                        const customInput = document.createElement('input');
                        customInput.type = 'text';
                        customInput.className = 'target-addr-custom';
                        customInput.placeholder = '请输入自定义IP地址';
                        customInput.style.marginLeft = '10px';
                        customInput.style.padding = '6px 10px';
                        customInput.style.border = '1px solid #ddd';
                        customInput.style.borderRadius = '4px';
                        customInput.style.fontSize = '14px';

                        // 替换选择框为输入框
                        const parent = this.parentElement;
                        parent.replaceChild(customInput, this);

                        // 聚焦到输入框
                        customInput.focus();

                        // 监听输入框变化
                        customInput.addEventListener('change', function() {
                            if (this.value) {
                                // 更新规则
                                const ruleItem = this.closest('.rule-item');
                                const listenAddr = ruleItem.querySelector('.listen-addr').value;
                                const listenPort = ruleItem.querySelector('.listen-port').value;
                                const targetPort = ruleItem.querySelector('.target-port').value;

                                fetch('/api/updateRule', {
                                    method: 'POST',
                                    headers: {
                                        'Content-Type': 'application/json'
                                    },
                                    body: JSON.stringify({
                                        id: ruleId,
                                        listenAddr: listenAddr,
                                        listenPort: listenPort,
                                        targetAddr: this.value,
                                        targetPort: targetPort
                                    })
                                })
                                .then(response => response.json())
                                .then(data => {
                                    if (data.success) {
                                        // 重新加载模板规则
                                        const templateSelect = document.getElementById('templateSelect');
                                        const templateName = templateSelect.value;
                                        if (templateName !== 'default') {
                                            fetch('/api/getTemplates')
                                                .then(response => response.json())
                                                .then(data => {
                                                    const template = data.find(t => t.name === templateName);
                                                    if (template) {
                                                        renderTemplateRules(template);
                                                    }
                                                });
                                        }
                                    }
                                });
                            } else {
                                // 如果输入框为空，恢复选择框
                                parent.replaceChild(targetAddrSelect, this);
                            }
                        });
                    }
                });
            }
        }

        // 从模板切换TCP转发
        function toggleTCPForwardFromTemplate(index, templateName) {
            // 通过模板名称获取模板对象
            fetch('/api/getTemplates')
                .then(response => response.json())
                .then(data => {
                    const template = data.find(t => t.name === templateName);
                    if (template) {
                        const rule = template.rules[index];

                        // 检查当前状态
                        fetch('/api/isTCPRunning?listenAddr=' + rule.listenAddr + '&listenPort=' + rule.listenPort)
                            .then(function(response) { return response.json(); })
                            .then(function(data) {
                                if (data.running) {
                                    // 停止TCP转发
                                    fetch('/api/stopTCPForward', {
                                        method: 'POST',
                                        headers: {
                                            'Content-Type': 'application/json'
                                        },
                                        body: JSON.stringify({
                                            listenAddr: rule.listenAddr,
                                            listenPort: rule.listenPort
                                        })
                                    })
                                    .then(function(response) { return response.json(); })
                                    .then(function(result) {
                                        if (result.success) {
                                            showMessage('TCP转发已停止', 'success');
                                            const templateSelect = document.getElementById('templateSelect');
                                            const templateName = templateSelect.value;
                                            if (templateName !== 'default') {
                                                fetch('/api/getTemplates')
                                                    .then(response => response.json())
                                                    .then(data => {
                                                        const template = data.find(t => t.name === templateName);
                                                        if (template) {
                                                            renderTemplateRules(template);
                                                        }
                                                    });
                                            }
                                        } else {
                                            showMessage('停止TCP转发失败: ' + result.error, 'error');
                                        }
                                    });
                                } else {
                                    // 启动TCP转发
                                    fetch('/api/startTCPForward', {
                                        method: 'POST',
                                        headers: {
                                            'Content-Type': 'application/json'
                                        },
                                        body: JSON.stringify({
                                            listenAddr: rule.listenAddr,
                                            listenPort: rule.listenPort,
                                            targetAddr: rule.targetAddr,
                                            targetPort: rule.targetPort
                                        })
                                    })
                                    .then(function(response) { return response.json(); })
                                    .then(function(result) {
                                        if (result.success) {
                                            showMessage('TCP转发已启动', 'success');
                                            const templateSelect = document.getElementById('templateSelect');
                                            const templateName = templateSelect.value;
                                            if (templateName !== 'default') {
                                                fetch('/api/getTemplates')
                                                    .then(response => response.json())
                                                    .then(data => {
                                                        const template = data.find(t => t.name === templateName);
                                                        if (template) {
                                                            renderTemplateRules(template);
                                                        }
                                                    });
                                            }
                                        } else {
                                            showMessage('启动TCP转发失败: ' + result.error, 'error');
                                        }
                                    });
                                }
                            });
                    }
                });
        }

        // 从模板切换UDP转发
        function toggleUDPForwardFromTemplate(index, templateName) {
            // 通过模板名称获取模板对象
            fetch('/api/getTemplates')
                .then(response => response.json())
                .then(data => {
                    const template = data.find(t => t.name === templateName);
                    if (template) {
                        const rule = template.rules[index];

                        // 检查当前状态
                        fetch('/api/isUDPRunning?listenAddr=' + rule.listenAddr + '&listenPort=' + rule.listenPort)
                            .then(function(response) { return response.json(); })
                            .then(function(data) {
                                if (data.running) {
                                    // 停止UDP转发
                                    fetch('/api/stopUDPForward', {
                                        method: 'POST',
                                        headers: {
                                            'Content-Type': 'application/json'
                                        },
                                        body: JSON.stringify({
                                            listenAddr: rule.listenAddr,
                                            listenPort: rule.listenPort
                                        })
                                    })
                                    .then(function(response) { return response.json(); })
                                    .then(function(result) {
                                        if (result.success) {
                                            showMessage('UDP转发已停止', 'success');
                                            const templateSelect = document.getElementById('templateSelect');
                                            const templateName = templateSelect.value;
                                            if (templateName !== 'default') {
                                                fetch('/api/getTemplates')
                                                    .then(response => response.json())
                                                    .then(data => {
                                                        const template = data.find(t => t.name === templateName);
                                                        if (template) {
                                                            renderTemplateRules(template);
                                                        }
                                                    });
                                            }
                                        } else {
                                            showMessage('停止UDP转发失败: ' + result.error, 'error');
                                        }
                                    });
                                } else {
                                    // 启动UDP转发
                                    fetch('/api/startUDPForward', {
                                        method: 'POST',
                                        headers: {
                                            'Content-Type': 'application/json'
                                        },
                                        body: JSON.stringify({
                                            listenAddr: rule.listenAddr,
                                            listenPort: rule.listenPort,
                                            targetAddr: rule.targetAddr,
                                            targetPort: rule.targetPort
                                        })
                                    })
                                    .then(function(response) { return response.json(); })
                                    .then(function(result) {
                                        if (result.success) {
                                            showMessage('UDP转发已启动', 'success');
                                            const templateSelect = document.getElementById('templateSelect');
                                            const templateName = templateSelect.value;
                                            if (templateName !== 'default') {
                                                fetch('/api/getTemplates')
                                                    .then(response => response.json())
                                                    .then(data => {
                                                        const template = data.find(t => t.name === templateName);
                                                        if (template) {
                                                            renderTemplateRules(template);
                                                        }
                                                    });
                                            }
                                        } else {
                                            showMessage('启动UDP转发失败: ' + result.error, 'error');
                                        }
                                    });
                                }
                            });
                    }
                });
        }

        // 渲染目标IP选项
        function renderTargetIPOptions(selectedAddr) {
            let options = '<option value="">选择目标IP</option>';
            if (window.localIPs) {
                window.localIPs.forEach(function(ipInfo) {
                    const selected = ipInfo.ip === selectedAddr ? 'selected' : '';
                    options += '<option value="' + ipInfo.ip + '" ' + selected + '>' + ipInfo.ip + ' (' + ipInfo.name + ')</option>';
                });
            }
            // 检查是否是自定义IP
            const isCustom = selectedAddr && (!window.localIPs || !window.localIPs.some(function(ipInfo) { return ipInfo.ip === selectedAddr; }));
            if (isCustom) {
                options += '<option value="' + selectedAddr + '" selected>' + selectedAddr + '</option>';
            } else {
                options += '<option value="custom">自定义</option>';
            }
            return options;
        }

        // 一键开启此模板所有转发
        function startTemplateForward() {
            const templateSelect = document.getElementById('templateSelect');
            const templateName = templateSelect.value;
            if (!templateName) {
                showMessage('请先选择要开启的模板', 'info');
                return;
            }

            fetch('/api/startTemplateForward', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({ name: templateName })
            })
            .then(response => response.json())
            .then(data => {
                if (data.success) {
                    if (templateName && templateName !== 'default') {
                        // 当前在模板视图中，重新渲染模板规则以更新状态
                        fetch('/api/getTemplates')
                            .then(response => response.json())
                            .then(data => {
                                const template = data.find(t => t.name === templateName);
                                if (template) {
                                    renderTemplateRules(template);
                                }
                            });
                    } else {
                        // 当前在所有记录视图中，加载所有规则
                        loadRules();
                    }
                    showMessage('模板转发已开启', 'success');
                }
            })
            .catch(error => {
                console.error('Failed to start template forward:', error);
            });
        }

        // 一键关闭此模板所有转发
        function stopTemplateForward() {
            const templateSelect = document.getElementById('templateSelect');
            const templateName = templateSelect.value;
            if (!templateName) {
                showMessage('请先选择要关闭的模板', 'info');
                return;
            }

            fetch('/api/stopTemplateForward', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({ name: templateName })
            })
            .then(response => response.json())
            .then(data => {
                if (data.success) {
                    if (templateName && templateName !== 'default') {
                        // 当前在模板视图中，重新渲染模板规则以更新状态
                        fetch('/api/getTemplates')
                            .then(response => response.json())
                            .then(data => {
                                const template = data.find(t => t.name === templateName);
                                if (template) {
                                    renderTemplateRules(template);
                                }
                            });
                    } else {
                        // 当前在所有记录视图中，加载所有规则
                        loadRules();
                    }
                    showMessage('模板转发已关闭', 'success');
                }
            })
            .catch(error => {
                console.error('Failed to stop template forward:', error);
            });
        }

        // 删除此模板
        function deleteTemplate() {
            const templateSelect = document.getElementById('templateSelect');
            const templateName = templateSelect.value;
            if (!templateName) {
                showMessage('请先选择要删除的模板', 'info');
                return;
            }

            if (confirm('确定要删除此模板吗？删除后将无法恢复。')) {
                fetch('/api/deleteTemplate', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json'
                    },
                    body: JSON.stringify({ name: templateName })
                })
                .then(response => response.json())
                .then(data => {
                    if (data.success) {
                        // 重新加载模板列表
                        loadTemplates();
                        // 显示所有规则
                        loadRules();
                        showMessage('模板删除成功', 'success');
                    }
                })
                .catch(error => {
                    console.error('Failed to delete template:', error);
                    showMessage('模板删除失败', 'error');
                });
            }
        }

        // 切换TCP转发
        function toggleTCPForward(index) {
            const rule = rules[index];
            
            // 检查当前状态
            fetch('/api/isTCPRunning?listenAddr=' + rule.listenAddr + '&listenPort=' + rule.listenPort)
                .then(function(response) { return response.json(); })
                .then(function(data) {
                    if (data.running) {
                        // 停止TCP转发
                        fetch('/api/stopTCPForward', {
                            method: 'POST',
                            headers: {
                                'Content-Type': 'application/json'
                            },
                            body: JSON.stringify({
                                listenAddr: rule.listenAddr,
                                listenPort: rule.listenPort
                            })
                        })
                        .then(function(response) { return response.json(); })
                        .then(function(result) {
                            if (result.success) {
                                showMessage('TCP转发已停止', 'success');
                                loadRules();
                            } else {
                                showMessage('停止TCP转发失败: ' + result.error, 'error');
                            }
                        });
                    } else {
                        // 启动TCP转发
                        fetch('/api/startTCPForward', {
                            method: 'POST',
                            headers: {
                                'Content-Type': 'application/json'
                            },
                            body: JSON.stringify({
                                listenAddr: rule.listenAddr,
                                listenPort: rule.listenPort,
                                targetAddr: rule.targetAddr,
                                targetPort: rule.targetPort
                            })
                        })
                        .then(function(response) { return response.json(); })
                        .then(function(result) {
                            if (result.success) {
                                showMessage('TCP转发已启动', 'success');
                                loadRules();
                            } else {
                                showMessage('启动TCP转发失败: ' + result.error, 'error');
                            }
                        });
                    }
                });
        }

        // 切换UDP转发
        function toggleUDPForward(index) {
            const rule = rules[index];
            
            // 检查当前状态
            fetch('/api/isUDPRunning?listenAddr=' + rule.listenAddr + '&listenPort=' + rule.listenPort)
                .then(function(response) { return response.json(); })
                .then(function(data) {
                    if (data.running) {
                        // 停止UDP转发
                        fetch('/api/stopUDPForward', {
                            method: 'POST',
                            headers: {
                                'Content-Type': 'application/json'
                            },
                            body: JSON.stringify({
                                listenAddr: rule.listenAddr,
                                listenPort: rule.listenPort
                            })
                        })
                        .then(function(response) { return response.json(); })
                        .then(function(result) {
                            if (result.success) {
                                showMessage('UDP转发已停止', 'success');
                                loadRules();
                            } else {
                                showMessage('停止UDP转发失败: ' + result.error, 'error');
                            }
                        });
                    } else {
                        // 启动UDP转发
                        fetch('/api/startUDPForward', {
                            method: 'POST',
                            headers: {
                                'Content-Type': 'application/json'
                            },
                            body: JSON.stringify({
                                listenAddr: rule.listenAddr,
                                listenPort: rule.listenPort,
                                targetAddr: rule.targetAddr,
                                targetPort: rule.targetPort
                            })
                        })
                        .then(function(response) { return response.json(); })
                        .then(function(result) {
                            if (result.success) {
                                showMessage('UDP转发已启动', 'success');
                                loadRules();
                            } else {
                                showMessage('启动UDP转发失败: ' + result.error, 'error');
                            }
                        });
                    }
                });
        }

        // 显示消息
        function showMessage(message, type) {
            const statusMessage = document.getElementById('statusMessage');
            statusMessage.textContent = message;
            statusMessage.className = 'status-message status-' + type;
            statusMessage.style.display = 'block';

            // 3秒后自动隐藏
            setTimeout(() => {
                statusMessage.style.display = 'none';
            }, 3000);
        }

        // 加载日志
        function loadLog() {
            fetch('/api/getLog')
                .then(response => response.text())
                .then(data => {
                    const logContent = document.getElementById('logContent');
                    logContent.innerHTML = '';
                    
                    // 按行分割日志
                    const lines = data.split('\n');
                    lines.forEach(line => {
                        if (line.trim() !== '') {
                            const p = document.createElement('p');
                            p.textContent = line;
                            logContent.appendChild(p);
                        }
                    });
                    
                    // 滚动到底部
                    logContent.scrollTop = logContent.scrollHeight;
                })
                .catch(error => {
                    console.error('Failed to load log:', error);
                });
        }

        // 定期加载日志
        setInterval(loadLog, 3000);

        // 页面加载时加载日志
        window.onload = function() {
            initApp();
            loadLog();
        };
    </script>
</body>
</html>
`
}

// IPInfo IP地址信息
type IPInfo struct {
	Name string `json:"name"`
	IP   string `json:"ip"`
}

// serveHTML 提供HTML页面
func serveHTML(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(getHTMLContent()))
}

// apiGetLocalIPs 获取本地网卡IP地址
func apiGetLocalIPs(w http.ResponseWriter, r *http.Request) {
	var ipInfos []IPInfo

	// 获取所有网络接口
	interfaces, err := net.Interfaces()
	if err != nil {
		log.Printf("Failed to get network interfaces: %v", err)
		json.NewEncoder(w).Encode([]IPInfo{})
		return
	}

	// 遍历所有网络接口
	for _, iface := range interfaces {
		// 跳过禁用的接口
		if iface.Flags&net.FlagUp == 0 {
			continue
		}

		// 获取接口的IP地址
		addrs, err := iface.Addrs()
		if err != nil {
			log.Printf("Failed to get addresses for interface %s: %v", iface.Name, err)
			continue
		}

		// 遍历所有IP地址
		for _, addr := range addrs {
			// 检查是否是IPv4地址
			if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				if ipnet.IP.To4() != nil {
					ipInfos = append(ipInfos, IPInfo{
						Name: iface.Name,
						IP:   ipnet.IP.String(),
					})
				}
			}
		}
	}

	// 添加本地回环地址
	ipInfos = append(ipInfos, IPInfo{
		Name: "本地回环",
		IP:   "127.0.0.1",
	})

	// 返回JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ipInfos)
}

// apiGetRules 获取规则
func apiGetRules(w http.ResponseWriter, r *http.Request) {
	// 创建规则副本
	rulesCopy := make([]Rule, len(rules))
	copy(rulesCopy, rules)

	// 按 Seq 字段降序排序副本，确保最新的在前
	sort.Slice(rulesCopy, func(i, j int) bool {
		return rulesCopy[i].Seq > rulesCopy[j].Seq
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rulesCopy)
}

// apiGetTemplates 获取模板
func apiGetTemplates(w http.ResponseWriter, r *http.Request) {
	// 按创建时间降序排序，最新的模板在前
	sorted := make([]Template, len(templates))
	copy(sorted, templates)
	sort.Slice(sorted, func(i, j int) bool {
		ti := parseCreatedAt(sorted[i].CreatedAt)
		tj := parseCreatedAt(sorted[j].CreatedAt)
		return tj.After(ti)
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sorted)
}

// parseCreatedAt 尝试把 CreatedAt 字符串解析为时间，空字符串返回零时间
func parseCreatedAt(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	t, err := time.Parse("2006-01-02 15:04:05", s)
	if err != nil {
		return time.Time{}
	}
	return t
}

// apiAddRule 添加规则
func apiAddRule(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 生成唯一ID
	id := uuid.New().String()

	// 计算新规则的序号（当前最大序号+1）
	maxSeq := 0
	for _, rule := range rules {
		if rule.Seq > maxSeq {
			maxSeq = rule.Seq
		}
	}
	seq := maxSeq + 1

	// 创建新规则
	newRule := Rule{
		ID:         id,
		Seq:        seq,
		ListenAddr: "",
		ListenPort: "",
		TargetAddr: "",
		TargetPort: "",
	}

	// 添加到规则列表
	rules = append(rules, newRule)

	// 保存规则
	if err := storage.SaveRules(rules); err != nil {
		log.Printf("Failed to save rules: %v", err)
	}

	// 返回成功
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// apiDeleteRules 删除规则
func apiDeleteRules(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 解析请求体
	var req struct {
		IDs []string `json:"ids"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Failed to decode request body: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// 过滤规则
	var newRules []Rule
	for _, rule := range rules {
		keep := true
		for _, id := range req.IDs {
			if rule.ID == id {
				keep = false
				break
			}
		}
		if keep {
			newRules = append(newRules, rule)
		}
	}

	// 更新规则列表（不再重新计算序号）
	rules = newRules

	// 保存规则
	if err := storage.SaveRules(rules); err != nil {
		log.Printf("Failed to save rules: %v", err)
	}

	// 更新所有模板，过滤掉被删除的规则ID
	for i, template := range templates {
		var newTemplateRules []string
		for _, ruleID := range template.Rules {
			keep := true
			for _, id := range req.IDs {
				if ruleID == id {
					keep = false
					break
				}
			}
			if keep {
				newTemplateRules = append(newTemplateRules, ruleID)
			}
		}
		templates[i].Rules = newTemplateRules
	}

	// 保存模板
	if err := storage.SaveTemplates(templates); err != nil {
		log.Printf("Failed to save templates: %v", err)
	}

	// 返回成功
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// apiUpdateRule 更新规则
func apiUpdateRule(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 解析请求体
	var req struct {
		ID         string `json:"id"`
		ListenAddr string `json:"listenAddr"`
		ListenPort string `json:"listenPort"`
		TargetAddr string `json:"targetAddr"`
		TargetPort string `json:"targetPort"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Failed to decode request body: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// 查找规则
	for i, rule := range rules {
		if rule.ID == req.ID {
			// 更新规则
			rules[i].ListenAddr = req.ListenAddr
			rules[i].ListenPort = req.ListenPort
			rules[i].TargetAddr = req.TargetAddr
			rules[i].TargetPort = req.TargetPort
			break
		}
	}

	// 保存规则
	if err := storage.SaveRules(rules); err != nil {
		log.Printf("Failed to save rules: %v", err)
	}

	// 返回成功
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// apiSaveAsTemplate 保存为模板
func apiSaveAsTemplate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 解析请求体
	var req struct {
		Name string   `json:"name"`
		IDs  []string `json:"ids"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Failed to decode request body: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// 检查是否已存在同名模板
	exists := false
	for i, template := range templates {
		if template.Name == req.Name {
			// 模板已存在，将新规则ID添加到模板中，避免重复
			for _, newID := range req.IDs {
				// 检查规则ID是否已存在于模板中
				existsInTemplate := false
				for _, existingID := range template.Rules {
					if existingID == newID {
						existsInTemplate = true
						break
					}
				}
				// 如果规则ID不存在于模板中，添加它
				if !existsInTemplate {
					templates[i].Rules = append(templates[i].Rules, newID)
				}
			}
			exists = true
			break
		}
	}

	// 如果不存在，添加新模板
	if !exists {
		newTemplate := Template{
			Name:      req.Name,
			Rules:     req.IDs,
			CreatedAt: time.Now().Format("2006-01-02 15:04:05"),
		}
		templates = append(templates, newTemplate)
	}

	// 保存模板
	if err := storage.SaveTemplates(templates); err != nil {
		log.Printf("Failed to save templates: %v", err)
	}

	// 返回成功
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// apiApplyTemplate 应用模板
func apiApplyTemplate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 解析请求体
	var req struct {
		Name string `json:"name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Failed to decode request body: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// 查找模板
	var template *Template
	for i, t := range templates {
		if t.Name == req.Name {
			template = &templates[i]
			break
		}
	}

	if template == nil {
		log.Printf("Template %s not found", req.Name)
		http.Error(w, "Template not found", http.StatusNotFound)
		return
	}

	// 根据模板中的规则ID列表获取对应的规则详情
	var templateRules []Rule
	for _, ruleID := range template.Rules {
		for _, rule := range rules {
			if rule.ID == ruleID {
				templateRules = append(templateRules, rule)
				break
			}
		}
	}

	// 返回模板规则，不添加到主规则列表
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "rules": templateRules})
}

// Result 操作结果
type Result struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// apiStartTCPForward 启动TCP转发
func apiStartTCPForward(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 解析请求体
	var req struct {
		ListenAddr string `json:"listenAddr"`
		ListenPort string `json:"listenPort"`
		TargetAddr string `json:"targetAddr"`
		TargetPort string `json:"targetPort"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Failed to decode request body: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// 启动TCP转发
	err := forwarder.StartTCPForward(req.ListenAddr, req.ListenPort, req.TargetAddr, req.TargetPort)
	if err != nil {
		log.Printf("Failed to start TCP forward: %v", err)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Result{Success: false, Error: err.Error()})
		return
	}

	// 返回成功
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(Result{Success: true})
}

// apiStopTCPForward 停止TCP转发
func apiStopTCPForward(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 解析请求体
	var req struct {
		ListenAddr string `json:"listenAddr"`
		ListenPort string `json:"listenPort"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Failed to decode request body: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// 停止TCP转发
	err := forwarder.StopTCPForward(req.ListenAddr, req.ListenPort)
	if err != nil {
		log.Printf("Failed to stop TCP forward: %v", err)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Result{Success: false, Error: err.Error()})
		return
	}

	// 返回成功
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(Result{Success: true})
}

// apiStartUDPForward 启动UDP转发
func apiStartUDPForward(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 解析请求体
	var req struct {
		ListenAddr string `json:"listenAddr"`
		ListenPort string `json:"listenPort"`
		TargetAddr string `json:"targetAddr"`
		TargetPort string `json:"targetPort"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Failed to decode request body: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// 启动UDP转发
	err := forwarder.StartUDPForward(req.ListenAddr, req.ListenPort, req.TargetAddr, req.TargetPort)
	if err != nil {
		log.Printf("Failed to start UDP forward: %v", err)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Result{Success: false, Error: err.Error()})
		return
	}

	// 返回成功
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(Result{Success: true})
}

// apiStopUDPForward 停止UDP转发
func apiStopUDPForward(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 解析请求体
	var req struct {
		ListenAddr string `json:"listenAddr"`
		ListenPort string `json:"listenPort"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Failed to decode request body: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// 停止UDP转发
	err := forwarder.StopUDPForward(req.ListenAddr, req.ListenPort)
	if err != nil {
		log.Printf("Failed to stop UDP forward: %v", err)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Result{Success: false, Error: err.Error()})
		return
	}

	// 返回成功
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(Result{Success: true})
}

// apiIsTCPRunning 检查TCP转发是否运行
func apiIsTCPRunning(w http.ResponseWriter, r *http.Request) {
	// 解析查询参数
	listenAddr := r.URL.Query().Get("listenAddr")
	listenPort := r.URL.Query().Get("listenPort")

	// 检查TCP转发是否运行
	running := forwarder.IsTCPRunning(listenAddr, listenPort)

	// 返回结果
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"running": running})
}

// apiIsUDPRunning 检查UDP转发是否运行
func apiIsUDPRunning(w http.ResponseWriter, r *http.Request) {
	// 解析查询参数
	listenAddr := r.URL.Query().Get("listenAddr")
	listenPort := r.URL.Query().Get("listenPort")

	// 检查UDP转发是否运行
	running := forwarder.IsUDPRunning(listenAddr, listenPort)

	// 返回结果
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"running": running})
}

// apiStartTemplateForward 启动模板所有转发
func apiStartTemplateForward(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 解析请求体
	var req struct {
		Name string `json:"name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Failed to decode request body: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// 查找模板
	var template *Template
	for i, t := range templates {
		if t.Name == req.Name {
			template = &templates[i]
			break
		}
	}

	if template == nil {
		log.Printf("Template %s not found", req.Name)
		http.Error(w, "Template not found", http.StatusNotFound)
		return
	}

	// 根据模板中的规则ID列表获取对应的规则详情并启动转发
	for _, ruleID := range template.Rules {
		for _, rule := range rules {
			if rule.ID == ruleID {
				// 启动TCP转发
				forwarder.StartTCPForward(rule.ListenAddr, rule.ListenPort, rule.TargetAddr, rule.TargetPort)
				// 启动UDP转发
				forwarder.StartUDPForward(rule.ListenAddr, rule.ListenPort, rule.TargetAddr, rule.TargetPort)
				break
			}
		}
	}

	// 返回成功
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// apiStopTemplateForward 停止模板所有转发
func apiStopTemplateForward(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 解析请求体
	var req struct {
		Name string `json:"name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Failed to decode request body: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// 查找模板
	var template *Template
	for i, t := range templates {
		if t.Name == req.Name {
			template = &templates[i]
			break
		}
	}

	if template == nil {
		log.Printf("Template %s not found", req.Name)
		http.Error(w, "Template not found", http.StatusNotFound)
		return
	}

	// 根据模板中的规则ID列表获取对应的规则详情并停止转发
	for _, ruleID := range template.Rules {
		for _, rule := range rules {
			if rule.ID == ruleID {
				// 停止TCP转发
				forwarder.StopTCPForward(rule.ListenAddr, rule.ListenPort)
				// 停止UDP转发
				forwarder.StopUDPForward(rule.ListenAddr, rule.ListenPort)
				break
			}
		}
	}

	// 返回成功
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// apiGetQRCode 生成二维码
func apiGetQRCode(w http.ResponseWriter, r *http.Request) {
	// 解析查询参数
	listenAddr := r.URL.Query().Get("listenAddr")
	listenPort := r.URL.Query().Get("listenPort")

	if listenAddr == "" || listenPort == "" {
		http.Error(w, "Missing required parameters", http.StatusBadRequest)
		return
	}

	// 生成二维码数据
	data := listenAddr + ":" + listenPort

	// 生成二维码
	qr, err := qrcode.New(data, qrcode.Medium)
	if err != nil {
		log.Printf("Failed to create QR code: %v", err)
		http.Error(w, "Failed to generate QR code", http.StatusInternalServerError)
		return
	}

	// 将二维码写入ResponseWriter
	w.Header().Set("Content-Type", "image/png")
	png.Encode(w, qr.Image(200))
}

// apiDeleteTemplate 删除模板
func apiDeleteTemplate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 解析请求体
	var req struct {
		Name string `json:"name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Failed to decode request body: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, "Template name is required", http.StatusBadRequest)
		return
	}

	// 过滤模板
	var newTemplates []Template
	for _, template := range templates {
		if template.Name != req.Name {
			newTemplates = append(newTemplates, template)
		}
	}

	// 更新模板列表
	templates = newTemplates

	// 保存模板
	if err := storage.SaveTemplates(templates); err != nil {
		log.Printf("Failed to save templates: %v", err)
	}

	// 返回成功
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// apiUpdateTemplate 更新模板
func apiUpdateTemplate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 解析请求体
	var req struct {
		OldName string `json:"oldName"`
		NewName string `json:"newName"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Failed to decode request body: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.OldName == "" || req.NewName == "" {
		http.Error(w, "Both old and new template names are required", http.StatusBadRequest)
		return
	}

	// 查找并更新模板
	updated := false
	for i, template := range templates {
		if template.Name == req.OldName {
			// 更新模板名称
			templates[i].Name = req.NewName
			updated = true
			break
		}
	}

	if !updated {
		log.Printf("Template %s not found", req.OldName)
		http.Error(w, "Template not found", http.StatusNotFound)
		return
	}

	// 保存模板
	if err := storage.SaveTemplates(templates); err != nil {
		log.Printf("Failed to save templates: %v", err)
	}

	// 返回成功
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// apiGetLog 获取日志
func apiGetLog(w http.ResponseWriter, r *http.Request) {
	// 读取日志文件
	logData, err := os.ReadFile(filepath.Join(".", "db", "log.txt"))
	if err != nil {
		log.Printf("Failed to read log file: %v", err)
		http.Error(w, "Failed to read log file", http.StatusInternalServerError)
		return
	}

	// 设置响应头
	w.Header().Set("Content-Type", "text/plain")

	// 返回日志内容
	w.Write(logData)
}

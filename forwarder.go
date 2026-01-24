package main

import (
	"fmt"
	"log"
	"net"
	"sync"
)

// Forwarder 端口转发器
type Forwarder struct {
	tcpListeners map[string]*net.Listener
	udpListeners map[string]*net.UDPConn
	mu           sync.Mutex
}

// NewForwarder 创建新的端口转发器
func NewForwarder() *Forwarder {
	return &Forwarder{
		tcpListeners: make(map[string]*net.Listener),
		udpListeners: make(map[string]*net.UDPConn),
	}
}

// StartTCPForward 启动TCP端口转发
func (f *Forwarder) StartTCPForward(listenAddr, listenPort, targetAddr, targetPort string) error {
	key := fmt.Sprintf("tcp:%s:%s", listenAddr, listenPort)

	f.mu.Lock()
	defer f.mu.Unlock()

	// 检查是否已经在运行
	if _, exists := f.tcpListeners[key]; exists {
		return fmt.Errorf("TCP forward already running on %s:%s", listenAddr, listenPort)
	}

	// 监听本地端口
	addr := fmt.Sprintf("%s:%s", listenAddr, listenPort)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}

	// 保存监听器
	f.tcpListeners[key] = &listener

	// 启动转发协程
	go f.handleTCPForward(listener, targetAddr, targetPort)

	log.Printf("Started TCP forward: %s:%s -> %s:%s", listenAddr, listenPort, targetAddr, targetPort)
	return nil
}

// StopTCPForward 停止TCP端口转发
func (f *Forwarder) StopTCPForward(listenAddr, listenPort string) error {
	key := fmt.Sprintf("tcp:%s:%s", listenAddr, listenPort)

	f.mu.Lock()
	defer f.mu.Unlock()

	// 检查是否在运行
	listener, exists := f.tcpListeners[key]
	if !exists {
		return fmt.Errorf("TCP forward not running on %s:%s", listenAddr, listenPort)
	}

	// 关闭监听器
	if err := (*listener).Close(); err != nil {
		return fmt.Errorf("failed to close listener: %w", err)
	}

	// 删除监听器
	delete(f.tcpListeners, key)

	log.Printf("Stopped TCP forward: %s:%s", listenAddr, listenPort)
	return nil
}

// StartUDPForward 启动UDP端口转发
func (f *Forwarder) StartUDPForward(listenAddr, listenPort, targetAddr, targetPort string) error {
	key := fmt.Sprintf("udp:%s:%s", listenAddr, listenPort)

	f.mu.Lock()
	defer f.mu.Unlock()

	// 检查是否已经在运行
	if _, exists := f.udpListeners[key]; exists {
		return fmt.Errorf("UDP forward already running on %s:%s", listenAddr, listenPort)
	}

	// 解析监听地址
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%s", listenAddr, listenPort))
	if err != nil {
		return fmt.Errorf("failed to resolve listen address: %w", err)
	}

	// 监听UDP端口
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s:%s: %w", listenAddr, listenPort, err)
	}

	// 保存连接
	f.udpListeners[key] = conn

	// 启动转发协程
	go f.handleUDPForward(conn, targetAddr, targetPort)

	log.Printf("Started UDP forward: %s:%s -> %s:%s", listenAddr, listenPort, targetAddr, targetPort)
	return nil
}

// StopUDPForward 停止UDP端口转发
func (f *Forwarder) StopUDPForward(listenAddr, listenPort string) error {
	key := fmt.Sprintf("udp:%s:%s", listenAddr, listenPort)

	f.mu.Lock()
	defer f.mu.Unlock()

	// 检查是否在运行
	conn, exists := f.udpListeners[key]
	if !exists {
		return fmt.Errorf("UDP forward not running on %s:%s", listenAddr, listenPort)
	}

	// 关闭连接
	if err := conn.Close(); err != nil {
		return fmt.Errorf("failed to close UDP connection: %w", err)
	}

	// 删除连接
	delete(f.udpListeners, key)

	log.Printf("Stopped UDP forward: %s:%s", listenAddr, listenPort)
	return nil
}

// IsTCPRunning 检查TCP转发是否运行
func (f *Forwarder) IsTCPRunning(listenAddr, listenPort string) bool {
	key := fmt.Sprintf("tcp:%s:%s", listenAddr, listenPort)

	f.mu.Lock()
	defer f.mu.Unlock()

	_, exists := f.tcpListeners[key]
	return exists
}

// IsUDPRunning 检查UDP转发是否运行
func (f *Forwarder) IsUDPRunning(listenAddr, listenPort string) bool {
	key := fmt.Sprintf("udp:%s:%s", listenAddr, listenPort)

	f.mu.Lock()
	defer f.mu.Unlock()

	_, exists := f.udpListeners[key]
	return exists
}

// handleTCPForward 处理TCP转发
func (f *Forwarder) handleTCPForward(listener net.Listener, targetAddr, targetPort string) {
	target := fmt.Sprintf("%s:%s", targetAddr, targetPort)

	for {
		// 接受新连接
		conn, err := listener.Accept()
		if err != nil {
			// 检查是否是因为关闭监听器导致的错误
			if netErr, ok := err.(net.Error); ok && netErr.Temporary() {
				log.Printf("Temporary error accepting connection: %v", err)
				continue
			}
			log.Printf("Error accepting connection: %v", err)
			break
		}

		// 处理连接
		go func(conn net.Conn) {
			defer conn.Close()

			// 连接到目标服务器
			targetConn, err := net.Dial("tcp", target)
			if err != nil {
				log.Printf("Error connecting to target %s: %v", target, err)
				return
			}
			defer targetConn.Close()

			// 双向转发数据
			forwardData(conn, targetConn)
		}(conn)
	}
}

// handleUDPForward 处理UDP转发
func (f *Forwarder) handleUDPForward(conn *net.UDPConn, targetAddr, targetPort string) {
	// 解析目标地址
	target, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%s", targetAddr, targetPort))
	if err != nil {
		log.Printf("Error resolving target address: %v", err)
		return
	}

	// 缓冲区
	buf := make([]byte, 65535)

	for {
		// 读取UDP数据
		n, addr, err := conn.ReadFromUDP(buf)
		if err != nil {
			log.Printf("Error reading UDP data: %v", err)
			break
		}

		// 转发数据到目标
		_, err = conn.WriteToUDP(buf[:n], target)
		if err != nil {
			log.Printf("Error forwarding UDP data: %v", err)
			continue
		}

		// 从目标读取响应并转发回客户端
		go func(clientAddr *net.UDPAddr) {
			responseBuf := make([]byte, 65535)
			targetConn, err := net.DialUDP("udp", nil, target)
			if err != nil {
				log.Printf("Error connecting to target for response: %v", err)
				return
			}
			defer targetConn.Close()

			// 设置读取超时
			// targetConn.SetReadDeadline(time.Now().Add(5 * time.Second))

			n, err := targetConn.Read(responseBuf)
			if err != nil {
				// 忽略超时错误
				return
			}

			// 转发响应回客户端
			_, err = conn.WriteToUDP(responseBuf[:n], clientAddr)
			if err != nil {
				log.Printf("Error forwarding UDP response: %v", err)
			}
		}(addr)
	}
}

// forwardData 双向转发数据
func forwardData(src, dst net.Conn) {
	var wg sync.WaitGroup

	// 从src读取数据并写入dst
	wg.Add(1)
	go func() {
		defer wg.Done()
		buf := make([]byte, 4096)
		for {
			n, err := src.Read(buf)
			if err != nil {
				break
			}
			if n > 0 {
				if _, err := dst.Write(buf[:n]); err != nil {
					break
				}
			}
		}
	}()

	// 从dst读取数据并写入src
	wg.Add(1)
	go func() {
		defer wg.Done()
		buf := make([]byte, 4096)
		for {
			n, err := dst.Read(buf)
			if err != nil {
				break
			}
			if n > 0 {
				if _, err := src.Write(buf[:n]); err != nil {
					break
				}
			}
		}
	}()

	wg.Wait()
}

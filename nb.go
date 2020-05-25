package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const timeout = 5

func main() {
	//log.SetFlags(log.Ldate | log.Lmicroseconds | log.Lshortfile)
	log.SetFlags(log.Ldate | log.Lmicroseconds)

	printProjectInfo()

	args := os.Args
	argc := len(os.Args)
	if argc <= 2 {
		printUsage()
		os.Exit(0)
	}

	//TODO:support UDP protocol

	/*var logFileError error
	if argc > 5 && args[4] == "-log" {
		logPath := args[5] + "/" + time.Now().Format("2006_01_02_15_04_05") // "2006-01-02 15:04:05"
		logPath += args[1] + "-" + strings.Replace(args[2], ":", "_", -1) + "-" + args[3] + ".log"
		logPath = strings.Replace(logPath, `\`, "/", -1)
		logPath = strings.Replace(logPath, "//", "/", -1)
		logFile, logFileError = os.OpenFile(logPath, os.O_APPEND|os.O_CREATE, 0666)
		if logFileError != nil {
			log.Fatalln("[x ]", "日志文件路径错误.", logFileError.Error())
		}
		log.Println("[√]", "打开测试日志文件成功. 路径:", logPath)
	}*/

	switch args[1] {
	case "-listen":
		if argc < 3 {
			log.Fatalln(`-listen 需要两个参数, 例如 "nb -listen 2019 2020".`)
		}
		port1 := checkPort(args[2])
		port2 := checkPort(args[3])
		log.Println("[√]", "开始监听端口:", port1, "与端口:", port2)
		port2port(port1, port2)
		break
	case "-tran":
		if argc < 3 {
			log.Fatalln(`-tran 需要两个参数, 例如 "nb -tran 2019 192.168.1.2:3389".`)
		}
		localPort := checkPort(args[2])

		conn, err := net.ResolveTCPAddr("tcp", args[3])
		if err != nil {
			log.Fatalln("[x ]", "目标地址错误. 地址应该像这样[ip:port] 或者 [domain:port]. ")
		}
		remoteIP := conn.IP.String()
		remotePort := strconv.Itoa(conn.Port)
		remoteIPAndPort := remoteIP + ":" + remotePort

		log.Println("[√]", "开始传输端口:", localPort, "到地址:", remoteIPAndPort)
		port2host(localPort, remoteIPAndPort)
		break
	case "-slave":
		if argc < 3 {
			log.Fatalln(`-slave 需要两个参数, 例如 "nb -slave 127.0.0.1:3389 8.8.8.8:2019".`)
		}
		var address1, address2 string

		conn, err := net.ResolveTCPAddr("tcp", args[2])
		if err != nil {
			log.Fatalln("[x ]", "地址错误. 地址应该像这样[ip:port] 或者 [domain:port]. ")
		}
		address1 = conn.IP.String() + ":" + strconv.Itoa(conn.Port)

		conn, err = net.ResolveTCPAddr("tcp", args[3])
		if err != nil {
			log.Fatalln("[x ]", "地址错误. 地址应该像这样[ip:port] 或者 [domain:port]. ")
		}
		address2 = conn.IP.String() + ":" + strconv.Itoa(conn.Port)

		log.Println("[√]", "开始连接地址:", address1, "与地址:", address2)
		host2host(address1, address2)
		break
	default:
		printUsage()
	}
}

func printProjectInfo() {
	fmt.Println("+----------------------------------------------------------------+")
	fmt.Println("| NATBypass V2.0                                                 |")
	fmt.Println("| Submit issue at : https://github.com/coflery/NATBypass         |")
	fmt.Println("+----------------------------------------------------------------+")
	fmt.Println()
	// 睡1秒,因为fmt不是线程安全的.如果不这样做,fmt.Print会在log.Print之后采能打印出
	time.Sleep(time.Second)
}
func printUsage() {
	fmt.Println(`用法: "-listen port1 port2" 例: "nb -listen 2019 2020" `)
	fmt.Println(`      "-tran port1 ip:port2" 例: "nb -tran 2019 192.168.1.2:3389" `)
	fmt.Println(`      "-slave ip1:port1 ip2:port2" 例: "nb -slave 127.0.0.1:3389 8.8.8.8:2019" `)
	fmt.Println(`============================================================`)
	fmt.Println(`可选参数: "-log logpath" . 例: "nb -listen 2019 2020 -log d:/nb" `)
	fmt.Println(`日志文件名格式: Y_m_d_H_i_s-agrs1-args2-args3.log`)
	fmt.Println(`============================================================`)
	fmt.Println(`如果你需要更多帮助,请阅读"README.md". `)
}

func checkPort(port string) string {
	PortNum, err := strconv.Atoi(port)
	if err != nil {
		log.Fatalln("[x ]", "端口号应该是数字")
	}
	if PortNum < 1 || PortNum > 65535 {
		log.Fatalln("[x ]", "端口号应该在1到65535之间")
	}
	return port
}

func port2port(port1 string, port2 string) {
	listen1 := startServer("0.0.0.0:" + port1)
	listen2 := startServer("0.0.0.0:" + port2)
	log.Println("[√]", "监听端口:", port1, "与", port2, "成功. 等待连接...")
	for {
		conn1 := accept(listen1)
		conn2 := accept(listen2)
		if conn1 == nil || conn2 == nil {
			log.Println("[x ]", "接受连接失败. 在 ", timeout, " 秒后重试. ")
			time.Sleep(timeout * time.Second)
			continue
		}
		forward(conn1, conn2)
	}
}

func port2host(allowPort string, targetAddress string) {
	server := startServer("0.0.0.0:" + allowPort)
	for {
		conn := accept(server)
		if conn == nil {
			continue
		}
		//println(targetAddress)
		go func(targetAddress string) {
			log.Println("[+]", "开始连接主机:["+targetAddress+"]")
			target, err := net.Dial("tcp", targetAddress)
			if err != nil {
				// temporarily unavailable, don't use fatal.
				log.Println("[x ]", "连接目标地址 ["+targetAddress+"] 失败. 在 ", timeout, "秒后重试. ")
				conn.Close()
				log.Println("[←]", "关闭本地连接:["+conn.LocalAddr().String()+"] 和远程:["+conn.RemoteAddr().String()+"]")
				time.Sleep(timeout * time.Second)
				return
			}
			log.Println("[→]", "连接目标地址 ["+targetAddress+"] 成功.")
			forward(target, conn)
		}(targetAddress)
	}
}

func host2host(address1, address2 string) {
	for {
		log.Println("[+]", "尝试连接主机:["+address1+"] 与 ["+address2+"]")
		var host1, host2 net.Conn
		var err error
		for {
			host1, err = net.Dial("tcp", address1)
			if err == nil {
				log.Println("[→]", "连接 ["+address1+"] 成功.")
				break
			} else {
				log.Println("[x ]", "连接目标地址 ["+address1+"] 失败. 在 ", timeout, " 秒后重试. ")
				time.Sleep(timeout * time.Second)
			}
		}
		for {
			host2, err = net.Dial("tcp", address2)
			if err == nil {
				log.Println("[→]", "连接 ["+address2+"] 成功.")
				break
			} else {
				log.Println("[x ]", "连接目标地址 ["+address2+"] 失败. 在 ", timeout, " 秒后重试. ")
				time.Sleep(timeout * time.Second)
			}
		}
		forward(host1, host2)
	}
}

func startServer(address string) net.Listener {
	log.Println("[+]", "尝试开始服务于端口:["+address+"]")
	server, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalln("[x ]", "监听地址 ["+address+"] 失败.")
	}
	log.Println("[√]", "开始监听地址:["+address+"]")
	return server
	/*defer server.Close()

	for {
		conn, err := server.Accept()
		log.Println("接受一个新连接失败. 远端地址:[" + conn.RemoteAddr().String() +
			"], 本地地址:[" + conn.LocalAddr().String() + "]")
		if err != nil {
			log.Println("接受一个新连接失败.", err.Error())
			continue
		}
		//go recvConnMsg(conn)
	}*/
}

func accept(listener net.Listener) net.Conn {
	conn, err := listener.Accept()
	if err != nil {
		log.Println("[x ]", "接受连接 ["+conn.RemoteAddr().String()+"] 失败.", err.Error())
		return nil
	}
	log.Println("[√]", "收到一个新连接. 远端地址:["+conn.RemoteAddr().String()+"], 本地地址:["+conn.LocalAddr().String()+"]")
	return conn
}

func forward(conn1 net.Conn, conn2 net.Conn) {
	log.Printf("[+] 开始传输. [%s],[%s] <-> [%s],[%s] \n", conn1.LocalAddr().String(), conn1.RemoteAddr().String(), conn2.LocalAddr().String(), conn2.RemoteAddr().String())
	var wg sync.WaitGroup
	// wait tow goroutines
	wg.Add(2)
	go connCopy(conn1, conn2, &wg)
	go connCopy(conn2, conn1, &wg)
	//blocking when the wg is locked
	wg.Wait()
}

func connCopy(conn1 net.Conn, conn2 net.Conn, wg *sync.WaitGroup) {
	//TODO:log, record the data from conn1 and conn2.
	logFile := openLog(conn1.LocalAddr().String(), conn1.RemoteAddr().String(), conn2.LocalAddr().String(), conn2.RemoteAddr().String())
	if logFile != nil {
		w := io.MultiWriter(conn1, logFile)
		io.Copy(w, conn2)
	} else {
		io.Copy(conn1, conn2)
	}
	conn1.Close()
	log.Println("[←]", "关闭本地连接:["+conn1.LocalAddr().String()+"] 和远程:["+conn1.RemoteAddr().String()+"]")
	//conn2.Close()
	//log.Println("[←]", "关闭本地连接:["+conn2.LocalAddr().String()+"] 和远程:["+conn2.RemoteAddr().String()+"]")
	wg.Done()
}
func openLog(address1, address2, address3, address4 string) *os.File {
	args := os.Args
	argc := len(os.Args)
	var logFileError error
	var logFile *os.File
	if argc > 5 && args[4] == "-log" {
		address1 = strings.Replace(address1, ":", "_", -1)
		address2 = strings.Replace(address2, ":", "_", -1)
		address3 = strings.Replace(address3, ":", "_", -1)
		address4 = strings.Replace(address4, ":", "_", -1)
		timeStr := time.Now().Format("yyyy_mm_dd_HH_mm_ss") // "2000-12-31 23:59:59"
		logPath := args[5] + "/" + timeStr + args[1] + "-" + address1 + "_" + address2 + "-" + address3 + "_" + address4 + ".log"
		logPath = strings.Replace(logPath, `\`, "/", -1)
		logPath = strings.Replace(logPath, "//", "/", -1)
		logFile, logFileError = os.OpenFile(logPath, os.O_APPEND|os.O_CREATE, 0666)
		if logFileError != nil {
			log.Fatalln("[x ]", "日志文件路径错误.", logFileError.Error())
		}
		log.Println("[√]", "打开测试日志文件成功. 路径:", logPath)
	}
	return logFile
}

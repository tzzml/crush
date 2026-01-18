package cmd

import (
	"os"
	"strconv"

	internalcmd "github.com/charmbracelet/crush/internal/cmd"
)

// Execute 是命令入口点，检查是否需要启动 API 服务器
func Execute() {
	// 检查 --server 标志
	hasServer := false
	port := 8080
	host := "localhost"

	for i, arg := range os.Args {
		if arg == "--server" || arg == "-s" {
			hasServer = true
		}
		if (arg == "--port" || arg == "-p") && i+1 < len(os.Args) {
			if p, err := strconv.Atoi(os.Args[i+1]); err == nil {
				port = p
			}
		}
		if arg == "--host" && i+1 < len(os.Args) {
			host = os.Args[i+1]
		}
	}

	if hasServer {
		// 启动 API 服务器
		StartServer(port, host)
		return
	}

	// 否则，调用原有的 internal/cmd.Execute()
	internalcmd.Execute()
}

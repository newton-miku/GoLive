// Package log 提供分级日志输出控制。
package log

import (
	"fmt"
	"os"
)

// DebugMode 调试模式，默认关闭。设为 true 时输出调试日志。
// 生产环境应保持关闭，避免泄露数据库等隐私信息。
var DebugMode = os.Getenv("GOLIVE_DEBUG") != ""

func init() {
	if DebugMode {
		fmt.Println("[日志] 调试模式已启用（包含敏感数据，请勿在线上环境使用）")
	}
}

// Debugf 仅在调试模式下输出。
func Debugf(format string, v ...any) {
	if DebugMode {
		fmt.Printf("[调试] "+format, v...)
	}
}

// Debug 仅在调试模式下输出。
func Debug(v ...any) {
	if DebugMode {
		fmt.Println("[调试]", fmt.Sprint(v...))
	}
}

// Infof 始终输出信息级别日志。
func Infof(format string, v ...any) {
	fmt.Printf(format, v...)
}

// Info 始终输出信息级别日志。
func Info(v ...any) {
	fmt.Println(fmt.Sprint(v...))
}
